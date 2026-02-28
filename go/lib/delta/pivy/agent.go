package pivy

import (
	"crypto/ecdh"
	"encoding/binary"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func ResolveAgentSocketPath() (string, error) {
	path := os.Getenv("PIVY_AUTH_SOCK")
	if path == "" {
		return "", errors.Errorf("PIVY_AUTH_SOCK not set")
	}
	return path, nil
}

// NewAgentIdentity creates an Identity that performs ECDH via pivy-agent's
// ecdh@joyent.com extension.
func NewAgentIdentity(pubkey *ecdh.PublicKey) *Identity {
	return &Identity{
		ecdhFunc: agentECDH(pubkey),
	}
}

// AgentECDHFunc returns an ECDHFunc that calls pivy-agent at the given socket.
func AgentECDHFunc(socketPath string, recipientPubkey *ecdh.PublicKey) ECDHFunc {
	return func(ephPubBytes []byte) ([]byte, error) {
		return callAgentECDH(socketPath, recipientPubkey, ephPubBytes)
	}
}

func agentECDH(recipientPubkey *ecdh.PublicKey) ECDHFunc {
	return func(ephPubBytes []byte) ([]byte, error) {
		socketPath, err := ResolveAgentSocketPath()
		if err != nil {
			return nil, err
		}

		return callAgentECDH(socketPath, recipientPubkey, ephPubBytes)
	}
}

func callAgentECDH(
	socketPath string,
	recipientPubkey *ecdh.PublicKey,
	ephemeralPubkey []byte,
) ([]byte, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, errors.Wrapf(err, "connecting to pivy-agent at %s", socketPath)
	}
	defer conn.Close()

	client := agent.NewClient(conn)

	extClient, ok := client.(agent.ExtendedAgent)
	if !ok {
		return nil, errors.Errorf("SSH agent client does not support extensions")
	}

	// The ephemeral pubkey from the age stanza is in compressed form (33 bytes).
	// Decompress to get an ecdh.PublicKey for SSH wire format conversion.
	ephPub, err := DecompressP256Point(ephemeralPubkey)
	if err != nil {
		return nil, errors.Wrapf(err, "decompressing ephemeral pubkey for agent")
	}

	// Build the extension request payload.
	// The ecdh@joyent.com extension parses both keys with sshkey_froms(),
	// so both must be in SSH wire format:
	//   string(ssh_key) string(ssh_key) uint32(flags)
	recipientSSHKey, err := pubkeyToSSHWireFormat(recipientPubkey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	ephemeralSSHKey, err := pubkeyToSSHWireFormat(ephPub)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	innerPayload := ssh.Marshal(struct {
		RecipientKey []byte
		EphemeralKey []byte
		Flags        uint32
	}{
		RecipientKey: recipientSSHKey,
		EphemeralKey: ephemeralSSHKey,
		Flags:        0,
	})

	// pivy-agent's extension dispatch uses eh_string=B_TRUE for ecdh@joyent.com,
	// which means it calls sshbuf_froms() to unwrap a length-prefixed string
	// before passing the buffer to the handler. Go's agent.Extension() sends
	// contents as raw bytes (ssh:"rest"), so we must add the outer string
	// wrapper that pivy-agent expects.
	payload := make([]byte, 4+len(innerPayload))
	binary.BigEndian.PutUint32(payload[:4], uint32(len(innerPayload)))
	copy(payload[4:], innerPayload)

	response, err := extClient.Extension("ecdh@joyent.com", payload)
	if err != nil {
		return nil, errors.Wrapf(err, "ecdh@joyent.com extension call")
	}

	// The response includes the SSH_AGENT_SUCCESS type byte followed by the
	// shared secret as a length-prefixed string: [u8 type] [u32 len] [secret]
	secret, err := parseECDHResponse(response)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func parseECDHResponse(response []byte) ([]byte, error) {
	if len(response) < 5 {
		return nil, errors.Errorf(
			"ecdh response too short: %d bytes",
			len(response),
		)
	}

	// First byte is SSH_AGENT_SUCCESS (0x06), skip it
	rest := response[1:]

	// Remaining bytes are the shared secret as a length-prefixed SSH string
	secretLen := binary.BigEndian.Uint32(rest[:4])

	if uint32(len(rest)-4) < secretLen {
		return nil, errors.Errorf(
			"ecdh response secret truncated: expected %d bytes, got %d",
			secretLen,
			len(rest)-4,
		)
	}

	return rest[4 : 4+secretLen], nil
}

func pubkeyToSSHWireFormat(pub *ecdh.PublicKey) ([]byte, error) {
	// SSH wire format for ECDSA keys:
	//   string("ecdsa-sha2-nistp256") + string("nistp256") + string(uncompressed_point)
	uncompressed := pub.Bytes() // 0x04 || x || y

	key := struct {
		KeyType string
		Curve   string
		Point   []byte
	}{
		KeyType: "ecdsa-sha2-nistp256",
		Curve:   "nistp256",
		Point:   uncompressed,
	}

	return ssh.Marshal(key), nil
}
