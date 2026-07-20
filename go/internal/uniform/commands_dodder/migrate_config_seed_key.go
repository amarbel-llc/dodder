package commands_dodder

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("migrate-config-seed-key", &MigrateConfigSeedKey{
		backup: true,
	})
}

type MigrateConfigSeedKey struct {
	path   string
	backup bool
}

var (
	_ interfaces.CommandComponentWriter = (*MigrateConfigSeedKey)(nil)
	_ command.CommandWithArgs           = (*MigrateConfigSeedKey)(nil)
)

func (cmd MigrateConfigSeedKey) GetDescription() command.Description {
	return command.Description{
		Short: "re-encode config-seed private key in canonical split-HRP form",
		Long: "Rewrites the `private-key` markl-id in dodder's config-seed " +
			"from the legacy combined-HRP wire form (pre-madder v0.3.16) to " +
			"the canonical split-HRP wire form. The underlying key bytes are " +
			"unchanged; only the checksum encoding is fixed. Run once if " +
			"dodder fails to load this repo with an " +
			"ErrLegacyCombinedHRPWireForm on the dodder-repo-private_key " +
			"purpose. See madder#167 and madder#170 for the migration story.",
	}
}

func (cmd *MigrateConfigSeedKey) GetArgs() []command.ArgGroup { return nil }

func (cmd *MigrateConfigSeedKey) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(
		&cmd.path,
		"path",
		"",
		"explicit path to config-seed (default: $XDG_DATA_HOME/dodder/config-seed)",
	)
	flagSet.BoolVar(
		&cmd.backup,
		"backup",
		true,
		"write a .bak.<unix-seconds> sibling before overwriting",
	)
}

func (cmd MigrateConfigSeedKey) Run(req command.Request) {
	req.AssertNoMoreArgs()

	dryRun := repo_config_cli.FromAny(req.Utility.GetConfigAny()).IsDryRun()

	path := cmd.path
	if path == "" {
		resolved, err := defaultConfigSeedPath()
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}
		path = resolved
	}

	result, err := env_repo.MigrateLegacyCombinedHRPConfigSeed(
		path,
		cmd.backup,
		dryRun,
	)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	if result.AlreadyCanonical {
		ui.Out().Printf(
			"%s: private-key already in canonical split-HRP wire form, nothing to do",
			path,
		)
		return
	}

	ui.Out().Printf("legacy combined-HRP wire form detected at %s", path)
	ui.Out().Printf("  purpose: %s", result.Purpose)
	ui.Out().Printf("  format:  %s", result.FormatId)
	ui.Out().Printf("  data:    %d bytes (preserved)", result.DataBytes)
	ui.Out().Printf("  before:  %s", result.LegacyWire)
	ui.Out().Printf("  after:   %s", result.CanonicalWire)

	if dryRun {
		ui.Out().Print("dry-run: not writing")
		return
	}

	if result.BackupPath != "" {
		ui.Out().Printf("wrote backup: %s", result.BackupPath)
	}
	ui.Out().Printf("rewrote %s", path)
}

func defaultConfigSeedPath() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "dodder", "config-seed"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrapf(err, "resolving user home dir")
	}

	return filepath.Join(home, ".local", "share", "dodder", "config-seed"), nil
}
