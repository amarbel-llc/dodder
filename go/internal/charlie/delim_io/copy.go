package delim_io

import (
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

// Copies each `delim` suffixed segment from src to dst, and for each segment,
// adds the passed in prefix string.
//
// Useful for taking a Reader and adding a prefix for every line, like how `git`
// shows `remote: <line>` for all remote stderr output.
// TODO extract into an io.Writer-like object
func CopyWithPrefixOnDelim(
	delimiter byte,
	prefix string,
	destination ui.Printer,
	source io.Reader,
	includeLineNumbers bool,
) (n int64, err error) {
	delimiterString := string([]byte{delimiter})

	bufferedReader, repool := pool.GetBufferedReader(source)
	defer repool()

	var (
		isEOF      bool
		lineNumber int
	)

	var stringBuilder strings.Builder

	for !isEOF {
		var rawLine string

		rawLine, err = bufferedReader.ReadString(delimiter)
		n1 := len(rawLine)
		n += int64(n1)

		if err != nil && !errors.IsEOF(err) {
			err = errors.Wrap(err)
			return n, err
		}

		if errors.IsEOF(err) {
			isEOF = true
			err = nil

			if n1 == 0 {
				break
			}
		}

		stringBuilder.WriteString(prefix)

		if includeLineNumbers {
			fmt.Fprintf(&stringBuilder, "%d:", lineNumber)
		}

		stringBuilder.WriteString(strings.TrimSuffix(rawLine, delimiterString))

		destination.Print(stringBuilder.String())
		stringBuilder.Reset()

		lineNumber++
	}

	return n, err
}
