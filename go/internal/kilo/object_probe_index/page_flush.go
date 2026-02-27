package object_probe_index

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
)

func (page *page) makeSeqReadFromFile(
	bufferedReader *bufio.Reader,
) interfaces.SeqError[*row] {
	return func(yield func(*row, error) bool) {
		if bufferedReader == nil {
			return
		}

		var row row

		for {
			n, err := page.readRowFrom(&row, bufferedReader)

			if err == io.EOF && n == 0 {
				return
			} else if err == io.EOF && n > 0 {
				yield(nil, errors.Wrap(io.ErrUnexpectedEOF))
				return
			} else if err != nil {
				yield(nil, errors.Wrap(err))
				return
			}

			if !yield(&row, nil) {
				return
			}
		}
	}
}

func (page *page) flushNew(
	bufferedReader *bufio.Reader,
	bufferedWriter *bufio.Writer,
) (err error) {
	seq := quiter.MergeSeqErrorLeft(
		quiter.MakeSeqErrorFromSeq(page.added.All()),
		page.makeSeqReadFromFile(bufferedReader),
		func(left, right *row) cmp.Result {
			return cmp.Bytes(left.Digest.GetBytes(), right.Digest.GetBytes())
		},
	)

	// Deduplicate consecutive rows with the same digest, keeping only the last
	// in each group. This matches the behavior of the old heap-based merge which
	// consumed all equal entries before writing. We must copy into a local row
	// because the file reader reuses the same row struct across iterations.
	equaler := page.added.GetEqualer()
	var pending row
	hasPending := false

	for current, errIter := range seq {
		if errIter != nil {
			err = errors.Wrap(errIter)
			return err
		}

		if hasPending {
			if !equaler.Equals(&pending, current) {
				if _, err = page.writeRowTo(&pending, bufferedWriter); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}
		}

		rowResetter{}.ResetWith(&pending, current)
		hasPending = true
	}

	if hasPending {
		if _, err = page.writeRowTo(&pending, bufferedWriter); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
