package workspace_config_blobs

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

var (
	_ ConfigWithParentPath         = V1{}
	_ ConfigWithSyncBaseline       = V1{}
	_ ConfigWithDefaultQueryString = V1{}
	_ ConfigWithDryRun             = V1{}
)
