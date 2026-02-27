package descriptions

import (
	"io"
	"strings"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/internal/delta/doddish"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

// TODO-P1 move to catgut.String
type Description struct {
	wasSet bool
	value  string
}

func Make(v string) Description {
	return Description{
		wasSet: true,
		value:  v,
	}
}

func (description Description) String() string {
	return description.value
}

func (description Description) StringWithoutNewlines() string {
	return strings.ReplaceAll(description.value, "\n", " ")
}

func (description *Description) TodoSetManyCatgutStrings(
	vs ...*catgut.String,
) (err error) {
	var s catgut.String

	if _, err = s.Append(vs...); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return description.Set(s.String())
}

func (description *Description) TodoSetSlice(v catgut.Slice) (err error) {
	return description.Set(v.String())
}

func (description *Description) readFromRuneScannerAfterNewline(
	rs *doddish.Scanner,
	sb *strings.Builder,
) (err error) {
	if !rs.ConsumeSpacesOrErrorOnFalse() {
		return err
	}

	var r rune

	r, _, err = rs.ReadRune()
	isEOF := err == io.EOF

	if err != nil && !isEOF {
		err = errors.Wrap(err)
		return err
	}

	if r == '-' || r == '%' || r == '#' {
		if err = rs.UnreadRune(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	sb.WriteRune(' ')
	sb.WriteRune(r)

	if !rs.ConsumeSpacesOrErrorOnFalse() {
		return err
	}

	if err = description.readFromRuneScannerOrdinary(rs, sb); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (description *Description) readFromRuneScannerOrdinary(
	rs *doddish.Scanner,
	sb *strings.Builder,
) (err error) {
	for {
		var r rune

		r, _, err = rs.ReadRune()
		isEOF := err == io.EOF

		if err != nil && !isEOF {
			err = errors.Wrap(err)
			return err
		}

		if r == '\n' {
			if err = description.readFromRuneScannerAfterNewline(rs, sb); err != nil {
				err = errors.Wrap(err)
				return err
			}

			break
		}

		if isEOF {
			err = nil
			if r != utf8.RuneError {
				sb.WriteRune(r)
			}

			break
		}

		sb.WriteRune(r)
	}

	return err
}

func (description *Description) ReadFromBoxScanner(rs *doddish.Scanner) (err error) {
	var sb strings.Builder

	if err = description.readFromRuneScannerOrdinary(rs, &sb); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = description.Set(sb.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (description *Description) Set(value string) (err error) {
	description.wasSet = true

	v1 := strings.TrimSpace(value)

	if v0 := description.String(); v0 != "" && v0 != v1 {
		description.value = v0 + " " + v1
	} else {
		description.value = v1
	}

	return err
}

func (description Description) WasSet() bool {
	return description.wasSet
}

func (description *Description) Reset() {
	description.wasSet = false
	description.value = ""
}

func (description *Description) ResetWith(other Description) {
	description.wasSet = other.wasSet
	description.value = other.value
}

func (description Description) IsEmpty() bool {
	return description.value == ""
}

func (description Description) Equals(b Description) (ok bool) {
	// if !a.wasSet {
	// 	return false
	// }

	return description.value == b.value
}

func (description Description) Less(b Description) (ok bool) {
	return description.value < b.value
}

func (description Description) MarshalBinary() (text []byte, err error) {
	text = []byte(description.value)
	return text, err
}

func (description *Description) UnmarshalBinary(text []byte) (err error) {
	description.wasSet = true
	description.value = string(text)
	return err
}

func (description Description) MarshalText() (text []byte, err error) {
	text = []byte(description.value)
	return text, err
}

func (description *Description) UnmarshalText(text []byte) (err error) {
	description.wasSet = true
	description.value = string(text)
	return err
}
