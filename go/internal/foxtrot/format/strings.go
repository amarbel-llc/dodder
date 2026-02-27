package format

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeFormatStringRightAligned(
	f string,
	args ...any,
) interfaces.FuncWriter {
	return func(w io.Writer) (n int64, err error) {
		f = fmt.Sprintf(f+" ", args...)

		diff := string_format_writer.LenStringMax + 1 - utf8.RuneCountInString(
			f,
		)

		if diff > 0 {
			f = strings.Repeat(" ", diff) + f
		}

		var n1 int

		if n1, err = io.WriteString(w, f); err != nil {
			n = int64(n1)
			err = errors.Wrap(err)
			return n, err
		}

		n = int64(n1)

		return n, err
	}
}
