package commands_dodder

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("dormant-edit", &DormantEdit{})
}

func (cmd DormantEdit) GetDescription() command.Description {
	return command.Description{
		Short: "edit dormant tags in an editor",
	}
}

type DormantEdit struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*DormantEdit)(nil)

// GetArgs returns nil: dormant-edit pops args but ignores them with a warning.
func (cmd *DormantEdit) GetArgs() []command.ArgGroup { return nil }

func (cmd DormantEdit) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command dormant-edit ignores passed in arguments.")
	}

	// dormant-edit edits dormant tags, which live inside the config blob;
	// like edit-config it commits the result via UpdateKonfig, so mirror
	// the additive config-log append here too. Capture the pre-edit digest
	// as a stable clone (GetBlobDigest aliases the config sku metadata that
	// Reset overwrites in place) so no-op edits can be skipped.
	var previousDigest markl.Id
	previousDigest.ResetWithMarklId(
		localWorkingCopy.GetConfigStore().
			GetConfig().
			GetSku().
			GetBlobDigest(),
	)

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

	if _, err := localWorkingCopy.GetStore().UpdateKonfig(digest); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	// Additive config-log append (UpdateKonfig still writes the konfig
	// object above; a later task removes it). Skip no-op edits.
	if !markl.Equals(&previousDigest, digest) {
		log := config_log.Make(
			localWorkingCopy.GetEnvRepo(),
			command_components_dodder.InventoryLists{}.MakeInventoryListCoderCloset(
				localWorkingCopy.GetEnvRepo(),
			),
		)

		if _, err := log.Append(
			digest,
			localWorkingCopy.GetStore().GetTai(),
		); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}
}
