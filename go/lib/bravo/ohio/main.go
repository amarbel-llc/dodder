package ohio

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

type PipedReader interface {
	Close() (n int64, err error)
	io.Writer
}

type readFromDone struct {
	n   int64
	err error
}

type pipedReaderFrom struct {
	*io.PipeWriter
	ch chan readFromDone
}

func MakePipedReaderFrom(r io.ReaderFrom) PipedReader {
	var p pipedReaderFrom

	var pr *io.PipeReader
	pr, p.PipeWriter = io.Pipe()
	p.ch = make(chan readFromDone)

	go func() {
		var msg readFromDone

		if msg.n, msg.err = r.ReadFrom(pr); msg.err != nil {
			if !errors.IsEOF(msg.err) {
				pr.CloseWithError(msg.err)
			}
		}

		p.ch <- msg
	}()

	return p
}

func (p pipedReaderFrom) Close() (n int64, err error) {
	if p.PipeWriter == nil {
		return n, err
	}

	p.PipeWriter.Close()
	out := <-p.ch
	n = out.n
	err = out.err

	return n, err
}

type pipedDecoderFrom struct {
	*io.PipeWriter
	ch chan readFromDone
}

func MakePipedDecoder[O any](
	object O,
	decoder interfaces.DecoderFromBufferedReader[O],
) PipedReader {
	var p pipedDecoderFrom

	var pr *io.PipeReader
	pr, p.PipeWriter = io.Pipe()
	p.ch = make(chan readFromDone)

	go func() {
		var msg readFromDone

		bufferedReader, repoolBufferedReader := pool.GetBufferedReader(pr)
		defer repoolBufferedReader()

		if msg.n, msg.err = decoder.DecodeFrom(
			object,
			bufferedReader,
		); msg.err != nil {
			if !errors.IsEOF(msg.err) {
				pr.CloseWithError(msg.err)
			}
		}

		p.ch <- msg
	}()

	return p
}

func (p pipedDecoderFrom) Close() (n int64, err error) {
	if p.PipeWriter == nil {
		return n, err
	}

	p.PipeWriter.Close()
	out := <-p.ch
	n = out.n
	err = out.err

	return n, err
}
