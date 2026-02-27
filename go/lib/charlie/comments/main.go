package comments

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func Change(_ string)                 {}
func Comment(_ ...string)             {}
func Decide(_ string)                 {}
func GoRefactor(before, after string) {}
func Optimize(_ string)               {}
func Parallelize()                    {}
func Performance(_ string)            {}
func Refactor(_ string)               {}
func Remove()                         {}

func Implement() (err error) {
	return errors.WrapSkip(1, errors.Err501NotImplemented)
}
