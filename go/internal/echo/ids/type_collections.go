package ids

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_ptr"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/internal/delta/thyme"
)

func init() {
	collections_value.RegisterGobValue[thyme.Time](nil)
}

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
