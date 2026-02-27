package markl

import (
	"bytes"
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func MakeLockCoder[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](
	lock domain_interfaces.Lock[KEY, KEY_PTR],
	requireValue bool,
) lockBinaryMarshaler[KEY, KEY_PTR] {
	return lockBinaryMarshaler[KEY, KEY_PTR]{
		requireValue: requireValue,
		lock:         lock,
	}
}

func MakeMutableLockCoder[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](
	lock domain_interfaces.LockMutable[KEY, KEY_PTR],
	requireValue bool,
) mutableLockBinaryMarshaler[KEY, KEY_PTR] {
	return mutableLockBinaryMarshaler[KEY, KEY_PTR]{
		requireValue: requireValue,
		lock:         lock,
	}
}

func MakeLockCoderValueNotRequired[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](
	lock domain_interfaces.Lock[KEY, KEY_PTR],
) lockBinaryMarshaler[KEY, KEY_PTR] {
	return MakeLockCoder[KEY, KEY_PTR](lock, false)
}

func MakeMutableLockCoderValueNotRequired[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](
	lock domain_interfaces.LockMutable[KEY, KEY_PTR],
) mutableLockBinaryMarshaler[KEY, KEY_PTR] {
	return MakeMutableLockCoder(lock, false)
}

func MakeMutableLockCoderValueRequired[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](
	lock domain_interfaces.LockMutable[KEY, KEY_PTR],
) mutableLockBinaryMarshaler[KEY, KEY_PTR] {
	return MakeMutableLockCoder(lock, true)
}

type lockBinaryMarshaler[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
] struct {
	requireValue bool
	lock         domain_interfaces.Lock[KEY, KEY_PTR]
}

type mutableLockBinaryMarshaler[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
] struct {
	requireValue bool
	lock         domain_interfaces.LockMutable[KEY, KEY_PTR]
}

func (marshaler lockBinaryMarshaler[KEY, KEY_PTR]) String() string {
	lock := marshaler.lock

	if marshaler.requireValue && lock.GetValue().IsEmpty() {
		panic(fmt.Sprintf("marshaler requires non empty lock for %q", lock.GetKey()))
	} else if lock.GetValue().IsEmpty() {
		return lock.GetKey().String()
	} else {
		return fmt.Sprintf("%s@%s", lock.GetKey(), lock.GetValue())
	}
}

func (marshaler mutableLockBinaryMarshaler[KEY, KEY_PTR]) String() string {
	lock := marshaler.lock

	if marshaler.requireValue && lock.GetValue().IsEmpty() {
		panic(fmt.Sprintf("marshaler requires non empty lock for %q", lock.GetKey()))
	} else if lock.GetValue().IsEmpty() {
		return lock.GetKey().String()
	} else {
		return fmt.Sprintf("%s@%s", lock.GetKey(), lock.GetValue())
	}
}

func (marshaler mutableLockBinaryMarshaler[KEY, KEY_PTR]) Set(
	value string,
) (err error) {
	lock := marshaler.lock

	key := lock.GetKeyMutable()

	left, right, ok := strings.Cut(value, "@")

	if !ok {
		if err = key.Set(value); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	if err = key.Set(left); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = lock.GetValueMutable().Set(right); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (marshaler mutableLockBinaryMarshaler[KEY, KEY_PTR]) MarshalBinary() (data []byte, err error) {
	return marshaler.AppendBinary(nil)
}

func (marshaler mutableLockBinaryMarshaler[KEY, KEY_PTR]) AppendBinary(
	bites []byte,
) ([]byte, error) {
	bites = fmt.Append(bites, marshaler.lock.GetKey().String())

	if marshaler.lock.GetValue().IsEmpty() {
		var err error

		if marshaler.requireValue {
			err = errors.Errorf("empty type signature for %q", marshaler.lock.GetKey())
		}

		return bites, err
	}

	bites = append(bites, '\x00')
	formatId := marshaler.lock.GetValue().GetMarklFormat().GetMarklFormatId()
	bites = fmt.Append(bites, formatId)
	bites = append(bites, '\x00')
	bites = append(bites, marshaler.lock.GetValue().GetBytes()...)

	return bites, nil
}

func (marshaler mutableLockBinaryMarshaler[KEY, KEY_PTR]) UnmarshalBinary(
	bites []byte,
) (err error) {
	if len(bites) == 0 {
		return err
	}

	var formatAndBytes []byte

	{
		var key []byte
		var ok bool

		key, formatAndBytes, ok = bytes.Cut(bites, []byte{'\x00'})

		if !ok {
			if err = marshaler.lock.GetKeyMutable().Set(string(bites)); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		}

		if err = marshaler.lock.GetKeyMutable().Set(string(key)); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	{
		format, valueBytes, ok := bytes.Cut(formatAndBytes, []byte{'\x00'})

		if !ok {
			err = errors.Errorf("expected empty byte between format id and id bytes")
			return err
		}

		id := marshaler.lock.GetValueMutable()
		id.Reset()

		if err = id.SetMarklId(string(format), valueBytes); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
