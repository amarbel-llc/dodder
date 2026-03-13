package commands_dodder

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/import_plan"
	"code.linenisgreat.com/dodder/go/internal/romeo/remote_transfer"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("import", &Import{})
}

type Import struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.InventoryLists
	command_components_madder.BlobStore
	command_components_madder.Complete

	repo.ImporterOptions

	Proto sku.Proto

	BlobStoreId blob_store_id.Id
	PlanFormat  string
}

var _ interfaces.CommandComponentWriter = (*Import)(nil)

func (cmd *Import) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.ImporterOptions.SetFlagDefinitions(flagDefinitions)
	cmd.Proto.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.Var(
		cmd.Complete.GetFlagValueBlobIds(&cmd.BlobStoreId),
		"blob_store-id",
		"The name of the existing madder blob store to use",
	)

	flagDefinitions.StringVar(
		&cmd.PlanFormat,
		"plan-format",
		"summary",
		"output format for the import plan: summary or objects",
	)
}

func (cmd Import) Run(req command.Request) {
	inventoryListPaths := req.PopArgs()

	if len(inventoryListPaths) == 0 {
		errors.ContextCancelWithBadRequestf(
			req,
			"expected at least one inventory list path",
		)
	}

	local := cmd.MakeLocalWorkingCopy(req)

	cmd.DedupingFormatId = markl.PurposeV5MetadataDigestWithoutTai
	cmd.CheckedOutPrinter = local.PrinterCheckedOutConflictsForRemoteTransfers()

	if !cmd.BlobStoreId.IsEmpty() {
		cmd.RemoteBlobStore = local.GetEnvRepo().GetEnvBlobStore().GetBlobStore(
			cmd.BlobStoreId,
		)
	}

	closet := local.GetInventoryListCoderCloset()

	builder := import_plan.MakeBuilder(
		local.GetStore().GetStreamIndex(),
		markl.PurposeV5MetadataDigestWithoutTai,
	)

	for i, path := range inventoryListPaths {
		builder.AddSourcePath(path)

		seq := cmd.MakeSeqFromPath(
			local,
			closet,
			path,
			nil,
		)

		for object, err := range seq {
			if err != nil {
				local.Cancel(errors.Wrapf(err, "reading %s", path))
				return
			}

			builder.AddObject(object, i)
		}
	}

	plan, err := builder.Build()
	if err != nil {
		local.Cancel(errors.Wrap(err))
		return
	}

	if local.GetConfig().IsDryRun() {
		switch cmd.PlanFormat {
		case "objects":
			plan.FormatObjects(os.Stderr)
		default:
			plan.FormatSummary(os.Stderr)
		}

		if plan.HasErrors {
			local.Cancel(errors.WithoutStack(errors.Errorf("plan has errors")))
		}

		return
	}

	importer := local.MakeImporter(
		cmd.ImporterOptions,
		sku.GetStoreOptionsImport(),
	)

	if err := remote_transfer.CommitPlan(
		local,
		local,
		local,
		importer,
		plan,
	); err != nil {
		local.Cancel(err)
	}
}
