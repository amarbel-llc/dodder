package queries

import (
	"maps"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type expSigilAndGenre struct {
	ids.Sigil
	ids.Genre

	exp

	Hidden sku.Query
}

func (expSigilAndGenre *expSigilAndGenre) IsEmpty() bool {
	return expSigilAndGenre.Sigil == ids.SigilUnknown &&
		expSigilAndGenre.Genre.IsEmpty() &&
		expSigilAndGenre.exp.IsEmpty()
}

func (expSigilAndGenre *expSigilAndGenre) GetSigil() ids.Sigil {
	return expSigilAndGenre.Sigil
}

func (expSigilAndGenre *expSigilAndGenre) addPinnedObjectId(
	buildState *buildState,
	pinnedObjectId pinnedObjectId,
) (err error) {
	if err = expSigilAndGenre.addExactObjectId(
		buildState,
		pinnedObjectId.ObjectId,
		pinnedObjectId.Sigil,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (expSigilAndGenre *expSigilAndGenre) addExactObjectId(
	buildState *buildState,
	objectId ObjectId,
	sigil ids.Sigil,
) (err error) {
	if objectId.ObjectId == nil {
		err = errors.ErrorWithStackf("nil object id")
		return err
	}

	expSigilAndGenre.Sigil.Add(sigil)
	expSigilAndGenre.expObjectIds.internal[objectId.GetObjectId().String()] = objectId
	expSigilAndGenre.Genre.Add(genres.Must(objectId))

	return err
}

func (expSigilAndGenre *expSigilAndGenre) ContainsObjectId(
	objectId *ids.ObjectId,
) bool {
	if !expSigilAndGenre.Genre.Contains(objectId.GetGenre()) {
		err := errors.ErrorWithStackf(
			"checking query %#v for object id %#v, %q, %q",
			expSigilAndGenre,
			objectId,
			expSigilAndGenre,
			objectId,
		)
		panic(err)
	}

	if len(expSigilAndGenre.expObjectIds.internal) == 0 {
		return false
	}

	_, ok := expSigilAndGenre.expObjectIds.internal[objectId.String()]

	return ok
}

func (a *expSigilAndGenre) Clone() (b *expSigilAndGenre) {
	b = &expSigilAndGenre{
		Sigil: a.Sigil,
		Genre: a.Genre,
		exp: exp{
			expObjectIds: expObjectIds{
				internal: make(
					map[string]ObjectId,
					len(a.expObjectIds.internal),
				),
				external: make(
					map[string]sku.ExternalObjectId,
					len(a.expObjectIds.external),
				),
			},
		},
		Hidden: a.Hidden,
	}

	bExp := a.expTagsOrTypes.Clone()
	b.expTagsOrTypes = *bExp

	maps.Copy(b.expObjectIds.internal, a.expObjectIds.internal)

	for k, v := range a.expObjectIds.external {
		b.expObjectIds.external[k] = v
	}

	return b
}

func (q *expSigilAndGenre) Add(m sku.Query) (err error) {
	q1, ok := m.(*expSigilAndGenre)

	if !ok {
		return q.expTagsOrTypes.Add(m)
	}

	if q1.Genre != q.Genre {
		err = errors.ErrorWithStackf(
			"expected %q but got %q",
			q.Genre,
			q1.Genre,
		)

		return err
	}

	if err = q.Merge(q1); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (expSigilAndGenre *expSigilAndGenre) Merge(
	exp *expSigilAndGenre,
) (err error) {
	expSigilAndGenre.Sigil.Add(exp.Sigil)

	if expSigilAndGenre.expObjectIds.internal == nil {
		expSigilAndGenre.expObjectIds.internal = make(
			map[string]ObjectId,
			len(exp.expObjectIds.internal),
		)
	}

	for _, internalHoistedId := range exp.expObjectIds.internal {
		idString := internalHoistedId.GetObjectId().String()
		expSigilAndGenre.expObjectIds.internal[idString] = internalHoistedId
	}

	if expSigilAndGenre.expObjectIds.external == nil {
		expSigilAndGenre.expObjectIds.external = make(
			map[string]sku.ExternalObjectId,
			len(exp.expObjectIds.external),
		)
	}

	for _, externalObjectId := range exp.expObjectIds.external {
		idString := externalObjectId.String()
		expSigilAndGenre.expObjectIds.external[idString] = externalObjectId
	}

	expSigilAndGenre.expTagsOrTypes.Children = append(
		expSigilAndGenre.expTagsOrTypes.Children,
		exp.expTagsOrTypes.Children...)

	return err
}

func (expSigilAndGenre *expSigilAndGenre) writeSigilAndGenreSuffix(
	sb *strings.Builder,
) {
	if expSigilAndGenre.IsEmpty() && !expSigilAndGenre.IsLatestOrUnknown() {
		sb.WriteString(expSigilAndGenre.Sigil.String())
	} else if !expSigilAndGenre.IsEmpty() {
		sb.WriteString(expSigilAndGenre.Sigil.String())
		sb.WriteString(expSigilAndGenre.Genre.String())
	}
}

func (expSigilAndGenre *expSigilAndGenre) StringDebug() string {
	var sb strings.Builder

	if expSigilAndGenre.expObjectIds.internal == nil ||
		len(expSigilAndGenre.expObjectIds.internal) == 0 {
		sb.WriteString(expSigilAndGenre.expTagsOrTypes.StringDebug())
	} else {
		sb.WriteString("[[")

		first := true

		for _, id := range expSigilAndGenre.expObjectIds.internal {
			if !first {
				sb.WriteString(", ")
			}

			sb.WriteString(id.String())

			first = false
		}

		sb.WriteString(", ")
		sb.WriteString(expSigilAndGenre.expTagsOrTypes.StringDebug())
		sb.WriteString("]")
	}

	expSigilAndGenre.writeSigilAndGenreSuffix(&sb)

	return sb.String()
}

func (expSigilAndGenre *expSigilAndGenre) SortedObjectIds() []string {
	out := make([]string, 0, expSigilAndGenre.expObjectIds.Len())

	for k := range expSigilAndGenre.expObjectIds.internal {
		out = append(out, k)
	}

	for k := range expSigilAndGenre.expObjectIds.external {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

func (expSigilAndGenre *expSigilAndGenre) String() string {
	var sb strings.Builder

	e := expSigilAndGenre.expTagsOrTypes.String()

	oids := expSigilAndGenre.SortedObjectIds()

	if len(oids) == 0 {
		sb.WriteString(e)
	} else if len(oids) == 1 && e == "" {
		for _, k := range oids {
			sb.WriteString(k)
		}
	} else {
		sb.WriteString("[")

		first := true

		for _, k := range oids {
			if !first {
				sb.WriteString(", ")
			}

			sb.WriteString(k)

			first = false
		}

		if e != "" {
			sb.WriteString(", ")
			sb.WriteString(expSigilAndGenre.expTagsOrTypes.String())
		}

		sb.WriteString("]")
	}

	expSigilAndGenre.writeSigilAndGenreSuffix(&sb)

	return sb.String()
}

func (expSigilAndGenre *expSigilAndGenre) ShouldHide(
	objectGetter sku.TransactedGetter,
	objectIdString string,
) bool {
	_, ok := expSigilAndGenre.expObjectIds.internal[objectIdString]

	if expSigilAndGenre.IncludesHidden() || expSigilAndGenre.Hidden == nil ||
		ok {
		return false
	}

	return expSigilAndGenre.Hidden.ContainsSku(objectGetter)
}

func (expSigilAndGenre *expSigilAndGenre) ContainsSku(
	objectGetter sku.TransactedGetter,
) (ok bool) {
	object := objectGetter.GetSku()

	objectIdString := object.ObjectId.String()

	genre := genres.Must(object)

	if expSigilAndGenre.ShouldHide(object, objectIdString) {
		return ok
	}

	if !expSigilAndGenre.Genre.ContainsOneOf(genre) {
		return ok
	}

	if _, ok = expSigilAndGenre.expObjectIds.internal[objectIdString]; ok {
		return ok
	}

	if len(expSigilAndGenre.expTagsOrTypes.Children) == 0 {
		ok = len(expSigilAndGenre.expObjectIds.internal) == 0
		return ok
	} else if !expSigilAndGenre.expTagsOrTypes.ContainsSku(objectGetter) {
		return ok
	}

	ok = true

	return ok
}

func (expSigilAndGenre *expSigilAndGenre) ContainsExternalSku(
	el sku.ExternalLike,
) (ok bool) {
	object := el.GetSku()

	genre := genres.Must(object)

	if !expSigilAndGenre.Genre.ContainsOneOf(genre) {
		return ok
	}

	objectIdString := object.ObjectId.String()

	if expSigilAndGenre.ShouldHide(el, objectIdString) {
		return ok
	}

	externalObjectIdString := el.GetExternalObjectId().String()
	ui.Log().Print(
		externalObjectIdString,
		expSigilAndGenre.expObjectIds.external,
		expSigilAndGenre.expObjectIds.internal,
	)

	if _, ok = expSigilAndGenre.expObjectIds.external[externalObjectIdString]; ok {
		return ok
	}

	if _, ok = expSigilAndGenre.expObjectIds.external[objectIdString]; ok {
		return ok
	}

	if _, ok = expSigilAndGenre.expObjectIds.internal[objectIdString]; ok {
		return ok
	}

	if len(expSigilAndGenre.expTagsOrTypes.Children) == 0 {
		ok = expSigilAndGenre.expObjectIds.IsEmpty()
		return ok
	} else if !expSigilAndGenre.expTagsOrTypes.ContainsSku(el) {
		return ok
	}

	ok = true

	return ok
}
