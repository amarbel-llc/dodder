package collections_ptr

import (
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
)

type SetterPolicy int

const (
	SetterPolicyAppend = SetterPolicy(iota)
	SetterPolicyReset
)

// TODO-P2 add Resetter2 and Pool
type Flag[
	VALUE interfaces.Value,
	VALUE_PTR interfaces.ValuePtr[VALUE],
] interface {
	interfaces.FlagValue
	SetMany(vs ...string) (err error)
	interfaces.Set[VALUE]
}

func MakeFlagCommas[
	VALUE interfaces.Value,
	VALUE_PTR interfaces.ValuePtr[VALUE],
](
	policy SetterPolicy,
) Flag[VALUE, VALUE_PTR] {
	return &flagCommas[VALUE, VALUE_PTR]{
		policy: policy,
		set:    MakeMutableValueSet[VALUE, VALUE_PTR](nil),
	}
}

type flagCommas[
	VALUE interfaces.Value,
	VALUE_PTR interfaces.ValuePtr[VALUE],
] struct {
	policy   SetterPolicy
	set      interfaces.SetMutable[VALUE]
	pool     interfaces.PoolPtr[VALUE, VALUE_PTR]
	resetter interfaces.ResetterPtr[VALUE, VALUE_PTR]
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) Len() int {
	return flags.set.Len()
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) ContainsKey(key string) bool {
	return flags.set.ContainsKey(key)
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) Key(element ELEMENT) string {
	return flags.set.Key(element)
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) Get(key string) (ELEMENT, bool) {
	return flags.set.Get(key)
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) All() interfaces.Seq[ELEMENT] {
	return flags.set.All()
}

func (flags flagCommas[ELEMENT, ELEMENT_PTR]) String() (out string) {
	if flags.set == nil {
		return out
	}

	sorted := quiter.SortedStrings(flags.set)

	sb := &strings.Builder{}
	first := true

	for _, e1 := range sorted {
		if !first {
			sb.WriteString(", ")
		}

		sb.WriteString(e1)

		first = false
	}

	out = sb.String()

	return out
}

func (flags *flagCommas[ELEMENT, ELEMENT_PTR]) SetMany(vs ...string) (err error) {
	for _, v := range vs {
		if err = flags.Set(v); err != nil {
			return err
		}
	}

	return err
}

func (flags *flagCommas[ELEMENT, ELEMENT_PTR]) Set(value string) (err error) {
	switch flags.policy {
	case SetterPolicyReset:
		flags.set.Reset()
	}

	elements := strings.SplitSeq(value, ",")

	for element := range elements {
		element = strings.TrimSpace(element)

		// TODO-P2 use iter.AddStringPtr
		if err = quiter.AddString[ELEMENT, ELEMENT_PTR](
			flags.set,
			element,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
