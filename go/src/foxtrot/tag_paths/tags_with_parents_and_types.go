package tag_paths

import (
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/cmp"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
)

type (
	TagsWithParentsAndTypes collections_slice.Slice[TagWithParentsAndTypes]
)

func (tagsWithParentsAndTypes *TagsWithParentsAndTypes) GetSlice() *collections_slice.Slice[TagWithParentsAndTypes] {
	return (*collections_slice.Slice[TagWithParentsAndTypes])(tagsWithParentsAndTypes)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) Len() int {
	return len(tagsWithParentsAndTypes)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) ContainsObjectIdTag(
	k *ids.ObjectId,
) (int, bool) {
	return tagsWithParentsAndTypes.containsObjectIdTag(k, true)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) ContainsObjectIdTagExact(
	k *ids.ObjectId,
) (int, bool) {
	return tagsWithParentsAndTypes.containsObjectIdTag(k, false)
}

// TODO make less fragile
func (tagsWithParentsAndTypes TagsWithParentsAndTypes) containsObjectIdTag(
	id *ids.ObjectId,
	partial bool,
) (int, bool) {
	tagString := id.String()
	offset := 0

	if ids.IsVirtual(id) {
		percent, percentRepool := catgut.GetPool().GetWithRepool()
		defer percentRepool()

		percent.Set("%")

		loc, ok := cmp.BinarySearchFuncIndex(
			tagsWithParentsAndTypes,
			percent,
			func(ewp TagWithParentsAndTypes, e *Tag) cmp.Result {
				return ewp.Tag.ComparePartial(e)
			},
		)

		if !ok {
			return loc, ok
		}

		offset = percent.Len()
		tagsWithParentsAndTypes = tagsWithParentsAndTypes[loc:]
	}

	return cmp.BinarySearchFuncIndex(
		tagsWithParentsAndTypes,
		tagString,
		func(left TagWithParentsAndTypes, right string) cmp.Result {
			return cmp.CompareUTF8BytesAndString(
				left.Tag.Bytes()[offset:],
				right,
				partial,
			)
		},
	)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) ContainsTag(e *Tag) (int, bool) {
	return cmp.BinarySearchFuncIndex(
		tagsWithParentsAndTypes,
		e,
		func(ewp TagWithParentsAndTypes, e *Tag) cmp.Result {
			return ewp.Tag.ComparePartial(e)
		},
	)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) ContainsString(
	value string,
) (int, bool) {
	return cmp.BinarySearchFuncIndex(
		tagsWithParentsAndTypes,
		value,
		func(ewp TagWithParentsAndTypes, c string) cmp.Result {
			return cmp.CompareUTF8BytesAndString(ewp.Tag.Bytes(), c, true)
		},
	)
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) GetMatching(
	e *Tag,
) (matching []TagWithParentsAndTypes) {
	i, ok := tagsWithParentsAndTypes.ContainsTag(e)

	if !ok {
		return matching
	}

	for _, ewp := range tagsWithParentsAndTypes[i:] {
		cmp := ewp.ComparePartial(e)

		if !cmp.IsEqual() {
			return matching
		}

		matching = append(matching, ewp)
	}

	return matching
}

// TODO return success
func (tagsWithParentsAndTypes *TagsWithParentsAndTypes) Add(
	e1 *Tag,
	p *PathWithType,
) (err error) {
	var e *Tag

	if e, _, err = e1.Clone(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	idx, ok := tagsWithParentsAndTypes.ContainsTag(e)

	var a TagWithParentsAndTypes

	if ok {
		a = (*tagsWithParentsAndTypes)[idx]
		a.Parents.AddNonEmptyPath(p)
		(*tagsWithParentsAndTypes)[idx] = a
	} else {
		a = TagWithParentsAndTypes{Tag: e}
		a.Parents.AddNonEmptyPath(p)

		if idx == tagsWithParentsAndTypes.Len() {
			*tagsWithParentsAndTypes = append(*tagsWithParentsAndTypes, a)
		} else {
			*tagsWithParentsAndTypes = slices.Insert(*tagsWithParentsAndTypes, idx, a)
		}
	}

	return err
}

// TODO return success
func (tagsWithParentsAndTypes *TagsWithParentsAndTypes) Remove(e1 *Tag) (err error) {
	var e *Tag

	if e, _, err = e1.Clone(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	idx, ok := tagsWithParentsAndTypes.ContainsTag(e)

	if !ok {
		return err
	}

	*tagsWithParentsAndTypes = slices.Delete(*tagsWithParentsAndTypes, idx, idx+1)

	return err
}

func (tagsWithParentsAndTypes TagsWithParentsAndTypes) StringCommaSeparatedExplicit() string {
	var sb strings.Builder

	first := true

	for _, ewp := range tagsWithParentsAndTypes {
		if ewp.Parents.Len() != 0 {
			continue
		}

		sb.Write(ewp.Tag.Bytes())

		if !first {
			sb.WriteString(", ")
		}

		first = false
	}

	return sb.String()
}
