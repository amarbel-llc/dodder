package stream_index_fixed

import (
	"io"
	"math"
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/charlie/page_id"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type overflowWriter struct {
	mu     sync.Mutex
	file   *os.File
	offset int32
}

func makeOverflowWriter(pageId page_id.PageId) (result *overflowWriter, err error) {
	ovId := overflowPageId(pageId)
	path := ovId.Path()

	var file *os.File

	if file, err = os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0o644,
	); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	var info os.FileInfo

	if info, err = file.Stat(); err != nil {
		err = errors.Wrap(err)
		file.Close()
		return nil, err
	}

	if info.Size() > math.MaxInt32 {
		file.Close()
		err = errors.Errorf("overflow file too large: %d", info.Size())
		return nil, err
	}

	result = &overflowWriter{
		file:   file,
		offset: int32(info.Size()),
	}

	return result, err
}

func makeOverflowWriterForTempFile(
	file *os.File,
) *overflowWriter {
	return &overflowWriter{
		file:   file,
		offset: 0,
	}
}

func (overflowWriter *overflowWriter) Write(data []byte) (offset int32, length uint16, err error) {
	if len(data) > math.MaxUint16 {
		err = errOverflowTooLarge
		return offset, length, err
	}

	overflowWriter.mu.Lock()
	defer overflowWriter.mu.Unlock()

	offset = overflowWriter.offset
	length = uint16(len(data))

	if _, err = ohio.WriteAllOrDieTrying(overflowWriter.file, data); err != nil {
		err = errors.Wrap(err)
		return offset, length, err
	}

	overflowWriter.offset += int32(len(data))

	return offset, length, err
}

func (overflowWriter *overflowWriter) Close() error {
	if overflowWriter.file == nil {
		return nil
	}

	return overflowWriter.file.Close()
}

func (overflowWriter *overflowWriter) ReaderAt() io.ReaderAt {
	return overflowWriter.file
}

type overflowReaderAt struct {
	readerAt io.ReaderAt
}

func (overflowReaderAt *overflowReaderAt) ReadAt(
	offset int32,
	length uint16,
) (data []byte, err error) {
	data = make([]byte, length)

	if _, err = overflowReaderAt.readerAt.ReadAt(data, int64(offset)); err != nil {
		if err == io.EOF {
			err = nil
		} else {
			err = errors.Wrap(err)
			return nil, err
		}
	}

	return data, err
}

func overflowPageId(pageId page_id.PageId) page_id.PageId {
	return page_id.PageId{
		Prefix: "Overflow",
		Dir:    pageId.Dir,
		Index:  pageId.Index,
	}
}
