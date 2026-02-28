package markl

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Slice []Id

func (slice *Slice) ReadFrom(reader io.Reader) (n int64, err error) {
	bufferedReader, repool := pool.GetBufferedReader(reader)
	defer repool()

	var isEOF bool

	for !isEOF {
		var line string
		line, err = bufferedReader.ReadString('\n')

		if err == io.EOF {
			err = nil
			isEOF = true
		} else if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		if line == "" {
			continue
		}

		var id Id

		if err = id.Set(strings.TrimSpace(line)); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		*slice = append(*slice, id)
	}

	return n, err
}
