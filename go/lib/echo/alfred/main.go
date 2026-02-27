package alfred

import (
	"io"
)

type Writer interface {
	WriteItem(i *Item)
	io.Closer
}
