package quiter

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func PrintPointer[T any, TPtr interfaces.Ptr[T]](e TPtr) (err error) {
	// log.Debug().Caller(1, "%s -> %s", unsafe.Pointer(e), e)
	return err
}

func MakeIterDebug[T any](f interfaces.FuncIter[T]) interfaces.FuncIter[T] {
	return func(e T) (err error) {
		// log.Debug().FunctionName(2)
		return f(e)
	}
}
