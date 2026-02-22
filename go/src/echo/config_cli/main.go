package config_cli

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/cli"
	"code.linenisgreat.com/dodder/go/src/delta/debug"
)

type Config struct {
	Debug   debug.Options
	Verbose bool
	Quiet   bool
	Todo    bool
	dryRun  bool

	// CustomOut and CustomErr override os.Stdout/os.Stderr when set.
	// Used by MCP handlers to capture command output into buffers.
	CustomOut io.Writer `toml:"-"`
	CustomErr io.Writer `toml:"-"`
}

var _ interfaces.CommandComponentWriter = (*Config)(nil)

func (config *Config) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cli.FlagSetVarWithCompletion(flagSet, &config.Debug, "debug")
	flagSet.BoolVar(&config.Todo, "todo", false, "")
	flagSet.BoolVar(&config.dryRun, "dry-run", false, "")
	flagSet.BoolVar(&config.Verbose, "verbose", false, "")
	flagSet.BoolVar(&config.Quiet, "quiet", false, "")
}

func Default() (config *Config) {
	return &Config{}
}

func (config Config) IsDryRun() bool {
	return config.dryRun
}

func (config *Config) SetDryRun(v bool) {
	config.dryRun = v
}

func (config Config) GetConfigCLI() Config {
	return config
}

func (config Config) GetVerbose() bool {
	return config.Verbose
}

func (config Config) GetQuiet() bool {
	return config.Quiet
}

func (config Config) GetTodo() bool {
	return config.Todo
}

// FromAny extracts a Config from an any value
func FromAny(v any) Config {
	return *v.(*Config)
}
