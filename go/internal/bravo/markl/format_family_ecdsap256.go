package markl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"io"
	"math/big"
	"net"
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/pivy"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var ErrEcdsaP256SSHAgentNotConnected, IsErrEcdsaP256SSHAgentNotConnected = errors.MakeTypedSentinel[pkgErrDisamb](
	"ecdsa P256 SSH agent signer not connected",
)

func EcdsaP256Verify(pub, message, sig domain_interfaces.MarklId) (err error) {
	compressed := pub.GetBytes()

	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), compressed)
	if x == nil {
		return errors.Errorf("invalid compressed P-256 point")
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	sigBytes := sig.GetBytes()
	if len(sigBytes) != 64 {
		return errors.Errorf("invalid ECDSA P256 signature length: %d", len(sigBytes))
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])

	digest := sha256.Sum256(message.GetBytes())

	if !ecdsa.Verify(pubKey, digest[:], r, s) {
		return errors.Err422UnprocessableEntity.Errorf(
			"invalid ECDSA P256 signature: %q",
			sig.StringWithFormat(),
		)
	}

	return nil
}

func parseSSHEcdsaSignatureBlob(blob []byte) ([]byte, error) {
	var parsed struct {
		R *big.Int
		S *big.Int
	}

	if err := ssh.Unmarshal(blob, &parsed); err != nil {
		return nil, errors.Wrapf(err, "parsing SSH ECDSA signature blob")
	}

	fixed := make([]byte, 64)

	rBytes := parsed.R.Bytes()
	sBytes := parsed.S.Bytes()

	if len(rBytes) > 32 || len(sBytes) > 32 {
		return nil, errors.Errorf(
			"ECDSA signature component too large: r=%d s=%d",
			len(rBytes),
			len(sBytes),
		)
	}

	copy(fixed[32-len(rBytes):32], rBytes)
	copy(fixed[64-len(sBytes):64], sBytes)

	return fixed, nil
}

type ecdsaP256AgentSigner struct {
	agentClient agent.Agent
	key         *agent.Key
}

func (s *ecdsaP256AgentSigner) PublicKey() ssh.PublicKey {
	return s.key
}

func (s *ecdsaP256AgentSigner) Sign(
	rand io.Reader,
	data []byte,
) (*ssh.Signature, error) {
	return s.agentClient.Sign(s.key, data)
}

func ConnectEcdsaP256AgentSigner(
	compressed []byte,
) (ssh.Signer, io.Closer, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, nil, errors.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to connect to SSH agent")
	}

	agentClient := agent.NewClient(conn)

	keys, err := agentClient.List()
	if err != nil {
		conn.Close()
		return nil, nil, errors.Wrapf(err, "failed to list SSH agent keys")
	}

	for _, key := range keys {
		if key.Type() != "ecdsa-sha2-nistp256" {
			continue
		}

		parsed, err := parseSSHPublicKey(key)
		if err != nil {
			continue
		}

		ecdsaPub, ok := parsed.CryptoPublicKey().(*ecdsa.PublicKey)
		if !ok {
			continue
		}

		ecdhPub, err := ecdsaPub.ECDH()
		if err != nil {
			continue
		}

		signerCompressed := pivy.CompressP256Point(ecdhPub)

		if len(signerCompressed) != len(compressed) {
			continue
		}

		match := true
		for i := range compressed {
			if signerCompressed[i] != compressed[i] {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		return &ecdsaP256AgentSigner{
			agentClient: agentClient,
			key:         key,
		}, conn, nil
	}

	conn.Close()

	return nil, nil, errors.Errorf(
		"no matching ECDSA P256 key found in SSH agent",
	)
}

var ecdsaP256FormatOnce sync.Once

func RegisterEcdsaP256SSHFormat(signer ssh.Signer) {
	ecdsaP256FormatOnce.Do(func() {
		formats[FormatIdEcdsaP256SSH] = FormatSec{
			Id:          FormatIdEcdsaP256SSH,
			Size:        33,
			PubFormatId: FormatIdEcdsaP256Pub,
			GetPublicKey: func(id domain_interfaces.MarklId) ([]byte, error) {
				return id.GetBytes(), nil
			},
			SigFormatId: FormatIdEcdsaP256Sig,
			Sign: func(
				sec, mes domain_interfaces.MarklId,
				readerRand io.Reader,
			) ([]byte, error) {
				sshSig, err := signer.Sign(readerRand, mes.GetBytes())
				if err != nil {
					return nil, errors.Wrap(err)
				}

				return parseSSHEcdsaSignatureBlob(sshSig.Blob)
			},
		}
	})
}

func resetEcdsaP256SSHFormatForTesting() {
	makeStubEcdsaP256SSHFormat()
	ecdsaP256FormatOnce = sync.Once{}
}

func makeStubEcdsaP256SSHFormat() {
	formats[FormatIdEcdsaP256SSH] = FormatSec{
		Id:          FormatIdEcdsaP256SSH,
		Size:        33,
		PubFormatId: FormatIdEcdsaP256Pub,
		GetPublicKey: func(_ domain_interfaces.MarklId) ([]byte, error) {
			return nil, errors.Wrap(ErrEcdsaP256SSHAgentNotConnected)
		},
		SigFormatId: FormatIdEcdsaP256Sig,
		Sign: func(
			_, _ domain_interfaces.MarklId,
			_ io.Reader,
		) ([]byte, error) {
			return nil, errors.Wrap(ErrEcdsaP256SSHAgentNotConnected)
		},
	}
}
