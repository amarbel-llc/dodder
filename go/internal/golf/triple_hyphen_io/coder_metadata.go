package triple_hyphen_io

import (
	"bufio"
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/format"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type TypedMetadataCoder[BLOB any] struct{}

func (TypedMetadataCoder[BLOB]) DecodeFrom(
	typedBlob *TypedBlob[BLOB],
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	// TODO scan for type directly
	if n, err = format.ReadLines(
		bufferedReader,
		ohio.MakeLineReaderRepeat(
			ohio.MakeLineReaderKeyValues(
				map[string]interfaces.FuncSetString{
					"!": typedBlob.Type.Set,
				},
			),
		),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (TypedMetadataCoder[BLOB]) EncodeTo(
	typedBlob *TypedBlob[BLOB],
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	var n1 int
	n1, err = fmt.Fprintf(
		bufferedWriter,
		"! %s\n",
		typedBlob.Type.StringSansOp(),
	)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
