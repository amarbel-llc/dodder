package organize_text

import (
	"fmt"
	"sort"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
)

func MakeSkuMapWithOrder(c int) (out SkuMapWithOrder) {
	out.m = make(map[string]skuTypeWithIndex, c)
	return out
}

type skuTypeWithIndex struct {
	sku sku.SkuType
	int
}

type SkuMapWithOrder struct {
	m    map[string]skuTypeWithIndex
	next int
}

func (smwo *SkuMapWithOrder) AsExternalLikeSet() sku.SkuTypeSetMutable {
	elms := sku.MakeSkuTypeSetMutable()

	for _, sk := range smwo.AllSkuAndIndex() {
		errors.PanicIfError(elms.Add(sk))
	}

	return elms
}

func (smwo *SkuMapWithOrder) AsTransactedSet() sku.TransactedMutableSet {
	tms := sku.MakeTransactedMutableSet()

	for _, el := range smwo.AllSkuAndIndex() {
		errors.PanicIfError(tms.Add(el.GetSkuExternal()))
	}

	return tms
}

func (sm *SkuMapWithOrder) Del(sk sku.SkuType) error {
	delete(sm.m, keyer.GetKey(sk))
	return nil
}

func (sm *SkuMapWithOrder) Add(sk sku.SkuType) error {
	k := keyer.GetKey(sk)
	entry, ok := sm.m[k]

	if !ok {
		entry.int = sm.next
		entry.sku = sk
		sm.next++
	}

	sm.m[k] = entry

	return nil
}

func (sm *SkuMapWithOrder) Len() int {
	return len(sm.m)
}

func (sm *SkuMapWithOrder) Clone() (out SkuMapWithOrder) {
	out = MakeSkuMapWithOrder(sm.Len())

	for _, v := range sm.m {
		out.Add(v.sku)
	}

	return out
}

func (sm SkuMapWithOrder) Sorted() (out []sku.SkuType) {
	out = make([]sku.SkuType, 0, sm.Len())

	for _, v := range sm.m {
		out = append(out, v.sku)
	}

	sort.Slice(out, func(i, j int) bool {
		iObject := out[i].GetSkuExternal()
		jObject := out[j].GetSkuExternal()

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

	return out
}

func (smwo *SkuMapWithOrder) AllSkuAndIndex() interfaces.Seq2[int, sku.SkuType] {
	return func(yield func(int, sku.SkuType) bool) {
		for i, sk := range smwo.Sorted() {
			if !yield(i, sk) {
				break
			}
		}
	}
}

type Changes struct {
	Before, After  SkuMapWithOrder
	Added, Removed SkuMapWithOrder
	Changed        SkuMapWithOrder
}

func (c Changes) String() string {
	return fmt.Sprintf(
		"Before: %d, After: %d, Added: %d, Removed: %d, Changed: %d",
		c.Before.Len(),
		c.After.Len(),
		c.Added.Len(),
		c.Removed.Len(),
		c.Changed.Len(),
	)
}

// TODO combine with above
type OrganizeResults struct {
	Before, After *Text
	Original      sku.SkuTypeSet
	QueryGroup    *queries.Query
}

func ChangesFrom(
	po options_print.Options,
	a, b *Text,
	original sku.SkuTypeSet,
) (c Changes, err error) {
	if c, err = ChangesFromResults(
		po,
		OrganizeResults{
			Before:   a,
			After:    b,
			Original: original,
		}); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	return c, err
}

func ChangesFromResults(
	po options_print.Options,
	results OrganizeResults,
) (c Changes, err error) {
	if err = applyToText(po, results.Before); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	if c.Before, err = results.Before.GetSkus(results.Original); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	if c.After, err = results.After.GetSkus(results.Original); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	c.Changed = c.After.Clone()
	c.Removed = c.Before.Clone()

	for _, sk := range c.After.m {
		if err = c.Removed.Del(sk.sku); err != nil {
			err = errors.Wrap(err)
			return c, err
		}
	}

	for _, sk := range c.Removed.AllSkuAndIndex() {
		if err = results.Before.RemoveFromTransacted(sk); err != nil {
			err = errors.Wrap(err)
			return c, err
		}

		if err = c.Changed.Add(sk); err != nil {
			err = errors.Wrap(err)
			return c, err
		}
	}

	return c, err
}

func applyToText(
	po options_print.Options,
	t *Text,
) (err error) {
	if po.BoxPrintTagsAlways {
		return err
	}

	for el := range t.Options.Skus.All() {
		sk := el.GetSkuExternal()

		if sk.GetMetadata().GetDescription().IsEmpty() {
			continue
		}

		sk.GetMetadataMutable().ResetTags()
	}

	return err
}
