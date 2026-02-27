package collections_ptr

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type Set[
	T any,
	TPtr interfaces.Ptr[T],
] struct {
	K interfaces.StringKeyerPtr[T, TPtr]
	E map[string]TPtr
}

func (s Set[T, TPtr]) AllKeys() interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for k := range s.E {
			if !yield(k) {
				break
			}
		}
	}
}

func (s Set[T, TPtr]) All() interfaces.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range s.E {
			if !yield(*e) {
				break
			}
		}
	}
}

func (s Set[T, TPtr]) AllPtr() interfaces.Seq[TPtr] {
	return func(yield func(TPtr) bool) {
		for _, e := range s.E {
			if !yield(e) {
				break
			}
		}
	}
}

func (s Set[T, TPtr]) Len() int {
	if s.E == nil {
		return 0
	}

	return len(s.E)
}

func (s Set[T, TPtr]) Key(e T) string {
	return s.K.GetKey(e)
}

func (s Set[T, TPtr]) KeyPtr(e TPtr) string {
	return s.K.GetKeyPtr(e)
}

func (s Set[T, TPtr]) GetPtr(k string) (e TPtr, ok bool) {
	e, ok = s.E[k]

	return e, ok
}

func (s Set[T, TPtr]) Get(k string) (e T, ok bool) {
	var e1 TPtr

	if e1, ok = s.E[k]; ok {
		e = *e1
	}

	return e, ok
}

func (s Set[T, TPtr]) Any() (e T) {
	for _, e1 := range s.E {
		return *e1
	}

	return e
}

func (s Set[T, TPtr]) ContainsKey(k string) (ok bool) {
	if k == "" {
		return ok
	}

	_, ok = s.E[k]

	return ok
}

func (s Set[T, TPtr]) Contains(e T) (ok bool) {
	return s.ContainsKey(s.Key(e))
}

func (s Set[T, TPtr]) EachKey(wf interfaces.FuncIterKey) (err error) {
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

func (s Set[T, TPtr]) Add(v T) (err error) {
	s.E[s.Key(v)] = TPtr(&v)
	return err
}

func (s Set[T, TPtr]) AddPtr(v TPtr) (err error) {
	s.E[s.K.GetKeyPtr(v)] = v
	return err
}
