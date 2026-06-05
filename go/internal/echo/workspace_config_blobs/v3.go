package workspace_config_blobs

//go:generate tommy generate
type V3 struct {
	V2
	// Ignore is a list of gitignore-style patterns; matching paths are
	// excluded from store_fs's external-item scan (issue #232). Dot-
	// prefixed paths (.git, .dodder, .madder, …) are already skipped by
	// the scan, so this targets non-dot dirs (node_modules, target, …).
	Ignore []string `toml:"ignore,omitempty"`
}

func (blob V3) GetIgnorePatterns() []string {
	return blob.Ignore
}

var (
	_ ConfigWithIgnore             = V3{}
	_ ConfigWithHaustoria          = V3{}
	_ ConfigWithParentPath         = V3{}
	_ ConfigWithSyncBaseline       = V3{}
	_ ConfigWithDefaultQueryString = V3{}
	_ ConfigWithDryRun             = V3{}
)
