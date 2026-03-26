package string_format_writer

import (
	"code.linenisgreat.com/dodder/go/internal/_/fields"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func colorForType(t fields.Type) string {
	switch t {
	case fields.TypeId:
		return colorBlue
	case fields.TypeHash:
		return colorItalic
	case fields.TypeError:
		return colorRed
	case fields.TypeType:
		return colorYellow
	case fields.TypeUserData:
		return colorCyan
	case fields.TypeHeading:
		return colorRed
	default:
		return colorNone
	}
}

type color[T any] struct {
	options            ColorOptions
	color              fields.Type
	stringFormatWriter interfaces.StringEncoderTo[T]
}

func MakeColor[T any](
	o ColorOptions,
	fsw interfaces.StringEncoderTo[T],
	c fields.Type,
) interfaces.StringEncoderTo[T] {
	if o.OffEntirely {
		return fsw
	} else {
		return &color[T]{
			color:              c,
			stringFormatWriter: fsw,
		}
	}
}

func (f *color[T]) EncodeStringTo(
	e T,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	if f.options.OffEntirely {
		return f.stringFormatWriter.EncodeStringTo(e, sw)
	}

	var n1 int

	n1, err = sw.WriteString(colorForType(f.color))
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var n2 int64
	n2, err = f.stringFormatWriter.EncodeStringTo(e, sw)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = sw.WriteString(string(colorReset))
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
