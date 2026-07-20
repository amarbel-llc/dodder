package commands_dodder

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("edit-config", &EditConfig{})
}

type EditConfig struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*EditConfig)(nil)

// GetArgs returns nil: edit-config pops args but ignores them with a warning.
func (cmd *EditConfig) GetArgs() []command.ArgGroup { return nil }

func (cmd EditConfig) GetDescription() command.Description {
	return command.Description{
		Short: "edit the repository configuration",
	}
}

func (cmd EditConfig) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command edit-config ignores passed in arguments.")
	}

	// Capture the current config blob digest before the edit so a no-op
	// edit (digest unchanged) can be skipped when appending to the config
	// log below, keeping the log free of redundant entries. The digest is
	// cloned into a stable local value: GetBlobDigest returns a pointer
	// into the long-lived config sku's metadata, which Reset overwrites in
	// place with the new digest, so an un-cloned alias would always appear
	// equal to the post-edit digest.
	var previousDigest markl.Id
	previousDigest.ResetWithMarklId(
		localWorkingCopy.GetConfigStore().
			GetConfig().
			GetSku().
			GetBlobDigest(),
	)

	// Capture the config's current type before the edit. Editing config
	// does not change its type, so the pre-edit type is the right type to
	// stamp on the new config-log entry.
	configType := localWorkingCopy.GetConfigStore().
		GetConfig().
		GetSku().
		GetType().
		ToType()

	var digest mad_domain_interfaces.MarklId

	{
		var err error

		if digest, err = editKonfigInVim(localWorkingCopy); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}

	if err := localWorkingCopy.Reset(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	if err := localWorkingCopy.Lock(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	defer localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)

	// Append the new config state to the repo-local config log. Config
	// mutation is log-only: the konfig object is no longer written. Skip
	// no-op edits so the log only records real changes.
	if !markl.Equals(&previousDigest, digest) {
		log := config_log.Make(
			localWorkingCopy.GetEnvRepo(),
			command_components_dodder.InventoryLists{}.MakeInventoryListCoderCloset(
				localWorkingCopy.GetEnvRepo(),
			),
		)

		if err := log.Append(
			digest,
			configType,
			localWorkingCopy.GetStore().GetTai(),
		); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}

		// Print the just-appended entry as commit confirmation. Head only
		// reads the log file, so it is safe under the still-held lock.
		head, repoolHead, err := log.Head()
		if err != nil {
			localWorkingCopy.Cancel(err)
			return
		}

		defer repoolHead()

		if err := localWorkingCopy.PrinterConfigCommit()(head); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}
}
