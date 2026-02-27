package organize_text

import (
	"sort"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
)

type Refiner struct {
	Enabled         bool
	UsePrefixJoints bool
}

func (atc *Refiner) shouldMergeAllChildrenIntoParent(a *Assignment) (ok bool) {
	switch {
	case a.Parent.IsRoot:
		fallthrough

	default:
		ok = false
	}

	return ok
}

func (atc *Refiner) shouldMergeIntoParent(a *Assignment) bool {
	ui.Log().Printf("checking node should merge: %s", a)

	if a.Parent == nil {
		ui.Log().Print("parent is nil")
		return false
	}

	if a.Parent.IsRoot {
		ui.Log().Print("parent is root")
		return false
	}

	childTag := quiter_set.Any(a.Transacted.GetMetadata().GetTags())
	parentTag := quiter_set.Any(a.Parent.Transacted.GetMetadata().GetTags())

	if a.Transacted.GetMetadata().GetTags().Len() == 1 && ids.IsEmpty(childTag) {
		ui.Log().Print("1 tag, and it's empty, merging")
		return true
	}

	if a.Transacted.GetMetadata().GetTags().Len() == 0 {
		ui.Log().Print("tags length is 0, merging")
		return true
	}

	if a.Parent.Transacted.GetMetadata().GetTags().Len() != 1 {
		ui.Log().Print("parent tags length is not 1")
		return false
	}

	if a.Transacted.GetMetadata().GetTags().Len() != 1 {
		ui.Log().Print("tags length is not 1")
		return false
	}

	equal := quiter_set.Equals(
		a.Transacted.GetMetadata().GetTags(),
		a.Parent.Transacted.GetMetadata().GetTags())

	if !equal {
		ui.Log().Print("parent tags not equal")
		return false
	}

	if ids.IsDependentLeaf(parentTag) {
		ui.Log().Print("is prefix joint")
		return false
	}

	if ids.IsDependentLeaf(childTag) {
		ui.Log().Print("is prefix joint")
		return false
	}

	return true
}

func (atc *Refiner) renameForPrefixJoint(a *Assignment) (err error) {
	if !atc.UsePrefixJoints {
		return err
	}

	if a == nil {
		ui.Log().Printf("assignment is nil")
		return err
	}

	if a.Parent == nil {
		ui.Log().Printf("parent is nil: %#v", a)
		return err
	}

	if a.Parent.Transacted.GetMetadata().GetTags().Len() == 0 {
		return err
	}

	if a.Parent.Transacted.GetMetadata().GetTags().Len() != 1 {
		return err
	}

	parentTag := quiter_set.Any(a.Parent.Transacted.GetMetadata().GetTags())

	if ids.IsDependentLeaf(parentTag) {
		return err
	}

	childTag := quiter_set.Any(a.Transacted.GetMetadata().GetTags())

	if ids.IsDependentLeaf(childTag) {
		return err
	}

	if !ids.HasParentPrefix(childTag, parentTag) {
		ui.Log().Print("parent is not prefix joint")
		return err
	}

	if childTag.Equals(parentTag) {
		ui.Log().Print("parent is is equal to child")
		return err
	}

	var ls ids.TagStruct

	if ls, err = ids.LeftSubtract(childTag, parentTag); err != nil {
		err = errors.Wrap(err)
		return err
	}

	objects.SetTags(a.Transacted.GetMetadataMutable(), ids.MakeTagSetMutable(ls))

	return err
}

// passed-in assignment may be nil?
func (atc *Refiner) Refine(a *Assignment) (err error) {
	if !atc.Enabled {
		return err
	}

	if !a.IsRoot {
		if atc.shouldMergeIntoParent(a) {
			ui.Log().Print("merging into parent")
			p := a.Parent

			if err = p.consume(a); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return atc.Refine(p)
		}
	}

	if err = atc.applyPrefixJoints(a); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = atc.renameForPrefixJoint(a); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, child := range a.Children {
		if err = atc.Refine(child); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = atc.applyPrefixJoints(a); err != nil {
		err = errors.Wrap(err)
		return err
	}

	a.SortChildren()

	return err
}

func (atc Refiner) applyPrefixJoints(a *Assignment) (err error) {
	if !atc.UsePrefixJoints {
		return err
	}

	if a.Transacted.GetMetadata().GetTags() == nil || a.Transacted.GetMetadata().GetTags().Len() == 0 {
		return err
	}

	childPrefixes := atc.childPrefixes(a)

	if len(childPrefixes) == 0 {
		return err
	}

	groupingPrefix := childPrefixes[0]

	var na *Assignment

	if a.Transacted.GetMetadata().GetTags().Len() == 1 &&
		quiter_set.Any(a.Transacted.GetMetadata().GetTags()).Equals(groupingPrefix.Tag) {
		na = a
	} else {
		na = newAssignment(a.GetDepth() + 1)
		objects.SetTags(na.Transacted.GetMetadataMutable(), ids.MakeTagSetMutable(groupingPrefix.Tag))
		a.addChild(na)
	}

	for _, c := range groupingPrefix.assignments {
		if c.Parent != na {
			if err = c.removeFromParent(); err != nil {
				err = errors.Wrap(err)
				return err
			}

			na.addChild(c)
		}

		subtractedTags := ids.SubtractPrefix(
			c.Transacted.GetMetadata().GetTags(),
			groupingPrefix.Tag,
		)
		objects.SetTags(c.Transacted.GetMetadataMutable(), subtractedTags)
	}

	return err
}

type tagBag struct {
	Tag         ids.TagStruct
	assignments []*Assignment
}

func (a Refiner) childPrefixes(node *Assignment) (out []tagBag) {
	assignmentMap := make(map[string][]*Assignment)
	out = make([]tagBag, 0, len(node.Children))

	if node.Transacted.GetMetadata().GetTags().Len() == 0 {
		return out
	}

	for _, child := range node.Children {
		expanded := expansion.ExpandMany(
			child.Transacted.GetMetadata().GetTags().All(),
			expansion.ExpanderRight,
		)

		for tag := range expanded {
			if tag.String() == "" {
				continue
			}

			var n []*Assignment
			ok := false

			if n, ok = assignmentMap[tag.String()]; !ok {
				n = make([]*Assignment, 0)
			}

			n = append(n, child)

			assignmentMap[tag.String()] = n
		}
	}

	for e, n := range assignmentMap {
		if len(n) > 1 {
			var e1 ids.TagStruct

			errors.PanicIfError(e1.Set(e))

			out = append(out, tagBag{Tag: e1, assignments: n})
		}
	}

	sort.Slice(
		out,
		func(i, j int) bool {
			if len(out[i].assignments) == len(out[j].assignments) {
				return len(
					out[i].Tag.String(),
				) > len(
					out[j].Tag.String(),
				)
			} else {
				return len(out[i].assignments) > len(out[j].assignments)
			}
		},
	)

	return out
}
