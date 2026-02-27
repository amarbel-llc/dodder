package env_dir

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type RelativePath interface {
	Rel(string) string
}

func (env env) MakeRelativePathStringFormatWriter() interfaces.StringEncoderTo[string] {
	return relativePathStringFormatWriter(env)
}

type relativePathStringFormatWriter env

func (formatter relativePathStringFormatWriter) EncodeStringTo(
	path string,
	writer interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var n1 int

	{
		// if p, err = filepath.Rel(s.cwd, p); err != nil {
		// 	err = errors.Wrap(err)
		// 	return
		// }

		p1, _ := filepath.Rel(formatter.xdgInitArgs.Cwd, path)

		if p1 != "" {
			path = p1
		}
	}

	n1, err = writer.WriteString(path)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
