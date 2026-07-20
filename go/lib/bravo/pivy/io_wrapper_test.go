//go:build test && debug

package pivy

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"io"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestIOWrapperRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	wrapper := &IOWrapper{
		RecipientPubkey: privKey.PublicKey(),
		DecryptECDH:     softwareECDH(privKey),
	}

	plaintext := []byte("hello pivy encrypted world")

	// Encrypt
	var cipherBuf bytes.Buffer
	w, err := wrapper.WrapWriter(&cipherBuf)
	t.AssertNoError(err)

	_, err = w.Write(plaintext)
	t.AssertNoError(err)

	t.AssertNoError(w.Close())

	// Verify ciphertext is different from plaintext
	t.AssertFalse(bytes.Equal(cipherBuf.Bytes(), plaintext), "ciphertext equals plaintext")

	// Decrypt
	r, err := wrapper.WrapReader(bytes.NewReader(cipherBuf.Bytes()))
	t.AssertNoError(err)

	decrypted, err := io.ReadAll(r)
	t.AssertNoError(err)

	t.AssertNoError(r.Close())

	t.AssertEqual(plaintext, decrypted)
}

func TestIOWrapperStreamingLargePayload(t1 *testing.T) {
	t := ui.MakeT(t1)
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	t.AssertNoError(err)

	wrapper := &IOWrapper{
		RecipientPubkey: privKey.PublicKey(),
		DecryptECDH:     softwareECDH(privKey),
	}

	// 256 KiB payload — spans multiple age STREAM chunks (64 KiB each)
	plaintext := make([]byte, 256*1024)
	_, err = rand.Read(plaintext)
	t.AssertNoError(err)

	var cipherBuf bytes.Buffer
	w, err := wrapper.WrapWriter(&cipherBuf)
	t.AssertNoError(err)

	_, err = w.Write(plaintext)
	t.AssertNoError(err)

	t.AssertNoError(w.Close())

	r, err := wrapper.WrapReader(bytes.NewReader(cipherBuf.Bytes()))
	t.AssertNoError(err)

	decrypted, err := io.ReadAll(r)
	t.AssertNoError(err)

	t.AssertNoError(r.Close())

	t.AssertEqual(plaintext, decrypted)
}
