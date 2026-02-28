package genres

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func ConfirmTypeFromReader(t Genre, r *bufio.Reader) (err error) {
	var t1 Genre

	if t1, err = FromReader(r); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if t1 != t {
		err = errors.BadRequest(
			ErrWrongGenre{
				Expected: t,
				Actual:   t,
			},
		)
	}

	return err
}

func FromReader(r *bufio.Reader) (t Genre, err error) {
	var line string

	if line, err = r.ReadString('\n'); err != nil {
		err = errors.Wrap(err)
		return t, err
	}

	if err = t.Set(line); err != nil {
		err = errors.Wrap(err)
		return t, err
	}

	return t, err
}
