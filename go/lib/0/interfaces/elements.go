package interfaces

type (
	Lessor[ELEMENT any] interface {
		Less(ELEMENT, ELEMENT) bool
	}

	// TODO-P2 rename
	Equaler[ELEMENT any] interface {
		Equals(ELEMENT, ELEMENT) bool
	}

	ResetterPtr[
		ELEMENT any,
		ELEMENT_PTR Ptr[ELEMENT],
	] interface {
		Reset(ELEMENT_PTR)
		ResetWith(ELEMENT_PTR, ELEMENT_PTR)
	}

	Resetter[ELEMENT any] interface {
		Reset(ELEMENT)
		ResetWith(ELEMENT, ELEMENT)
	}

	Equatable[ELEMENT any] interface {
		Equals(ELEMENT) bool
	}

	Resetable interface {
		Reset()
	}

	ResetableWithError interface {
		Reset() error
	}

	ResetablePtr[ELEMENT any] interface {
		Ptr[ELEMENT]
		ResetWith(ELEMENT)
		Reset()
	}
)
