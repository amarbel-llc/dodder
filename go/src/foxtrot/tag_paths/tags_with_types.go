package tag_paths

import (
	"fmt"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
)

type Tags struct {
	Paths PathsWithTypes // TODO implement
	All   TagsWithParentsAndTypes
}

func (tags *Tags) String() string {
	return fmt.Sprintf("[Paths: %s, All: %s]", tags.Paths, tags.All)
}

func (tags *Tags) Reset() {
	// TODO pool *Path's
	tags.Paths.Reset()
	tags.All.GetSlice().Reset()
}

// TODO improve performance
func (tags *Tags) ResetWith(b *Tags) {
	tags.Paths = slices.Grow(tags.Paths, len(b.Paths))

	for _, p := range b.Paths {
		tags.AddPath(p.Clone())
	}
	// a.Paths = a.Paths[:cap(a.Paths)]
	// nPaths := copy(a.Paths, b.Paths)

	// a.All = slices.Grow(a.All, len(b.All))
	// a.All = a.All[:cap(a.All)]
	// nAll := copy(a.All, b.All)
	// ui.Debug().Print(nPaths, nAll, a, b)
}

func (tags *Tags) AddSuperFrom(
	b *Tags,
	prefix *Tag,
) (err error) {
	for _, ep := range b.Paths {
		ui.Log().Print("adding", prefix, ep)
		if prefix.ComparePartial(ep.First()).IsEqual() {
			continue
		}

		prefixPath := makePath(prefix)
		prefixPath.Add(ep.Path...)

		c := &PathWithType{
			Path: prefixPath,
			Type: TypeSuper,
		}

		if err = tags.AddPath(c); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (tags *Tags) AddTagOld(tag ids.Tag) (err error) {
	s, _ := catgut.MakeFromString(tag.String())
	return tags.AddTag(s)
}

func (tags *Tags) AddTag(e *Tag) (err error) {
	if e.IsEmpty() {
		return err
	}

	path := MakePathWithType(e)

	if err = tags.AddPathWithType(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) AddSelf(e *Tag) (err error) {
	if e.IsEmpty() {
		return err
	}

	p := MakePathWithType(e)
	p.Type = TypeSelf

	if err = tags.AddPathWithType(p); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) AddPathWithType(pwt *PathWithType) (err error) {
	_, alreadyExists := tags.Paths.AddPath(pwt)

	if alreadyExists {
		return err
	}

	for _, e := range pwt.Path {
		if err = tags.All.Add(e, pwt); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (tags *Tags) AddPath(p *PathWithType) (err error) {
	if err = tags.AddPathWithType(p); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) Set(v string) (err error) {
	vs := strings.Split(v, ",")

	for _, v := range vs {
		var e ids.TagStruct

		if err = e.Set(v); err != nil {
			err = errors.Wrap(err)
			return err
		}

		es, _ := catgut.MakeFromString(e.String())

		if err = tags.AddTag(es); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
