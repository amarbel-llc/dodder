package organize_text

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/charlie/expansion"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type PrefixSet struct {
	count    int
	innerMap map[string]objSet
}

type Segments struct {
	Ungrouped objSet
	Grouped   PrefixSet
}

func MakePrefixSet(c int) (s PrefixSet) {
	s.innerMap = make(map[string]objSet, c)
	return s
}

func MakePrefixSetFrom(
	objectSet objSet,
) (prefixSet PrefixSet) {
	prefixSet = MakePrefixSet(objectSet.Len())
	for element := range objectSet.All() {
		prefixSet.Add(element)
	}
	return prefixSet
}

func (prefixSet PrefixSet) Len() int {
	return prefixSet.count
}

func (prefixSet *PrefixSet) AddSku(object sku.SkuType) (err error) {
	if object.GetState() == checked_out_state.Unknown {
		err = errors.ErrorWithStackf(
			"unacceptable state: %s",
			object.GetState(),
		)
		err = errors.Wrapf(err, "Sku: %s", sku.String(object.GetSku()))
		return err
	}

	clonedSku, _ := sku.CloneSkuType(object)
	o := obj{
		sku: clonedSku,
	}

	if err = prefixSet.Add(&o); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// this splits on right-expanded
func (prefixSet *PrefixSet) Add(object *obj) (err error) {
	var addedAny bool

	{
		seq := expansion.ExpandMany(
			object.GetSkuExternal().GetMetadata().GetIndex().GetImplicitTags().All(),
			expansion.ExpanderRight,
		)

		for tag := range seq {
			prefixSet.addPair(tag.String(), object)
			addedAny = true
		}
	}

	{
		seq := expansion.ExpandMany(
			object.GetSkuExternal().GetMetadata().GetTags().All(),
			expansion.ExpanderRight,
		)

		for tag := range seq {
			prefixSet.addPair(tag.String(), object)
			addedAny = true
		}
	}

	if addedAny {
		return err
	}

	prefixSet.addPair("", object)

	return err
}

func (prefixSet PrefixSet) Subtract(
	objects objSet,
) (output PrefixSet) {
	output = MakePrefixSet(len(prefixSet.innerMap))

	for tag, aSet := range prefixSet.innerMap {
		for object := range aSet.All() {
			if !quiter_set.Contains(objects, object) {
				output.addPair(tag, object)
			}
		}
	}

	return output
}

func (prefixSet *PrefixSet) addPair(
	tagString string,
	object *obj,
) {
	if tagString == object.GetSkuExternal().ObjectId.String() {
		tagString = ""
	}

	existingSet, ok := prefixSet.innerMap[tagString]

	if !ok {
		existingSet = makeObjSet()
		prefixSet.innerMap[tagString] = existingSet
	}

	var existingObj *obj
	existingObj, ok = existingSet.Get(existingSet.Key(object))

	if ok && existingObj.tipe.IsDirectOrSelf() {
		object.tipe.SetDirect()
	} else if !ok {
		prefixSet.count += 1
	}

	existingSet.Add(object)
}

func (prefixSet PrefixSet) AllObjectSets() interfaces.Seq2[string, objSet] {
	return func(yield func(string, objSet) bool) {
		for tagString, objects := range prefixSet.innerMap {
			if !yield(tagString, objects) {
				break
			}
		}
	}
}

func (prefixSet PrefixSet) AllObjects() interfaces.Seq2[string, *obj] {
	return func(yield func(string, *obj) bool) {
		for tagString, objects := range prefixSet.innerMap {
			for object := range objects.All() {
				if !yield(tagString, object) {
					break
				}
			}
		}
	}
}

func (prefixSet PrefixSet) Match(
	tag ids.TagStruct,
) (out Segments) {
	out.Ungrouped = makeObjSet()
	out.Grouped = MakePrefixSet(len(prefixSet.innerMap))

	for prefixTag, objects := range prefixSet.innerMap {
		if prefixTag == "" {
			continue
		}

		for object := range objects.All() {
			objectTags := object.GetSkuExternal().GetTags()

			intersection := ids.IntersectPrefixes(objectTags, tag)

			exactMatch := intersection.Len() == 1 &&
				quiter_set.Any(intersection).Equals(tag)

			if intersection.Len() == 0 && !exactMatch {
				continue
			}

			for _, e2 := range quiter.CollectSlice(intersection) {
				out.Grouped.addPair(e2.String(), object)
			}
		}
	}

	return out
}

func (prefixSet PrefixSet) Subset(
	tag ids.TagStruct,
) (out Segments) {
	out.Ungrouped = makeObjSet()
	out.Grouped = MakePrefixSet(len(prefixSet.innerMap))

	tagString, tagStringRepool := catgut.MakeFromString(tag.String())
	defer tagStringRepool()

	for prefixTag, objects := range prefixSet.innerMap {
		if prefixTag == "" {
			continue
		}

		for object := range objects.All() {
			intersection := object.GetSkuExternal().GetMetadata().GetIndex().GetTagPaths().All.GetMatching(
				tagString,
			)

			hasDirect := false || len(intersection) == 0

			type match struct {
				string
				tag_paths.Type
			}

			toAddGrouped := make([]match, 0)

		OUTER:
			for _, objectTagMatch := range intersection {
				objectTagMatchString := objectTagMatch.Tag.String()

				for _, objectTagMatchParents := range objectTagMatch.Parents {
					toAddGrouped = append(toAddGrouped, match{
						string: objectTagMatchString,
						Type:   objectTagMatchParents.Type,
					})

					if objectTagMatchParents.Type == tag_paths.TypeDirect &&
						objectTagMatch.Tag.Len() == tagString.Len() {
						hasDirect = true
						break OUTER
					}
				}
			}

			if hasDirect {
				out.Ungrouped.Add(object)
			} else {
				for _, tagToAdd := range toAddGrouped {
					objectClone := object.cloneWithType(tagToAdd.Type)
					out.Grouped.addPair(tagToAdd.string, objectClone)
				}
			}
		}
	}

	return out
}
