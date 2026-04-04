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
	Orgmode   *OrgmodeConfig            `toml:"orgmode,omitempty"`
	Folders   map[string]FolderConfig   `toml:"folders,omitempty"`
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

// OrgmodeConfig holds orgmode haustoria connection parameters. Supports
// WebDAV or SFTP as the file transport.
type OrgmodeConfig struct {
	Transport string          `toml:"transport"` // "webdav" or "sftp"
	WebDAV    *OrgmodeWebDAV  `toml:"webdav,omitempty"`
	SFTP      *OrgmodeSFTP    `toml:"sftp,omitempty"`
}

// OrgmodeWebDAV holds WebDAV connection parameters for orgmode sync.
type OrgmodeWebDAV struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
}

// OrgmodeSFTP holds SFTP connection parameters for orgmode sync.
type OrgmodeSFTP struct {
	Host           string `toml:"host"`
	Port           int    `toml:"port"`
	User           string `toml:"user"`
	PrivateKeyPath string `toml:"private-key-path"`
}

// FolderConfig maps a remote folder to a dodder type and optional tags.
type FolderConfig struct {
	Path string   `toml:"path"`
	Type string   `toml:"type"`
	Tags []string `toml:"tags,omitempty"`
}

// ResolveOrgmode merges TOML config values with environment variables for
// orgmode haustoria. Env vars: ORGMODE_WEBDAV_URL, ORGMODE_WEBDAV_USERNAME,
// ORGMODE_WEBDAV_PASSWORD, ORGMODE_SFTP_HOST, ORGMODE_SFTP_USER,
// ORGMODE_SFTP_PASSWORD.
func (orgmodeConfig OrgmodeConfig) ResolveOrgmode() (ResolvedOrgmodeConfig, error) {
	resolved := ResolvedOrgmodeConfig{
		Transport: orgmodeConfig.Transport,
	}

	switch orgmodeConfig.Transport {
	case "webdav", "":
		resolved.Transport = "webdav"

		if orgmodeConfig.WebDAV != nil {
			resolved.WebDAVURL = orgmodeConfig.WebDAV.URL
			resolved.WebDAVUsername = orgmodeConfig.WebDAV.Username
		}

		if resolved.WebDAVURL == "" {
			resolved.WebDAVURL = os.Getenv("ORGMODE_WEBDAV_URL")
		}
		if resolved.WebDAVUsername == "" {
			resolved.WebDAVUsername = os.Getenv("ORGMODE_WEBDAV_USERNAME")
		}

		resolved.WebDAVPassword = os.Getenv("ORGMODE_WEBDAV_PASSWORD")

		if resolved.WebDAVURL == "" {
			return resolved, fmt.Errorf("orgmode WebDAV URL required: set [haustoria.orgmode.webdav] url or ORGMODE_WEBDAV_URL")
		}

	case "sftp":
		if orgmodeConfig.SFTP != nil {
			resolved.SFTPHost = orgmodeConfig.SFTP.Host
			resolved.SFTPPort = orgmodeConfig.SFTP.Port
			resolved.SFTPUser = orgmodeConfig.SFTP.User
			resolved.SFTPPrivateKeyPath = orgmodeConfig.SFTP.PrivateKeyPath
		}

		if resolved.SFTPHost == "" {
			resolved.SFTPHost = os.Getenv("ORGMODE_SFTP_HOST")
		}
		if resolved.SFTPUser == "" {
			resolved.SFTPUser = os.Getenv("ORGMODE_SFTP_USER")
		}

		resolved.SFTPPassword = os.Getenv("ORGMODE_SFTP_PASSWORD")

		if resolved.SFTPHost == "" {
			return resolved, fmt.Errorf("orgmode SFTP host required: set [haustoria.orgmode.sftp] host or ORGMODE_SFTP_HOST")
		}
		if resolved.SFTPUser == "" {
			return resolved, fmt.Errorf("orgmode SFTP user required: set [haustoria.orgmode.sftp] user or ORGMODE_SFTP_USER")
		}

	default:
		return resolved, fmt.Errorf("unknown orgmode transport: %s (supported: webdav, sftp)", orgmodeConfig.Transport)
	}

	return resolved, nil
}

// ResolvedOrgmodeConfig holds fully resolved orgmode connection parameters.
type ResolvedOrgmodeConfig struct {
	Transport          string
	WebDAVURL          string
	WebDAVUsername     string
	WebDAVPassword     string
	SFTPHost           string
	SFTPPort           int
	SFTPUser           string
	SFTPPassword       string
	SFTPPrivateKeyPath string
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
