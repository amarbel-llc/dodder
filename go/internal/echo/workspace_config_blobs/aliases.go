package workspace_config_blobs

import (
	workspace_config_value_blobs "code.linenisgreat.com/dodder/go/internal/delta/workspace_config_value_blobs"
)

type (
	V0 = workspace_config_value_blobs.V0
	V1 = workspace_config_value_blobs.V1
	V2 = workspace_config_value_blobs.V2
	V3 = workspace_config_value_blobs.V3

	HaustoriaConfig = workspace_config_value_blobs.HaustoriaConfig
	CalDAVConfig    = workspace_config_value_blobs.CalDAVConfig
	CalendarConfig  = workspace_config_value_blobs.CalendarConfig
	OrgmodeConfig   = workspace_config_value_blobs.OrgmodeConfig
	OrgmodeWebDAV   = workspace_config_value_blobs.OrgmodeWebDAV
	OrgmodeSFTP     = workspace_config_value_blobs.OrgmodeSFTP
	FolderConfig    = workspace_config_value_blobs.FolderConfig
)

var (
	_ ConfigWithDefaultQueryString = V0{}

	_ ConfigWithParentPath         = V1{}
	_ ConfigWithSyncBaseline       = V1{}
	_ ConfigWithDefaultQueryString = V1{}
	_ ConfigWithDryRun             = V1{}

	_ ConfigWithHaustoria          = V2{}
	_ ConfigWithParentPath         = V2{}
	_ ConfigWithSyncBaseline       = V2{}
	_ ConfigWithDefaultQueryString = V2{}
	_ ConfigWithDryRun             = V2{}

	_ ConfigWithIgnore             = V3{}
	_ ConfigWithHaustoria          = V3{}
	_ ConfigWithParentPath         = V3{}
	_ ConfigWithSyncBaseline       = V3{}
	_ ConfigWithDefaultQueryString = V3{}
	_ ConfigWithDryRun             = V3{}
)
