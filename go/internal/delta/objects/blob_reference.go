package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type blobReferenceEntry struct {
	Key   markl.Id
	Alias string
}

type BlobReferences struct {
	entries collections_slice.Slice[blobReferenceEntry]
}

func (refs BlobReferences) All() interfaces.Seq[markl.Id] {
	return func(yield func(markl.Id) bool) {
		for entry := range refs.entries.All() {
			if !yield(entry.Key) {
				return
			}
		}
	}
}

func (refs *BlobReferences) Add(id markl.Id) {
	for _, entry := range refs.entries {
		if markl.Equals(&entry.Key, &id) {
			return
		}
	}

	refs.entries.Append(blobReferenceEntry{Key: id})
}

func (refs *BlobReferences) SetAlias(id markl.Id, alias string) error {
	for index := range refs.entries {
		entry := &refs.entries[index]

		if markl.Equals(&entry.Key, &id) {
			entry.Alias = alias
			return nil
		}
	}

	return errors.Errorf("blob reference not found: %s", id.String())
}

func (refs BlobReferences) GetAlias(id markl.Id) string {
	for _, entry := range refs.entries {
		if markl.Equals(&entry.Key, &id) {
			return entry.Alias
		}
	}

	return ""
}

func (refs *BlobReferences) Reset() {
	refs.entries.Reset()
}

func (refs *BlobReferences) ResetWith(other BlobReferences) {
	refs.entries.Reset()
	refs.entries.Grow(other.entries.Len())

	for entry := range other.entries.All() {
		var clone blobReferenceEntry
		clone.Key.ResetWith(entry.Key)
		clone.Alias = entry.Alias
		refs.entries.Append(clone)
	}
}
