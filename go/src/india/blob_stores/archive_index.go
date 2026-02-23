package blob_stores

// ArchiveIndex is implemented by blob stores backed by archive files.
// It exposes the in-memory index for listing archives and their blob IDs.
type ArchiveIndex interface {
	AllArchiveEntryChecksums() map[string][]string // archiveChecksum -> []blobIdString
}
