# Bug: pivy-encrypted blob store fails to decrypt on read

## Symptom

Writing to a pivy-encrypted blob store succeeds, but reading back fails:

```
$ echo wow | madder write .pivy-test -
ok 1 - blake2b256-40mtcwggatwwql4pp9ty93nyugn3r3ppvzs48uza0ze9zltneh3qez5yrs -

$ madder cat .pivy-test blake2b256-40mtcwggatwwql4pp9ty93nyugn3r3ppvzs48uza0ze9zltneh3qez5yrs
failed to decompress: Unknown frame descriptor
```

## Blob store config

```toml
hash_buckets = [2]
hash_type-id = 'blake2b256'
encryption = 'pivy_ecdh_p256_pub-q2r9256et986jsl0rs0k0hmurkscjazfff5tz8mas5s4x9qh79ckx6ajkl9'
compression-type = 'zstd'
lock-internal-files = true
```

## Root cause chain (confirmed via debug instrumentation)

### 1. Agent ECDH call fails, triggering silent fallback to unencrypted read

The `newFileReaderFromReadSeeker` function (`go/internal/echo/env_dir/blob_reader.go:113`)
has a fallback: if `NewReader` fails with the configured encryption, it retries
with `encryption=nil`. This causes it to read raw age ciphertext as if it were
plaintext, which the zstd decompressor rejects as "Unknown frame descriptor".

Debug output confirmed:
```
DEBUG MakeConfig: encryption=pivy_ecdh_p256_pub-... (nil=false, isNull=false)
DEBUG MakeConfig: ioWrapper type=*pivy.IOWrapper (err=<nil>)
DEBUG newFileReaderFromReadSeeker: NewReader failed: identity did not match any
  of the recipients: incorrect identity for recipient block
DEBUG MakeConfig: encryption=<nil> (nil=true)
DEBUG decrypter first 16 bytes: 61 67 65 2d 65 6e 63 72 79 70 74 69 6f 6e 2e 6f
  (that's "age-encryption.o" -- raw ciphertext)
DEBUG encryption type: ohio.NopeIOWrapper
failed to decompress: Unknown frame descriptor
```

### 2. The agent ECDH call itself fails

The `age.Decrypt` call returns "incorrect identity for recipient block" because
`tryUnwrap` in `go/lib/delta/pivy/identity.go` fails at the AEAD open step --
the shared secret from the agent doesn't produce the correct wrapping key.

Debug output:
```
DEBUG tryUnwrap: ephPubBytes len=33, first bytes=03 5c 52 b0 1d a9 8a 06
DEBUG callAgentECDH: ephemeralPubkey len=33
DEBUG tryUnwrap: ecdhFunc error: agent: generic extension failure
```

### 3. The pivy-agent extension expects SSH wire format, not raw bytes

The `ecdh@joyent.com` extension in pivy-agent (`pivy/src/pivy-agent.c:1735`)
parses BOTH the recipient key and the ephemeral key using `sshkey_froms()`:

```c
if ((r = sshkey_froms(buf, &key)) ||        // recipient key
    (r = sshkey_froms(buf, &partner))) {     // ephemeral key
```

`sshkey_froms` expects SSH wire format:
`string("ecdsa-sha2-nistp256") + string("nistp256") + string(uncompressed_point)`

But `callAgentECDH` in `go/lib/delta/pivy/agent.go` was sending:
- `RecipientKey`: SSH wire format (correct)
- `EphemeralKey`: raw compressed bytes, 33 bytes (wrong)

### 4. The response format also needs parsing

The pivy-agent response (`pivy-agent.c:1816-1817`) is:

```c
sshbuf_put_u8(msg, SSH_AGENT_SUCCESS)       // type byte (0x06)
sshbuf_put_string(msg, secret, seclen)      // length-prefixed shared secret
```

Go's `ExtendedAgent.Extension()` returns the complete raw response including the
type byte. The code was treating the entire response as the bare shared secret.

## Fixes applied so far (in `go/lib/delta/pivy/agent.go`)

1. Decompress the ephemeral pubkey from 33-byte compressed to `*ecdh.PublicKey`
2. Convert ephemeral pubkey to SSH wire format via `pubkeyToSSHWireFormat()`
3. Parse the response to extract the shared secret from the SSH wire envelope

### 4. Missing outer string wrapper (the actual root cause of extension failure)

The previous status said "the extension call no longer returns generic extension
failure" but this was **not verified by testing** (commit message: "have not yet
been verified working with a YubiKey").

Analysis of the pivy-agent dispatch code revealed the real problem:

**pivy-agent's `exthandlers` table** (`pivy-agent.c:2324`) registers
`ecdh@joyent.com` with `eh_string = B_TRUE`. The `process_extension` dispatcher
(`pivy-agent.c:2362`) calls `sshbuf_froms(se_request, &inner)` for `eh_string`
extensions, which reads the remaining request bytes as a **length-prefixed
string** (`u32(len) + bytes`) before passing the unwrapped inner bytes to the
handler.

The **C reference client** (`piv.c:7485`) does `sshbuf_put_stringb(req, buf)`,
wrapping the entire payload in a length-prefixed string.

**Go's `agent.Extension()`** (`x/crypto/ssh/agent/client.go:834`) uses
`ssh:"rest"` for Contents, which appends raw bytes — **no outer string wrapper**.

Without the wrapper, pivy-agent's `sshbuf_froms` reads the first `u32` of the
payload (which is `len(recipientSSHKey)` ≈ 104) as the string length, consuming
only the recipient key blob. Then `sshkey_froms` inside `process_ext_ecdh` tries
to parse from this mangled buffer and fails → `send_extfail` → Go sees
`"agent: generic extension failure"`.

**Fix:** Prepend `u32(len(payload))` before passing to `Extension()`.

## Status: fix applied, needs YubiKey testing

All four fixes are now in `callAgentECDH`. The wire format should now match the
C reference client. Remaining verification:

1. **Test with YubiKey** — write and read back a blob to confirm end-to-end
   decryption works.
2. **If still failing**, add debug instrumentation to dump the raw response from
   the extension call (secret length should be 32 bytes for P-256 x-coordinate).

## Key files

- `go/lib/delta/pivy/agent.go` -- agent ECDH client (the buggy code)
- `go/lib/delta/pivy/identity.go` -- age identity unwrap using ECDHFunc
- `go/lib/delta/pivy/recipient.go` -- age recipient wrap (encryption, works)
- `go/lib/delta/pivy/io_wrapper.go` -- IOWrapper bridging age and pivy
- `go/internal/echo/env_dir/blob_reader.go` -- blob reader with fallback chain
- `go/internal/echo/env_dir/blob_config.go` -- config construction
- `pivy/src/pivy-agent.c:1718` -- process_ext_ecdh (server side)
- `pivy/src/piv.c:7454` -- piv_box_open_agent (reference client)
- `pivy/src/piv.c:5051` -- piv_ecdh (card-side ECDH operation)

## Other issues noticed

### `info-repo` missing blob store flag

The `info-repo` madder command doesn't support specifying which blob store to
query. Other madder commands use `command_components_madder.EnvBlobStore` /
`command_components_madder.BlobStoreLocal` for this. Should be added for
consistency.
