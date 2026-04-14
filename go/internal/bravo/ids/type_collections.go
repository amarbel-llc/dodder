package ids

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/collections_ptr"
)

type (
	TypeSet        = interfaces.Set[TypeStruct]
	TypeMutableSet = interfaces.SetMutable[TypeStruct]
)

func MakeTypeSet(es ...TypeStruct) (s TypeSet) {
	return TypeSet(collections_ptr.MakeValueSetValue(nil, es...))
}

func MakeTypeSetStrings(vs ...string) (s TypeSet, err error) {
	return collections_ptr.MakeValueSetString[TypeStruct](nil, vs...)
}

func MakeMutableTypeSet(hs ...TypeStruct) TypeMutableSet {
	return MakeTypeMutableSet(hs...)
}

func MakeTypeMutableSet(hs ...TypeStruct) TypeMutableSet {
	return TypeMutableSet(collections_ptr.MakeMutableValueSetValue(nil, hs...))
}
