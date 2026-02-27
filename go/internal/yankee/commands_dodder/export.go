package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/age"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl_age_id"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
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
