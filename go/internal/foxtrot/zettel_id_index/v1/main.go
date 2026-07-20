package zettel_id_index

import (
	"bufio"
	"encoding"
	"io"
	"math/rand"
	"sync"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/coordinates"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type index struct {
	namedBlobAccess mad_domain_interfaces.NamedBlobAccess

	lock *sync.RWMutex
	path string

	bitset collections.Bitset

	oldHinweisenStore *zettel_id_provider.Provider

	didRead    bool
	hasChanges bool

	nonRandomSelection bool
}

func MakeIndex(
	configCli repo_config_cli.Config,
	directoryLayout directory_layout.RepoMutable,
	namedBlobAccess mad_domain_interfaces.NamedBlobAccess,
) (i *index, err error) {
	i = &index{
		lock:               &sync.RWMutex{},
		path:               directoryLayout.FileCacheObjectId(),
		nonRandomSelection: configCli.UsePredictableZettelIds(),
		namedBlobAccess:    namedBlobAccess,
		bitset:             collections.MakeBitset(0),
	}

	if i.oldHinweisenStore, err = zettel_id_provider.New(directoryLayout); err != nil {
		if errors.IsNotExist(err) {
			ui.TodoP4("determine which layer handles no-create kasten")
			err = nil
		} else {
			err = errors.Wrap(err)
			return i, err
		}
	}

	return i, err
}

func (index *index) Flush() (err error) {
	index.lock.RLock()

	if !index.hasChanges {
		ui.Log().Print("no changes")
		index.lock.RUnlock()
		return err
	}

	index.lock.RUnlock()

	var namedBlobWriter mad_domain_interfaces.BlobWriter

	if namedBlobWriter, err = index.namedBlobAccess.MakeNamedBlobWriter(index.path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, namedBlobWriter.Close)

	w := bufio.NewWriter(namedBlobWriter)

	defer errors.Deferred(&err, w.Flush)

	marshaler := index.bitset.(encoding.BinaryMarshaler)

	var bs []byte

	if bs, err = marshaler.MarshalBinary(); err != nil {
		err = errors.Wrapf(err, "failed to write encoded zettel id")
		return err
	}

	if _, err = w.Write(bs); err != nil {
		err = errors.Wrapf(err, "failed to write encoded zettel id")
		return err
	}

	return err
}

func (index *index) readIfNecessary() (err error) {
	index.lock.RLock()

	if index.didRead {
		index.lock.RUnlock()
		return err
	}

	index.lock.RUnlock()

	index.lock.Lock()
	defer index.lock.Unlock()

	ui.Log().Print("reading")

	index.didRead = true

	var namedBlobReader mad_domain_interfaces.BlobReader

	if namedBlobReader, err = index.namedBlobAccess.MakeNamedBlobReader(
		index.path,
	); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	defer errors.DeferredCloser(&err, namedBlobReader)

	r := bufio.NewReader(namedBlobReader)

	var bs []byte

	if bs, err = io.ReadAll(r); err != nil {
		err = errors.Wrap(err)
		return err
	}

	unmarshaler := index.bitset.(encoding.BinaryUnmarshaler)

	if err = unmarshaler.UnmarshalBinary(bs); err != nil {
		ui.Log().Printf("failed to read zettel id cache (stale format?), rebuilding: %s", err)
		err = nil
	}

	return err
}

