package interfaces

type (
	FuncSetString     func(string) error
	FuncString[T any] func(T) string
)

type Stringer interface {
	String() string
}

type StringerSetter interface {
	Stringer
	Setter
}

type StringerPtr[T any] interface {
	Stringer
	Ptr[T]
}

type StringerSetterPtr[T any] interface {
	Stringer
	Setter
	Ptr[T]
}

type Setter interface {
	Set(string) error
}

type SetterPtr[T any] interface {
	Ptr[T]
	Setter
}

type StringSetterPtr[T any] interface {
	Stringer
	Ptr[T]
	Setter
}

type StringerWithHeadAndTail interface {
	Stringer
	GetHead() string
	GetTail() string
}
