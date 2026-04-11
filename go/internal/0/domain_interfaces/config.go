package domain_interfaces

type (
	ConfigDryRunGetter interface {
		IsDryRun() bool
	}

	ConfigDryRunSetter interface {
		SetDryRun(bool)
	}

	MutableConfigDryRun interface {
		ConfigDryRunGetter
		ConfigDryRunSetter
	}

	Config interface {
		UsePredictableZettelIds() bool
		GetTypeStringFromExtension(t string) string
		GetTypeExtension(string) string
		ConfigDryRunGetter
	}

	MutableConfig interface {
		Config
		MutableConfigDryRun
	}

	// CLIConfigProvider provides base CLI configuration.
	// Note: debug.Options is not included because this package cannot import
	// delta/debug. Pass debug.Options separately where needed.
	CLIConfigProvider interface {
		GetVerbose() bool
		GetQuiet() bool
		GetTodo() bool
		IsDryRun() bool
	}

	// RepoCLIConfigProvider extends CLIConfigProvider with repository-specific
	// fields for dodder.
	RepoCLIConfigProvider interface {
		CLIConfigProvider
		GetBasePath() string
		GetIgnoreWorkspace() bool
	}
)
