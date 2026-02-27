package store_abbr

import (
	"encoding/gob"
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/options_print"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/kilo/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/tridex"
)

type indexCodable struct {
	SeenIds  map[genres.Genre]interfaces.TridexMutable
	MarklIds indexNotZettelId[markl.Id, *markl.Id]
	ZettelId indexZettelId
}

type indexAbbr struct {
	options_print.Options

	lock    sync.Locker
	once    *sync.Once
	envRepo env_repo.Env

	path string

	indexCodable

	didRead    bool
	hasChanges bool
}

var _ sku.IdIndex = &indexAbbr{}

func NewIndex(
	options options_print.Options,
	envRepo env_repo.Env,
) (index *indexAbbr, err error) {
	index = &indexAbbr{
		Options: options,
		lock:    &sync.Mutex{},
		once:    &sync.Once{},
		path:    envRepo.DirDataIndex("Abbr"),
		envRepo: envRepo,
		indexCodable: indexCodable{
			SeenIds: map[genres.Genre]interfaces.TridexMutable{
				genres.Repo:   tridex.Make(),
				genres.Tag:    tridex.Make(),
				genres.Type:   tridex.Make(),
				genres.Zettel: tridex.Make(),
			},
			MarklIds: indexNotZettelId[markl.Id, *markl.Id]{
				ObjectIds: tridex.Make(),
			},
			ZettelId: indexZettelId{
				Heads: tridex.Make(),
				Tails: tridex.Make(),
			},
		},
	}

	index.ZettelId.readFunc = index.readIfNecessary
	index.MarklIds.readFunc = index.readIfNecessary

	return index, err
}

func (index *indexAbbr) Flush() (err error) {
	index.lock.Lock()
	defer index.lock.Unlock()

	if !index.hasChanges {
		ui.Log().Print("no changes")
		return err
	}

	var namedBlobWriter domain_interfaces.BlobWriter

	if namedBlobWriter, err = index.envRepo.MakeNamedBlobWriter(
		index.path,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, namedBlobWriter)

	bufferedWriter, repool := pool.GetBufferedWriter(namedBlobWriter)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	enc := gob.NewEncoder(bufferedWriter)

	if err = enc.Encode(index.indexCodable); err != nil {
		err = errors.Wrapf(err, "failed to write encoded object id")
		return err
	}

	return err
}

func (index *indexAbbr) readIfNecessary() (err error) {
	index.once.Do(
		func() {
			if index.didRead {
				return
			}

			ui.Log().Print("reading")

			index.didRead = true

			var namedBlobReader io.ReadCloser

			if namedBlobReader, err = index.envRepo.MakeNamedBlobReader(index.path); err != nil {
				if errors.IsNotExist(err) {
					err = nil
				} else {
					err = errors.Wrap(err)
				}

				return
			}

			defer errors.DeferredCloser(&err, namedBlobReader)

			bufferedReader, repool := pool.GetBufferedReader(namedBlobReader)
			defer repool()

			dec := gob.NewDecoder(bufferedReader)

			ui.Log().Print("starting decode")

			if err = dec.Decode(&index.indexCodable); err != nil {
				ui.Log().Print("finished decode unsuccessfully")
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return
			}
		},
	)

	return err
}

func (index *indexAbbr) GetAbbr() (out ids.Abbr) {
	out.ZettelId.Expand = index.GetZettelIds().ExpandStringString
	out.BlobId.Expand = index.GetBlobIds().ExpandStringString

	if index.AbbreviateZettelIds {
		out.ZettelId.Abbreviate = index.GetZettelIds().Abbreviate
	}

	if index.AbbreviateMarklIds {
		out.BlobId.Abbreviate = index.GetBlobIds().Abbreviate
	}

	return out
}

func (index *indexAbbr) AddObjectToIdIndex(
	object *sku.Transacted,
) (err error) {
	genre := genres.Must(object.GetGenre())

	switch genre {
	case genres.Config:
		return err

	case genres.InventoryList:
		return err

	case genres.Zettel, genres.Type, genres.Tag, genres.Repo:

	default:
		err = errors.ErrorWithStackf(
			"unsupported object id: %qv",
			object.GetObjectId(),
		)
		return err
	}

	if err = index.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	index.hasChanges = true

	objectIdString := object.GetObjectId().String()
	index.SeenIds[genre].Add(objectIdString)

	if genre == genres.Zettel {
		var zettelId ids.ZettelId

		if err = zettelId.SetWithSeq(object.GetObjectId().ToSeq()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		index.ZettelId.Heads.Add(zettelId.GetHead())
		index.ZettelId.Tails.Add(zettelId.GetTail())
	}

	for tag := range object.GetMetadata().AllTags() {
		index.SeenIds[genres.Tag].Add(tag.String())
	}

	// TODO add other markl ids
	index.MarklIds.ObjectIds.Add(
		object.GetBlobDigest().String(),
	)

	index.SeenIds[genres.Type].Add(object.GetType().String())
	index.SeenIds[genres.Repo].Add(object.GetRepoId().String())

	return err
}

func (index *indexAbbr) GetZettelIds() sku.IdAbbrIndexGeneric[ids.ZettelId, *ids.ZettelId] {
	return &index.ZettelId
}

func (index *indexAbbr) GetBlobIds() sku.IdAbbrIndexGeneric[markl.Id, *markl.Id] {
	return &index.MarklIds
}

func (index *indexAbbr) GetSeenIds() map[genres.Genre]interfaces.Collection[string] {
	output := make(
		map[genres.Genre]interfaces.Collection[string],
		len(index.SeenIds),
	)

	for genre, tridex := range index.SeenIds {
		output[genre] = tridex
	}

	return output
}
