package ids

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"code.linenisgreat.com/dodder/go/src/alfa/cmp"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/src/charlie/delim_io"
	"code.linenisgreat.com/dodder/go/src/charlie/doddish"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/src/delta/thyme"
	chai "github.com/brandondube/tai"
)

type tai = chai.TAI

func init() {
	register(Tai{})
	collections_value.RegisterGobValue[Tai](nil)
}

type Tai struct {
	wasSet bool
	tai
}

func NowTai() Tai {
	return Tai{
		wasSet: true,
		tai:    chai.Now(),
	}
}

func TaiFromTime(t1 thyme.Time) (t2 Tai) {
	t2 = TaiFromTimeWithIndex(t1, 0)
	return t2
}

func TaiFromTime1(t1 time.Time) (t2 Tai) {
	t2.wasSet = true
	t2.tai = chai.FromTime(t1)

	return t2
}

func TaiFromTimeWithIndex(t1 thyme.Time, n int) (t2 Tai) {
	t2.wasSet = true
	t2.tai = chai.FromTime(t1.GetTime())
	t2.Asec += int64(n * chai.Attosecond)

	return t2
}

func (tai Tai) AsTime() (t1 thyme.Time) {
	t1 = thyme.Tyme(tai.tai.AsTime().Local())
	return t1
}

func (tai Tai) Before(b Tai) bool {
	return tai.tai.Before(b.tai)
}

func (tai Tai) After(b Tai) bool {
	return tai.tai.After(b.tai)
}

func (tai Tai) GetGenre() domain_interfaces.Genre {
	return genres.InventoryList
}

func (tai Tai) Parts() [3]string {
	a := strings.TrimRight(fmt.Sprintf("%018d", tai.Asec), "0")

	if a == "" {
		a = "0"
	}

	return [3]string{strconv.FormatInt(tai.Sec, 10), ".", a}
}

func (tai Tai) GetObjectIdString() string {
	return tai.String()
}

func (tai Tai) String() (v string) {
	a := strings.TrimRight(fmt.Sprintf("%018d", tai.Asec), "0")

	if a == "" {
		a = "0"
	}

	v = fmt.Sprintf("%s.%s", strconv.FormatInt(tai.Sec, 10), a)

	// if v == "0.0" {
	// 	panic("empty tai")
	// }

	return v
}

func (tai Tai) StringDefaultFormat() string {
	f := string_format_writer.StringFormatDateTime + ".000000000"
	return tai.Format(f)
}

func (tai Tai) StringBoxFormat() string {
	return tai.Format(string_format_writer.StringFormatDateTime)
}

func (tai Tai) Format(v string) string {
	return tai.AsTime().Format(v)
}

