package string_format_writer

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

type Field struct {
	ColorType
	Separator rune
	// TODO switch to using io.StringWriter instead of key and value
	Key, Value         string
	DisableValueQuotes bool
	NoTruncate         bool
	NeedsNewline       bool
}

type BoxHeader struct {
	Value        string
	RightAligned bool
}

type HeaderWriter[T any] interface {
	WriteBoxHeader(*BoxHeader, T) error
}

type Box struct {
	Header                   BoxHeader
	Contents                 collections_slice.Slice[Field]
	Trailer                  collections_slice.Slice[Field]
	EachFieldOnANewline      bool
	IdFieldsSeparatedByLines bool
}

type boxStringEncoder struct {
	ColorOptions
	truncate CliFormatTruncation
	rightAligned
}

func MakeBoxStringEncoder(
	truncate CliFormatTruncation,
	co ColorOptions,
) *boxStringEncoder {
	return &boxStringEncoder{
		truncate:     truncate,
		ColorOptions: co,
	}
}

func (encoder *boxStringEncoder) EncodeStringTo(
	box Box,
	writer interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var n1 int64
	var n2 int

	separatorSameLine := " "
	separatorNextLine := "\n" + StringIndentWithSpace

	if box.Header.Value != "" {
		headerWriter := writer

		if box.Header.RightAligned {
			headerWriter = rightAligned2{writer}
		}

		n2, err = headerWriter.WriteString(box.Header.Value)
		n += int64(n2)

		if err != nil {
			err = errors.Wrapf(err, "Headers: %#v", box.Header)
			return n, err
		}
	}

	n1, err = encoder.writeStringFormatField(
		writer,
		Field{
			Value:     "[",
			ColorType: ColorTypeNormal,
		},
	)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	for i, field := range box.Contents {
		if i > 0 {
			if field.NeedsNewline {
				n2, err = writer.WriteString(separatorNextLine)
				n += int64(n2)

				if err != nil {
					err = errors.Wrap(err)
					return n, err
				}
			} else {
				n2, err = fmt.Fprint(writer, separatorSameLine)
				n += int64(n2)

				if err != nil {
					err = errors.Wrap(err)
					return n, err
				}
			}
		}

		n1, err = encoder.writeStringFormatField(writer, field)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	if separatorSameLine == "\n" {
		n2, err = writer.WriteString(separatorSameLine)
		n += int64(n2)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	closingBracket := "]"

	if len(box.Trailer) > 0 && false {
		closingBracket = "\n" + StringIndent + " ]"
	}

	n1, err = encoder.writeStringFormatField(
		writer,
		Field{
			Value:     closingBracket,
			ColorType: ColorTypeNormal,
		},
	)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	for _, field := range box.Trailer {
		n2, err = fmt.Fprint(writer, separatorSameLine)
		n += int64(n2)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n1, err = encoder.writeStringFormatField(writer, field)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}

func (f *boxStringEncoder) writeStringFormatField(
	w interfaces.WriterAndStringWriter,
	field Field,
) (n int64, err error) {
	var n1 int

	if field.Key != "" {
		if field.Separator == '\x00' {
			field.Separator = '='
		}

		n1, err = fmt.Fprintf(w, "%s%c", field.Key, field.Separator)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	preColor, postColor, ellipsis := field.ColorType, colorReset, ""

	if f.OffEntirely {
		preColor, postColor = "", ""
	}

	trunc := f.truncate

	if trunc == CliFormatTruncation66CharEllipsis {
		trunc = 66
	}

	if !field.NoTruncate && trunc > 0 && len(field.Value) > int(trunc) {
		field.Value = field.Value[:trunc+1]
		ellipsis = "…"
	}

	format := "%s%s%s%s"

	if (strings.ContainsRune(field.Value, ' ') || field.ColorType == ColorTypeUserData) &&
		!field.DisableValueQuotes {
		format = "%s%q%s%s"
	}

	n1, err = fmt.Fprintf(w, format, preColor, field.Value, postColor, ellipsis)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
