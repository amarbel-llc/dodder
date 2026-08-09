package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
)

func init() {
	utility.AddCmd("export-inventory_lists", &ExportInventoryLists{})
}

// ExportInventoryLists exports the repo's inventory-list history WITHOUT
// consulting the stream index: the list objects come straight from the
// inventory-list log (inventory_list_store.AllInventoryLists reads
// FileInventoryListLog directly), and -contents additionally decodes each
// list's blob for its member objects. `export`, by contrast, answers a
// query through the index — an index that is silently incomplete or
// corrupt silently narrows its output. This command is the recovery
// counterpart: its output depends only on the log and the blob store, so
// it stays trustworthy exactly when the index is in doubt (dodder#359-era
// legacy-store recovery; task: personal-data consolidation toolbox).
type ExportInventoryLists struct {
	command_components_dodder.LocalWorkingCopy

	Contents bool
}

var (
	_ interfaces.CommandComponentWriter = (*ExportInventoryLists)(nil)
	_ command.CommandWithArgs           = (*ExportInventoryLists)(nil)
)

func (cmd ExportInventoryLists) GetDescription() command.Description {
	return command.Description{
		Short: "export inventory lists from the log, bypassing the stream index",
		Long: "Write the repo's inventory-list history to stdout as an " +
			"inventory list stream, reading the inventory-list log directly " +
			"instead of querying the stream index. By default emits the list " +
			"objects themselves (the :b history, sorted); with -contents, " +
			"each list is followed by its decoded member objects (the full " +
			"object history, importable into a fresh repo). Because nothing " +
			"here touches the index, the output is complete even when the " +
			"index is stale, incomplete, or corrupt — the recovery " +
			"counterpart to `export`, whose query path inherits any index " +
			"defect. Duplicate objects across lists (e.g. config states) are " +
			"emitted as-is; import deduplicates.",
	}
}

func (cmd *ExportInventoryLists) GetArgs() []command.ArgGroup { return nil }

func (cmd *ExportInventoryLists) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.Contents,
		"contents",
		false,
		"also emit each list's decoded member objects (full object history)",
	)
}

func (cmd ExportInventoryLists) Run(req command.Request) {
	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	inventoryListStore := localWorkingCopy.GetStore().GetInventoryListStore()

	var seq sku.Seq

	if cmd.Contents {
		// Each list object is yielded first, then its members — both come
		// from the log + blob store only.
		seq = func(yield func(*sku.Transacted, error) bool) {
			for objectWithList, iterErr := range inventoryListStore.AllInventoryListObjectsAndContents() {
				if iterErr != nil {
					if !yield(nil, errors.Wrap(iterErr)) {
						return
					}

					continue
				}

				if !yield(objectWithList.Object, nil) {
					return
				}
			}
		}
	} else {
		seq = inventoryListStore.AllInventoryListsSorted()
	}

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
		localWorkingCopy.GetUIFile(),
	)
	defer repoolBufferedWriter()
	defer errors.ContextMustFlush(localWorkingCopy, bufferedWriter)

	inventoryListCoderCloset := localWorkingCopy.GetInventoryListCoderCloset()

	if _, err := inventoryListCoderCloset.WriteTypedBlobToWriter(
		req,
		ids.GetOrPanic(
			localWorkingCopy.GetImmutableConfigPublic().GetInventoryListTypeId(),
		).TypeStruct,
		seq,
		bufferedWriter,
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
