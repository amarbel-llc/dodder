package ohio

import (
	"bufio"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
