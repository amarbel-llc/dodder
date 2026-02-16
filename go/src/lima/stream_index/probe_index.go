package stream_index

import (
	"code.linenisgreat.com/dodder/go/src/alfa/collections_map"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/object_probe_index"
)

type probeIndex struct {
	defaultObjectDigestMarklFormatId string
	index                            *object_probe_index.Index
	additionProbes                   collections_map.Map[string, *sku.Transacted]
}

func (index *Index) PrintAllProbes() (err error) {
	if index.probeIndex.index.PrintAll(index.envRepo); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
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

func (index *Index) VerifyObjectProbes(
	object *sku.Transacted,
) (err error) {
	for probeId := range object.AllProbeIds(
		index.probeIndex.index.GetHashType(),
		index.probeIndex.defaultObjectDigestMarklFormatId,
	) {
		if probeId.Id.IsNull() {
			continue
		}

		loc, err := index.probeIndex.readOneMarklIdLoc(probeId.Id)
		if err != nil {
			return errors.Wrapf(err, "probe %q not found in index", probeId.Key)
		}

		checkObject, checkObjectRepool := sku.GetTransactedPool().GetWithRepool()
		defer checkObjectRepool()

		if !index.readOneLoc(loc, checkObject) {
			return errors.Errorf("probe %q location invalid", probeId.Key)
		}

		// Only verify exact object match for unique probes.
		// The "objectId" probe points to the latest version, so historical
		// objects will not match. The "objectId+tai" probe is unique per
		// object+timestamp and should match exactly.
		if probeId.Key == "objectId" {
			// For objectId probe, just verify it points to the same object ID
			// (may be a different/newer TAI for historical objects)
			if checkObject.GetObjectId().String() != object.GetObjectId().String() {
				return errors.Errorf(
					"probe %q points to wrong object id: expected %s, got %s",
					probeId.Key,
					object.GetObjectId(),
					checkObject.GetObjectId(),
				)
			}
		} else {
			// For all other probes, verify exact match
			if checkObject.GetObjectId().String() != object.GetObjectId().String() ||
				checkObject.GetTai().String() != object.GetTai().String() {
				return errors.Errorf(
					"probe %q points to wrong object: expected %s@%s, got %s@%s",
					probeId.Key,
					object.GetObjectId(), object.GetTai(),
					checkObject.GetObjectId(), checkObject.GetTai(),
				)
			}
		}
	}

	return err
}
