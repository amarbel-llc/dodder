# PIV ECDH Encryption Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add PIV ECDH encryption support to madder blob stores via pivy-agent, using age's STREAM format for chunked streaming payloads.

**Architecture:** `delta/pivy/` implements `age.Recipient` (local ECDH wrap) and `age.Identity` (agent-backed unwrap), delegating to `age.Encrypt`/`age.Decrypt` for streaming. Format registration in `echo/markl/` wires it into the config system. A discovery command lists available keys from pivy-agent.

**Tech Stack:** Go stdlib (`crypto/ecdh`, `crypto/sha512`, `crypto/elliptic`), `golang.org/x/crypto/ssh/agent`, `golang.org/x/crypto/chacha20poly1305`, `filippo.io/age`

**Design doc:** `docs/plans/2026-02-27-pivy-ecdh-encryption-design.md`

---

### Task 1: Recipient — Wrap (Key Encapsulation)

**Files:**
- Create: `go/src/delta/pivy/recipient.go`
- Create: `go/src/delta/pivy/recipient_test.go`

The `Recipient` wraps a file key into a `pivy-ecdh-p256` age stanza using local
ECDH. No agent needed.

**Step 1: Write the failing test**

Create `go/src/delta/pivy/recipient_test.go`:

```go
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
	// age file keys are 16 bytes, wrapped = 16 + 16 = 32
	if len(s.Body) != 32 {
		t.Fatalf("body length: got %d, want 32", len(s.Body))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestRecipientWrapProducesValidStanza -v`
Expected: FAIL — package does not exist

**Step 3: Write minimal implementation**

Create `go/src/delta/pivy/recipient.go`:

```go
package pivy

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"filippo.io/age"
	"golang.org/x/crypto/chacha20poly1305"
)

const StanzaTypePivyEcdhP256 = "pivy-ecdh-p256"

// TODO: add pivy-ecdh-p384 and pivy-ecdh-p521 stanza types

type Recipient struct {
	Pubkey *ecdh.PublicKey
}

func (r *Recipient) Wrap(fileKey []byte) ([]*age.Stanza, error) {
	ephemeralPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	sharedSecret, err := ephemeralPriv.ECDH(r.Pubkey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.Wrap(err)
	}

	wrappingKey := deriveWrappingKey(sharedSecret, nonce)

	// Zeroize shared secret
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}

	aead, err := chacha20poly1305.New(wrappingKey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// Zeroize wrapping key
	for i := range wrappingKey {
		wrappingKey[i] = 0
	}

	wrappedKey := aead.Seal(nil, make([]byte, chacha20poly1305.NonceSize), fileKey, nil)

	ephPubBytes := compressP256Point(ephemeralPriv.PublicKey())

	stanza := &age.Stanza{
		Type: StanzaTypePivyEcdhP256,
		Args: []string{
			base64.RawStdEncoding.EncodeToString(ephPubBytes),
			base64.RawStdEncoding.EncodeToString(nonce),
		},
		Body: wrappedKey,
	}

	return []*age.Stanza{stanza}, nil
}

func deriveWrappingKey(sharedSecret, nonce []byte) []byte {
	h := sha512.New()
	h.Write(sharedSecret)
	h.Write(nonce)
	sum := h.Sum(nil)
	return sum[:32]
}

func compressP256Point(pub *ecdh.PublicKey) []byte {
	// ecdh.PublicKey.Bytes() returns uncompressed point (0x04 || x || y)
	raw := pub.Bytes()
	// Compressed: 0x02 or 0x03 prefix (based on y parity) || x
	x := raw[1:33]
	y := raw[33:65]
	compressed := make([]byte, 33)
	if y[31]&1 == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}
	copy(compressed[1:], x)
	return compressed
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestRecipientWrapProducesValidStanza -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/delta/pivy/recipient.go go/src/delta/pivy/recipient_test.go
git commit -m "feat(delta/pivy): add Recipient with ECDH P-256 key wrapping"
```

---

### Task 2: Identity — Unwrap (Software ECDH, No Agent)

**Files:**
- Create: `go/src/delta/pivy/identity.go`
- Modify: `go/src/delta/pivy/recipient_test.go` (add round-trip test)

