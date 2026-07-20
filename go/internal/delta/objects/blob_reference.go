package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/cmp"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type blobReferenceEntry struct {
	Key      markl.Id
	TypeLock markl.Lock[ids.SeqId, *ids.SeqId]
	Alias    string
}

type BlobReferences struct {
	entries collections_slice.Slice[blobReferenceEntry]
}

func blobReferenceEntryCompareKey(left, right blobReferenceEntry) cmp.Result {
	return cmp.CompareUTF8String(left.Key.String(), right.Key.String(), false)
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

func (refs *BlobReferences) Add(
	id markl.Id,
	typeLock markl.Lock[ids.SeqId, *ids.SeqId],
) {
	for _, entry := range refs.entries {
		if markl.Equals(&entry.Key, &id) {
			return
		}
	}

	refs.entries.Append(blobReferenceEntry{Key: id, TypeLock: typeLock})
	refs.entries.SortWithComparer(blobReferenceEntryCompareKey)
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

func (refs BlobReferences) GetTypeLock(
	id markl.Id,
) markl.Lock[ids.SeqId, *ids.SeqId] {
	for _, entry := range refs.entries {
		if markl.Equals(&entry.Key, &id) {
			return entry.TypeLock
		}
	}

	return markl.Lock[ids.SeqId, *ids.SeqId]{}
}

func (refs *BlobReferences) GetTypeLockMutable(
	id markl.Id,
) *markl.Lock[ids.SeqId, *ids.SeqId] {
	for index := range refs.entries {
		entry := &refs.entries[index]

		if markl.Equals(&entry.Key, &id) {
			return &entry.TypeLock
		}
	}

	return nil
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
		clone.TypeLock.ResetWith(entry.TypeLock)
		clone.Alias = entry.Alias
		refs.entries.Append(clone)
	}
}
