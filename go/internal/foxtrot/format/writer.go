package format

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func Write(
	w io.Writer,
	wffs ...interfaces.FuncWriter,
) (n int64, err error) {
	for _, wf := range wffs {
		var n1 int64

		if n1, err = wf(w); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n += n1
	}

	return n, err
}
