package pivy

import (
	"crypto/ecdh"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
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

	// Build the extension request payload.
	// The ecdh@joyent.com extension expects:
	//   [recipient_pubkey as ssh wire format] [ephemeral_pubkey bytes] [flags uint32]
	recipientSSHKey, err := pubkeyToSSHWireFormat(recipientPubkey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	payload := ssh.Marshal(struct {
		RecipientKey []byte
		EphemeralKey []byte
		Flags        uint32
	}{
		RecipientKey: recipientSSHKey,
		EphemeralKey: ephemeralPubkey,
		Flags:        0,
	})

	response, err := extClient.Extension("ecdh@joyent.com", payload)
	if err != nil {
		return nil, errors.Wrapf(err, "ecdh@joyent.com extension call")
	}

	if len(response) == 0 {
		return nil, errors.Errorf("empty response from ecdh@joyent.com")
	}

	return response, nil
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
