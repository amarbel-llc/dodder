package hyphence

import (
	"bufio"
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/format"
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
					"@": typedBlob.BlobDigest.Set,
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

	if !typedBlob.BlobDigest.IsNull() {
		n1, err = fmt.Fprintf(
			bufferedWriter,
			"@ %s\n",
			&typedBlob.BlobDigest,
		)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
