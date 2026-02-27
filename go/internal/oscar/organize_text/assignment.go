package organize_text

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

// TODO move to object_factory
func newAssignment(depth int) *Assignment {
	assignment := &Assignment{
		Depth:    depth,
		objects:  make(map[string]struct{}),
		Objects:  make(Objects, 0),
		Children: make([]*Assignment, 0),
	}

	sku.TransactedResetter.Reset(&assignment.Transacted)

	return assignment
}

type Assignment struct {
	sku.Transacted

	IsRoot  bool
	Depth   int
	objects map[string]struct{}
	Objects
	Children []*Assignment
	Parent   *Assignment
}

func (assignment *Assignment) AddObject(object *obj) (err error) {
	key := keyer.GetKey(object.sku)
	_, ok := assignment.objects[key]

	if ok {
		return err
	}

	assignment.objects[key] = struct{}{}

	return assignment.Objects.Add(object)
}

func (assignment Assignment) GetDepth() int {
	if assignment.Parent == nil {
		return 0
	} else {
		return assignment.Parent.GetDepth() + 1
	}
}

func (assignment Assignment) MaxDepth() (depth int) {
	depth = assignment.GetDepth()

	for _, child := range assignment.Children {
		childDepth := child.MaxDepth()

		if depth < childDepth {
			depth = childDepth
		}
	}

	return depth
}

func (assignment Assignment) AlignmentSpacing() int {
	childTag := quiter_set.Any(assignment.Transacted.GetMetadata().GetTags())

	if assignment.Transacted.GetMetadata().GetTags().Len() == 1 && ids.IsDependentLeaf(childTag) {
		return assignment.Parent.AlignmentSpacing() + len(
			quiter_set.Any(assignment.Parent.Transacted.GetMetadata().GetTags()).String(),
		)
	}

	return 0
}

func (assignment Assignment) MaxLen() (maxLength int) {
	for _, object := range assignment.Objects.All() {
		objectIdLength := object.sku.GetSkuExternal().ObjectId.Len()

		if objectIdLength > maxLength {
			maxLength = objectIdLength
		}
	}

	for _, child := range assignment.Children {
		childMaxLength := child.MaxLen()

		if childMaxLength > maxLength {
			maxLength = childMaxLength
		}
	}

	return maxLength
}

func (assignment Assignment) String() (s string) {
	if assignment.Parent != nil {
		s = assignment.Parent.String() + "."
	}

	return s + quiter.StringCommaSeparated(assignment.Transacted.GetMetadata().GetTags())
}

func (assignment *Assignment) makeChild(e ids.TagStruct) (b *Assignment) {
	b = newAssignment(assignment.GetDepth() + 1)
	objects.SetTags(b.Transacted.GetMetadataMutable(), ids.MakeTagSetMutable(e))
	assignment.addChild(b)
	return b
}

func (assignment *Assignment) addChild(c *Assignment) {
	if assignment == c {
		panic("child and parent are the same")
	}

	if c.Parent != nil && c.Parent == assignment {
		panic("child already has self as parent")
	}

	if c.Parent != nil {
		panic("child already has a parent")
	}

	assignment.Children = append(assignment.Children, c)
	c.Parent = assignment
}

func (assignment *Assignment) parentOrRoot() (p *Assignment) {
	switch assignment.Parent {
	case nil:
		return assignment

	default:
		return assignment.Parent
	}
}

func (assignment *Assignment) nthParent(n int) (p *Assignment, err error) {
	if n < 0 {
		n = -n
	}

	if n == 0 {
		p = assignment
		return p, err
	}

	if assignment.Parent == nil {
		err = errors.ErrorWithStackf("cannot get nth parent as parent is nil")
		return p, err
	}

	return assignment.Parent.nthParent(n - 1)
}

func (assignment *Assignment) removeFromParent() (err error) {
	return assignment.Parent.removeChild(assignment)
}

func (assignment *Assignment) removeChild(c *Assignment) (err error) {
	if c.Parent != assignment {
		err = errors.ErrorWithStackf("attempting to remove child from wrong parent")
		return err
	}

	if len(assignment.Children) == 0 {
		err = errors.ErrorWithStackf(
			"attempting to remove child when there are no children",
		)
		return err
	}

	cap1 := 0
	cap2 := len(assignment.Children) - 1

	if cap2 > 0 {
		cap1 = cap2
	}

	nc := make([]*Assignment, 0, cap1)

	for _, c1 := range assignment.Children {
		if c1 == c {
			continue
		}

		nc = append(nc, c1)
	}

	c.Parent = nil
	assignment.Children = nc

	return err
}

