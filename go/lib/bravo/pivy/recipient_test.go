//go:build test && debug

package pivy

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestRecipientWrapProducesValidStanza(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	_, err = rand.Read(fileKey)
	t.AssertNoError(err)

	stanzas, err := recipient.Wrap(fileKey)
	t.AssertNoError(err)

	t.AssertLen(1, stanzas, "stanzas")

	s := stanzas[0]

	t.AssertEqual(StanzaTypePivyEcdhP256, s.Type)

	t.AssertLen(2, s.Args, "stanza args")

	// Body is wrapped file key: 16 bytes key + 16 bytes poly1305 tag = 32
	t.AssertLen(32, s.Body, "stanza body")
}

func TestWrapUnwrapRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	_, err = rand.Read(fileKey)
	t.AssertNoError(err)

	stanzas, err := recipient.Wrap(fileKey)
	t.AssertNoError(err)

	identity := &Identity{
		ecdhFunc: softwareECDH(privKey),
	}

	decryptedKey, err := identity.Unwrap(stanzas)
	t.AssertNoError(err)

	t.AssertEqual(fileKey, decryptedKey)
}

func TestUnwrapWrongKeyFails(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	wrongKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	_, err = rand.Read(fileKey)
	t.AssertNoError(err)

	stanzas, err := recipient.Wrap(fileKey)
	t.AssertNoError(err)

	identity := &Identity{
		ecdhFunc: softwareECDH(wrongKey),
	}

	_, err = identity.Unwrap(stanzas)
	t.AssertError(err)
}

func TestResolveAgentSocketPathFromEnv(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Setenv("PIVY_AUTH_SOCK", "/tmp/test-pivy-agent.sock")

	path, err := ResolveAgentSocketPath()
	t.AssertNoError(err)

	t.AssertEqualStrings("/tmp/test-pivy-agent.sock", path)
}

func TestResolveAgentSocketPathUnset(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Setenv("PIVY_AUTH_SOCK", "")

	_, err := ResolveAgentSocketPath()
	t.AssertError(err)
}

func TestNewAgentIdentity(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	// NewAgentIdentity constructs an Identity that would call the agent.
	// We can't test the actual agent call without pivy-agent running,
	// but we verify the constructor works.
	identity := NewAgentIdentity(privKey.PublicKey())
	t.AssertNotNil(identity, "agent identity")
}
