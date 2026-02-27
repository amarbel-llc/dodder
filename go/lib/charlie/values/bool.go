package values

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Bool struct {
	wasSet bool
	value  bool
}

func MakeBool(v bool) Bool {
	return Bool{
		wasSet: true,
		value:  v,
	}
}

func (sv *Bool) Set(v string) (err error) {
	v = strings.ToLower(strings.TrimSpace(v))
	v1 := false

	switch v {
	case "", "t", "true", "y", "yes":
		v1 = true
	}

	sv.SetBool(v1)

	return err
}

func (sv *Bool) SetBool(v bool) {
	sv.wasSet = true
	sv.value = v
}

func (sv Bool) Bool() bool {
	return sv.value
}

func (sv Bool) String() string {
	return fmt.Sprintf("%t", sv.value)
}

func (a Bool) Equals(b Bool) bool {
	return a.value == b.value && a.wasSet && b.wasSet
}

func (a Bool) WasSet() bool {
	return a.wasSet
}

func (a *Bool) Reset() {
	a.value = false
	a.wasSet = false
}

func (a *Bool) ResetWith(b Bool) {
	a.wasSet = true
	a.value = b.value
}

func (a *Bool) MarshalBinary() ([]byte, error) {
	b := uint8(0)

	if a.value {
		b = 1
	}

	return []byte{b}, nil
}

func (a *Bool) UnmarshalBinary(b []byte) (err error) {
	if len(b) != 1 {
		err = errors.ErrorWithStackf("expected exactly 1 byte but got %d", b)
		return err
	}

	b1 := b[0]

	switch b1 {
	case 0:
		a.SetBool(false)

	case 1:
		a.SetBool(true)

	default:
		err = errors.ErrorWithStackf("unexpected value: %d", b1)
		return err
	}

	return err
}
