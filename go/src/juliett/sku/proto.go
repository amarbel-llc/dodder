package sku

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/comments"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/golf/repo_configs"
)

func MakeProto(defaults repo_configs.Defaults) (proto Proto) {
	var tipe ids.TypeStruct
	var tags ids.TagSet

	if defaults != nil {
		tipe = defaults.GetDefaultType()
		tags = ids.MakeTagSetFromSlice(defaults.GetDefaultTags()...)
	}

	proto.Metadata.GetTypeMutable().ResetWithType(tipe)
	objects.SetTags(&proto.Metadata, tags)

	return proto
}

type Proto struct {
	Metadata objects.MetadataStruct
}

var _ interfaces.CommandComponentWriter = (*Proto)(nil)

func (proto *Proto) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	proto.Metadata.SetFlagDefinitions(f)
}

func (proto Proto) Equals(metadata objects.MetadataMutable) (ok bool) {
	var okType, okMetadata bool

	if !ids.IsEmpty(proto.Metadata.GetType()) &&
		ids.Equals(proto.Metadata.GetType(), metadata.GetType()) {
		okType = true
	}

	if objects.Equaler.Equals(&proto.Metadata, metadata) {
		okMetadata = true
	}

	ok = okType && okMetadata

	return ok
}

func (proto Proto) Make() (object *Transacted, repool interfaces.FuncRepool) {
	comments.Change("add type")
	comments.Change("add description")
	object, repool = GetTransactedPool().GetWithRepool()

	proto.Apply(object, genres.Zettel)

	return object, repool
}

func (proto Proto) ApplyType(
	metadataLike objects.GetterMutable,
	genreGetter domain_interfaces.GenreGetter,
) (ok bool) {
	metadata := metadataLike.GetMetadataMutable()

	g := genreGetter.GetGenre()
	ui.Log().Print(metadataLike, g)

	switch g {
	case genres.Zettel, genres.Unknown:
		if ids.IsEmpty(metadata.GetType()) &&
			!ids.IsEmpty(proto.Metadata.GetType()) &&
			!ids.Equals(metadata.GetType(), proto.Metadata.GetType()) {
			ok = true
			metadata.GetTypeMutable().ResetWith(proto.Metadata.GetType())
		}
	}

	return ok
}

func (proto Proto) Apply(
	metadataLike objects.GetterMutable,
	genreGetter domain_interfaces.GenreGetter,
) (changed bool) {
	metadata := metadataLike.GetMetadataMutable()

	if proto.ApplyType(metadataLike, genreGetter) {
		changed = true
	}

	if proto.Metadata.GetDescription().WasSet() &&
		!metadata.GetDescription().Equals(proto.Metadata.GetDescription()) {
		changed = true
		metadata.GetDescriptionMutable().ResetWith(proto.Metadata.GetDescription())
	}

	if proto.Metadata.GetTags().Len() > 0 {
		changed = true
	}

	for tag := range proto.Metadata.AllTags() {
		errors.PanicIfError(metadata.AddTagPtr(tag))
	}

	return changed
}

func (proto Proto) ApplyWithBlobFD(
	metadataGetter objects.GetterMutable,
	blobFD *fd.FD,
) (err error) {
	metadataMutable := metadataGetter.GetMetadataMutable()

	if ids.IsEmpty(metadataMutable.GetType()) &&
		!ids.IsEmpty(proto.Metadata.GetType()) &&
		!ids.Equals(metadataMutable.GetType(), proto.Metadata.GetType()) {
		metadataMutable.GetTypeMutable().ResetWith(proto.Metadata.GetType())
	} else {
		// TODO-P4 use konfig
		ext := blobFD.Ext()

		if ext != "" {
			if err = metadataMutable.GetTypeMutable().SetType(blobFD.Ext()); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	desc := blobFD.FileNameSansExt()

	if proto.Metadata.GetDescription().WasSet() &&
		!metadataMutable.GetDescription().Equals(proto.Metadata.GetDescription()) {
		desc = proto.Metadata.GetDescription().String()
	}

	if err = metadataMutable.GetDescriptionMutable().Set(desc); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for e := range proto.Metadata.AllTags() {
		errors.PanicIfError(metadataMutable.AddTagPtr(e))
	}

	return err
}
