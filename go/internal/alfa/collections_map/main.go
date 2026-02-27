package collections_map

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

type Map[KEY comparable, VALUE any] map[KEY]VALUE

var _ interfaces.Collection[string] = Map[string, string]{}

func (mapp Map[KEY, VALUE]) Len() int {
	return len(mapp)
}

func (mapp Map[KEY, VALUE]) Any() (key KEY) {
	for someKey := range mapp {
		key = someKey
		break
	}

	return key
}

func (mapp Map[KEY, VALUE]) All() interfaces.Seq[KEY] {
	return func(yield func(KEY) bool) {
		for key := range mapp {
			if !yield(key) {
				break
			}
		}
	}
}

func (mapp Map[KEY, VALUE]) AllPairs() interfaces.Seq2[KEY, VALUE] {
	return func(yield func(KEY, VALUE) bool) {
		for key, value := range mapp {
			if !yield(key, value) {
				break
			}
		}
	}
}

func (mapp Map[KEY, VALUE]) Reset() {
	clear(mapp)
}

func (mapp Map[KEY, VALUE]) Get(key KEY) (value VALUE, ok bool) {
	value, ok = mapp[key]
	return value, ok
}

func (mapp Map[KEY, VALUE]) Set(key KEY, value VALUE) {
	mapp[key] = value
}

func (mapp Map[KEY, VALUE]) ResetWith(other Map[KEY, VALUE]) {
	mapp.ResetWithSeq(other.AllPairs())
}

func (mapp Map[KEY, VALUE]) ResetWithSeq(other interfaces.Seq2[KEY, VALUE]) {
	mapp.Reset()

	for key, value := range other {
		mapp.Set(key, value)
	}
}
