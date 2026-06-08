package workspace_config_value_blobs

//go:generate tommy generate
type V1 struct {
	V0
	ParentPath string `toml:"parent-path,omitempty"`
	SyncTai    string `toml:"sync-tai,omitempty"`
	SyncDigest string `toml:"sync-digest,omitempty"`
}

func (blob V1) GetParentPath() string {
	return blob.ParentPath
}

func (blob V1) GetSyncTai() string {
	return blob.SyncTai
}

func (blob V1) GetSyncDigest() string {
	return blob.SyncDigest
}
