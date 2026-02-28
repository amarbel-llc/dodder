package objects

var Resetter resetter

type resetter struct{}

func (resetter) Reset(metadatuh MetadataMutable) {
	{
		metadata := metadatuh.(*metadata)
		metadata.Description.Reset()
		metadata.sigRepo.Reset()
		metadata.pubRepo.Reset()
		metadata.ResetTags()
		resetIndex(&metadata.Index)
		metadata.Type.Reset()
		metadata.Tai.Reset()
		metadata.DigBlob.Reset()
		metadata.digSelf.Reset()
		metadata.sigMother.Reset()
	}
}

func (resetter) ResetWithExceptFields(dst MetadataMutable, src Metadata) {
	{
		dst := dst.(*metadata)
		src := src.(*metadata)

		dst.Description = src.Description

		dst.SetTagsFast(src.GetTags())

		resetIndexWith(&dst.Index, &src.Index)

		dst.sigRepo.ResetWith(src.sigRepo)
		dst.pubRepo.ResetWith(src.pubRepo)

		dst.Type.ResetWith(src.Type)
		dst.Tai = src.Tai

		dst.DigBlob.ResetWith(src.DigBlob)
		dst.digSelf.ResetWith(src.digSelf)
		dst.sigMother.ResetWith(src.sigMother)
	}
}

func (resetter resetter) ResetWith(dst MetadataMutable, src Metadata) {
	{
		dst := dst.(*metadata)
		src := src.(*metadata)
		resetter.ResetWithExceptFields(dst, src)
		dst.Index.Fields.ResetWith(src.Index.Fields)
	}
}

func resetIndex(a *index) {
	a.Comments.Reset()
	a.Dormant.Reset()
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