Before wiring up the agent, implement `Identity.Unwrap` using software ECDH.
This enables testing the full Wrap/Unwrap round-trip without hardware. The agent
call will be added in Task 3.

**Step 1: Write the failing test**

Add to `go/src/delta/pivy/recipient_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestWrapUnwrapRoundTrip -v`
Expected: FAIL — `Identity` type not defined

**Step 3: Write minimal implementation**

Create `go/src/delta/pivy/identity.go`:

```go
package pivy

import (
	"crypto/ecdh"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"filippo.io/age"
	"golang.org/x/crypto/chacha20poly1305"
)

// ECDHFunc performs ECDH given an ephemeral public key and returns the shared
// secret. For software keys this is a local operation. For agent-backed keys
// this calls the pivy-agent extension.
type ECDHFunc func(ephemeralPubkey []byte) (sharedSecret []byte, err error)

type Identity struct {
	ecdhFunc ECDHFunc
}

// TODO: add support for ecdh-rebox@joyent.com as alternative unwrap path

func (id *Identity) Unwrap(stanzas []*age.Stanza) ([]byte, error) {
	for _, s := range stanzas {
		if s.Type != StanzaTypePivyEcdhP256 {
			continue
		}

		fileKey, err := id.tryUnwrap(s)
		if err != nil {
			continue // trial decryption: AEAD failure means wrong recipient
		}

		return fileKey, nil
	}

	return nil, age.ErrIncorrectIdentity
}

func (id *Identity) tryUnwrap(s *age.Stanza) ([]byte, error) {
	if len(s.Args) != 2 {
		return nil, errors.Errorf("expected 2 args, got %d", len(s.Args))
	}

	ephPubBytes, err := base64.RawStdEncoding.DecodeString(s.Args[0])
	if err != nil {
		return nil, errors.Wrap(err)
	}

	nonce, err := base64.RawStdEncoding.DecodeString(s.Args[1])
	if err != nil {
		return nil, errors.Wrap(err)
	}

	sharedSecret, err := id.ecdhFunc(ephPubBytes)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	wrappingKey := deriveWrappingKey(sharedSecret, nonce)

	// Zeroize shared secret
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}

	aead, err := chacha20poly1305.New(wrappingKey)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// Zeroize wrapping key
	for i := range wrappingKey {
		wrappingKey[i] = 0
	}

	fileKey, err := aead.Open(nil, make([]byte, chacha20poly1305.NonceSize), s.Body, nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return fileKey, nil
}

func softwareECDH(privKey *ecdh.PrivateKey) ECDHFunc {
	return func(ephPubBytes []byte) ([]byte, error) {
		ephPub, err := decompressP256Point(ephPubBytes)
		if err != nil {
			return nil, err
		}

		return privKey.ECDH(ephPub)
	}
}

func decompressP256Point(compressed []byte) (*ecdh.PublicKey, error) {
	if len(compressed) != 33 {
		return nil, errors.Errorf(
			"invalid compressed point length: %d",
			len(compressed),
		)
	}

	x := new(big.Int).SetBytes(compressed[1:33])
	curve := elliptic.P256()
	y := decompressY(curve, x, compressed[0] == 0x03)

	if y == nil {
		return nil, errors.Errorf("invalid point: not on curve")
	}

	// Build uncompressed point: 0x04 || x || y
	uncompressed := make([]byte, 65)
	uncompressed[0] = 0x04
	xBytes := x.Bytes()
	yBytes := y.Bytes()
	copy(uncompressed[1+32-len(xBytes):33], xBytes)
	copy(uncompressed[33+32-len(yBytes):65], yBytes)

	return ecdh.P256().NewPublicKey(uncompressed)
}

func decompressY(curve elliptic.Curve, x *big.Int, odd bool) *big.Int {
	params := curve.Params()
	p := params.P

	// y^2 = x^3 - 3x + b (mod p)
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Mod(x3, p)

	threeX := new(big.Int).Mul(big.NewInt(3), x)
	threeX.Mod(threeX, p)

	y2 := new(big.Int).Sub(x3, threeX)
	y2.Add(y2, params.B)
	y2.Mod(y2, p)

	// y = sqrt(y^2) mod p
	y := new(big.Int).ModSqrt(y2, p)
	if y == nil {
		return nil
	}

	// Choose the correct root based on parity
	if odd != (y.Bit(0) == 1) {
		y.Sub(p, y)
	}

	return y
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -v`
Expected: PASS (all 3 tests)

