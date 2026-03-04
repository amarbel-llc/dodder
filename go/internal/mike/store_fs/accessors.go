package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/filesystem_ops"
)

func (store *Store) GetFileEncoder() FileEncoder {
	return store.fileEncoder
}

func (store *Store) GetFsOps() filesystem_ops.V0 {
	return store.fsOps
}
