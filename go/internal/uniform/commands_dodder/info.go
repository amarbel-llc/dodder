package commands_dodder

import (
	"bufio"
	"strings"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/env_vars"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
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

// infoEnvDir builds the dodder env_dir for the `env`/`xdg` info keys via the
// shared operate-path resolver, so `info -repo_id <id> xdg` reports exactly
// the paths the operate commands act on: a system `//name` roots at the system
// root (#280), a multi-dot `..name` resolves the Nth same-named ancestor
// store-aware (#281), and other scopes use the nearest-ancestor walk.
func infoEnvDir(
	req command.Request,
	config repo_config_cli.Config,
) mad_env_dir.Env {
	return command_components_dodder.MakeOperateEnvDir(
		req,
		config,
		dodder_env.XDGUtilityName,
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
