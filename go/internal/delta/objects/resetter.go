package objects

var Resetter resetter

type resetter struct{}

func (resetter) Reset(metadatuh MetadataMutable) {
	{
		metadata := metadatuh.(*metadata)
		metadata.description.Reset()
		metadata.sigRepo.Reset()
		metadata.pubRepo.Reset()
		metadata.contents.Reset()
		metadata.blobRefs.Reset()
		resetIndex(&metadata.idx)
		metadata.typ.Reset()
		metadata.tai.Reset()
		metadata.digBlob.Reset()
		metadata.digSelf.Reset()
		metadata.sigMother.Reset()
	}
}

func (resetter) ResetWithExceptFields(dst MetadataMutable, src Metadata) {
	{
		dst := dst.(*metadata)
		src := src.(*metadata)

		dst.description = src.description

		dst.contents.GetSliceMutable().ResetWith(src.contents.GetSlice())
		dst.blobRefs.ResetWith(src.blobRefs)

		resetIndexWith(&dst.idx, &src.idx)

		dst.sigRepo.ResetWith(src.sigRepo)
		dst.pubRepo.ResetWith(src.pubRepo)

		dst.typ.ResetWith(src.typ)
		dst.tai = src.tai

		dst.digBlob.ResetWith(src.digBlob)
		dst.digSelf.ResetWith(src.digSelf)
		dst.sigMother.ResetWith(src.sigMother)
	}
}

func (resetter resetter) ResetWith(dst MetadataMutable, src Metadata) {
	{
		dst := dst.(*metadata)
		src := src.(*metadata)
		resetter.ResetWithExceptFields(dst, src)
		dst.idx.Fields.ResetWith(src.idx.Fields)
	}
}

func resetIndex(a *index) {
	a.Comments.Reset()
	a.Dormant.Reset()
	a.Fields.Reset()
	a.SelfWithoutTai.Reset()
	a.SetImplicitTags(nil)
	a.TagPaths.Reset()
}

func resetIndexWith(dst, src *index) {
	dst.Comments.ResetWith(src.Comments)
	dst.Dormant.ResetWith(src.Dormant)
	dst.SetImplicitTags(src.GetImplicitTags())
	dst.TagPaths.ResetWith(&src.TagPaths)
}
