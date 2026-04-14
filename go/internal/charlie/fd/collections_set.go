package fd

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/collections_value"
)

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
