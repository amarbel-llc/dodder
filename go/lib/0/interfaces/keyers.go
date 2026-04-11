package interfaces

import "fmt"

type StringKeyer[T any] interface {
	GetKey(T) string
}

type StringKeyerPtr[T any, TPtr Ptr[T]] interface {
	StringKeyer[T]
	GetKeyPtr(TPtr) string
}

type CompoundKeyer[T any] []StringKeyer[T]

func (ck CompoundKeyer[T]) GetKey(e T) string {
	for _, k := range ck {
		if key := k.GetKey(e); key != "" {
			return key
		}
	}

	panic(fmt.Sprintf("no valid key found for %#v", e))
}
