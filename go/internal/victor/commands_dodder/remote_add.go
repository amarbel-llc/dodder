package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
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

	if err := remoteObject.GetObjectIdMutable().SetWithSeq(id.ToSeq()); err != nil {
		req.Cancel(err)
	}

	// TODO connect to remote and get public key and validate

	cmd.proto.Apply(remoteObject.GetMetadataMutable(), genres.Repo)

	builder := import_plan.MakeLocalBuilder()
	if err := builder.AddObject(remoteObject, 0); err != nil {
		req.Cancel(err)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		req.Cancel(buildErr)
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: local.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	if _, err := local.ExecutePlan(plan); err != nil {
		req.Cancel(err)
	}
}
