package id_fmts

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/flags"
	"code.linenisgreat.com/dodder/go/internal/charlie/catgut"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

type tagsReader struct{}

func MakeTagsReader() (reader *tagsReader) {
	reader = &tagsReader{}

	return reader
}

func (reader *tagsReader) ReadStringFormat(
	tags ids.TagSetMutable,
	ringBuffer *catgut.RingBuffer,
) (n int64, err error) {
	var readable catgut.Slice

	if readable, err = ringBuffer.PeekUptoAndIncluding(
		'\n',
	); err != nil && err != io.EOF {
		err = errors.Wrap(err)
		return n, err
	}

	if readable.Len() == 1 {
		return n, err
	}

	seq := flags.SplitCommasAndTrimAndMake[ids.TagStruct](readable.String())

	for tag, iterr := range seq {
		if errors.Is(iterr, ids.ErrEmptyTag) {
			continue
		} else if iterr != nil {
			err = errors.Wrap(iterr)
			return n, err
		}

		tags.Add(tag)
	}

	n = int64(readable.Len())
	ringBuffer.AdvanceRead(readable.Len())

	return n, err
}
