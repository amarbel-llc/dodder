package stream_index_fixed

import "code.linenisgreat.com/dodder/go/internal/_/key_bytes"

const (
	EntryWidth             = 256
	OverflowTrailerSize    = 7 // int32 offset + uint16 length + 1 flag byte
	InlineCapacity         = EntryWidth - 1
	InlineCapacityOverflow = EntryWidth - OverflowTrailerSize
)

// fixedFieldOrder defines the hot-first field ordering for fixed-length
// entries. Hot fields (used for filtering without overflow reads) come first.
var fixedFieldOrder = []key_bytes.Binary{
	// Hot fields — always inline for fast filtering
	key_bytes.Sigil,
	key_bytes.ObjectId,
	key_bytes.Tai,
	key_bytes.Type,
	key_bytes.Blob,

	// Cold fields — overflow if no room
	key_bytes.RepoPubKey,
	key_bytes.RepoSig,
	key_bytes.Description,
	key_bytes.Tag,
	key_bytes.SigParentMetadataParentObjectId,
	key_bytes.DigestMetadataParentObjectId,
	key_bytes.DigestMetadataWithoutTai,
	key_bytes.CacheTagImplicit,
	key_bytes.CacheTags,
}
