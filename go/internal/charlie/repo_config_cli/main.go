package repo_config_cli

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/0/options_tools"
	"code.linenisgreat.com/dodder/go/internal/0/repo_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/descriptions"
	"code.linenisgreat.com/dodder/go/lib/charlie/config_cli"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type Config struct {
	config_cli.Config
	BasePath string
	RepoId   repo_id.Id

	IgnoreHookErrors bool
	Hooks            string
	IgnoreWorkspace  bool

	CheckoutCacheEnabled bool
	PredictableZettelIds bool

	printOptionsOverlay options_print.Overlay
	ToolOptions         options_tools.Options

	descriptions.Description
}

var _ interfaces.CommandComponentWriter = (*Config)(nil)

func (config Config) GetPrintOptionsOverlay() options_print.Overlay {
	return config.printOptionsOverlay
}

// TODO add support for all flags
// TODO move to store_config
func (config *Config) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	config.Config.SetFlagDefinitions(flagSet)

	flagSet.StringVar(&config.BasePath, "dir-dodder", "", "")

	flagSet.BoolVar(
		&config.IgnoreWorkspace,
		"ignore-workspace",
		false,
		"ignore any workspaces that may be present and checkout the object in a temporary workspace",
	)

	flagSet.BoolVar(
		&config.CheckoutCacheEnabled,
		"checkout-cache-enabled",
		false,
		"",
	)

	flagSet.BoolVar(
		&config.PredictableZettelIds,
		"predictable-zettel-ids",
		false,
		"generate new zettel ids in order",
	)

	config.printOptionsOverlay.AddToFlags(flagSet)
	config.ToolOptions.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&config.IgnoreHookErrors,
		"ignore-hook-errors",
		false,
		"ignores errors coming out of hooks",
	)

	flagSet.StringVar(&config.Hooks, "hooks", "", "")

	flagSet.Var(&config.Description, "comment", "Comment for inventory list")

	flagSet.Var(&config.RepoId, "repo_id", "repo location: . (cwd) or / (system)")
}

func Default() (config *Config) {
	config = &Config{
		Config: *(config_cli.Default()),
	}

	if envRepoId := os.Getenv("DODDER_REPO_ID"); envRepoId != "" {
		if err := config.RepoId.Set(envRepoId); err != nil {
			// env var is invalid — ignore, let flag override or error later
		}
	}

	return config
}

// func (config Config) GetPrintOptions() options_print.Options {
// 	return options_print.MakeDefaultConfig(config.printOptionsOverlay)
// }

func (config Config) UsePredictableZettelIds() bool {
	return config.PredictableZettelIds
}

func (config Config) GetConfigCLI() config_cli.Config {
	return config.Config
}

func (config Config) GetBasePath() string {
	return config.BasePath
}

func (config Config) GetIgnoreWorkspace() bool {
	return config.IgnoreWorkspace
}

func (config Config) GetRepoId() repo_id.Id {
	return config.RepoId
}

// FromAny extracts a Config from an any value (typically from command.Utility.GetConfigAny()).
// Panics if the value is not a *Config.
func FromAny(v any) Config {
	return *v.(*Config)
}
