package remote_transfer

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type deduper struct {
	formatId                         string
	lookupLock                       *sync.RWMutex
	lookup                           map[string]struct{}
	id                               markl.Id
	defaultObjectDigestMarklFormatId string
}

func (deduper *deduper) initialize(
	options repo.ImporterOptions,
	envRepo env_repo.Env,
) {
	if options.DedupingFormatId != "" {
		deduper.formatId = options.DedupingFormatId
		deduper.lookupLock = &sync.RWMutex{}
		deduper.lookup = make(map[string]struct{})
		deduper.defaultObjectDigestMarklFormatId = envRepo.GetObjectDigestType()
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
		return ErrSkipped
	}

	deduper.lookupLock.RUnlock()

	deduper.lookupLock.Lock()
	deduper.lookup[string(bites)] = struct{}{}
	deduper.lookupLock.Unlock()

	return err
}
