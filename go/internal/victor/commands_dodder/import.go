package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/remote_transfer"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/charmbracelet/huh"
)

func init() {
	utility.AddCmd("import", &Import{})
}

type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return fmt.Sprintf("%v", *f) }
func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (cmd Import) GetDescription() command.Description {
	return command.Description{
		Short: "import objects from inventory list files",
	}
}

type Import struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.InventoryLists
	command_components_dodder.BlobStore

	repo.ImporterOptions

	Proto sku.Proto

	BlobStoreId blob_store_id.Id
	PlanFormat  string
	Interactive bool
	OmitTags    stringSliceFlag
}

var _ interfaces.CommandComponentWriter = (*Import)(nil)

func (cmd *Import) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "inventory-list-paths",
			Description: "paths to inventory list files to import",
			Required:    true,
			Variadic:    true,
		}},
	}}
}

func (cmd *Import) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.ImporterOptions.SetFlagDefinitions(flagDefinitions)
	cmd.Proto.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.Var(
		cmd.BlobStore.GetFlagValueBlobIds(&cmd.BlobStoreId),
		"blob_store-id",
		"The name of the existing madder blob store to use",
	)

	flagDefinitions.StringVar(
		&cmd.PlanFormat,
		"plan-format",
		"summary",
		"output format for the import plan: summary or objects",
	)

	flagDefinitions.BoolVar(
		&cmd.Interactive,
		"interactive",
		false,
		"interactively resolve blobless types by selecting local replacements",
	)

	flagDefinitions.BoolVar(
		&cmd.Interactive,
		"i",
		false,
		"shorthand for -interactive",
	)

	flagDefinitions.Var(
		&cmd.OmitTags,
		"omit-tags",
		"regex pattern for tags to strip during import (repeatable)",
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

	builder := import_plan.MakeImportBuilder(
		local.GetStore().GetStreamIndex(),
		markl.PurposeV5MetadataDigestWithoutTai,
	)

	if len(cmd.OmitTags) > 0 {
		transform, err := import_plan.MakeOmitTagsTransform(cmd.OmitTags)
		if err != nil {
			local.Cancel(errors.Wrap(err))
			return
		}

		builder.AddTransform(transform)
	}

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

			if err := builder.AddObject(object, i); err != nil {
				local.Cancel(errors.Wrap(err))
				return
			}
		}
	}

	plan, err := builder.Build()
	if err != nil {
		local.Cancel(errors.Wrap(err))
		return
	}

	if cmd.Interactive && plan.HasErrors {
		remapping := promptBloblessTypeResolution(local, plan)
		if len(remapping) > 0 {
			plan.ResolveBloblessTypes(remapping)
		}
	}

	if local.GetConfig().IsDryRun() {
		switch cmd.PlanFormat {
		case "objects":
			plan.FormatObjects(os.Stderr)
		default:
			printOptions := local.GetConfig().GetPrintOptions().
				WithPrintSigs(true)
			colorOptions := local.FormatColorOptionsErr(printOptions)

			boxFormatter := local.StringFormatWriterSkuBoxTransacted(
				printOptions,
				colorOptions,
				string_format_writer.CliFormatTruncation66CharEllipsis,
			)

			boxFormatter.SetAbbr(plan.Abbr)
			plan.FormatSummary(os.Stderr, boxFormatter)
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

func promptBloblessTypeResolution(
	local *local_working_copy.Repo,
	plan *import_plan.Plan,
) map[string]string {
	bloblessTypes := plan.BloblessTypes()
	if len(bloblessTypes) == 0 {
		return nil
	}

	if !local.GetEnv().GetIn().IsTty() {
		fmt.Fprintln(
			os.Stderr,
			"stdin is not a tty, skipping interactive blobless type resolution",
		)

		return nil
	}

	remapping := make(map[string]string)
	streamIndex := local.GetStore().GetStreamIndex()

	for _, typeString := range bloblessTypes {
		objectId, repool, err := ids.MakeObjectId(typeString)
		if err != nil {
			repool()
			continue
		}

		var localType sku.Transacted
		hasLocal := sku.ReadOneObjectId(streamIndex, objectId, &localType)
		repool()

		options := []huh.Option[string]{
			huh.NewOption("Skip (keep as error)", ""),
		}

		if hasLocal && !localType.GetBlobDigest().IsNull() {
			options = []huh.Option[string]{
				huh.NewOption(
					fmt.Sprintf("Use local %s", typeString),
					typeString,
				),
				huh.NewOption("Skip (keep as error)", ""),
			}
		}

		var result string

		huh.NewSelect[string]().
			Title(fmt.Sprintf("Blobless type: %s", typeString)).
			Options(options...).
			Value(&result).
			Run()

		if result != "" {
			remapping[typeString] = result
		}
	}

	return remapping
}
