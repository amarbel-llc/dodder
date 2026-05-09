package remote_http

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/http_statuses"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

// These tests cover the response-body verification path of
// RoundTripperBufioWrappedSigner: the verifyingBodyReader. The full
// sign-on-server / verify-on-client round trip cannot be unit-tested
// today because Server.addSignatureIfNecessary reaches into a concrete
// *local_working_copy.Repo for keys (no interface). That refactor is
// tracked in #153 and BATS exercises the round trip in #150.

type signerTestKeys struct {
	priv markl.Id
	pub  markl.Id
}

func makeSignerTestKeys(t1 *testing.T) signerTestKeys {
	t1.Helper()

	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t1.Fatalf("ed25519.GenerateKey: %v", err)
	}
	edPub := edPriv.Public().(ed25519.PublicKey)

	var sec markl.Id
	if err := sec.SetPurposeId(markl.PurposeRepoPrivateKeyV1); err != nil {
		t1.Fatalf("sec.SetPurposeId: %v", err)
	}
	if err := sec.SetMarklId(markl.FormatIdEd25519Sec, []byte(edPriv)); err != nil {
		t1.Fatalf("sec.SetMarklId: %v", err)
	}

	var pub markl.Id
	if err := pub.SetPurposeId(markl.PurposeRepoPubKeyV1); err != nil {
		t1.Fatalf("pub.SetPurposeId: %v", err)
	}
	if err := pub.SetMarklId(markl.FormatIdEd25519Pub, []byte(edPub)); err != nil {
		t1.Fatalf("pub.SetMarklId: %v", err)
	}

	return signerTestKeys{priv: sec, pub: pub}
}

// signBodyForTest computes the SHA-256 digest of body and signs it with
// priv using PurposeRequestRepoSigV1, returning the encoded signature.
func signBodyForTest(t1 *testing.T, priv markl.Id, body []byte) string {
	t1.Helper()

	hash, repoolHash := markl.FormatHashSha256.GetHash()
	defer repoolHash()

	if _, err := hash.Write(body); err != nil {
		t1.Fatalf("hash.Write: %v", err)
	}

	digest, repoolDigest := hash.GetMarklId()
	defer repoolDigest()

	var sig markl.Id
	if err := priv.Sign(digest, &sig, markl.PurposeRequestRepoSigV1); err != nil {
		t1.Fatalf("priv.Sign: %v", err)
	}

	return sig.String()
}

func makeReaderForBody(
	body []byte,
	trailer http.Header,
	pub markl.Id,
) (*verifyingBodyReader, func()) {
	hash, repoolHash := markl.FormatHashSha256.GetHash()

	resp := &http.Response{
		Body:    io.NopCloser(bytes.NewReader(body)),
		Trailer: trailer,
	}

	reader := &verifyingBodyReader{
		body:       resp.Body,
		response:   resp,
		hash:       hash,
		repoolHash: repoolHash,
		pubkey:     pub,
	}

	cleanup := func() {
		_ = reader.Close()
	}

	return reader, cleanup
}

func TestVerifyingBodyReaderSuccess(t1 *testing.T) {
	t := ui.MakeT(t1)

	keys := makeSignerTestKeys(t.T)

	body := []byte("hello, signed world")
	sig := signBodyForTest(t.T, keys.priv, body)

	trailer := http.Header{headerRepoSig: []string{sig}}

	reader, cleanup := makeReaderForBody(body, trailer, keys.pub)
	defer cleanup()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll: unexpected err: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body mismatch: got %q, want %q", got, body)
	}
}

func TestVerifyingBodyReaderEmptyBodySuccess(t1 *testing.T) {
	t := ui.MakeT(t1)

	keys := makeSignerTestKeys(t.T)

	body := []byte{}
	sig := signBodyForTest(t.T, keys.priv, body)

	trailer := http.Header{headerRepoSig: []string{sig}}

	reader, cleanup := makeReaderForBody(body, trailer, keys.pub)
	defer cleanup()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll on empty body: unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty body, got %d bytes: %q", len(got), got)
	}
}

func TestVerifyingBodyReaderMissingTrailerFails(t1 *testing.T) {
	t := ui.MakeT(t1)

	keys := makeSignerTestKeys(t.T)

	body := []byte("body without a trailer")

	reader, cleanup := makeReaderForBody(body, http.Header{}, keys.pub)
	defer cleanup()

	_, err := io.ReadAll(reader)
	if err == nil {
		t.Fatalf("expected error for missing trailer, got nil")
	}
	if !strings.Contains(err.Error(), "trailer") {
		t.Fatalf("expected error to mention 'trailer', got: %v", err)
	}
}

func TestVerifyingBodyReaderMalformedSignatureFails(t1 *testing.T) {
	t := ui.MakeT(t1)

	keys := makeSignerTestKeys(t.T)

	body := []byte("body with mangled trailer")
	trailer := http.Header{headerRepoSig: []string{"not-a-valid-markl-id"}}

	reader, cleanup := makeReaderForBody(body, trailer, keys.pub)
	defer cleanup()

	_, err := io.ReadAll(reader)
	if err == nil {
		t.Fatalf("expected error for malformed signature, got nil")
	}
}

func TestVerifyingBodyReaderTamperedBodyFailsVerification(t1 *testing.T) {
	t := ui.MakeT(t1)

	keys := makeSignerTestKeys(t.T)

	originalBody := []byte("the original body")
	sig := signBodyForTest(t.T, keys.priv, originalBody)

	trailer := http.Header{headerRepoSig: []string{sig}}

	tamperedBody := []byte("the tampered body")

	reader, cleanup := makeReaderForBody(tamperedBody, trailer, keys.pub)
	defer cleanup()

	_, err := io.ReadAll(reader)
	if err == nil {
		t.Fatalf("expected verification error for tampered body, got nil")
	}
	if !errors.IsHTTPError(err, http_statuses.Code422UnprocessableEntity) {
		t.Fatalf("expected 422 Unprocessable Entity (verification failure), got: %v", err)
	}
}

func TestVerifyingBodyReaderWrongPubkeyFailsVerification(t1 *testing.T) {
	t := ui.MakeT(t1)

	signerKeys := makeSignerTestKeys(t.T)
	verifierKeys := makeSignerTestKeys(t.T)

	body := []byte("signed by one key, verified with another")
	sig := signBodyForTest(t.T, signerKeys.priv, body)

	trailer := http.Header{headerRepoSig: []string{sig}}

	reader, cleanup := makeReaderForBody(body, trailer, verifierKeys.pub)
	defer cleanup()

	_, err := io.ReadAll(reader)
	if err == nil {
		t.Fatalf("expected verification error for wrong pubkey, got nil")
	}
	if !errors.IsHTTPError(err, http_statuses.Code422UnprocessableEntity) {
		t.Fatalf("expected 422 Unprocessable Entity (verification failure), got: %v", err)
	}
}
