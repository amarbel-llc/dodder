package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

// TODO switch to using compound command pattern from blob_store_init.go
func init() {
	utility.AddCmd(
		"remote-add",
		&RemoteAdd{})
}

type RemoteAdd struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.RemoteTransfer

	complete command_components_dodder.Complete

	proto sku.Proto
}

var _ interfaces.CommandComponentWriter = (*RemoteAdd)(nil)

func (cmd *RemoteAdd) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.RemoteTransfer.SetFlagDefinitions(flagSet)

	flagSet.Var(
		cmd.complete.GetFlagValueMetadataTags(&cmd.proto.Metadata),
		"tags",
		"tags added for new objects in `checkin`, `new`, `organize`",
	)

	cmd.proto.Metadata.SetFlagSetDescription(
		flagSet,
		"description to use for the new repo",
	)
}

func (cmd RemoteAdd) Run(req command.Request) {
	local := cmd.MakeLocalWorkingCopy(req)
	_, remoteObject := cmd.MakeRemoteAndObject(req, local)

	var id ids.RepoId

	if err := id.Set(req.PopArg("repo-id")); err != nil {
		req.Cancel(err)
	}

	req.AssertNoMoreArgs()

	if err := remoteObject.ObjectId.SetWithSeq(id.ToSeq()); err != nil {
		req.Cancel(err)
	}

	// TODO connect to remote and get public key and validate

	cmd.proto.Apply(remoteObject.GetMetadataMutable(), genres.Repo)

	req.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

	if err := local.GetStore().CreateOrUpdateDefaultProto(
		remoteObject,
		sku.StoreOptions{
			ApplyProto: true,
		},
	); err != nil {
		req.Cancel(err)
	}

	req.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))
}
