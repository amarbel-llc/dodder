package organize_text

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type constructor2 struct {
	Text
	all PrefixSet
}

func (c *constructor2) collectExplicitAndImplicitFor(
	skus sku.SkuTypeSet,
	re ids.TagStruct,
) (explicitCount, implicitCount int, err error) {
	res, resRepool := catgut.MakeFromString(re.String())
	defer resRepool()

	for checkedOut := range skus.All() {
		object := checkedOut.GetSkuExternal()

		for _, tag := range object.GetMetadata().GetIndex().GetTagPaths().All {
			if tag.Tag.String() == object.ObjectId.String() {
				continue
			}

			cmp := tag.ComparePartial(res)

			if !cmp.IsEqual() {
				continue
			}

			if len(tag.Parents) == 0 { // TODO use Type
				explicitCount++
				break
			}

			for _, p := range tag.Parents {
				if p.Type == tag_paths.TypeDirect {
					explicitCount++
				} else {
					implicitCount++
				}
			}
		}
	}

	return explicitCount, implicitCount, err
}

func (c *constructor2) preparePrefixSetsAndRootsAndExtras() (err error) {
	anchored := ids.MakeTagSetMutable()
	extras := ids.MakeTagSetMutable()

	for re := range c.TagSet.All() {
		var explicitCount, implicitCount int

		if explicitCount, implicitCount, err = c.collectExplicitAndImplicitFor(
			c.Options.Skus,
			re,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ui.Log().Print(re, "explicit", explicitCount, "implicit", implicitCount)

		// TODO [radi/du !task project-2021-zit-etiketten_and_organize zz-inbox] fix issue with `zit organize project-2021-zit` causing an extra tag…
		if explicitCount == c.Options.Skus.Len() {
			if err = anchored.Add(re); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else if explicitCount > 0 {
			if err = extras.Add(re); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	c.TagSet = anchored
	c.ExtraTags = extras

	return err
}

func (c *constructor2) populate() (err error) {
	allUsed := makeObjSet()

	for e := range c.ExtraTags.All() {
		ee := c.makeChild(e)

		segments := c.all.Subset(e)

		if err = c.makeChildrenWithPossibleGroups(
			ee,
			segments.Grouped,
			c.GroupingTags,
			allUsed,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = c.makeChildrenWithoutGroups(
			ee,
			func(f interfaces.FuncIter[*obj]) error {
				for element := range segments.Ungrouped.All() {
					if err := f(element); err != nil {
						return err
					}
				}
				return nil
			},
			allUsed,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	c.all = c.all.Subtract(allUsed)

	if err = c.makeChildrenWithPossibleGroups(
		c.Assignment,
		c.all,
		c.GroupingTags,
		allUsed,
	); err != nil {
		err = errors.Wrapf(err, "Assignment: %#v", c.Assignment)
		return err
	}

	return err
}

func (c *constructor2) makeChildrenWithoutGroups(
	parent *Assignment,
	fi func(interfaces.FuncIter[*obj]) error,
	used objSet,
) (err error) {
	if err = fi(used.Add); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = c.makeAndAddUngrouped(parent, fi); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (c *constructor2) makeChildrenWithPossibleGroups(
	parent *Assignment,
	prefixSet PrefixSet,
	groupingTags ids.TagSlice,
	used objSet,
) (err error) {
	if groupingTags.Len() == 0 {
		for _, tz := range prefixSet.AllObjects() {
			var z *obj

			if z, err = c.cloneObj(tz); err != nil {
				err = errors.Wrap(err)
				return err
			}

			parent.AddObject(z)

			if err = used.Add(tz); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		return err
	}

	segments := prefixSet.Subset(groupingTags[0])

	if err = c.makeAndAddUngrouped(parent, func(f interfaces.FuncIter[*obj]) error {
		for element := range segments.Ungrouped.All() {
			if err := f(element); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = c.addGroupedChildren(
		parent,
		segments.Grouped,
		groupingTags,
		used,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	parent.SortChildren()

	return err
}

func (c *constructor2) addGroupedChildren(
	parent *Assignment,
	grouped PrefixSet,
	groupingTags ids.TagSlice,
	used objSet,
) (err error) {
	for eStr, zs := range grouped.AllObjectSets() {
		var e ids.TagStruct
		if eStr != "" {
			e = ids.MustTag(eStr)
		}

		if e.IsEmpty() || c.TagSet.ContainsKey(e.String()) {
			if err = c.makeAndAddUngrouped(parent, func(f interfaces.FuncIter[*obj]) error {
				for element := range zs.All() {
					if err := f(element); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				err = errors.Wrap(err)
				return err
			}

			for element := range zs.All() {
				if err = used.Add(element); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			continue
		}

		child := newAssignment(parent.GetDepth() + 1)
		objects.SetTags(child.Transacted.GetMetadataMutable(), ids.MakeTagSetMutable(e))
		groupingTags.DropFirst()

		psv := MakePrefixSetFrom(zs)

		if err = c.makeChildrenWithPossibleGroups(
			child,
			psv,
			groupingTags,
			used,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		parent.addChild(child)
	}

	return err
}

func (c *constructor2) makeAndAddUngrouped(
	parent *Assignment,
	fi func(interfaces.FuncIter[*obj]) error,
) (err error) {
	if err = fi(
		func(tz *obj) (err error) {
			var z *obj

			if z, err = c.cloneObj(tz); err != nil {
				err = errors.Wrap(err)
				return err
			}

			parent.AddObject(z)

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}
	return err
}

func (c *constructor2) cloneObj(
	named *obj,
) (z *obj, err error) {
	clonedSku, _ := sku.CloneSkuType(named.sku)
	z = &obj{
		tipe: named.tipe,
		sku:  clonedSku,
	}

	// TODO explore using shas as keys
	// if named.External.GetSkuExternal().Metadata.Shas.SelfMetadataWithoutTai.IsNull() {
	// 	panic("empty sha")
	// }

	// if z.External.GetSkuExternal().Metadata.Shas.SelfMetadataWithoutTai.IsNull() {
	// 	panic("empty sha")
	// }

	return z, err
}
