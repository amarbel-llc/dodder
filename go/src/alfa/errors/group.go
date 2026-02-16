package errors

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/pool"
)

type Group []error

func (group Group) Error() string {
	return fmt.Sprintf("error group: %d errors", group.Len())
}

func (group Group) Unwrap() []error {
	return group
}

func (group Group) Len() int {
	return len(group)
}

var groupPool = pool.MakeSlice[error, Group]()