**Step 5: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/delta/pivy/identity.go go/src/delta/pivy/recipient_test.go
git commit -m "feat(delta/pivy): add Identity with software ECDH unwrap"
```

---

### Task 3: Agent Client

**Files:**
- Create: `go/src/delta/pivy/agent.go`
- Modify: `go/src/delta/pivy/recipient_test.go` (add agent constructor test)

Implement the SSH agent extension client and wire it into `Identity` via
`ECDHFunc`. The agent test uses `softwareECDH` as a stand-in since we can't
rely on pivy-agent in CI, but the constructor is tested.

**Step 1: Write the failing test**

Add to `go/src/delta/pivy/recipient_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestResolveAgentSocketPath -v`
Expected: FAIL — `ResolveAgentSocketPath` not defined

**Step 3: Write minimal implementation**

Create `go/src/delta/pivy/agent.go`:

```go
package pivy

import (
	"crypto/ecdh"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// TODO: add support for reading socket path from blob store config TOML
// TODO: add -pivy-agent-sock CLI flag
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

	// Response is the raw shared secret bytes
	if len(response) == 0 {
		return nil, errors.Errorf("empty response from ecdh@joyent.com")
	}

	return response, nil
}

func pubkeyToSSHWireFormat(pub *ecdh.PublicKey) ([]byte, error) {
	// SSH wire format for ECDSA keys: string("ecdsa-sha2-nistp256") + string("nistp256") + string(uncompressed_point)
	// We use ssh.Marshal to build this.
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
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -v`
Expected: PASS (all 6 tests)

**Step 5: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/delta/pivy/agent.go go/src/delta/pivy/recipient_test.go
git commit -m "feat(delta/pivy): add agent client for ecdh@joyent.com extension"
```

---

### Task 4: IOWrapper — Streaming Encrypt/Decrypt

**Files:**
- Create: `go/src/delta/pivy/io_wrapper.go`
- Create: `go/src/delta/pivy/io_wrapper_test.go`

The `IOWrapper` delegates to `age.Encrypt`/`age.Decrypt` with the pivy
`Recipient`/`Identity`. This is the integration point with dodder's archive
system.

**Step 1: Write the failing test**

Create `go/src/delta/pivy/io_wrapper_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestIOWrapper -v`
Expected: FAIL — `IOWrapper` type not defined

**Step 3: Write minimal implementation**

Create `go/src/delta/pivy/io_wrapper.go`:

```go
package pivy

import (
	"crypto/ecdh"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"filippo.io/age"
)

type IOWrapper struct {
	RecipientPubkey *ecdh.PublicKey
	DecryptECDH     ECDHFunc
}

func (iow *IOWrapper) WrapWriter(w io.Writer) (io.WriteCloser, error) {
	out, err := age.Encrypt(w, &Recipient{Pubkey: iow.RecipientPubkey})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return out, nil
}

func (iow *IOWrapper) WrapReader(r io.Reader) (io.ReadCloser, error) {
	identity := &Identity{ecdhFunc: iow.DecryptECDH}

	out, err := age.Decrypt(r, identity)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return ohio.NopCloser(out), nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -v`
Expected: PASS (all 8 tests)

**Step 5: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/delta/pivy/io_wrapper.go go/src/delta/pivy/io_wrapper_test.go
git commit -m "feat(delta/pivy): add IOWrapper with streaming encrypt/decrypt via age"
```

---

### Task 5: Format Registration

**Files:**
- Create: `go/src/echo/markl/format_family_pivyecdhp256.go`
- Modify: `go/src/echo/markl/format.go` (add constant, register in init)
- Modify: `go/src/echo/markl/purposes.go` (add to madder purpose)

Register the pivy format so the config system can resolve
`pivy_ecdh_p256_pub`-encoded markl.Ids into IOWrappers.

**Step 1: Write the failing test**

Create `go/src/delta/pivy/format_registration_test.go`:

```go
//go:build test && debug

package pivy_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"testing"

	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TestFormatRegistered(t *testing.T) {
	format, err := markl.GetFormatOrError(markl.FormatIdPivyEcdhP256Pub)
	if err != nil {
		t.Fatalf("format not registered: %v", err)
	}

	if format.GetSize() != 33 {
		t.Fatalf("size: got %d, want 33", format.GetSize())
	}
}

func TestFormatGetIOWrapper(t *testing.T) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	compressed := compressP256PointForTest(privKey.PublicKey())

	var id markl.Id
	if err := id.SetMarklId(markl.FormatIdPivyEcdhP256Pub, compressed); err != nil {
		t.Fatalf("SetMarklId: %v", err)
	}

	ioWrapper, err := id.GetIOWrapper()
	if err != nil {
		t.Fatalf("GetIOWrapper: %v", err)
	}

	if ioWrapper == nil {
		t.Fatal("expected non-nil IOWrapper")
	}
}

