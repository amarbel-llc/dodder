package zettel_id_index

import (
	"bufio"
	"encoding/gob"
	"math/rand"
	"sync"
	"time"

	"code.linenisgreat.com/dodder/go/internal/_/coordinates"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/collections"
)

type index struct {
	namedBlobAccess domain_interfaces.NamedBlobAccess

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
	namedBlobAccess domain_interfaces.NamedBlobAccess,
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

	var namedBlobWriter domain_interfaces.BlobWriter

	if namedBlobWriter, err = index.namedBlobAccess.MakeNamedBlobWriter(index.path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, namedBlobWriter.Close)

	w := bufio.NewWriter(namedBlobWriter)

	defer errors.Deferred(&err, w.Flush)

	enc := gob.NewEncoder(w)

	if err = enc.Encode(index.bitset); err != nil {
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

	var namedBlobReader domain_interfaces.BlobReader

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

	defer namedBlobReader.Close()

	r := bufio.NewReader(namedBlobReader)

	dec := gob.NewDecoder(r)

	if err = dec.Decode(index.bitset); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *index) Reset() (err error) {
	lMax := index.oldHinweisenStore.Left().Len() - 1
	rMax := index.oldHinweisenStore.Right().Len() - 1

	if lMax == 0 {
		err = errors.ErrorWithStackf("left zettel id are empty")
		return err
	}

	if rMax == 0 {
		err = errors.ErrorWithStackf("right zettel id are empty")
		return err
	}

	index.bitset = collections.MakeBitsetOn(lMax * rMax)

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

	rand.Seed(time.Now().UnixNano())

	if index.bitset.CountOn() == 0 {
		err = errors.Wrap(zettel_id_provider.ErrZettelIdsExhausted{})
		return h, err
	}

	ri := 0

	if index.bitset.CountOn() > 1 {
		ri = rand.Intn(index.bitset.CountOn() - 1)
	}

	m := 0
	j := 0

	if err = index.bitset.EachOff(
		func(n int) (err error) {
			if index.nonRandomSelection {
				if m == 0 {
					m = n
					return err
				}

				if n > m {
					return err
				}

				m = n
			} else {
				j++
				m = n

				if j == ri {
					err = errors.MakeErrStopIteration()
					return err
				}
			}

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
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

	if err = index.bitset.EachOff(
		func(n int) (err error) {
			n += 1
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
