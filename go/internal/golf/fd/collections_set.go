package fd

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

func init() {
	collections_value.RegisterGobValue[*FD](nil)
}

type (
	Set        = interfaces.Set[*FD]
	MutableSet = interfaces.SetMutable[*FD]
)

func MakeSet(ts ...*FD) Set {
	return collections_value.MakeValueSetFromSlice[*FD](
		nil,
		ts...,
	)
}

func MakeMutableSet(ts ...*FD) MutableSet {
	return collections_value.MakeMutableValueSet[*FD](
		nil,
		ts...,
	)
}

func MakeMutableSetSha() MutableSet {
	return collections_value.MakeMutableValueSet[*FD](
		KeyerSha{},
	)
}

type KeyerSha struct{}

func (KeyerSha) GetKey(fd *FD) string {
	return fd.digest.String()
}
