package workspace_config_blobs

import (
	"fmt"
	"os"
)

//go:generate tommy generate
type V2 struct {
	V1
	Haustoria HaustoriaConfig `toml:"haustoria,omitempty"`
}

type HaustoriaConfig struct {
	Type      string                    `toml:"type"`
	CalDAV    *CalDAVConfig             `toml:"caldav,omitempty"`
	Calendars map[string]CalendarConfig `toml:"calendars,omitempty"`
}

type CalDAVConfig struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
}

// CalendarConfig maps a CalDAV calendar to a dodder type and optional tags.
type CalendarConfig struct {
	URL        string            `toml:"url"`
	Type       string            `toml:"type"`
	Tags       []string          `toml:"tags,omitempty"`
	StatusTags map[string]string `toml:"status-tags,omitempty"`
}

// Resolve merges TOML config values with environment variables.
// TOML values take precedence; env vars are the fallback.
// Env vars: CALDAV_URL, CALDAV_USERNAME, CALDAV_PASSWORD.
// Password is always from CALDAV_PASSWORD (never stored in config).
func (c CalDAVConfig) Resolve() (ResolvedCalDAVConfig, error) {
	resolved := ResolvedCalDAVConfig{
		URL:      c.URL,
		Username: c.Username,
	}

	if resolved.URL == "" {
		resolved.URL = os.Getenv("CALDAV_URL")
	}
	if resolved.Username == "" {
		resolved.Username = os.Getenv("CALDAV_USERNAME")
	}

	resolved.Password = os.Getenv("CALDAV_PASSWORD")

	if resolved.URL == "" {
		return resolved, fmt.Errorf("CalDAV URL required: set [haustoria.caldav] url or CALDAV_URL")
	}
	if resolved.Username == "" {
		return resolved, fmt.Errorf("CalDAV username required: set [haustoria.caldav] username or CALDAV_USERNAME")
	}
	if resolved.Password == "" {
		return resolved, fmt.Errorf("CalDAV password required: set CALDAV_PASSWORD")
	}

	return resolved, nil
}

type ResolvedCalDAVConfig struct {
	URL      string
	Username string
	Password string
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
