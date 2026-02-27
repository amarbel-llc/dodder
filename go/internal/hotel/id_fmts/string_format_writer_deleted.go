package id_fmts

import (
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/golf/fd"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type fdDeletedStringWriterFormat struct {
	dryRun               bool
	rightAlignedWriter   interfaces.StringEncoderTo[string]
	fdStringFormatWriter interfaces.StringEncoderTo[*fd.FD]
}

func MakeFDDeletedStringWriterFormat(
	dryRun bool,
	fdStringFormatWriter interfaces.StringEncoderTo[*fd.FD],
) *fdDeletedStringWriterFormat {
	return &fdDeletedStringWriterFormat{
		dryRun:               dryRun,
		rightAlignedWriter:   string_format_writer.MakeRightAligned(),
		fdStringFormatWriter: fdStringFormatWriter,
	}
}

func (f *fdDeletedStringWriterFormat) EncodeStringTo(
	fd *fd.FD,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var (
		n1 int
		n2 int64
	)

	prefix := string_format_writer.StringDeleted

	if f.dryRun {
		prefix = string_format_writer.StringWouldDelete
	}

	n2, err = f.rightAlignedWriter.EncodeStringTo(prefix, sw)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = sw.WriteString("[")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = f.fdStringFormatWriter.EncodeStringTo(
		fd,
		sw,
	)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = sw.WriteString("]")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
