package store_abbr

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/tridex"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type InMemoryIndex struct {
	zettelId indexZettelId
	blobIds  indexNotZettelId[markl.Id, *markl.Id]
	pubKeys  interfaces.TridexMutable
	sigs     interfaces.TridexMutable
}

func NewInMemoryIndex() *InMemoryIndex {
	noop := func() error { return nil }

	return &InMemoryIndex{
		zettelId: indexZettelId{
			readFunc: noop,
			Heads:    tridex.Make(),
			Tails:    tridex.Make(),
		},
		blobIds: indexNotZettelId[markl.Id, *markl.Id]{
			readFunc:  noop,
			ObjectIds: tridex.Make(),
		},
		pubKeys: tridex.Make(),
		sigs:    tridex.Make(),
	}
}

func (idx *InMemoryIndex) AddObject(object *sku.Transacted) (err error) {
	genre := genres.Make(object.GetGenre())

	switch genre {
	case genres.Config, genres.InventoryList:
		return err
	}

	if genre == genres.Zettel {
		var zettelId ids.ZettelId

		if err = zettelId.SetWithSeq(object.GetObjectId().ToSeq()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		idx.zettelId.Heads.Add(zettelId.GetHead())
		idx.zettelId.Tails.Add(zettelId.GetTail())
	}

	blobDigest := object.GetBlobDigest()
	if !blobDigest.IsNull() {
		idx.blobIds.ObjectIds.Add(blobDigest.String())
	}

	metadata := object.GetMetadata()

	pubKey := metadata.GetRepoPubKey()
	if !pubKey.IsNull() {
		idx.pubKeys.Add(pubKey.String())
	}

	objectSig := metadata.GetObjectSig()
	if !objectSig.IsNull() {
		idx.sigs.Add(objectSig.String())
	}

	motherSig := metadata.GetMotherObjectSig()
	if !motherSig.IsNull() {
		idx.sigs.Add(motherSig.String())
	}

	return err
}

func (idx *InMemoryIndex) GetAbbr() ids.Abbr {
	return ids.Abbr{
		ZettelId: domain_interfaces.Abbreviator{
			Abbreviate: idx.zettelId.Abbreviate,
		},
		BlobId: domain_interfaces.Abbreviator{
			Abbreviate: idx.blobIds.Abbreviate,
		},
		PubKey: domain_interfaces.Abbreviator{
			Abbreviate: func(k domain_interfaces.Abbreviatable) (string, error) {
				return idx.pubKeys.Abbreviate(k.String()), nil
			},
		},
		Sig: domain_interfaces.Abbreviator{
			Abbreviate: func(k domain_interfaces.Abbreviatable) (string, error) {
				return idx.sigs.Abbreviate(k.String()), nil
			},
		},
	}
}
