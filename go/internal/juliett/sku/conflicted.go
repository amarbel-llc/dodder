package sku

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
)

type ParentNegotiator interface {
	FindBestCommonAncestor(Conflicted) (*Transacted, error)
}

// TODO consider making this a ConflictedWithBase and ConflictedWithoutBase
// and an interface for both
type Conflicted struct {
	*CheckedOut
	Local, Base, Remote *Transacted
}

func (conflicted Conflicted) String() string {
	return fmt.Sprintf(
		"Local: %q\nBase: %q\nRemote: %q\n",
		String(conflicted.Local),
		String(conflicted.Base),
		String(conflicted.Remote),
	)
}

func (conflicted *Conflicted) FindBestCommonAncestor(
	negotiator ParentNegotiator,
) (err error) {
	if negotiator == nil {
		return err
	}

	if conflicted.Base, err = negotiator.FindBestCommonAncestor(*conflicted); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (conflicted Conflicted) GetCollection() Collection {
	return conflicted
}

func (conflicted Conflicted) Len() int {
	if conflicted.Base == nil {
		return 2
	} else {
		return 3
	}
}

func (conflicted Conflicted) Any() *Transacted {
	return conflicted.Local
}

func (conflicted Conflicted) All() interfaces.Seq[*Transacted] {
	return func(yield func(*Transacted) bool) {
		if !yield(conflicted.Local) {
			return
		}

		if conflicted.Base != nil && !yield(conflicted.Base) {
			return
		}

		if !yield(conflicted.Remote) {
			return
		}
	}
}

func (conflicted Conflicted) IsAllInlineType(
	typeChecker ids.InlineTypeChecker,
) bool {
	if typeChecker == nil {
		panic("nil type checker")
	}

	if conflicted.Local == nil {
		panic("nil local")
	}

	if !typeChecker.IsInlineType(conflicted.Local.GetType()) {
		return false
	}

	if conflicted.Base != nil &&
		!typeChecker.IsInlineType(conflicted.Base.GetType()) {
		return false
	}

	if !typeChecker.IsInlineType(conflicted.Remote.GetType()) {
		return false
	}

	return true
}

func (conflicted *Conflicted) MergeTags() (err error) {
	if conflicted.Base == nil {
		return err
	}

	left := ids.CloneTagSetMutable(conflicted.Local.GetTags())
	middle := ids.CloneTagSetMutable(conflicted.Base.GetTags())
	right := ids.CloneTagSetMutable(conflicted.Remote.GetTags())

	same := ids.MakeTagSetMutable()
	deleted := ids.MakeTagSetMutable()

	removeFromAllButAddTo := func(
		tag ids.TagStruct, tagSet ids.TagSetMutable,
	) (err error) {
		if err = tagSet.Add(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}

		quiter_set.Del(left, tag)
		quiter_set.Del(middle, tag)

		quiter_set.Del(right, tag)

		return err
	}

	for tag := range middle.All() {
		if left.ContainsKey(left.Key(tag)) &&
			right.ContainsKey(right.Key(tag)) {
			if err = removeFromAllButAddTo(tag, same); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else if left.ContainsKey(left.Key(tag)) || right.ContainsKey(right.Key(tag)) {
			if err = removeFromAllButAddTo(tag, deleted); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	for tag := range left.All() {
		if err = same.Add(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for tag := range right.All() {
		if err = same.Add(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	tags := ids.CloneTagSet(same)

	objects.SetTags(conflicted.Local.GetMetadataMutable(), tags)
	objects.SetTags(conflicted.Base.GetMetadataMutable(), tags)
	objects.SetTags(conflicted.Remote.GetMetadataMutable(), tags)

	return err
}

func (conflicted *Conflicted) ReadConflictMarker(
	seq interfaces.SeqError[*Transacted],
) (err error) {
	idx := 0

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return err
		}

		switch idx {
		case 0:
			conflicted.Local = object

		case 1:
			conflicted.Base = object

		case 2:
			conflicted.Remote = object

		default:
			err = errors.ErrorWithStackf("too many objects in conflict file")
			return err
		}

		idx++
	}

	if idx == 0 {
		err = errors.ErrorWithStackf("no objects in conflict file")
		return err
	}

	// Conflicts can exist between objects without a base
	if idx == 2 {
		conflicted.Remote = conflicted.Base
		conflicted.Base = nil
	}

	return err
}
