package object_probe_index

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/page_id"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_ui"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

const (
	DigitWidth = 1
	PageCount  = 1 << (DigitWidth * 4)
)

type Index struct {
	hashType markl.FormatHash
	rowWidth int
	pages    [PageCount]page
}

func MakePermitDuplicates(
	envRepo env_repo.Env,
	path string,
	hashType markl.FormatHash,
) (index *Index, err error) {
	index = &Index{hashType: hashType}
	err = index.initialize(rowEqualerComplete{}, envRepo, path)
	return index, err
}

func MakeNoDuplicates(
	envRepo env_repo.Env,
	dir string,
	hashType markl.FormatHash,
) (index *Index, err error) {
	index = &Index{hashType: hashType}
	err = index.initialize(rowEqualerDigestOnly{}, envRepo, dir)
	return index, err
}

func (index *Index) initialize(
	equaler interfaces.Equaler[*row],
	envRepo env_repo.Env,
	dir string,
) (err error) {
	index.rowWidth = index.hashType.GetSize() + 1 + 8 + 8

	for pageIndex := range index.pages {
		page := &index.pages[pageIndex]
		page.initialize(
			equaler,
			envRepo,
			page_id.PageIdFromPath(uint8(pageIndex), dir),
			index.hashType,
			index.rowWidth,
		)
	}

	return err
}

func (index *Index) GetHashType() markl.FormatHash {
	return index.hashType
}

func (index *Index) AddDigest(
	probeId ids.ProbeIdWithObjectId,
	loc Loc,
) (err error) {
	if probeId.Id.IsNull() {
		return err
	}

	id := probeId.Id

	if probeId.Id.GetMarklFormat().GetMarklFormatId() != index.hashType.GetMarklFormatId() {
		replacementId, repool := index.hashType.GetMarklIdForMarklId(probeId.Id)
		defer repool()

		id = replacementId
	}

	var pageIndex uint8

	if pageIndex, err = page_id.PageIndexForDigest(
		DigitWidth,
		id,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = index.pages[pageIndex].AddMarklId(id, loc); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return err
	}

	return err
}

func (index *Index) ReadOne(
	originalId domain_interfaces.MarklId,
) (loc Loc, err error) {
	id := originalId

	if id.GetMarklFormat().GetMarklFormatId() != index.hashType.GetMarklFormatId() {
		replacementId, repool := index.hashType.GetMarklIdForMarklId(id)
		defer repool()

		id = replacementId
	}

	var pageIndex uint8

	if pageIndex, err = page_id.PageIndexForDigest(
		DigitWidth,
		id,
	); err != nil {
		return loc, err
	}

	if loc, err = index.pages[pageIndex].ReadOne(id); err != nil {
		return loc, err
	}

	return loc, err
}

func (index *Index) ReadMany(
	originalId domain_interfaces.MarklId,
	locations *[]Loc,
) (err error) {
	id := originalId

	if id.GetMarklFormat().GetMarklFormatId() != index.hashType.GetMarklFormatId() {
		replacementId, repool := index.hashType.GetMarklIdForMarklId(id)
		defer repool()

		id = replacementId
	}

	var pageIndex uint8

	if pageIndex, err = page_id.PageIndexForDigest(
		DigitWidth,
		id,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = index.pages[pageIndex].ReadMany(id, locations); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) PrintAll(env env_ui.Env) (err error) {
	for pageIndex := range index.pages {
		page := &index.pages[pageIndex]

		if err = page.PrintAll(env); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (index *Index) Reset() (err error) {
	for pageIndex := range index.pages {
		page := &index.pages[pageIndex]

		if page.file != nil {
			if err = page.file.Close(); err != nil {
				err = errors.Wrap(err)
				return err
			}

			page.file = nil
		}

		page.added.Reset()
	}

	return err
}

func (index *Index) Flush() (err error) {
	waitGroup := errors.MakeWaitGroupParallel()

	for pageIndex := range index.pages {
		page := &index.pages[pageIndex]
		waitGroup.Do(page.Flush)
	}

	return waitGroup.GetError()
}
