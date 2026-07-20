package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("set-parent", &SetParent{})
}

// SetParent (#287b) records the workspace's current parent repo identity
// (pubkey) into the workspace config. It is the explicit, non-interactive
// migration path for legacy workspaces (a `parent-path` with no pinned
// pubkey), and the way to re-pin after a parent legitimately moved or was
// replaced. It always pins whatever the parent currently resolves to — it
// does NOT verify against an existing pin (that is push/pull's job).
type SetParent struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer
}

var (
	_ interfaces.CommandComponentWriter = (*SetParent)(nil)
	_ command.CommandWithArgs           = (*SetParent)(nil)
)

func (cmd SetParent) GetDescription() command.Description {
	return command.Description{
		Short: "pin the workspace's parent repo identity (pubkey)",
	}
}

func (cmd *SetParent) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{}
}

func (cmd *SetParent) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(flagSet)
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)
}

func (cmd SetParent) Run(req command.Request) {
	local := cmd.MakeLocalWorkingCopy(req)

	cmd.ResolveImplicitDirectPath(local)

	if !cmd.ResolvedFromParent() {
		req.Cancel(errors.BadRequestf(
			"this workspace has no parent to pin (run inside a repo-backed " +
				"workspace created with `dodder init-workspace`)",
		))
		return
	}

	var remote repo.Repo

	if cmd.IsHomeRepoParent() {
		remote = cmd.MakeHomeRepoRemote(req)
	} else {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	}

	pubkey := remote.GetImmutableConfigPublic().GetPublicKey().StringWithFormat()

	if err := local.GetEnvWorkspace().PinParentPubkey(pubkey); err != nil {
		req.Cancel(err)
		return
	}

	local.GetUI().Printf("pinned parent: %s", pubkey)
}
