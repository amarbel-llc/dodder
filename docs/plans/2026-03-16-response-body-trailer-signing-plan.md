# Response Body Trailer Signing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Sign HTTP response bodies as a trailer so clients can verify response integrity without the server buffering the entire response.

**Architecture:** The server hashes the response body as it streams via `io.TeeReader` into a `markl_io.MakeWriter`, signs the resulting digest with the repo private key, and appends the signature as an HTTP trailer. The client wraps `resp.Body` in a verifying reader that independently hashes the body, then checks the trailer signature on EOF using the server's public key.

**Tech Stack:** Go `net/http` trailers, `markl_io.MakeWriter`, `markl.Id.Sign`/`Verify`, `domain_interfaces.FormatHash`

**Rollback:** Revert the commits on this branch. Purely additive — existing challenge-response nonce signing is untouched.

---

### Task 1: Register PurposeRequestRepoSigV1

**Files:**
- Modify: `go/internal/bravo/markl/purposes.go:122-123`

**Step 1: Add registration**

In `purposes.go` init(), after the existing request auth registrations (line 123), add:

```go
	makePurpose(
		PurposeRequestRepoSigV1,
		PurposeTypeRequestAuth,
		FormatIdEd25519Sig,
		FormatIdEcdsaP256Sig,
	)
```

**Step 2: Verify build**

Run: `cd go && go build ./internal/bravo/markl/`
Expected: success

**Step 3: Commit**

```
fix: register PurposeRequestRepoSigV1 for response body signing
```

---

### Task 2: Add HashFormat field to RoundTripperBufioWrappedSigner

**Files:**
- Modify: `go/internal/tango/remote_http/round_tripper_wrapped_signer.go:19-22`
- Modify: `go/internal/tango/remote_http/round_tripper_stdio.go:26-58`
- Modify: `go/internal/tango/remote_http/round_tripper_unix_socket.go:19-42`

**Step 1: Add field**

In `round_tripper_wrapped_signer.go`, add `HashFormat` to the struct:

```go
type RoundTripperBufioWrappedSigner struct {
	PublicKey  domain_interfaces.MarklId
	HashFormat domain_interfaces.FormatHash
	roundTripperBufio
}
```

Add import for `domain_interfaces`.

**Step 2: Set HashFormat in InitializeWithLocal**

In `round_tripper_stdio.go` `InitializeWithLocal`, after `roundTripper.PublicKey = pubkey`:

```go
	roundTripper.HashFormat = config.GetDefaultBlobStoreConfig().GetDefaultHashType()
```

Check that `store_config.Config` has this path. If not, the caller
(`command_components_dodder/remote.go`) passes the repo — use
`repo.GetBlobStore().GetDefaultHashType()` instead and thread it through.

**Step 3: Set HashFormat in UnixSocket Initialize**

In `round_tripper_unix_socket.go` `Initialize`, after `roundTripper.PublicKey = pubkey`:

```go
	roundTripper.HashFormat = remote.Repo.GetBlobStore().GetDefaultHashType()
```

**Step 4: SSH case — leave nil**

`InitializeWithSSH` does not set `PublicKey` (TOFU), and similarly leaves
`HashFormat` nil. Verification will be skipped when `HashFormat` is nil.

**Step 5: Verify build**

Run: `cd go && go build ./internal/tango/remote_http/`
Expected: success

**Step 6: Commit**

```
feat: add HashFormat to RoundTripperBufioWrappedSigner for body signing
```

---

### Task 3: Server — sign response body as trailer in makeHandler

**Files:**
- Modify: `go/internal/tango/remote_http/server.go:432-473` (response writing section of makeHandler)

**Step 1: Add imports**

Add `"code.linenisgreat.com/dodder/go/internal/alfa/markl_io"` to imports in server.go.

**Step 2: Modify the response writing closure**

Replace the response writing section inside the `RunContextWithPrintTicker` closure
(lines 434-472) with body hashing and trailer signing. The key changes:

1. Pre-declare the trailer before `WriteHeader`
2. Get hash from blob store, create `markl_io.MakeWriter(hash, nil)`
3. Wrap `response.Body` in `io.TeeReader` writing to the digest writer
4. After `io.Copy`, sign digest and set trailer
5. For nil body, flush for HTTP/2 safety

```go
			func(ctx errors.Context) {
				header := responseWriter.Header()

				for key, values := range response.Headers() {
					for _, value := range values {
						header.Add(key, value)
					}
				}

				if response.StatusCode == 0 {
					response.StatusCode = http.StatusOK
				}

				// Pre-declare trailer before WriteHeader
				header.Set("Trailer", headerRepoSig)

				responseWriter.WriteHeader(response.StatusCode)

				if response.Body == nil {
					server.signEmptyBodyTrailer(responseWriter)
					return
				}

				hashFormat := server.Repo.GetBlobStore().GetDefaultHashType()
				hash, repoolHash := hashFormat.GetHash()
				defer repoolHash()

				digestWriter := markl_io.MakeWriter(hash, nil)

				if _, err := io.Copy(
					io.MultiWriter(responseWriter, digestWriter, &progressWriter),
					response.Body,
				); err != nil {
					// ... same error handling as before ...
				}

				server.signBodyTrailer(
					digestWriter.GetMarklId(),
					responseWriter,
				)
			},
```

**Step 3: Add signBodyTrailer helper**

Add to server.go:

```go
func (server *Server) signBodyTrailer(
	bodyDigest domain_interfaces.MarklId,
	responseWriter http.ResponseWriter,
) {
	sec := server.Repo.GetImmutableConfigPrivate().Blob.GetPrivateKey()

	var sig markl.Id

	if err := sec.Sign(
		bodyDigest,
		&sig,
		markl.PurposeRequestRepoSigV1,
	); err != nil {
		ui.Err().Print(err)
		return
	}

	responseWriter.Header().Set(headerRepoSig, sig.String())
}
```

