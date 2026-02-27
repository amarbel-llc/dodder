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
