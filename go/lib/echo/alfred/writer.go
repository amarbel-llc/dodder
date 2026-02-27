package alfred

import (
	"bufio"
	"encoding/json"
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type writer struct {
	wBuf        *bufio.Writer
	jsonEncoder *json.Encoder

	chItem chan *Item
	chDone chan struct{}

	afterFirstWrite bool

	ItemPool
}

func NewWriter(out io.Writer, itemPool ItemPool) (w *writer, err error) {
	w = &writer{
		wBuf:     bufio.NewWriter(out),
		chItem:   make(chan *Item),
		chDone:   make(chan struct{}),
		ItemPool: itemPool,
	}

	w.jsonEncoder = json.NewEncoder(w.wBuf)

	if err = w.open(); err != nil {
		err = errors.Wrap(err)
		return w, err
	}

	return w, err
}

func (w *writer) open() (err error) {
	if _, err = w.wBuf.WriteString(
		"{",
	); err != nil {
		close(w.chDone)
		err = errors.Wrap(err)
		return err
	}

	if _, err = w.wBuf.WriteString(
		`"cache": {"seconds": 1, "loosereload": true},`,
	); err != nil {
		close(w.chDone)
		err = errors.Wrap(err)
		return err
	}

	if _, err = w.wBuf.WriteString(
		`"items":[`,
	); err != nil {
		close(w.chDone)
		err = errors.Wrap(err)
		return err
	}

	go func() {
		for i := range w.chItem {
			// var err error

			if _ = w.writeItem(i); err != nil {
				// err = errors.Wrap(err)
				// TODO-P4
			}
		}

		close(w.chDone)
	}()

	return err
}

func (w *writer) WriteItem(i *Item) {
	w.chItem <- i
}

func (w *writer) setAfterFirstWrite() {
	w.afterFirstWrite = true
}

func (w *writer) writeItem(i *Item) (err error) {
	if i == nil {
		panic("item was nil")
	}

	defer w.setAfterFirstWrite()

	if w.afterFirstWrite {
		if _, err = io.WriteString(w.wBuf, ","); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = w.jsonEncoder.Encode(i); err != nil {
		err = errors.Wrap(err)
		return err
	}

	w.Put(i)

	return err
}

func (w *writer) Close() (err error) {
	close(w.chItem)
	<-w.chDone

	if _, err = w.wBuf.WriteString("]}\n"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = w.wBuf.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
