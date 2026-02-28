package markl

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
)

type Lock[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
] struct {
	Key   KEY
	Value Id
}

func MakeLock[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
]() Lock[KEY, KEY_PTR] {
	return Lock[KEY, KEY_PTR]{}
}

func MakeLockWith[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](key KEY, value domain_interfaces.MarklId) Lock[KEY, KEY_PTR] {
	lock := MakeLock[KEY, KEY_PTR]()

	lock.GetKeyMutable().ResetWith(key)

	if value != nil {
		lock.Value.ResetWithMarklId(value)
	}

	return lock
}

var _ interfaces.Resetable = &Lock[values.String, *values.String]{}

func (tuple Lock[KEY, KEY_PTR]) GetKey() KEY {
	return tuple.Key
}

func (tuple *Lock[KEY, KEY_PTR]) GetKeyMutable() KEY_PTR {
	return KEY_PTR(&tuple.Key)
}

func (tuple Lock[KEY, KEY_PTR]) GetValue() domain_interfaces.MarklId {
	return tuple.Value
}

func (tuple *Lock[KEY, KEY_PTR]) GetValueMutable() domain_interfaces.MarklIdMutable {
	return &tuple.Value
}

func (tuple *Lock[KEY, KEY_PTR]) Reset() {
	tuple.GetKeyMutable().Reset()
	tuple.Value.Reset()
}

func (tuple *Lock[KEY, KEY_PTR]) ResetWith(
	other Lock[KEY, KEY_PTR],
) {
	tuple.GetKeyMutable().ResetWith(other.Key)
	tuple.Value.ResetWithMarklId(other.Value)
}

func (tuple Lock[KEY, KEY_PTR]) IsEmpty() bool {
	return tuple.Key.IsEmpty() && tuple.Value.IsEmpty()
}

func (tuple Lock[KEY, KEY_PTR]) Equals(
	other Lock[KEY, KEY_PTR],
) bool {
	if tuple.Key.String() != other.Key.String() {
		return false
	}

	if !Equals(tuple.Value, other.Value) {
		return false
	}

	return true
}

func LockEquals[
	KEY interfaces.Value,
	KEY_PTR interfaces.ValuePtr[KEY],
](left, right domain_interfaces.Lock[KEY, KEY_PTR]) bool {
	if left.GetKey().String() != right.GetKey().String() {
		return false
	}

	if !Equals(left.GetValue(), right.GetValue()) {
		return false
	}

	return true
}
