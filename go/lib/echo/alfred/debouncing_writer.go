package alfred

import (
	"fmt"
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type countedItem struct {
	*Item
	count int
}

type debouncingWriter struct {
	sync.RWMutex
	items map[string]countedItem
	out   io.Writer
}

func NewDebouncingWriter(out io.Writer) (w *debouncingWriter, err error) {
	w = &debouncingWriter{
		items: make(map[string]countedItem),
		out:   out,
	}

	return w, err
}

func (w *debouncingWriter) WriteItem(i *Item) {
	w.RLock()
	entry := w.items[i.Uid]
	w.RUnlock()

	entry.Item = i
	entry.count++

	w.Lock()
	w.items[i.Uid] = entry
	w.Unlock()
}

func (w *debouncingWriter) Close() (err error) {
	var writer Writer

	if writer, err = NewWriter(w.out, MakeItemPool()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, i := range w.items {
		if i.Subtitle == "" {
			i.Subtitle = fmt.Sprintf("%d", i.count)
		} else {
			i.Subtitle = fmt.Sprintf("%d: %s", i.count, i.Subtitle)
		}

		writer.WriteItem(i.Item)
	}

	if err = writer.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
