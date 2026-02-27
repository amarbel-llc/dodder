package collections_ptr

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func MakeValueSetString[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringSetterPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...string,
) (s Set[ELEMENT, ELEMENT_PTR], err error) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	s.K = keyer

	for _, v := range es {
		var e ELEMENT
		e1 := ELEMENT_PTR(&e)

		if err = e1.Set(v); err != nil {
			err = errors.Wrap(err)
			return s, err
		}

		s.E[s.K.GetKeyPtr(e1)] = e1
	}

	return s, err
}

func MakeValueSetSeq[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringerPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	seq interfaces.Seq[ELEMENT_PTR],
	count int,
) (set Set[ELEMENT, ELEMENT_PTR]) {
	set.E = make(map[string]ELEMENT_PTR, count)

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	set.K = keyer

	for element := range seq {
		set.E[set.K.GetKeyPtr(element)] = element
	}

	return set
}

func MakeValueSetValue[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringerPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT,
) (s Set[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	s.K = keyer

	for i := range es {
		e := ELEMENT_PTR(&es[i])
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

func MakeValueSet[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringerPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT_PTR,
) (s Set[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	s.K = keyer

	for i := range es {
		e := es[i]
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

func MakeSetValue[ELEMENT any, ELEMENT_PTR interfaces.Ptr[ELEMENT]](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT,
) (s Set[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		panic("keyer was nil")
	}

	s.K = keyer

	for i := range es {
		e := ELEMENT_PTR(&es[i])
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

func MakeSet[ELEMENT any, ELEMENT_PTR interfaces.Ptr[ELEMENT]](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT_PTR,
) (s Set[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		panic("keyer was nil")
	}

	s.K = keyer

	for i := range es {
		e := es[i]
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

// constructs a mutable set of values using the given pointers
func MakeMutableValueSetValue[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringerPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT,
) (s MutableSet[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	s.K = keyer

	for i := range es {
		e := ELEMENT_PTR(&es[i])
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

// constructs a mutable set of values using the given pointers
func MakeMutableValueSet[
	ELEMENT interfaces.Stringer,
	ELEMENT_PTR interfaces.StringerPtr[ELEMENT],
](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT_PTR,
) (s MutableSet[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[ELEMENT, ELEMENT_PTR]{}
	}

	s.K = keyer

	for i := range es {
		e := es[i]
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}

func MakeMutableSet[ELEMENT any, ELEMENT_PTR interfaces.Ptr[ELEMENT]](
	keyer interfaces.StringKeyerPtr[ELEMENT, ELEMENT_PTR],
	es ...ELEMENT_PTR,
) (s MutableSet[ELEMENT, ELEMENT_PTR]) {
	s.E = make(map[string]ELEMENT_PTR, len(es))

	if keyer == nil {
		panic("keyer was nil")
	}

	s.K = keyer

	for i := range es {
		e := es[i]
		s.E[s.K.GetKeyPtr(e)] = e
	}

	return s
}
