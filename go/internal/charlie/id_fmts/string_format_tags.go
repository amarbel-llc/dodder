package id_fmts

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/alfa/flags"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
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

	// Headings are space-separated conjunction terms (cutting-garden RFC
	// 0015 / trellis): comma is disjunctive elsewhere in the query
	// grammar and is not accepted here, deliberately, with no legacy
	// fallback (dodder#374).
	seq := flags.SplitSpacesAndTrimAndMake[ids.TagStruct](readable.String())

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
