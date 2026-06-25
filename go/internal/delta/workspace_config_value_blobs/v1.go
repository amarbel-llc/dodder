package workspace_config_value_blobs

//go:generate tommy generate
type V1 struct {
	V0
	ParentPath   string `toml:"parent-path,omitempty"`
	ParentPubkey string `toml:"parent-pubkey,omitempty"`
	SyncTai      string `toml:"sync-tai,omitempty"`
	SyncDigest   string `toml:"sync-digest,omitempty"`
}

func (blob V1) GetParentPath() string {
	return blob.ParentPath
}

// GetParentPubkey returns the pinned parent repo public key (#287b). Empty
// for legacy V1 workspaces written before pinning existed, and for workspaces
// whose parent has not yet been pinned.
func (blob V1) GetParentPubkey() string {
	return blob.ParentPubkey
}

func (blob V1) GetSyncTai() string {
	return blob.SyncTai
}

func (blob V1) GetSyncDigest() string {
	return blob.SyncDigest
}
