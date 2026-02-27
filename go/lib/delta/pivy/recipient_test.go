//go:build test && debug

package pivy

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"
)

func TestRecipientWrapProducesValidStanza(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	if _, err := rand.Read(fileKey); err != nil {
		t.Fatal(err)
	}

	stanzas, err := recipient.Wrap(fileKey)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	if len(stanzas) != 1 {
		t.Fatalf("expected 1 stanza, got %d", len(stanzas))
	}

	s := stanzas[0]

	if s.Type != StanzaTypePivyEcdhP256 {
		t.Fatalf("stanza type: got %q, want %q", s.Type, StanzaTypePivyEcdhP256)
	}

	if len(s.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(s.Args))
	}

	// Body is wrapped file key: 16 bytes key + 16 bytes poly1305 tag = 32
	if len(s.Body) != 32 {
		t.Fatalf("body length: got %d, want 32", len(s.Body))
	}
}

func TestWrapUnwrapRoundTrip(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	if _, err := rand.Read(fileKey); err != nil {
		t.Fatal(err)
	}

	stanzas, err := recipient.Wrap(fileKey)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	identity := &Identity{
		ecdhFunc: softwareECDH(privKey),
	}

	decryptedKey, err := identity.Unwrap(stanzas)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}

	if len(decryptedKey) != len(fileKey) {
		t.Fatalf("key length: got %d, want %d", len(decryptedKey), len(fileKey))
	}

	for i := range fileKey {
		if decryptedKey[i] != fileKey[i] {
			t.Fatalf("key mismatch at byte %d", i)
		}
	}
}

func TestUnwrapWrongKeyFails(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	wrongKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	recipient := &Recipient{Pubkey: privKey.PublicKey()}

	fileKey := make([]byte, 16)
	if _, err := rand.Read(fileKey); err != nil {
		t.Fatal(err)
	}

	stanzas, err := recipient.Wrap(fileKey)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	identity := &Identity{
		ecdhFunc: softwareECDH(wrongKey),
	}

	_, err = identity.Unwrap(stanzas)
	if err == nil {
		t.Fatal("expected error unwrapping with wrong key")
	}
}

func TestResolveAgentSocketPathFromEnv(t *testing.T) {
	t.Setenv("PIVY_AUTH_SOCK", "/tmp/test-pivy-agent.sock")

	path, err := ResolveAgentSocketPath()
	if err != nil {
		t.Fatal(err)
	}

	if path != "/tmp/test-pivy-agent.sock" {
		t.Fatalf("got %q, want /tmp/test-pivy-agent.sock", path)
	}
}

func TestResolveAgentSocketPathUnset(t *testing.T) {
	t.Setenv("PIVY_AUTH_SOCK", "")

	_, err := ResolveAgentSocketPath()
	if err == nil {
		t.Fatal("expected error when PIVY_AUTH_SOCK is unset")
	}
}

func TestNewAgentIdentity(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// NewAgentIdentity constructs an Identity that would call the agent.
	// We can't test the actual agent call without pivy-agent running,
	// but we verify the constructor works.
	identity := NewAgentIdentity(privKey.PublicKey())
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
}
