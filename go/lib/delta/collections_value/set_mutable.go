package collections_value

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type MutableSet[
	T any,
] struct {
	K interfaces.StringKeyer[T]
	E map[string]T
}

func (s MutableSet[T]) Len() int {
	if s.E == nil {
		return 0
	}

	return len(s.E)
}

func (s MutableSet[T]) All() interfaces.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range s.E {
			if !yield(e) {
				break
			}
		}
	}
}

func (s MutableSet[T]) AllKeys() interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for k := range s.E {
			if !yield(k) {
				break
			}
		}
	}
}

func (s MutableSet[T]) Key(e T) string {
	return s.K.GetKey(e)
}

func (s MutableSet[T]) Get(k string) (e T, ok bool) {
	e, ok = s.E[k]

	return e, ok
}

func (s MutableSet[T]) ContainsKey(k string) (ok bool) {
	if k == "" {
		return ok
	}

	_, ok = s.E[k]

	return ok
}

func (s MutableSet[T]) Contains(e T) (ok bool) {
	return s.ContainsKey(s.Key(e))
}

func (s MutableSet[T]) Any() (v T) {
	for _, v1 := range s.E {
		v = v1
		break
	}

	return v
}

func (s MutableSet[T]) DelKey(k string) (err error) {
	delete(s.E, k)
	return err
}

func (s MutableSet[T]) Add(v T) (err error) {
	s.E[s.Key(v)] = v
	return err
}

func (s MutableSet[T]) EachKey(
	wf interfaces.FuncIterKey,
) (err error) {
	for v := range s.E {
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

func (a MutableSet[T]) Reset() {
	for k := range a.E {
		delete(a.E, k)
	}
}

func (a MutableSet[T]) CloneSetLike() interfaces.Set[T] {
	b := MakeSet[T](a.K)

	for k, v := range a.E {
		b.E[k] = v
	}

	return b
}

func (set MutableSet[T]) CloneMutableSetLike() interfaces.SetMutable[T] {
	return MakeMutableSet[T](set.K, set.Len(), set.All())
}
