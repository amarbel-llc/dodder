package script_value

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/google/shlex"
)

type Utility []string

func (u *Utility) Len() int {
	return len(*u)
}

func (u *Utility) IsEmpty() bool {
	return u.Len() == 0
}

func (u *Utility) Head() string {
	if u.Len() == 0 {
		return ""
	}

	return (*u)[0]
}

func (u *Utility) Tail() []string {
	if u.Len() < 2 {
		return nil
	}

	return (*u)[1:]
}

func (u *Utility) String() string {
	return fmt.Sprintf("%q", []string(*u))
}

func (u *Utility) Set(v string) (err error) {
	if *u, err = shlex.Split(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
