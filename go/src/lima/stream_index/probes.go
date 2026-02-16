package stream_index

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/page_id"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/object_probe_index"
)

func (index *Index) ReadOneMarklIdAdded(
	marklId domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	additionObject, ok := index.additionProbes.Get(string(marklId.GetBytes()))

	if ok {
		sku.TransactedResetter.ResetWith(object, additionObject)
		return ok
	}

	return ok
}

func (index *Index) ReadOneMarklId(
	marklId domain_interfaces.MarklId,
	object *sku.Transacted,
) (ok bool) {
	errors.PanicIfError(markl.AssertIdIsNotNull(marklId))

	var loc object_probe_index.Loc

	{
		var err error

		// TODO migrate to panic semantics
		if loc, err = index.readOneMarklIdLoc(marklId); err != nil {
			if errors.IsNotExist(err) || errors.IsErrNotFound(err) {
				return ok
			} else {
				panic(err)
			}
		}
	}

	ok = index.readOneLoc(loc, object)

	return ok
}

func (index *Index) ReadManyMarklId(
	marklId domain_interfaces.MarklId,
) (objects []*sku.Transacted, err error) {
	// TODO read from page additions if necessary
	var locs []object_probe_index.Loc

	if locs, err = index.readManyMarklIdLoc(marklId); err != nil {
		err = errors.Wrap(err)
		return objects, err
	}

	for _, loc := range locs {
		object, _ := sku.GetTransactedPool().GetWithRepool()

		if !index.readOneLoc(loc, object) {
			err = errors.Errorf("failed to read loc: %s", loc)
			return objects, err
		}

		objects = append(objects, object)
	}

	return objects, err
}

func (index *Index) ObjectExists(
	objectId *ids.ObjectId,
) (err error) {
	var pageIndex uint8

	objectIdString := objectId.String()

	if pageIndex, err = page_id.PageIndexForString(
		DigitWidth,
		objectIdString,
		index.hashType,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	page := index.GetPage(pageIndex)

	if page.objectIdStringExists(objectIdString) {
		return err
	}

	digest, digestRepool := index.hashType.GetMarklIdForString(objectIdString)
	defer digestRepool()

	if _, err = index.readOneMarklIdLoc(digest); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) ReadOneObjectId(
	objectId domain_interfaces.ObjectId,
	object *sku.Transacted,
) (err error) {
	objectIdString := objectId.String()

	if objectIdString == "" {
		panic("empty object id")
	}

	digest, repool := markl.FormatHashSha256.GetMarklIdForString(
		objectIdString,
	)
	defer repool()

	if !index.ReadOneMarklId(digest, object) {
		err = errors.MakeErrNotFoundString(objectIdString)
		return err
	}

	return err
}

func (index *Index) ReadManyObjectId(
	objectId ids.Id,
) (objects []*sku.Transacted, err error) {
	digest, digestRepool := markl.FormatHashSha256.GetMarklIdForString(objectId.String())
	defer digestRepool()

	if objects, err = index.ReadManyMarklId(digest); err != nil {
		err = errors.Wrap(err)
		return objects, err
	}

	return objects, err
}

// TODO switch to empty=not found semantics instead of error
func (index *Index) ReadOneObjectIdTai(
	objectId ids.Id,
	tai ids.Tai,
) (object *sku.Transacted, err error) {
	if tai.IsEmpty() {
		err = errors.MakeErrNotFoundString(tai.String())
		return object, err
	}

	key := objectId.String() + tai.String()

	digest, digestRepool := markl.FormatHashSha256.GetMarklIdForString(key)
	defer digestRepool()

	object, _ = sku.GetTransactedPool().GetWithRepool()

	if !index.ReadOneMarklId(digest, object) {
		err = errors.MakeErrNotFoundString(key)
		return object, err
	}

	return object, err
}

func (index *Index) readOneLoc(
	loc object_probe_index.Loc,
	object *sku.Transacted,
) (ok bool) {
	pageReader, pageReaderClose := index.makeProbePageReader(loc.Page)
	defer errors.Must(pageReaderClose)

	ok = pageReader.readOneCursor(loc.Cursor, object)

	return ok
}