func compressP256PointForTest(pub *ecdh.PublicKey) []byte {
	raw := pub.Bytes()
	x := raw[1:33]
	y := raw[33:65]
	compressed := make([]byte, 33)
	if y[31]&1 == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}
	copy(compressed[1:], x)
	return compressed
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -run TestFormat -v`
Expected: FAIL — `FormatIdPivyEcdhP256Pub` not defined

**Step 3: Write implementation**

Create `go/src/echo/markl/format_family_pivyecdhp256.go`:

```go
package markl

import (
	"crypto/ecdh"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/delta/pivy"
)

// TODO: add FormatIdPivyEcdhP384Pub and FormatIdPivyEcdhP521Pub

func PivyEcdhP256GetIOWrapper(
	id domain_interfaces.MarklId,
) (ioWrapper interfaces.IOWrapper, err error) {
	pubkey, err := ecdh.P256().NewPublicKey(id.GetBytes())
	if err != nil {
		err = errors.Wrapf(err, "parsing P-256 public key")
		return ioWrapper, err
	}

	socketPath, err := pivy.ResolveAgentSocketPath()
	if err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	ioWrapper = &pivy.IOWrapper{
		RecipientPubkey: pubkey,
		DecryptECDH:     pivy.AgentECDHFunc(socketPath, pubkey),
	}

	return ioWrapper, err
}
```

Add the constant to `go/src/echo/markl/format.go` alongside the existing format
IDs:

```go
FormatIdPivyEcdhP256Pub = "pivy_ecdh_p256_pub"
```

Add registration in the `init()` function in `go/src/echo/markl/format.go`:

```go
makeFormat(
	FormatSec{
		Id:           FormatIdPivyEcdhP256Pub,
		Size:         33,
		GetIOWrapper: PivyEcdhP256GetIOWrapper,
	},
)
```

Add to `PurposeMadderPrivateKeyV1` in `go/src/echo/markl/purposes.go`:

```go
// Find the existing makePurpose call for PurposeMadderPrivateKeyV1 and add
// FormatIdPivyEcdhP256Pub to the format ID list.
```

Also export `AgentECDHFunc` from `go/src/delta/pivy/agent.go` — add a public
constructor that the format registration can call:

```go
// AgentECDHFunc returns an ECDHFunc that calls pivy-agent at the given socket.
func AgentECDHFunc(socketPath string, recipientPubkey *ecdh.PublicKey) ECDHFunc {
	return func(ephPubBytes []byte) ([]byte, error) {
		return callAgentECDH(socketPath, recipientPubkey, ephPubBytes)
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/delta/pivy/ -v`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/echo/markl/format_family_pivyecdhp256.go go/src/echo/markl/format.go go/src/echo/markl/purposes.go go/src/delta/pivy/agent.go go/src/delta/pivy/format_registration_test.go
git commit -m "feat(echo/markl): register pivy_ecdh_p256_pub format with IOWrapper"
```

---

### Task 6: Key Discovery from pivy-agent

**Files:**
- Create: `go/src/echo/markl/pivy_agent_discover.go`

Parallel to `ssh_agent_discover.go` — connects to `PIVY_AUTH_SOCK`, lists keys,
filters for ECDSA P-256, returns markl.Ids.

**Step 1: Write the implementation**

Create `go/src/echo/markl/pivy_agent_discover.go`:

```go
package markl

import (
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func DiscoverPivyAgentECDHKeys() ([]Id, error) {
	socket := os.Getenv("PIVY_AUTH_SOCK")
	if socket == "" {
		return nil, errors.Errorf("PIVY_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to pivy-agent")
	}
	defer conn.Close()

	keys, err := agent.NewClient(conn).List()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list pivy-agent keys")
	}

	var ids []Id

	for _, key := range keys {
		if key.Type() != "ecdsa-sha2-nistp256" {
			continue
		}

		parsed, err := ssh.ParsePublicKey(key.Marshal())
		if err != nil {
			continue
		}

		cryptoPub, ok := parsed.(ssh.CryptoPublicKey)
		if !ok {
			continue
		}

		ecdsaPub := cryptoPub.CryptoPublicKey()

		// Extract the compressed P-256 point from the ECDSA public key.
		// crypto/ecdh can convert from crypto.PublicKey.
		ecdhPub, err := ecdhPubFromCryptoKey(ecdsaPub)
		if err != nil {
			continue
		}

		compressed := compressP256Point(ecdhPub)

		var id Id
		if err := id.SetMarklId(FormatIdPivyEcdhP256Pub, compressed); err != nil {
			continue
		}

		ids = append(ids, id)
	}

	return ids, nil
}
```

The helper functions `ecdhPubFromCryptoKey` and `compressP256Point` will need to
be added. `compressP256Point` can be extracted from `delta/pivy/recipient.go`
into a shared location, or duplicated here. Since `echo` cannot import `delta`
(hierarchy violation — `echo` is above `delta`), duplicating the ~10-line
compression function is the correct approach.

```go
import (
	"crypto/ecdh"
	"crypto/ecdsa"
)

func ecdhPubFromCryptoKey(pub interface{}) (*ecdh.PublicKey, error) {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		return k.ECDH()
	default:
		return nil, errors.Errorf("unsupported key type: %T", pub)
	}
}

func compressP256Point(pub *ecdh.PublicKey) []byte {
	raw := pub.Bytes()
	x := raw[1:33]
	y := raw[33:65]
	compressed := make([]byte, 33)
	if y[31]&1 == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}
	copy(compressed[1:], x)
	return compressed
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go build -tags test,debug ./src/echo/markl/`
Expected: compiles without errors

**Step 3: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/echo/markl/pivy_agent_discover.go
git commit -m "feat(echo/markl): add DiscoverPivyAgentECDHKeys for pivy-agent key discovery"
```

---

### Task 7: info-pivy_agent CLI Command

**Files:**
- Create: `go/src/yankee/commands_dodder/info_pivy_agent.go`

Parallel to `info_ssh_agent.go`. Prints one markl.Id per line for each
ECDSA P-256 key found in pivy-agent.

**Step 1: Write the implementation**

Create `go/src/yankee/commands_dodder/info_pivy_agent.go`:

```go
package commands_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
)

func init() {
	utility.AddCmd("info-pivy_agent", &InfoPivyAgent{})
}

type InfoPivyAgent struct{}

func (cmd InfoPivyAgent) Run(req command.Request) {
	keys, err := markl.DiscoverPivyAgentECDHKeys()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("no ECDSA P-256 keys found in pivy-agent")
		return
	}

	for _, key := range keys {
		text, err := key.MarshalText()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		fmt.Printf("%s\n", string(text))
	}
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go build -tags debug ./src/yankee/...`
Expected: compiles without errors

**Step 3: Verify command is registered**

Run: `just build` and then: `go/build/debug/dodder complete | grep pivy`
Expected: `info-pivy_agent` appears in completion output

**Step 4: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/yankee/commands_dodder/info_pivy_agent.go
git commit -m "feat(yankee): add info-pivy_agent command for key discovery"
```

---

### Task 8: Integration Test — Archive Encryption Round-Trip

**Files:**
- Modify: `go/src/echo/inventory_archive/data_writer_v1_test.go` (add pivy test)

Add a test that uses the pivy `IOWrapper` (with software ECDH) to encrypt and
decrypt archive entries. This validates that pivy plugs into the same path as
age.

**Step 1: Write the test**

Add to `go/src/echo/inventory_archive/data_writer_v1_test.go`:

```go
func TestV1PivyEncryptedRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeZstd

	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var encryption interfaces.IOWrapper = &pivy.IOWrapper{
		RecipientPubkey: privKey.PublicKey(),
		DecryptECDH:     pivy.SoftwareECDHForTesting(privKey),
	}

	entries := []struct {
		hash []byte
		data []byte
	}{
		{hash: sha256Hash([]byte("pivy1")), data: []byte("pivy encrypted v1 blob")},
		{hash: sha256Hash([]byte("pivy2")), data: []byte("another pivy encrypted v1 blob")},
	}

	writer, err := NewDataWriterV1(&buf, hashFormatId, ct, 0, encryption)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if err := writer.WriteFullEntry(e.hash, e.data); err != nil {
			t.Fatal(err)
		}
	}

	_, _, err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	reader, err := NewDataReaderV1(bytes.NewReader(buf.Bytes()), encryption)
	if err != nil {
		t.Fatal(err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatal(err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(readEntries))
	}

	for i, re := range readEntries {
		if !bytes.Equal(re.Data, entries[i].data) {
			t.Errorf("entry %d: data mismatch", i)
		}
	}
}
```

This requires exporting `softwareECDH` from `delta/pivy` for testing. Add to
`go/src/delta/pivy/identity.go`:

```go
// SoftwareECDHForTesting returns an ECDHFunc using a software private key.
// Intended for tests that need an IOWrapper without a running pivy-agent.
func SoftwareECDHForTesting(privKey *ecdh.PrivateKey) ECDHFunc {
	return softwareECDH(privKey)
}
```

**Step 2: Run the test**

Run: `cd /Users/sfriedenberg/eng/repos/dodder/go && go test -tags test,debug ./src/echo/inventory_archive/ -run TestV1PivyEncryptedRoundTrip -v`
Expected: PASS

**Step 3: Commit**

```bash
cd /Users/sfriedenberg/eng/repos/dodder && git add go/src/echo/inventory_archive/data_writer_v1_test.go go/src/delta/pivy/identity.go
git commit -m "test: add pivy ECDH encryption round-trip for inventory archive V1"
```

---

### Task 9: Verify Full Build and All Tests Pass

**Files:** None (verification only)

**Step 1: Run full unit test suite**

Run: `cd /Users/sfriedenberg/eng/repos/dodder && just test-go`
Expected: All tests pass, including the new pivy tests

**Step 2: Build debug and release binaries**

Run: `cd /Users/sfriedenberg/eng/repos/dodder && just build`
Expected: Builds successfully

**Step 3: Verify info-pivy_agent appears in completions**

Run: `go/build/debug/dodder complete | grep pivy`
Expected: `info-pivy_agent`

**Step 4: Run bats integration tests (if applicable)**

Run: `cd /Users/sfriedenberg/eng/repos/dodder && just test-bats-targets complete.bats`
Expected: Passes (info-pivy_agent should appear in completion list)

---

### Implementation Notes

**Dependency on `echo` from `delta`:** The `delta/pivy` package does NOT import
`echo/markl` — the format registration in `echo/markl/format_family_pivyecdhp256.go`
imports `delta/pivy`, not the other way around. This preserves the NATO
hierarchy (`delta` < `echo`).

**`compressP256Point` duplication:** This function exists in both `delta/pivy/`
and `echo/markl/`. Since `echo` cannot import `delta`, and this is a trivial
10-line function, duplication is acceptable. If a third location needs it, extract
to `charlie/` or a shared crypto utility package.

**Agent wire format:** The exact payload format for `ecdh@joyent.com` may need
adjustment based on testing against pivy-agent. The current implementation
follows pivy's C code (`pivy-agent.c:1718-1830`) but the SSH wire encoding may
differ. Task 3's `callAgentECDH` is the function to adjust.

**`age.ErrIncorrectIdentity`:** This is a sentinel error from `filippo.io/age`
that signals "this identity doesn't match any stanzas." `age.Decrypt` tries each
identity and moves on if this error is returned. The pivy `Identity.Unwrap`
returns this when no `pivy-ecdh-p256` stanza can be decrypted.
