package commands_dodder

import (
	"bufio"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/store_version"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/hotel/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
	"code.linenisgreat.com/dodder/go/internal/india/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/env_vars"
)

type Info struct{}

var _ interfaces.CommandComponentWriter = (*Info)(nil)

func init() {
	utility.AddCmd(
		"info",
		&Info{},
	)
}

func (cmd Info) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
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
			if ioWrapper, ok := defaultBlobStoreConfig.(domain_interfaces.BlobIOWrapper); ok {
				ui.GetUI().Print(
					ioWrapper.GetBlobCompression(),
				)
			} else {
				errors.ContextCancelWithBadRequestf(ui, "default blob store does not support compression")
			}

		case "age-encryption":
			if ioWrapper, ok := defaultBlobStoreConfig.(domain_interfaces.BlobIOWrapper); ok {
				ui.GetUI().Print(
					ioWrapper.GetBlobEncryption(),
				)
			} else {
				errors.ContextCancelWithBadRequestf(ui, "default blob store does not support encryption")
			}

		case "env":
			dir := env_dir.MakeDefault(req, env_dir.XDGUtilityNameDodder, config.Debug)
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
			dir := env_dir.MakeDefault(req, env_dir.XDGUtilityNameDodder, config.Debug)
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
