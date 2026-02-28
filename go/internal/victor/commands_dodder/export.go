package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/markl_age_id"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
	"code.linenisgreat.com/dodder/go/lib/echo/age"
)

func init() {
	utility.AddCmd(
		"export",
		&Export{
			CompressionType: compression_type.CompressionTypeEmpty,
		})
}

type Export struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	AgeIdentity     age.Identity
	CompressionType compression_type.CompressionType
}

var _ interfaces.CommandComponentWriter = (*Export)(nil)

func (cmd *Export) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(f)

	f.Var(&cmd.AgeIdentity, "age-identity", "")
	cmd.CompressionType.SetFlagDefinitions(f)
}

func (cmd Export) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(
				genres.InventoryList,
			),
		),
	)

	var list *sku.HeapTransacted

	{
		var err error

		if list, err = localWorkingCopy.MakeInventoryList(queryGroup); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	var ag markl_age_id.Id

	if err := ag.AddIdentity(cmd.AgeIdentity); err != nil {
		errors.ContextCancelWithErrorAndFormat(
			localWorkingCopy,
			err,
			"age-identity: %q",
			&cmd.AgeIdentity,
		)
	}

	var writeCloser io.WriteCloser = ohio.NopWriteCloser(localWorkingCopy.GetUIFile())

	defer errors.ContextMustClose(localWorkingCopy, writeCloser)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(writeCloser)
	defer repoolBufferedWriter()
	defer errors.ContextMustFlush(localWorkingCopy, bufferedWriter)

	inventoryListCoderCloset := localWorkingCopy.GetInventoryListCoderCloset()

	if _, err := inventoryListCoderCloset.WriteTypedBlobToWriter(
		req,
		ids.GetOrPanic(localWorkingCopy.GetImmutableConfigPublic().GetInventoryListTypeId()).TypeStruct,
		quiter.MakeSeqErrorFromSeq(list.All()),
		bufferedWriter,
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