func (assignment *Assignment) consume(b *Assignment) (err error) {
	for _, c := range b.Children {
		if err = c.removeFromParent(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		assignment.addChild(c)
	}

	for _, obj := range b.Objects.All() {
		assignment.AddObject(obj)
	}
	for _, obj := range b.Objects.All() {
		b.Objects.Del(obj)
	}

	if err = b.removeFromParent(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (assignment *Assignment) AllTags(tags ids.TagSetMutable) (err error) {
	if assignment == nil {
		return err
	}

	var expandedTags ids.TagSet

	if expandedTags, err = assignment.expandedTags(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for tag := range expandedTags.All() {
		if err = tags.Add(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = assignment.Parent.AllTags(tags); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (assignment *Assignment) expandedTags() (tags ids.TagSet, err error) {
	tags = ids.MakeTagSetFromSlice()

	if assignment.Transacted.GetMetadata().GetTags().Len() != 1 || assignment.Parent == nil {
		tags = ids.CloneTagSet(assignment.Transacted.GetMetadata().GetTags())
		return tags, err
	} else {
		tag := quiter_set.Any(assignment.Transacted.GetMetadata().GetTags())

		if ids.IsDependentLeaf(tag) {
			var parentExpandedTags ids.TagSet

			if parentExpandedTags, err = assignment.Parent.expandedTags(); err != nil {
				err = errors.Wrap(err)
				return tags, err
			}

			if parentExpandedTags.Len() > 1 {
				err = errors.ErrorWithStackf(
					"cannot infer full tag for assignment because parent assignment has more than one tags: %s",
					assignment.Parent.Transacted.GetMetadata().GetTags(),
				)

				return tags, err
			}

			parentTag := quiter_set.Any(parentExpandedTags)

			if ids.IsEmpty(parentTag) {
				err = errors.ErrorWithStackf("parent tag is empty")
				return tags, err
			}

			if err = tag.Set(fmt.Sprintf("%s%s", parentTag, tag)); err != nil {
				err = errors.Wrap(err)
				return tags, err
			}
		}

		tags = ids.MakeTagSetFromSlice(tag)
	}

	return tags, err
}

func (assignment *Assignment) SubtractFromSet(
	tagsToSubtract ids.TagSetMutable,
) (err error) {
	for assignmentTag := range assignment.Transacted.GetMetadata().AllTags() {
		for tagToSubtract := range tagsToSubtract.All() {
			if tagToSubtract.String() != assignmentTag.String() {
				continue
			}

			tagsToSubtract.DelKey(tagToSubtract.String())
		}

		tagsToSubtract.DelKey(assignmentTag.String())
	}

	if assignment.Parent == nil {
		return err
	}

	return assignment.Parent.SubtractFromSet(tagsToSubtract)
}

func (assignment *Assignment) Contains(e *ids.TagStruct) bool {
	if assignment.Transacted.GetMetadata().GetTags().ContainsKey(e.String()) {
		return true
	}

	if assignment.Parent == nil {
		return false
	}

	return assignment.Parent.Contains(e)
}

func (assignment *Assignment) SortChildren() {
	sort.Slice(assignment.Children, func(i, j int) bool {
		iTags := assignment.Children[i].Transacted.GetMetadata().GetTags()
		jTags := assignment.Children[j].Transacted.GetMetadata().GetTags()

		if iTags.Len() == 1 && jTags.Len() == 1 {
			iTag := strings.TrimPrefix(quiter_set.Any(iTags).String(), "-")
			jTag := strings.TrimPrefix(quiter_set.Any(jTags).String(), "-")

			iInt, iErr := strconv.ParseInt(iTag, 0, 64)
			jInt, jErr := strconv.ParseInt(jTag, 0, 64)

			if iErr == nil && jErr == nil {
				return iInt < jInt
			} else {
				return iTag < jTag
			}
		} else {
			iValue := quiter.StringCommaSeparated(iTags)
			jValue := quiter.StringCommaSeparated(jTags)

			return iValue < jValue
		}
	})
}
