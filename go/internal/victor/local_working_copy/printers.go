package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/internal/golf/id_fmts"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
)

// TODO migrate to StringFormatWriterSkuBoxCheckedOut
func (local *Repo) PrinterTransactedDeleted() interfaces.FuncIter[*sku.CheckedOut] {
	printOptions := local.config.GetConfig().GetPrintOptions().
		WithPrintBlobDigests(true).
		WithPrintTime(false)

	stringEncoder := local.StringFormatWriterSkuBoxCheckedOut(
		printOptions,
		local.FormatColorOptionsOut(printOptions),
		string_format_writer.CliFormatTruncation66CharEllipsis,
		box_format.CheckedOutHeaderDeleted{
			ConfigDryRunGetter: local.GetConfig(),
		},
	)

	return string_format_writer.MakeDelim(
		"\n",
		local.GetUIFile(),
		string_format_writer.MakeFunc(
			func(
				writer interfaces.WriterAndStringWriter,
				object *sku.CheckedOut,
			) (n int64, err error) {
				return stringEncoder.EncodeStringTo(object, writer)
			},
		),
	)
}

// TODO make generic external version
func (local *Repo) PrinterFDDeleted() interfaces.FuncIter[*fd.FD] {
	p := id_fmts.MakeFDDeletedStringWriterFormat(
		local.GetConfig().IsDryRun(),
		id_fmts.MakeFDCliFormat(
			local.FormatColorOptionsOut(local.GetConfig().GetPrintOptions()),
			local.envRepo.MakeRelativePathStringFormatWriter(),
		),
	)

	return string_format_writer.MakeDelim(
		"\n",
		local.GetUIFile(),
		p,
	)
}

func (local *Repo) PrinterHeader() interfaces.FuncIter[string] {
	if local.config.GetConfig().GetPrintOptions().PrintFlush {
		return string_format_writer.MakeDelim(
			"\n",
			local.GetErrFile(),
			string_format_writer.MakeDefaultDatePrefixFormatWriter(
				local,
				string_format_writer.MakeColor(
					local.FormatColorOptionsOut(
						local.GetConfig().GetPrintOptions(),
					),
					string_format_writer.MakeString[string](),
					string_format_writer.ColorTypeHeading,
				),
			),
		)
	} else {
		return func(v string) error { return ui.Log().Print(v) }
	}
}

func (local *Repo) PrinterCheckedOutConflictsForRemoteTransfers() interfaces.FuncIter[*sku.CheckedOut] {
	p := local.PrinterCheckedOut(box_format.CheckedOutHeaderState{})

	return func(co *sku.CheckedOut) (err error) {
		if co.GetState() != checked_out_state.Conflicted {
			return err
		}

		if err = p(co); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

func (local *Repo) MakePrinterBoxArchive(
	out interfaces.WriterAndStringWriter,
	includeTai bool,
) interfaces.FuncIter[*sku.Transacted] {
	boxFormat := box_format.MakeBoxTransactedArchive(
		local.GetEnv(),
		local.GetConfig().GetPrintOptions().WithPrintTai(includeTai),
	)

	return string_format_writer.MakeDelim(
		"\n",
		out,
		string_format_writer.MakeFunc(
			func(w interfaces.WriterAndStringWriter, o *sku.Transacted) (n int64, err error) {
				return boxFormat.EncodeStringTo(o, w)
			},
		),
	)
}
