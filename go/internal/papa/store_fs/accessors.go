package store_fs

func (store *Store) GetFileEncoder() FileEncoder {
	return store.fileEncoder
}
