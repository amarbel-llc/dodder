package errors

import (
	"io"
	"os"
)

func DeferredRecover(err *error) {
	if !debugBuild {
		return
	}

	if r := recover(); r != nil {
		switch r := r.(type) {
		default:
			*err = Join(*err, Errorf("panicked: %s", r))

		case error:
			*err = Join(*err, r)
		}
	}
}

func DeferredFlusher(
	err *error,
	f Flusher,
) {
	if err1 := f.Flush(); err1 != nil {
		if err == nil {
			panic(err)
		} else {
			*err = Join(*err, err1)
		}
	}
}

func DeferredYieldCloser[T any](
	yield func(T, error) bool,
	closer io.Closer,
) {
	if err := closer.Close(); err != nil {
		var t T
		yield(t, err)
	}
}

func DeferredCloser(
	err *error,
	closer io.Closer,
) {
	if err1 := closer.Close(); err1 != nil {
		if err == nil {
			panic(err)
		} else {
			*err = Join(*err, WrapSkip(1, err1))
		}
	}
}

func DeferredCloseAndRename(err *error, c io.Closer, oldpath, newpath string) {
	if err == nil {
		panic("deferred error interface is nil")
	}

	if err1 := c.Close(); err1 != nil {
		*err = Join(*err, err1)
		return
	}

	if err1 := os.Rename(oldpath, newpath); err1 != nil {
		*err = Join(*err, err1)
	}
}

func Deferred(
	err *error,
	ef func() error,
) {
	if err1 := ef(); err1 != nil {
		if err == nil {
			panic(err)
		} else {
			*err = Join(*err, err1)
		}
	}
}

func Must(funk FuncErr) {
	if err := funk(); err != nil {
		panic(err)
	}
}
