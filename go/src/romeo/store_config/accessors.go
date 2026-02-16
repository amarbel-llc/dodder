package store_config

import (
	"sort"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/expansion"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/file_extensions"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

func (config Config) GetFileExtensions() file_extensions.Config {
	return config.FileExtensions
}

func (compiled *compiled) getType(k ids.Id) (ct *sku.Transacted) {
	if k.GetGenre() != genres.Type {
		return ct
	}

	if ct1, ok := compiled.Types.Get(k.String()); ok {
		ct, _ = ct1.CloneTransacted()
	}

	return ct
}

func (compiled *compiled) getRepo(k ids.Id) (ct *sku.Transacted) {
	if k.GetGenre() != genres.Repo {
		return ct
	}

	if ct1, ok := compiled.Repos.Get(k.String()); ok {
		ct, _ = ct1.CloneTransacted()
	}

	return ct
}

// Returns the exactly matching Typ, or if it doesn't exist, returns the parent
// Typ or nil. (Parent Typ for `md-gdoc` would be `md`.)
func (compiled *compiled) GetApproximatedType(
	id domain_interfaces.ObjectId,
) (ct ApproximatedType) {
	if id.GetGenre() != genres.Type {
		return ct
	}

	expandedActual := compiled.getSortedTypesExpanded(id.String())
	if len(expandedActual) > 0 {
		ct.HasValue = true
		ct.Type = expandedActual[0]

		if ids.Equals(ct.Type.GetObjectId(), id) {
			ct.IsActual = true
		}
	}

	return ct
}

func (compiled *compiled) GetTagOrRepoIdOrType(
	objectIdString string,
) (object *sku.Transacted, err error) {
	var objectId ids.ObjectId

	if err = objectId.Set(objectIdString); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	switch objectId.GetGenre() {
	case genres.Tag:
		object, _ = compiled.getTag(&objectId)

	case genres.Repo:
		object = compiled.getRepo(&objectId)

	case genres.Type:
		object = compiled.getType(&objectId)

	default:
		err = genres.MakeErrUnsupportedGenre(&objectId)
		return object, err
	}

	return object, err
}

func (compiled *compiled) getTag(
	objectId ids.Id,
) (object *sku.Transacted, ok bool) {
	if objectId.GetGenre() != genres.Tag {
		return object, ok
	}

	compiled.lock.Lock()
	defer compiled.lock.Unlock()

	var cursor *tag

	seq := expansion.ExpandOneIntoIds[ids.TagStruct](
		objectId.String(),
		expansion.ExpanderRight,
	)

	for expandedTag := range seq {
		if cursor == nil {
			cursor, _ = compiled.Tags.Get(expandedTag.String())
			continue
		}

		next, ok := compiled.Tags.Get(expandedTag.String())

		if !ok {
			continue
		}

		if len(
			next.Transacted.GetObjectId().String(),
		) > len(
			cursor.Transacted.GetObjectId().String(),
		) {
			cursor = next
		}
	}

	if cursor != nil {
		object, _ = sku.GetTransactedPool().GetWithRepool()
		sku.Resetter.ResetWith(object, &cursor.Transacted)
	}

	return object, ok
}

// TODO-P3 merge all the below
func (compiled *compiled) getSortedTypesExpanded(
	typeString string,
) (expandedActual []*sku.Transacted) {
	expandedActual = make([]*sku.Transacted, 0)

	seq := expansion.ExpandOneIntoIds[ids.SeqId](
		typeString,
		expansion.ExpanderRight,
	)

	for expandedType := range seq {
		compiled.lock.Lock()
		typeObject, ok := compiled.Types.Get(expandedType.String())
		compiled.lock.Unlock()

		if ok {
			expandedActual = append(expandedActual, typeObject)
		}
	}

	sort.Slice(expandedActual, func(i, j int) bool {
		return len(
			expandedActual[i].GetObjectId().String(),
		) > len(
			expandedActual[j].GetObjectId().String(),
		)
	})

	return expandedActual
}

func (compiled *compiled) GetImplicitTags(tag ids.Tag) ids.TagSet {
	s, ok := compiled.ImplicitTags[tag.String()]

	if !ok || s == nil {
		return ids.MakeTagSetFromSlice()
	}

	return s
}
