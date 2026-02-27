package errors

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"slices"
	"syscall"
	"time"
)

var (
	As     = errors.As
	Unwrap = errors.Unwrap
)

func IsAny(err error, funcTargets ...FuncIs) bool {
	for _, funcTarget := range funcTargets {
		if funcTarget(err) {
			return true
		}
	}

	return false
}

func IsSentinel(err, target error) bool {
	return err == target
}

func Is(err, target error) bool {
	if err == io.EOF || target == io.EOF {
		panic("checking for EOF via errors.Is")
	}

	if debugBuild {
		return IsWithTimeout(err, target, time.Second)
	}

	return errors.Is(err, target)
}

func IsWithTimeout(err, target error, timeout time.Duration) bool {
	chBool := make(chan bool)
	defer close(chBool)

	go func() {
		chBool <- errors.Is(err, target)
	}()

	select {
	case result := <-chBool:
		return result

	case <-time.After(timeout):
		panic(fmt.Sprintf("calling errors.Is() timed out for %T", err))
	}
}

func IsNetTimeout(err error) (ok bool) {
	var netError net.Error

	if !As(err, &netError) {
		return ok
	}

	ok = netError.Timeout()

	return ok
}

func MakeIsErrno(targets ...syscall.Errno) FuncIs {
	return func(err error) bool {
		return IsErrno(err, targets...)
	}
}

func IsErrno(err error, targets ...syscall.Errno) (ok bool) {
	var errno syscall.Errno

	if !As(err, &errno) {
		return false
	}

	return slices.Contains(targets, errno)
}

func IsBrokenPipe(err error) bool {
	return IsErrno(err, syscall.EPIPE)
}

func IsTooManyOpenFiles(err error) bool {
	e := errors.Unwrap(err)
	return e.Error() == "too many open files"
}

// TODO remove
func IsNotNilAndNotEOF(err error) bool {
	if err == nil || err == io.EOF {
		return false
	}

	return true
}

func IsEOF(err error) bool {
	return err == io.EOF
}

func IsExist(err error) bool {
	return Is(err, fs.ErrExist)
}

func IsNotExist(err error) bool {
	return Is(err, fs.ErrNotExist)
}

func IsReadlinkInvalidArgument(err error) bool {
	var pathError *fs.PathError

	if errors.As(err, &pathError) {
		if pathError.Err.Error() == "invalid argument" {
			return true
		}
	}

	return false
}
