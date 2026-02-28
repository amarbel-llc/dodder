# Bug: `madder write` panics when PIVY_AUTH_SOCK is not set

## Symptom

Writing to a pivy-encrypted blob store panics with "PIVY_AUTH_SOCK not set"
even though encryption (write) only needs the public key, not the pivy agent:

```
$ madder init -encryption pivy_ecdh_p256_pub-qft20htscs7x4z2sjwx2qd6tvdanm894thyty4ty4jy3d72hn6lh6gcgng2 .local
TAP version 14
ok 1 - init /home/sasha/workspaces/madder-pivy-test/.madder/local/share/blob_stores/local/dodder-blob_store-config
1..1

$ echo wow | madder write .local -
TAP version 14
# switched to blob store: .local
PIVY_AUTH_SOCK not set
```

## Stack trace

```
├── # PivyEcdhP256GetIOWrapper
│     go/internal/bravo/markl/format_family_pivyecdhp256.go:23
├── # Id.GetIOWrapper
│     go/internal/bravo/markl/id_crypto_sec.go:152
├── # MakeConfig
│     go/internal/echo/env_dir/blob_config.go:25
├── # gopanic
│     runtime/panic.go:783
├── # PanicIfError
│     go/lib/bravo/errors/main.go:38
├── # MakeConfig
│     go/internal/echo/env_dir/blob_config.go:25
├── # localHashBucketed.makeEnvDirConfig
│     go/internal/foxtrot/blob_stores/store_local_hash_bucketed.go:81
├── # localHashBucketed.blobWriterTo
│     go/internal/foxtrot/blob_stores/store_local_hash_bucketed.go:173
├── # localHashBucketed.MakeBlobWriter
│     go/internal/foxtrot/blob_stores/store_local_hash_bucketed.go:146
├── # Write.doOne
│     go/internal/india/commands_madder/write.go:182
├── # Write.Run
│     go/internal/india/commands_madder/write.go:115
```

## Root cause

`MakeConfig` in `blob_config.go:25` calls `GetIOWrapper` unconditionally, which
calls `PivyEcdhP256GetIOWrapper`. This creates an `IOWrapper` that needs the
pivy agent for both encryption AND decryption. When `PIVY_AUTH_SOCK` is unset,
it panics via `PanicIfError`.

The write path only needs the public key (age recipient) for encryption. The
pivy agent (via `PIVY_AUTH_SOCK`) is only needed for the ECDH operation during
decryption (age identity unwrap).

## Expected behavior

- `madder write` to a pivy-encrypted blob store should succeed without
  `PIVY_AUTH_SOCK` set, since encryption uses only the embedded public key
- `madder cat` / `madder read` should require `PIVY_AUTH_SOCK` since decryption
  needs the pivy agent for ECDH
- `madder init` with `-encryption pivy_ecdh_p256_pub-*` SHOULD require
  `PIVY_AUTH_SOCK` to verify the key works, unless a
  `--skip-decryption-verification` flag is passed

## Likely fix

Split `IOWrapper` construction so that:

1. The encryption side (recipient) is always available from the public key
2. The decryption side (identity) is lazily constructed only when needed, and
   only fails if `PIVY_AUTH_SOCK` is unset at that point
3. `MakeConfig` should accept a mode or context indicating whether decryption
   capability is needed

## Key files

- `go/internal/bravo/markl/format_family_pivyecdhp256.go:23` -- `PivyEcdhP256GetIOWrapper`
- `go/internal/bravo/markl/id_crypto_sec.go:152` -- `Id.GetIOWrapper`
- `go/internal/echo/env_dir/blob_config.go:25` -- `MakeConfig` (panics here)
- `go/internal/foxtrot/blob_stores/store_local_hash_bucketed.go:81` -- `makeEnvDirConfig`
- `go/lib/delta/pivy/io_wrapper.go` -- IOWrapper bridging age and pivy
- `go/lib/delta/pivy/recipient.go` -- age recipient (encryption, public key only)
- `go/lib/delta/pivy/identity.go` -- age identity (decryption, needs agent)
