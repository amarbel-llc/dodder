package collections_value

import (
	"encoding/gob"
	"slices"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

// TODO move construction to another derived package
func MakeValueSetValue[ELEMENT interfaces.Stringer](
	keyer interfaces.StringKeyer[ELEMENT],
	elements ...ELEMENT,
) (set Set[ELEMENT]) {
	gob.Register(set)
	set.E = make(map[string]ELEMENT, len(elements))

	if keyer == nil {
		keyer = quiter.StringerKeyer[ELEMENT]{}
	}

	set.K = keyer

	for i := range elements {
		e := elements[i]
		set.E[set.K.GetKey(e)] = e
	}

	return set
}

// TODO move construction to another derived package
func MakeValueSet[ELEMENT interfaces.Stringer](
	keyer interfaces.StringKeyer[ELEMENT],
	seq interfaces.Seq[ELEMENT],
) (set Set[ELEMENT]) {
	gob.Register(set)
	set.E = make(map[string]ELEMENT, 0)

	if keyer == nil {
		keyer = quiter.StringerKeyer[ELEMENT]{}
	}

	set.K = keyer

	for element := range seq {
		set.E[set.K.GetKey(element)] = element
	}

	return set
}

func MakeValueSetFromSlice[ELEMENT interfaces.Stringer](
	keyer interfaces.StringKeyer[ELEMENT],
	elements ...ELEMENT,
) (set Set[ELEMENT]) {
	gob.Register(set)
	set.E = make(map[string]ELEMENT, len(elements))

	if keyer == nil {
		keyer = quiter.StringerKeyer[ELEMENT]{}
	}

	set.K = keyer

	for index := range elements {
		element := elements[index]
		set.E[set.K.GetKey(element)] = element
	}

	return set
}

func MakeSet[ELEMENT any](
	keyer interfaces.StringKeyer[ELEMENT],
	elements ...ELEMENT,
) (set Set[ELEMENT]) {
	gob.Register(set)
	set.E = make(map[string]ELEMENT, len(elements))

	if keyer == nil {
		panic("keyer was nil")
	}

	set.K = keyer

	for i := range elements {
		e := elements[i]
		set.E[set.K.GetKey(e)] = e
	}

	return set
}

// TODO move construction to another derived package
func MakeMutableValueSet[ELEMENT interfaces.Stringer](
	keyer interfaces.StringKeyer[ELEMENT],
	elements ...ELEMENT,
) (set MutableSet[ELEMENT]) {
	if keyer == nil {
		keyer = quiter.StringerKeyer[ELEMENT]{}
	}

	return MakeMutableSet(keyer, len(elements), slices.Values(elements))
}

func MakeMutableSet[ELEMENT any](
	keyer interfaces.StringKeyer[ELEMENT],
	count int,
	seq interfaces.Seq[ELEMENT],
) (set MutableSet[ELEMENT]) {
	gob.Register(set)
	set.E = make(map[string]ELEMENT, count)

	if keyer == nil {
		panic("keyer was nil")
	}

	set.K = keyer

	for e := range seq {
		set.E[set.K.GetKey(e)] = e
	}

	return set
}
