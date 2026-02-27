package thyme

import (
	"encoding/gob"
	"strconv"
	tyme "time"

	chai "github.com/brandondube/tai"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/values"
	"code.linenisgreat.com/dodder/go/lib/charlie/collections_value"
)

const (
	Epoch   = 1660007128
	RFC3339 = tyme.RFC3339
	// FormatDateTai  = "%y-%m-%d %H:%M"
)

var (
	Date  = tyme.Date
	Parse = tyme.Parse
)

func init() {
	ui.TodoP1("refactor into common")
	gob.Register(Time{})
	collections_value.RegisterGobValue[Time](nil)
}

type Time struct {
	time
}

type time = tyme.Time

func Now() Time {
	return Time{
		time: tyme.Now(),
	}
}

func Tyme(t tyme.Time) Time {
	return Time{
		time: t,
	}
}

func TimeWithIndex(t1 Time, n int) (t2 Time) {
	t2 = t1
	t2.Add(tyme.Nanosecond * tyme.Duration(n))

	return t2
}

func (t Time) GetTime() time {
	return t.time
}

func (t Time) String() string {
	return strconv.FormatInt(t.Unix(), 10)
}

func (t *Time) Set(v string) (err error) {
	var n int64

	if n, err = strconv.ParseInt(v, 10, 64); err != nil {
		err = errors.Wrapf(err, "failed to parse time: %s", v)
		return err
	}

	t.time = tyme.Unix(n, 0)

	return err
}

func (t Time) MarshalBinary() (text []byte, err error) {
	text = []byte(t.String())

	return text, err
}

func (t *Time) UnmarshalBinary(text []byte) (err error) {
	if err = t.Set(string(text)); err != nil {
		return err
	}

	return err
}

func (t Time) MarshalText() (text []byte, err error) {
	ui.Err().Printf(t.String())
	text = []byte(t.String())

	return text, err
}

func (t *Time) UnmarshalText(text []byte) (err error) {
	if err = t.Set(string(text)); err != nil {
		return err
	}

	return err
}

func (t Time) EqualsAny(t1 any) bool {
	return values.Equals(t, t1)
}

func (t Time) Equals(t1 Time) bool {
	return t.Unix() == t1.Unix()
}

func (a Time) EqualsSansIndex(b Time) bool {
	a1 := chai.FromTime(a.time)
	a1.Asec = 0

	b1 := chai.FromTime(b.time)
	b1.Asec = 0

	return a1.Eq(b1)
}

func (t Time) Less(t1 Time) bool {
	return t.Unix() < t1.Unix()
}

func (t *Time) Reset() {
	t.time = tyme.Time{}
}

func (t Time) IsEmpty() bool {
	return t.time.IsZero()
}
