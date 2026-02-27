package string_format_writer

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func MakeDelim[T any](
	delim string,
	w1 interfaces.WriterAndStringWriter,
	f interfaces.StringEncoderTo[T],
) func(T) error {
	w := bufio.NewWriter(w1)

	return quiter.MakeSyncSerializer(
		func(e T) (err error) {
			ui.TodoP3("modify flushing behavior based on w1 being a TTY")
			defer errors.DeferredFlusher(&err, w)

			if _, err = f.EncodeStringTo(e, w); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if _, err = io.WriteString(w, delim); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)
}
