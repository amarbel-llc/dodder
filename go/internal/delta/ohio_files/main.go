package ohio_files

import (
	"bufio"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
)

func CopyFileWithTransform(
	src, dst string,
	delim byte,
	transform func(string) string,
) (err error) {
	if src == "" {
		return err
	}

	var fileSrc, fileDst *os.File

	if fileSrc, err = files.Open(src); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, fileSrc.Close)

	if fileDst, err = files.CreateExclusiveWriteOnly(dst); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, fileDst.Close)

	var bufferedReader *bufio.Reader
	var bufferedWriter *bufio.Writer

	{
		var repool interfaces.FuncRepool
		bufferedReader, repool = pool.GetBufferedReader(fileSrc)
		defer repool()
	}

	{
		var repool interfaces.FuncRepool
		bufferedWriter, repool = pool.GetBufferedWriter(fileDst)
		defer repool()
	}

	defer errors.DeferredFlusher(&err, bufferedWriter)

	isEOF := false

	for !isEOF {
		var line string
		line, err = bufferedReader.ReadString(delim)

		if err == io.EOF {
			isEOF = true
			err = nil
		} else if err != nil {
			err = errors.Wrap(err)
			return err
		}

		// TODO-P2 sterilize line
		bufferedWriter.WriteString(transform(line))
	}

	return err
}

func CopyFileLines(
	src, dst string,
) (err error) {
	return CopyFileWithTransform(
		src,
		dst,
		'\n',
		func(value string) string {
			return value
		},
	)
}