func (tai *Tai) SetFromRFC3339(v string) (err error) {
	tai.wasSet = true

	var t1 time.Time

	if t1, err = thyme.Parse(thyme.RFC3339, v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	*tai = TaiFromTime1(t1)

	return err
}

func (tai *Tai) Set(value string) (err error) {
	tai.wasSet = true

	reader, repool := pool.GetStringReader(value)
	defer repool()

	delimiterReader, delimRepool := delim_io.Make('.', reader)
	defer delimRepool()

	idx := 0
	var val string

	var isEOF bool

	for !isEOF {
		val, err = delimiterReader.ReadOneString()

		if err == io.EOF {
			isEOF = true
			err = nil
		} else if err != nil {
			err = errors.Wrap(err)
			return err
		}

		switch idx {
		case 0:
			val = strings.TrimSpace(val)

			if len(val) == 0 {
				break
			}

			if tai.Sec, err = strconv.ParseInt(val, 10, 64); err != nil {
				err = errors.Wrapf(err, "failed to parse Sec time: %s", value)
				return err
			}

		case 1:
			val = strings.TrimSpace(val)
			val = strings.TrimRight(val, "0")

			if val == "" {
				break
			}

			var pre int64

			if pre, err = strconv.ParseInt(val, 10, 64); err != nil {
				err = errors.Wrapf(err, "failed to parse Asec time: %s", val)
				return err
			}

			tai.Asec = pre * int64(math.Pow10(18-len(val)))

		default:
			err = errors.ErrorWithStackf(
				"expected no more elements but got %s",
				val,
			)
			return err
		}

		idx++
	}

	return err
}

func (tai Tai) IsZero() (ok bool) {
	ok = (tai.Sec == 0 && tai.Asec == 0) || !tai.wasSet
	return ok
}

func (tai Tai) IsEmpty() (ok bool) {
	ok = tai.IsZero()
	return ok
}

func (tai Tai) GetTai() Tai {
	return tai
}

func (tai *Tai) Reset() {
	tai.Sec = 0
	tai.Asec = 0
	tai.wasSet = false
}

func (tai *Tai) ResetWith(b Tai) {
	tai.Sec = b.Sec
	tai.Asec = b.Asec
	tai.wasSet = b.wasSet
}

func (tai Tai) WriteTo(w io.Writer) (n int64, err error) {
	b := make([]byte, binary.MaxVarintLen64*2)
	binary.PutVarint(b[:binary.MaxVarintLen64], tai.Sec)
	binary.PutVarint(b[binary.MaxVarintLen64:], tai.Asec)
	var n1 int
	n1, err = ohio.WriteAllOrDieTrying(w, b)
	n += int64(n1)
	return n, err
}

func (tai *Tai) ReadFrom(r io.Reader) (n int64, err error) {
	b := make([]byte, binary.MaxVarintLen64*2)

	var n1 int
	n1, err = ohio.ReadAllOrDieTrying(r, b)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	tai.wasSet = true
	tai.Sec, _ = binary.Varint(b[:binary.MaxVarintLen64])
	tai.Asec, _ = binary.Varint(b[binary.MaxVarintLen64:])

	return n, err
}

func (tai Tai) MarshalText() (text []byte, err error) {
	ui.Err().Printf(tai.String())
	text = []byte(tai.String())

	return text, err
}

func (tai *Tai) UnmarshalText(text []byte) (err error) {
	if err = tai.Set(string(text)); err != nil {
		return err
	}

	return err
}

func (tai Tai) MarshalBinary() (text []byte, err error) {
	return tai.AppendBinary(nil)
}

func (tai Tai) AppendBinary(text []byte) ([]byte, error) {
	return append(text, []byte(tai.String())...), nil
}

func (tai *Tai) UnmarshalBinary(text []byte) (err error) {
	if err = tai.Set(string(text)); err != nil {
		return err
	}

	return err
}

func (tai Tai) Equals(t1 Tai) bool {
	if !tai.Eq(t1.tai) {
		return false
	}

	return true
}

func (tai Tai) Less(t1 Tai) bool {
	return tai.Before(t1)
}

func (tai Tai) SortCompare(t1 Tai) cmp.Result {
	if tai.Equals(t1) {
		return cmp.Equal
	} else if tai.Before(t1) {
		return cmp.Less
	} else {
		return cmp.Greater
	}
}

func MakeTaiRFC3339Value(t Tai) *TaiRFC3339Value {
	t1 := TaiRFC3339Value(t)
	return &t1
}

type TaiRFC3339Value Tai

func (t *TaiRFC3339Value) Set(v string) (err error) {
	t.wasSet = true

	var t1 time.Time

	if t1, err = thyme.Parse(thyme.RFC3339, v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	*t = TaiRFC3339Value(TaiFromTime1(t1))

	return err
}

func (t *TaiRFC3339Value) String() string {
	// TODO figure out why the pointer needs to be converted to Tai to execute
	// correctly
	return Tai(*t).Format(thyme.RFC3339)
}

func (id Tai) ToType() TypeStruct {
	panic(errors.Err405MethodNotAllowed)
}

func (id Tai) ToSeq() doddish.Seq {
	parts := id.Parts()

	return doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(parts[0]),
		},
		doddish.Token{
			Type:     doddish.TokenTypeOperator,
			Contents: []byte{'.'},
		},
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(parts[2]),
		},
	}
}
