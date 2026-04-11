package interfaces

type (
	FuncIter[ELEMENT any] func(ELEMENT) error

	FuncIterIO[ELEMENT any]            func(ELEMENT) (int64, error)
	FuncTransform[ELEMENT any, T1 any] func(ELEMENT) (T1, error)
	FuncIterKey                        func(string) error
	FuncIterWithKey[ELEMENT any]       func(string, ELEMENT) error
)

type (
	Lenner interface {
		Len() int
	}

	ContainsKeyer interface {
		ContainsKey(string) bool
	}

	Keyer[ELEMENT any] interface {
		Key(ELEMENT) string
	}

	Iterable[ELEMENT any] interface {
		All() Seq[ELEMENT]
	}

	Adder[ELEMENT any] interface {
		Add(ELEMENT) error
	}

	Collection[ELEMENT any] interface {
		Lenner
		Iterable[ELEMENT]
	}

	CollectionMutable[ELEMENT any] interface {
		Collection[ELEMENT]
		Resetable
	}

	SetGetter[ELEMENT any] interface {
		Get(string) (ELEMENT, bool)
	}

	Set[ELEMENT any] interface {
		ContainsKeyer
		Iterable[ELEMENT]
		Keyer[ELEMENT]
		Lenner
		SetGetter[ELEMENT]
	}

	SetMutable[ELEMENT any] interface {
		Set[ELEMENT]
		Adder[ELEMENT]
		DelKey(string) error
		Resetable
	}
)

type Delta[ELEMENT any] interface {
	GetAdded() Set[ELEMENT]
	GetRemoved() Set[ELEMENT]
}

type (
	Tridex interface {
		Collection[string]
		ContainsAbbreviation(string) bool
		ContainsExpansion(string) bool
		Abbreviate(string) string
		Expand(string) string
	}

	TridexMutable interface {
		Tridex
		Add(string)
		Remove(string)
	}
)
