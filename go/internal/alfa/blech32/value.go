package blech32

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO make generic
type Value struct {
	HRP  string // human-readable part
	Data []byte
}

func MakeValue(
	hrp string,
	data []byte,
) Value {
	return Value{
		HRP:  hrp,
		Data: data,
	}
}

func MakeValueWithExpectedHRP(
	expectedHRP string,
	input string,
) (value Value, err error) {
	if err = value.Set(input); err != nil {
		err = errors.Wrap(err)
		return value, err
	}

	if value.HRP != expectedHRP {
		err = errors.Errorf(
			"expected HRP %q but got %q",
			expectedHRP,
			value.HRP,
		)
		return value, err
	}

	return value, err
}

func (value Value) GetHRP() string {
	return value.HRP
}

func (value Value) GetData() []byte {
	return value.Data
}

func (value Value) String() string {
	var text []byte
	var err error

	if text, err = Encode(value.HRP, value.Data); err != nil {
		panic(errors.Wrap(err))
	}

	return string(text)
}

func (value *Value) Set(text string) (err error) {
	if len(text) == 0 {
		return err
	}

	if value.HRP, value.Data, err = DecodeString(text); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (value Value) MarshalText() (text []byte, err error) {
	if len(value.Data) == 0 {
		return text, err
	}

	if text, err = Encode(value.HRP, value.Data); err != nil {
		err = errors.Wrap(err)
		return text, err
	}

	return text, err
}

func (value *Value) UnmarshalText(text []byte) (err error) {
	if len(text) == 0 {
		return err
	}

	if value.HRP, value.Data, err = DecodeString(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (value Value) WriteToMerkleId(
	merkleId domain_interfaces.MarklIdMutable,
) (err error) {
	if err = merkleId.SetMarklId(value.HRP, value.Data); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
