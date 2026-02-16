package organize_text

import (
	"sort"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type (
	objSet = interfaces.SetMutable[*obj]
)

var objKeyer interfaces.StringKeyer[*obj]

func makeObjSet() objSet {
	return collections_value.MakeMutableValueSet(
		sku.GetExternalLikeKeyer[*obj](),
	)
}

type obj struct {
	sku  sku.SkuType
	tipe tag_paths.Type
}

func (o obj) GetObjectId() *ids.ObjectId {
	return o.sku.GetObjectId()
}

func (o obj) GetSku() *sku.Transacted {
	return o.sku.GetSku()
}

func (o obj) GetSkuExternal() *sku.Transacted {
	return o.sku.GetSkuExternal()
}

func (a *obj) cloneWithType(t tag_paths.Type) (b *obj) {
	clonedSku, _ := sku.CloneSkuType(a.sku)
	b = &obj{
		tipe: t,
		sku:  clonedSku,
	}

	return b
}

func (a *obj) GetExternalObjectId() sku.ExternalObjectId {
	return a.sku.GetExternalObjectId()
}

func (a *obj) String() string {
	return a.sku.String()
}

func sortObjSet(
	s interfaces.SetMutable[*obj],
) (out Objects) {
	out = quiter.CollectSlice(s)
	out.Sort()
	return out
}

func (objects Objects) Sort() {
	sort.Slice(objects, func(i, j int) bool {
		iObject := objects[i].GetSkuExternal()
		jObject := objects[j].GetSkuExternal()

		switch {
		case iObject.ObjectId.IsEmpty() && jObject.ObjectId.IsEmpty():
			return iObject.GetMetadata().GetDescription().String() < jObject.GetMetadata().GetDescription().String()

		case iObject.ObjectId.IsEmpty():
			return true

		case jObject.ObjectId.IsEmpty():
			return false

		default:
			return iObject.ObjectId.String() < jObject.ObjectId.String()
		}
	})
}

type Objects []*obj

func (os Objects) Len() int {
	return len(os)
}

func (os *Objects) All() interfaces.Seq2[int, *obj] {
	return func(yield func(int, *obj) bool) {
		for i, o := range *os {
			if !yield(i, o) {
				break
			}
		}
	}
}

func (os Objects) Any() *obj {
	for _, v := range os {
		return v
	}

	return nil
}

func (os *Objects) Add(v *obj) error {
	*os = append(*os, v)
	return nil
}

func (os *Objects) Del(v *obj) error {
	for i, v1 := range *os {
		if v == v1 {
			*os = append((*os)[:i], (*os)[i+1:]...)
			break
		}
	}

	return nil
}

// func (os Objects) Sort() {
// 	sort.Slice(os, func(i, j int) bool {
// 		ei, ej := os[i].sku, os[j].sku

// 		keyI := keyer.GetKey(ei)
// 		keyJ := keyer.GetKey(ej)

// 		return keyI < keyJ
// 	})
// }
