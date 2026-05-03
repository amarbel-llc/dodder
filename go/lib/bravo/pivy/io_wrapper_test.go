//go:build test && debug

package pivy

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"io"
	"testing"
)

func TestIOWrapperRoundTrip(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	wrapper := &IOWrapper{
		RecipientPubkey: privKey.PublicKey(),
		DecryptECDH:     softwareECDH(privKey),
	}

	plaintext := []byte("hello pivy encrypted world")

	// Encrypt
	var cipherBuf bytes.Buffer
	w, err := wrapper.WrapWriter(&cipherBuf)
	if err != nil {
		t.Fatalf("WrapWriter: %v", err)
	}

	if _, err := w.Write(plaintext); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify ciphertext is different from plaintext
	if bytes.Equal(cipherBuf.Bytes(), plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	// Decrypt
	r, err := wrapper.WrapReader(bytes.NewReader(cipherBuf.Bytes()))
	if err != nil {
		t.Fatalf("WrapReader: %v", err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close reader: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted %q != plaintext %q", decrypted, plaintext)
	}
}

func TestIOWrapperStreamingLargePayload(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	wrapper := &IOWrapper{
		RecipientPubkey: privKey.PublicKey(),
		DecryptECDH:     softwareECDH(privKey),
	}

	// 256 KiB payload — spans multiple age STREAM chunks (64 KiB each)
	plaintext := make([]byte, 256*1024)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatal(err)
	}

	var cipherBuf bytes.Buffer
	w, err := wrapper.WrapWriter(&cipherBuf)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Write(plaintext); err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := wrapper.WrapReader(bytes.NewReader(cipherBuf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("large payload round-trip mismatch")
	}
}
