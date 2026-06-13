package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/id_fmts"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/madder/go/pkgs/fd"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// TODO migrate to StringFormatWriterSkuBoxCheckedOut
func (local *Repo) PrinterTransactedDeleted() interfaces.FuncIter[*sku.CheckedOut] {
	printOptions := local.config.GetConfig().GetPrintOptions().
		WithPrintBlobDigests(true).
		WithPrintTime(false)

	stringEncoder := local.StringFormatWriterSkuBoxCheckedOut(
		printOptions,
		env_ui.FormatColorOptionsOut(local, printOptions),
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
			env_ui.FormatColorOptionsOut(local, local.GetConfig().GetPrintOptions()),
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
					env_ui.FormatColorOptionsOut(local, local.GetConfig().GetPrintOptions()),
					string_format_writer.MakeString[string](),
					fields.TypeHeading,
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

// PrinterConfigCommit prints a *sku.Transacted as a clean box line with
// blob digest on but tai and signatures off, producing the
// `[konfig @<digest> !toml-config-v2]` form. Used by edit-config and
// dormant-edit to confirm a committed config-log entry without the
// verbose tai/pubkey/sig fields that the archive printer (show-config
// -history) emits.
func (local *Repo) PrinterConfigCommit() interfaces.FuncIter[*sku.Transacted] {
	printOptions := local.GetConfig().GetPrintOptions().
		WithPrintBlobDigests(true).
		WithExcludeFields(true).
		WithPrintTai(false).
		WithPrintTime(false).
		WithPrintSigs(false)

	stringEncoder := local.StringFormatWriterSkuBoxTransacted(
		printOptions,
		env_ui.FormatColorOptionsOut(local, printOptions),
		string_format_writer.CliFormatTruncation66CharEllipsis,
	)

	return string_format_writer.MakeDelim(
		"\n",
		local.GetUIFile(),
		string_format_writer.MakeFunc(
			func(
				writer interfaces.WriterAndStringWriter,
				object *sku.Transacted,
			) (n int64, err error) {
				return stringEncoder.EncodeStringTo(object, writer)
			},
		),
	)
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
