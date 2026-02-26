package ohio

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func MakeLineSeqFromReader(
	reader *bufio.Reader,
) interfaces.SeqError[string] {
	return func(yield func(string, error) bool) {
		for {
			line, err := reader.ReadString('\n')

			if len(line) > 0 {
				if !yield(line, nil) {
					return
				}
			}

			if err != nil {
				if !errors.IsEOF(err) {
					yield("", errors.Wrap(err))
				}

				return
			}
		}
	}
}
