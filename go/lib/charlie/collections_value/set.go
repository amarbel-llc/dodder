package collections_value

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type Set[
	T any,
] struct {
	K interfaces.StringKeyer[T]
	E map[string]T
}

func (set Set[T]) AllKeys() interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for k := range set.E {
			if !yield(k) {
				break
			}
		}
	}
}

func (set Set[T]) All() interfaces.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range set.E {
			if !yield(e) {
				break
			}
		}
	}
}

func (set Set[T]) Len() int {
	if set.E == nil {
		return 0
	}

	return len(set.E)
}

func (set Set[T]) Key(e T) string {
	return set.K.GetKey(e)
}

func (set Set[T]) Get(k string) (e T, ok bool) {
	e, ok = set.E[k]

	return e, ok
}

func (set Set[T]) Any() (e T) {
	for _, e1 := range set.E {
		return e1
	}

	return e
}

func (set Set[T]) ContainsKey(k string) (ok bool) {
	if k == "" {
		return ok
	}

	_, ok = set.E[k]

	return ok
}

func (set Set[T]) Contains(e T) (ok bool) {
	return set.ContainsKey(set.Key(e))
}

// TODO remove in favor of iterators
func (set Set[T]) EachKey(wf interfaces.FuncIterKey) (err error) {
	for v := range set.E {
		if err = wf(v); err != nil {
			if errors.IsStopIteration(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}
	}

	return err
}

func (set Set[T]) Add(v T) (err error) {
	set.E[set.Key(v)] = v
	return err
}

func (set Set[T]) CloneSetLike() interfaces.Set[T] {
	return set
}

func (set Set[T]) CloneMutableSetLike() interfaces.SetMutable[T] {
	clone := MakeMutableSet[T](set.K, set.Len(), set.All())
	return clone
}
