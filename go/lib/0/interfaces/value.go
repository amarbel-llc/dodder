package interfaces

type (
	Value interface {
		Stringer
		IsEmpty() bool
	}

	ValuePtr[SELF Value] interface {
		Resetable
		ResetablePtr[SELF]
		StringerSetterPtr[SELF]
	}
)
