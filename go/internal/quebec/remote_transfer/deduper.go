package remote_transfer

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// dedup keys are computed with the deduper's own purpose
// (PurposeV5MetadataDigestWithoutTai) over unsigned metadata, so no
// object-digest purpose (per-object or repo default) is involved here.
type deduper struct {
	formatId   string
	lookupLock *sync.RWMutex
	lookup     map[string]struct{}
	id         markl.Id
}

func (deduper *deduper) initialize(
	options repo.ImporterOptions,
) {
	if options.DedupingFormatId != "" {
		deduper.formatId = options.DedupingFormatId
		deduper.lookupLock = &sync.RWMutex{}
		deduper.lookup = make(map[string]struct{})
	}
}

func (deduper *deduper) shouldCommit(object *sku.Transacted) (err error) {
	if deduper.lookup == nil {
		return err
	}

	id, idRepool := markl.GetId()
	defer idRepool()

	if err = object.CalculateDigestForPurpose(
		markl.PurposeV5MetadataDigestWithoutTai,
		id,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	bites := id.GetBytes()

	deduper.lookupLock.RLock()
	if _, exists := deduper.lookup[string(bites)]; exists {
		deduper.lookupLock.RUnlock()
		return ErrDeduped{
			ObjectId: object.GetObjectId().String(),
			Digest:   id.String(),
		}
	}

	deduper.lookupLock.RUnlock()

	deduper.lookupLock.Lock()
	deduper.lookup[string(bites)] = struct{}{}
	deduper.lookupLock.Unlock()

	return err
}
