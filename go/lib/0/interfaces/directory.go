package interfaces

type (
	EnvVars = map[string]string

	EnvVarsAdder interface {
		AddToEnvVars(EnvVars)
	}

	DirectoryLayoutPath interface {
		Stringer

		GetBaseEnvVar() DirectoryLayoutBaseEnvVar
		GetTarget() string

		GetTemplate() string
	}

	DirectoryLayoutBaseEnvVar interface {
		Stringer

		GetBaseEnvVarName() string
		GetBaseEnvVarValue() string

		MakePath(...string) DirectoryLayoutPath
	}

	DirectoryLayoutXDG interface {
		// GetLocationType() blob_store_id.LocationType

		GetDirHome() DirectoryLayoutBaseEnvVar
		GetDirCwd() DirectoryLayoutBaseEnvVar
		GetDirData() DirectoryLayoutBaseEnvVar
		GetDirConfig() DirectoryLayoutBaseEnvVar
		GetDirState() DirectoryLayoutBaseEnvVar
		GetDirCache() DirectoryLayoutBaseEnvVar
		GetDirRuntime() DirectoryLayoutBaseEnvVar

		CloneWithUtilityName(string) DirectoryLayoutXDG
	}
)