func (index *index) Reset() (err error) {
	lLen := index.oldHinweisenStore.Left().Len()
	rLen := index.oldHinweisenStore.Right().Len()

	// A repo bootstrapped from nothing has no zettel-id words until
	// `add-zettel-ids-*` seeds them, so either side may be empty. An empty
	// pool has no available coordinates: leave the bitset empty and return.
	// Falling through would compute Len()-1 == -1 for the empty side, and
	// because coordinates.Int is uint32, Int(-1) wraps to ~4.3e9 —
	// maxCoord.Id() then sizes an enormous bitset that hangs/OOMs genesis
	// (the old `lMax == 0` guards mis-checked Len()==1, never the empty
	// Len()==0 case). CreateZettelId reports "no available zettel ids" until
	// words are added.
	if lLen == 0 || rLen == 0 {
		index.bitset = collections.MakeBitset(0)
		index.hasChanges = true
		return err
	}

	lMax := lLen - 1
	rMax := rLen - 1

	// Compute the max coordinate ID to size the bitset. Coordinate IDs use
	// triangular number mapping, so the max ID is much larger than lMax*rMax.
	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(lMax),
		Right: coordinates.Int(rMax),
	}
	index.bitset = collections.MakeBitset(int(maxCoord.Id()) + 1)

	// Set only valid coordinate IDs as available (ON).
	for l := 0; l <= lMax; l++ {
		for r := 0; r <= rMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			index.bitset.Add(int(k.Id()))
		}
	}

	index.hasChanges = true

	return err
}

func (index *index) AddZettelId(k1 ids.Id) (err error) {
	if !k1.GetGenre().IsZettel() {
		err = genres.MakeErrUnsupportedGenre(k1)
		return err
	}

	var h ids.ZettelId

	if err = h.Set(k1.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = index.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var left, right int

	if left, err = index.oldHinweisenStore.Left().ZettelId(h.GetHead()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if right, err = index.oldHinweisenStore.Right().ZettelId(h.GetTail()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	k := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(left),
		Right: coordinates.Int(right),
	}

	n := k.Id()
	ui.Log().Printf("deleting %d, %s", n, h)

	index.lock.Lock()
	defer index.lock.Unlock()

	index.bitset.DelIfPresent(int(n))

	index.hasChanges = true

	return err
}

func (index *index) CreateZettelId() (h *ids.ZettelId, err error) {
	if err = index.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return h, err
	}

	if index.bitset.CountOn() == 0 {
		err = errors.ErrorWithStackf("no available zettel ids")
		return h, err
	}

	var ri int

	if index.nonRandomSelection {
		ri = 0
	} else if index.bitset.CountOn() > 1 {
		ri = rand.Intn(index.bitset.CountOn())
	}

	m, ok := index.bitset.NthOn(ri)
	if !ok {
		err = errors.Wrap(zettel_id_provider.ErrZettelIdsExhausted{})
		return h, err
	}

	index.bitset.DelIfPresent(int(m))

	index.hasChanges = true

	return index.makeHinweisButDontStore(m)
}

func (index *index) makeHinweisButDontStore(
	j int,
) (h *ids.ZettelId, err error) {
	k := &coordinates.ZettelIdCoordinate{}
	k.SetInt(coordinates.Int(j))

	if h, err = ids.MakeZettelIdFromProvidersAndCoordinates(
		k.Id(),
		index.oldHinweisenStore.Left(),
		index.oldHinweisenStore.Right(),
	); err != nil {
		err = errors.Wrapf(err, "trying to make hinweis for %s, %d", k, j)
		return h, err
	}

	return h, err
}

func (index *index) PeekZettelIds(m int) (hs []*ids.ZettelId, err error) {
	if err = index.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return hs, err
	}

	if m > index.bitset.CountOn() || m == 0 {
		m = index.bitset.CountOn()
	}

	hs = make([]*ids.ZettelId, 0, m)
	j := 0

	if err = index.bitset.EachOn(
		func(n int) (err error) {
			k := &coordinates.ZettelIdCoordinate{}
			k.SetInt(coordinates.Int(n))

			var h *ids.ZettelId

			if h, err = index.makeHinweisButDontStore(n); err != nil {
				err = errors.Wrapf(err, "# %d", n)
				return err
			}

			hs = append(hs, h)

			j++

			if j == m {
				err = errors.MakeErrStopIteration()
				return err
			}

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return hs, err
	}

	return hs, err
}
