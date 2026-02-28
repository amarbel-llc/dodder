package store_config

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

func init() {
	collections_value.RegisterGobValue[*tag](nil)
}

type implicitTagMap map[string]ids.TagSetMutable

func (implicitTags implicitTagMap) Contains(to, imp ids.Tag) bool {
	tagSet, ok := implicitTags[to.String()]

	if !ok || tagSet == nil {
		return false
	}

	if !tagSet.ContainsKey(imp.String()) {
		return false
	}

	return true
}

func (implicitTags implicitTagMap) Set(to, imp ids.Tag) (err error) {
	set, ok := implicitTags[to.String()]

	if !ok {
		set = ids.MakeTagSetMutable()
		implicitTags[to.String()] = set
	}

	ids.TagSetMutableAdd(set, imp)

	return err
}

type tag struct {
	Transacted sku.Transacted
	Computed   bool
}

func TagComparePtr(left, right *tag) cmp.Result {
	return sku.TransactedCompare(&left.Transacted, &right.Transacted)
}

func (tag *tag) Less(b *tag) bool {
	return sku.TransactedLessor.Less(&tag.Transacted, &b.Transacted)
}

func (tag *tag) Equals(b *tag) bool {
	if !tag.Transacted.Equals(&b.Transacted) {
		return false
	}

	if !quiter_set.Equals(
		tag.Transacted.GetMetadata().GetIndex().GetImplicitTags(),
		b.Transacted.GetMetadata().GetIndex().GetImplicitTags()) {
		return false
	}

	return true
}

func (tag *tag) Set(v string) (err error) {
	if err = tag.Transacted.ObjectId.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tag *tag) String() string {
	return tag.Transacted.GetObjectId().String()
}

func (compiled *compiled) AccumulateImplicitTags(
	tag ids.Tag,
) (err error) {
	compiledTag, ok := compiled.Tags.Get(tag.String())

	if !ok {
		return err
	}

	expandedTags := expansion.ExpandOneIntoIds[ids.TagStruct](
		tag.String(),
		expansion.ExpanderRight,
	)

	for expandedTag := range expandedTags {
		if ids.TagEquals(expandedTag, tag) {
			continue
		}

		if err = compiled.AccumulateImplicitTags(expandedTag); err != nil {
			err = errors.Wrap(err)
			return err
		}

		for implicitTag := range compiled.GetImplicitTags(expandedTag).All() {
			if err = compiled.ImplicitTags.Set(tag, implicitTag); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	for compiledTag := range compiledTag.Transacted.GetMetadata().AllTags() {
		if compiled.ImplicitTags.Contains(compiledTag, tag) {
			continue
		}

		if err = compiled.ImplicitTags.Set(tag, compiledTag); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = compiled.AccumulateImplicitTags(compiledTag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (compiled *compiled) addTag(
	daughter *sku.Transacted,
	mother *sku.Transacted,
) (didChange bool, err error) {
	compiled.lock.Lock()
	defer compiled.lock.Unlock()

	var tag tag

	sku.Resetter.ResetWith(&tag.Transacted, daughter)

	if didChange, err = quiter.AddOrReplaceIfGreater(
		compiled.Tags,
		&tag,
		TagComparePtr,
	); err != nil {
		err = errors.Wrap(err)
		return didChange, err
	}

	return didChange, err
}