**Step 4: Add signEmptyBodyTrailer helper**

For nil-body responses, hash an empty body and sign it. Scope the flush to a
function so `defer errors.DeferredFlusher` works:

```go
func (server *Server) signEmptyBodyTrailer(
	responseWriter http.ResponseWriter,
) {
	hashFormat := server.Repo.GetBlobStore().GetDefaultHashType()
	hash, repoolHash := hashFormat.GetHash()
	defer repoolHash()

	digestWriter := markl_io.MakeWriter(hash, nil)

	// Flush the digest writer to finalize the empty hash
	func() (err error) {
		defer errors.DeferredFlusher(&err, digestWriter)
		return nil
	}()

	server.signBodyTrailer(digestWriter.GetMarklId(), responseWriter)

	if flusher, ok := responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
```

Note: check whether `markl_io.writer` implements `Flusher`. If `MakeWriter` is
called with `nil` as the underlying writer, flushing it may be a no-op or
unnecessary. The hash sum is available via `GetMarklId()` without flushing. If
`GetMarklId()` works without closing/flushing the digest writer, simplify to
just call `GetMarklId()` directly on an unused writer (hash of empty input).

**Step 5: Verify build**

Run: `cd go && go build ./internal/tango/remote_http/`
Expected: success

**Step 6: Commit**

```
feat: sign response body and send signature as HTTP trailer
```

---

### Task 4: Client — verify response body signature on read

**Files:**
- Modify: `go/internal/tango/remote_http/round_tripper_wrapped_signer.go`

**Step 1: Add verifyingBodyReader type**

Add a new type that wraps the response body, hashes it during reads, and
verifies the trailer signature when EOF is reached:

```go
type verifyingBodyReader struct {
	body         io.ReadCloser
	response     *http.Response
	digestWriter *markl_io.Writer  // from markl_io.MakeWriter
	repoolHash   interfaces.FuncRepool
	pubkey       domain_interfaces.MarklId
	verified     bool
}

func (r *verifyingBodyReader) Read(p []byte) (n int, err error) {
	n, err = r.body.Read(p)

	if n > 0 {
		r.digestWriter.Write(p[:n])
	}

	if errors.IsEOF(err) && !r.verified {
		r.verified = true

		if verifyErr := r.verifyTrailer(); verifyErr != nil {
			return n, verifyErr
		}
	}

	return n, err
}

func (r *verifyingBodyReader) Close() error {
	if r.repoolHash != nil {
		defer r.repoolHash()
	}

	return r.body.Close()
}

func (r *verifyingBodyReader) verifyTrailer() error {
	sigString := r.response.Trailer.Get(headerRepoSig)

	if sigString == "" {
		return errors.Errorf("response body signature trailer missing")
	}

	var sig markl.Id

	if err := sig.Set(sigString); err != nil {
		return errors.Wrap(err)
	}

	bodyDigest := r.digestWriter.GetMarklId()

	var bodyDigestId markl.Id
	bodyDigestId.ResetWithMarklId(bodyDigest)

	if err := r.pubkey.Verify(bodyDigestId, sig); err != nil {
		return errors.Wrapf(err, "response body signature verification failed")
	}

	return nil
}
```

Note: check the exact type returned by `markl_io.MakeWriter` — it's a
`*markl_io.writer` (unexported). If it can't be stored directly, use
`domain_interfaces.MarklIdGetter` or store the hash and call `GetMarklId()` on
the hash directly. Alternatively, use `markl_io.MakeWriterWithRepool` and store
the returned `*writer` via the interface it implements.

**Step 2: Wrap response body in RoundTrip**

In `RoundTrip`, after the existing challenge-response verification (line 93),
before returning the response, wrap the body if `HashFormat` is available:

```go
	if roundTripper.HashFormat != nil && response.Body != nil {
		hash, repoolHash := roundTripper.HashFormat.GetHash()
		digestWriter := markl_io.MakeWriter(hash, nil)

		response.Body = &verifyingBodyReader{
			body:         response.Body,
			response:     response,
			digestWriter: digestWriter,
			repoolHash:   repoolHash,
			pubkey:       pubkey,  // already parsed from response header
		}
	}
```

Note: `pubkey` is a local variable already parsed at line 64. Use it directly.

**Step 3: Verify build**

Run: `cd go && go build ./internal/tango/remote_http/`
Expected: success

**Step 4: Commit**

```
feat: verify response body signature trailer on client
```

---

### Task 5: Integration test

**Files:**
- Modify: `zz-tests_bats/current_version/push.bats` (or a dedicated remote test file)

**Step 1: Verify existing push test still passes**

Run: `just test-bats-targets push.bats`
Expected: all push tests pass (the existing push_validates_blob_digest test
exercises the full remote transfer path which now includes trailer signing)

**Step 2: Run full test suite**

Run: `just test`
Expected: all tests pass

**Step 3: Commit (if any test adjustments needed)**

```
test: verify response body trailer signing in integration tests
```

---

### Task 6: Final verification and cleanup

**Step 1: Run unit tests**

Run: `cd go && go test -v -tags test,debug ./internal/tango/remote_http/`
Expected: pass (if any tests exist in this package)

**Step 2: Run go vet and repool analyzer**

Run: `cd go && just check`
Expected: no new warnings

**Step 3: Verify full test suite**

Run: `just test`
Expected: all pass

**Step 4: Final commit if cleanup needed**
