package object_id_provider

import (
	"bufio"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/coordinates"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

type provider []string

func newProvider(path string) (p provider, err error) {
	var f *os.File

	if f, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return p, err
	}

	defer errors.Deferred(&err, f.Close)

	r := bufio.NewReader(f)

	for {
		var line string
		line, err = r.ReadString('\n')

		if err == io.EOF {
			err = nil
			break
		}

		if err != nil {
			err = errors.Wrap(err)
			return p, err
		}

		p = append(p, Clean(line))
	}

	return p, err
}

func (p provider) MakeZettelIdFromCoordinates(i coordinates.Int) (s string, err error) {
	if len(p)-1 < int(i) {
		err = errors.ErrorWithStackf(
			"insuffient ids. requested %d, have %d, last %s",
			int(i),
			len(p)-1,
			p[len(p)-1],
		)

		return s, err
	}

	s = p[i]

	return s, err
}

func (p provider) Len() int {
	return len(p)
}

func (p provider) ZettelId(v string) (i int, err error) {
	v = Clean(v)

	var s string

	for i, s = range p {
		if s == v {
			return i, err
		}
	}

	err = ErrDoesNotExist{
		Value: v,
	}

	return i, err
}
