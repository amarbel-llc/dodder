package object_metadata_fmt_triple_hyphen

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
)

type formatter []funcWrite

func (formatter formatter) FormatMetadata(
	writer io.Writer,
	formatterContext FormatterContext,
) (n int64, err error) {
	return ohio.WriteSeq(writer, formatterContext, formatter...)
}
