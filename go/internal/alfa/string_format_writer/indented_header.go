package string_format_writer

import (
	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func MakeIndentedHeader(
	o ColorOptions,
) interfaces.StringEncoderTo[string] {
	return &indentedHeader{
		stringFormatWriter: MakeColor[string](
			o,
			MakeRightAligned(),
			fields.TypeHeading,
		),
	}
}

type indentedHeader struct {
	stringFormatWriter interfaces.StringEncoderTo[string]
}

func (f indentedHeader) EncodeStringTo(
	v string,
	w interfaces.WriterAndStringWriter,
) (n int64, err error) {
	// n1 int
	var n2 int64

	n2, err = f.stringFormatWriter.EncodeStringTo(v, w)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
