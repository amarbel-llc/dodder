package stream_index_fixed

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_probe_index"
	"code.linenisgreat.com/dodder/go/lib/alfa/collections_map"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type probeIndex struct {
	defaultObjectDigestMarklFormatId string
	index                            *object_probe_index.Index
	additionProbes                   collections_map.Map[string, *sku.Transacted]
}

func (index *probeIndex) Initialize(
	envRepo env_repo.Env,
	hashType markl.FormatHash,
) (err error) {
	index.defaultObjectDigestMarklFormatId = envRepo.GetObjectDigestType()

	if index.index, err = object_probe_index.MakeNoDuplicates(
		envRepo,
		envRepo.DirIndexObjectPointers(),
		hashType,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	index.additionProbes = make(collections_map.Map[string, *sku.Transacted])

	return err
}

func (index *probeIndex) Reset() (err error) {
	if err = index.index.Reset(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	index.additionProbes.Reset()

	return err
}

func (index *probeIndex) Flush() (err error) {
	if err = index.index.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	index.additionProbes.Reset()

	return err
}

func (index *probeIndex) readOneMarklIdLoc(
	blobId domain_interfaces.MarklId,
) (loc object_probe_index.Loc, err error) {
	if loc, err = index.index.ReadOne(blobId); err != nil {
		return loc, err
	}

	return loc, err
}

func (index *probeIndex) readManyMarklIdLoc(
	blobId domain_interfaces.MarklId,
) (locs []object_probe_index.Loc, err error) {
	if err = index.index.ReadMany(blobId, &locs); err != nil {
		return locs, err
	}

	return locs, err
}

func (index *probeIndex) saveOneObjectLoc(
	object *sku.Transacted,
	loc object_probe_index.Loc,
) (err error) {
	for probeId := range object.AllProbeIds(
		index.index.GetHashType(),
		index.defaultObjectDigestMarklFormatId,
	) {
		if err = index.index.AddDigest(
			ids.ProbeIdWithObjectId{
				ObjectId: object.GetObjectId(),
				ProbeId:  probeId,
			},
			loc,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
