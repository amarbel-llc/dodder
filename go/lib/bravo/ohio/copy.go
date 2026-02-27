package ohio

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

func CopyBuffered(dst io.Writer, src io.Reader) (written int64, err error) {
	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(src)
	defer repoolBufferedReader()

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(dst)
	defer repoolBufferedWriter()
	defer errors.DeferredFlusher(&err, bufferedWriter)

	if written, err = io.Copy(bufferedWriter, bufferedReader); err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func CopyReaderAtToWriter(
	readerAt io.ReaderAt,
	writer io.Writer,
	cursor Cursor,
) (err error) {
	buffer := make([]byte, 32*1024)
	var written int64

	for written < cursor.ContentLength {
		toRead := int64(len(buffer))
		if remaining := cursor.ContentLength - written; remaining < toRead {
			toRead = remaining
		}

		n, err := readerAt.ReadAt(buffer[:toRead], cursor.Offset+written)
		if n > 0 {
			if _, werr := writer.Write(buffer[:n]); werr != nil {
				return werr
			}
			written += int64(n)
		}
		if err != nil {
			if err == io.EOF && written == cursor.ContentLength {
				return nil
			}
			return err
		}
	}
	return nil
}
