# Archive Encryption Design

## Goal

Add per-entry encryption to inventory archive data files so blob content is
protected at rest, while preserving plaintext framing (headers, entry metadata,
indexes) for random access and opaque remote serving.

Also rename size fields from compression-centric terminology to git-style
`StoredSize`/`LogicalSize` while there are no backward-compatibility constraints.

## Threat Model

Protect blob data at rest on shared or untrusted filesystems. Which blobs exist
(hashes, counts, sizes) is not sensitive -- only the content is.

## Strategy: Encrypted Payloads, Plaintext Framing

Each entry's data payload is encrypted after compression. Archive structure
(header, entry hashes, logical/stored sizes, trailer checksum) remains plaintext.

```
Write: data -> compress -> encrypt -> archive
Read:  archive[offset] -> decrypt -> decompress -> data
```

Index and cache files are unaffected -- they reference stored (on-disk) payload
sizes, which remain correct regardless of whether the payload is encrypted.

## Field Renames

No users of these store types exist yet, so there are no backward-compatibility
requirements.

| Struct              | Old Field          | New Field     |
|---------------------|--------------------|---------------|
| DataEntry           | CompressedSize     | StoredSize    |
| DataEntry           | UncompressedSize   | LogicalSize   |
| DataEntryV1         | CompressedSize     | StoredSize    |
| DataEntryV1         | UncompressedSize   | LogicalSize   |
| IndexEntry          | CompressedSize     | StoredSize    |
| IndexEntryV1        | CompressedSize     | StoredSize    |
| CacheEntry          | CompressedSize     | StoredSize    |
| CacheEntryV1        | CompressedSize     | StoredSize    |
| archiveEntry        | CompressedSize     | StoredSize    |
| archiveEntryV1      | CompressedSize     | StoredSize    |

Wire format field names in comments: `compressed_size` -> `stored_size`,
`uncompressed_size` -> `logical_size`, `compressed_data`/`delta_data` ->
`payload`.

## Format Changes

### Flag

V0 header flags (2 bytes, currently all zeros):

```
FlagHasEncryption uint16 = 1 << 0
```

V1 header flags (bits 0-1 already used):

```
FlagHasEncryption uint16 = 1 << 2
```

Encryption is all-or-nothing per archive -- no per-entry toggle. The store
config determines whether encryption is active.

### Writer API

`NewDataWriter` and `NewDataWriterV1` gain an `encryption interfaces.IOWrapper`
parameter. When non-nil, `WriteEntry`/`WriteFullEntry`/`WriteDeltaEntry`
compress then encrypt the payload before writing. The `FlagHasEncryption` bit is
set in the header.

When encryption is `nil` or `NopeIOWrapper`, behavior is identical to today.

### Reader API

`NewDataReader` and `NewDataReaderV1` gain an `encryption interfaces.IOWrapper`
parameter. When the `FlagHasEncryption` flag is set, `ReadEntry` decrypts the
stored payload before decompressing.

### Pack Path

`packChunkArchive` (V0 and V1) resolves the config's `GetBlobEncryption()` to
an `IOWrapper` and passes it to the data writer.

### Read Path

`MakeBlobReader` resolves the same `IOWrapper` and passes it to the data reader.

## Key Management

Same key as the loose blob store. The archive config's `Encryption markl.Id`
field (already present in `TomlInventoryArchiveV0`/`V1`/`V2`) provides the key
via `GetBlobEncryption()` -> `GetIOWrapper()`.

## Testing

- Unit tests in `inventory_archive`: encrypted round-trip for V0 and V1
- Existing tests pass unchanged with `NopeIOWrapper`
- BATS integration tests exercise the full pack path

---

## Long-Term Vision

### PIV Token Keys

The current encryption backend is age X25519 (software keys). The goal is to
support PIV hardware tokens (via [pivy](https://github.com/arekinath/pivy)) for
secret key storage. The `IOWrapper` interface insulates the archive layer from
key management -- a PIV-backed `IOWrapper` implementation would slot in without
archive format changes.

### Opaque Remote Blob Stores

Remote blob stores (SFTP, HTTP, or future transports) should be able to operate
without access to private keys. They are "opaque": aware of which blobs they
contain and how they are stored, but unable to read the decrypted contents. They
serve encrypted payloads to authenticated clients who hold the private keys and
decrypt locally.

This design enables that directly:

- **Remote stores parse plaintext framing** -- archive headers, entry metadata
  (hashes, stored sizes), indexes, and caches are all readable without
  decryption. The remote can build and serve index lookups, answer "do you have
  blob X?" queries, and transfer specific entries by offset.

- **Clients decrypt after retrieval** -- the remote serves the encrypted payload
  bytes for a requested entry. The client applies its local `IOWrapper`
  (backed by a software key, age identity, or PIV token) to decrypt then
  decompress.

- **No key material on the remote** -- the remote never sees plaintext blob
  data. Compromise of the remote reveals which blobs exist and their sizes, but
  not their contents.

This architecture separates storage concerns (the remote's job) from secrecy
concerns (the client's job), allowing untrusted or semi-trusted hosts to
participate in the storage layer.
