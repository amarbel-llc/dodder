package workspace_config_blobs

type V1 struct {
	V0
	ParentPath string `toml:"parent-path,omitempty"`
}

func (blob V1) GetParentPath() string {
	return blob.ParentPath
}

var (
	_ ConfigWithParentPath         = V1{}
	_ ConfigWithDefaultQueryString = V1{}
	_ ConfigWithDryRun             = V1{}
)
