package sku

import (
	"code.linenisgreat.com/dodder/go/internal/_/external_state"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
)

var TransactedResetter transactedResetter

type transactedResetter struct{}

func (transactedResetter) Reset(dst *Transacted) {
	dst.ObjectId.Reset()
	objects.Resetter.Reset(&dst.Metadata)
	dst.ExternalType.Reset()
	dst.RepoId.Reset()
	dst.State = external_state.Unknown
}

func (transactedResetter) ResetWith(dst *Transacted, src *Transacted) {
	dst.ObjectId.ResetWithObjectId(&src.ObjectId)
	objects.Resetter.ResetWith(&dst.Metadata, &src.Metadata)
	dst.ExternalType = src.ExternalType
	dst.RepoId = src.RepoId
	dst.State = src.State
	dst.ExternalObjectId.ResetWith(&src.ExternalObjectId)
}

func (transactedResetter) ResetWithExceptFields(dst *Transacted, src *Transacted) {
	dst.ObjectId.ResetWithObjectId(&src.ObjectId)
	objects.Resetter.ResetWithExceptFields(dst.GetMetadataMutable(), src.GetMetadata())
	dst.ExternalType = src.ExternalType
	dst.RepoId = src.RepoId
	dst.State = src.State
	dst.ExternalObjectId.ResetWith(&src.ExternalObjectId)
}

var Resetter resetter

type resetter struct{}

func (resetter) Reset(sl TransactedGetter) {
	TransactedResetter.Reset(sl.GetSku())
}

func (resetter) ResetWith(dst, src TransactedGetter) {
	TransactedResetter.ResetWith(dst.GetSku(), src.GetSku())
}

var CheckedOutResetter checkedOutResetter

type checkedOutResetter struct{}

func (checkedOutResetter) Reset(dst *CheckedOut) {
	TransactedResetter.Reset(dst.GetSku())
	TransactedResetter.Reset(dst.GetSkuExternal().GetSku())
	dst.SetState(checked_out_state.Unknown)
}

func (checkedOutResetter) ResetWith(dst *CheckedOut, src *CheckedOut) {
	TransactedResetter.ResetWith(dst.GetSku(), src.GetSku())
	TransactedResetter.ResetWith(dst.GetSkuExternal().GetSku(), src.GetSkuExternal().GetSku())
	dst.SetState(src.state)
}
