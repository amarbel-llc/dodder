package errors

import (
	"fmt"
	"os"
)

type Signal struct {
	os.Signal
}

func (err Signal) Is(target error) bool {
	_, ok := target.(Signal)
	return ok
}

func (err Signal) Error() string {
	return fmt.Sprintf("received signal: %q", err.Signal)
}
