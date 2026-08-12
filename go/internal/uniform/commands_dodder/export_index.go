package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
)

func init() {
	utility.AddCmd("export-index", &ExportIndex{})
}

// ExportIndex dumps the stream index VERBATIM as an inventory list
// stream: every record the index holds, via ReadPrimitiveQuery with the
// permissive primitive group (history + hidden sigils, no genre mask) —
// below the query layer entirely. Complements the other two recovery
// exports: `export` answers a QUERY (genre-filtered, query-layer
// semantics applied) and `export-inventory_lists` reads the LOG (whose
// list blobs can be missing — take4 pathology P9). When the index is the
// sole surviving carrier of states whose list blobs are gone, this is
// the census: it emits config-genre states and anything else the query
// surface cannot address.
type ExportIndex struct {
	command_components_dodder.LocalWorkingCopy
}

var (
	_ interfaces.CommandComponentWriter = (*ExportIndex)(nil)
	_ command.CommandWithArgs           = (*ExportIndex)(nil)
)

func (cmd ExportIndex) GetDescription() command.Description {
	return command.Description{
		Short: "dump the stream index verbatim as an inventory list stream",
		Long: "Write every record in the stream index to stdout as an " +
			"inventory list, using the primitive read surface below the " +
			"query layer: no genre filter (config-genre states included), " +
			"history and hidden sigils on, no dormant/tag-realization " +
			"semantics applied. The recovery counterpart for the case where " +
			"the index is the only surviving carrier of object states whose " +
			"inventory-list blobs are missing: `export` filters through the " +
			"query system, `export-inventory_lists` requires the list blobs " +
			"to exist, and this command needs neither. NOTE the inverse " +
			"trust relationship: the index is derived, rebuildable state — " +
			"a reindex regenerates it from the log and its list blobs — so " +
			"this dump is authoritative ONLY for states whose log-side " +
			"carriers are lost.",
	}
}

func (cmd *ExportIndex) GetArgs() []command.ArgGroup { return nil }

func (cmd *ExportIndex) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}

func (cmd ExportIndex) Run(req command.Request) {
	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	streamIndex := localWorkingCopy.GetStore().GetStreamIndex()

	// Collect-and-clone, mirroring MakeInventoryList: ReadPrimitiveQuery
	// iterates pooled objects (possibly concurrently — hence the sync
	// serializer), so each record is cloned before it outlives the
	// callback. Zero-value records (no object id — index sentinels/probe
	// rows the primitive surface exposes) are skipped with a census: the
	// v2 coder refuses their null digests/sigs, and they carry no state.
	list := sku.MakeListTransacted()

	var skippedEmpty int

	if err := streamIndex.ReadPrimitiveQuery(
		sku.MakePrimitiveQueryGroup(),
		quiter.MakeSyncSerializer(
			func(object *sku.Transacted) (err error) {
				if object.GetObjectId().IsEmpty() {
					skippedEmpty++
					return err
				}

				cloned, _ := object.CloneTransacted() //repool:owned
				return list.Add(cloned)
			},
		),
	); err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	if skippedEmpty > 0 {
		ui.Err().Printf(
			"skipped %d empty index record(s) (no object id)",
			skippedEmpty,
		)
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
		quiter.MakeSeqErrorFromSeq(list.All()),
		bufferedWriter,
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
