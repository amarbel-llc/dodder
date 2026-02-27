package repo_config_cli

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/options_tools"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/internal/echo/config_cli"
	"code.linenisgreat.com/dodder/go/internal/echo/descriptions"
)

type Config struct {
	config_cli.Config
	BasePath string

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
}

func Default() (config *Config) {
	return &Config{
		Config: *(config_cli.Default()),
	}
	// config.printOptionsOverlay =
	// options_print.DefaultOverlay().GetPrintOptionsOverlay()
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

// FromAny extracts a Config from an any value (typically from command.Utility.GetConfigAny()).
// Panics if the value is not a *Config.
func FromAny(v any) Config {
	return *v.(*Config)
}
