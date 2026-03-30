package workspace_config_blobs

//go:generate tommy generate
type V2 struct {
	V1
	Haustoria HaustoriaConfig `toml:"haustoria,omitempty"`
}

type HaustoriaConfig struct {
	Type     string                `toml:"type"`
	CalDAV   *CalDAVConfig         `toml:"caldav,omitempty"`
	Mappings map[string]TypeMapping `toml:"mappings,omitempty"`
}

type CalDAVConfig struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Calendar string `toml:"calendar,omitempty"`
}

type TypeMapping struct {
	Component string `toml:"component"`
}

func (blob V2) GetHaustoriaConfig() HaustoriaConfig {
	return blob.Haustoria
}

var (
	_ ConfigWithHaustoria          = V2{}
	_ ConfigWithParentPath         = V2{}
	_ ConfigWithSyncBaseline       = V2{}
	_ ConfigWithDefaultQueryString = V2{}
	_ ConfigWithDryRun             = V2{}
)
