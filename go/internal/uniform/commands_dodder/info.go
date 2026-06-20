package commands_dodder

import (
	"bufio"
	"strings"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/env_vars"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type Info struct{}

var _ interfaces.CommandComponentWriter = (*Info)(nil)

func (cmd Info) GetDescription() command.Description {
	return command.Description{
		Short: "display repository information",
	}
}

func init() {
	utility.AddCmd(
		"info",
		&Info{},
	)
}

func (cmd *Info) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "info-keys",
			Description: "information keys to display (default: store-version)",
			Variadic:    true,
			EnumValues: []string{
				"store-version",
				"store-version-next",
				"compression-type",
				"age-encryption",
				"env",
				"xdg",
			},
		}},
	}}
}

func (cmd Info) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}

// infoEnvDir builds the dodder env_dir for the `env`/`xdg` info keys,
// honoring a system-scoped id by rooting at the system root via
// MakeDefaultAndInitialize (#280) — the same routing the operate path uses
// — so `info -repo_id //name xdg` reports the system paths rather than
// silently falling back to the user tree. Other scopes use the cwd-walk-up
// MakeDefault. CheckSupported still gates multi-dot before this runs.
func infoEnvDir(
	req command.Request,
	config repo_config_cli.Config,
) mad_env_dir.Env {
	if config.RepoId.GetLocationType() == scoped_id.LocationTypeXDGSystem {
		return env_dir.MakeDefaultAndInitialize(
			req,
			dodder_env.XDGUtilityName,
			config.Debug,
			repo_id.EffectiveId(config.RepoId),
		)
	}

	return env_dir.MakeDefault(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
		repo_id.EffectiveName(config.RepoId),
	)
}

func (cmd Info) Run(req command.Request) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	ui := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	args := req.PopArgs()

	if len(args) == 0 {
		args = []string{"store-version"}
	}

	defaultGenesisConfig := genesis_configs.DefaultWithVersion(
		store_version.VCurrent,
		ids.TypeInventoryListVCurrent,
	).Blob

	defaultBlobStoreConfig := blob_store_configs.Default().Blob

	for _, arg := range args {
		// TODO switch to underscore+hyphen string keys
		switch strings.ToLower(arg) {
		case "store-version":
			ui.GetUI().Print(defaultGenesisConfig.GetStoreVersion())

		case "store-version-next":
			ui.GetUI().Print(store_version.VNext)

		case "compression-type":
			if compressionType, ok := defaultBlobStoreConfig.(blob_store_configs.ConfigCompressionType); ok {
				ui.GetUI().Print(compressionType.GetCompressionType())
			} else {
				errors.ContextCancelWithBadRequestf(ui, "default blob store does not support compression")
			}

		case "age-encryption":
			if ioWrapper, ok := defaultBlobStoreConfig.(mad_domain_interfaces.BlobIOWrapper); ok {
				ui.GetUI().Print(
					ioWrapper.GetBlobEncryption(),
				)
			} else {
				errors.ContextCancelWithBadRequestf(ui, "default blob store does not support encryption")
			}

		case "env":
			if err := repo_id.CheckSupported(config.RepoId); err != nil {
				ui.Cancel(err)
			}
			dir := infoEnvDir(req, config)
			envVars := env_vars.Make(dir)
			var coder env_vars.BufferedCoderDotenv
			bufferedWriter := bufio.NewWriter(ui.GetOutFile())

			if _, err := coder.EncodeTo(envVars, bufferedWriter); err != nil {
				ui.Cancel(err)
			}

			if err := bufferedWriter.Flush(); err != nil {
				ui.Cancel(err)
			}

		case "xdg":
			if err := repo_id.CheckSupported(config.RepoId); err != nil {
				ui.Cancel(err)
			}
			dir := infoEnvDir(req, config)
			ecksDeeGee := dir.GetXDG()
			envVars := env_vars.Make(ecksDeeGee)
			var coder env_vars.BufferedCoderDotenv
			bufferedWriter := bufio.NewWriter(ui.GetOutFile())

			if _, err := coder.EncodeTo(envVars, bufferedWriter); err != nil {
				ui.Cancel(err)
			}

			if err := bufferedWriter.Flush(); err != nil {
				ui.Cancel(err)
			}
		}
	}
}
