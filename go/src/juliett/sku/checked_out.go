package sku

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/external_state"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
)

func InternalAndExternalEqualsWithoutTai(co SkuType) bool {
	internal := co.GetSku()
	external := co.GetSkuExternal().GetSku()

	return objects.EqualerSansTai.Equals(
		&external.Metadata,
		&internal.Metadata,
	)
}

type CheckedOut struct {
	internal Transacted
	external Transacted
	state    checked_out_state.State
}

var (
	_ TransactedGetter   = &CheckedOut{}
	_ ExternalLike       = &CheckedOut{}
	_ ExternalLikeGetter = &CheckedOut{}
)

func (checkedOut *CheckedOut) GetRepoId() ids.RepoId {
	return checkedOut.GetSkuExternal().RepoId
}

func (checkedOut *CheckedOut) GetSkuExternal() *Transacted {
	return &checkedOut.external
}

func (checkedOut *CheckedOut) GetSku() *Transacted {
	return &checkedOut.internal
}

func (checkedOut *CheckedOut) GetState() checked_out_state.State {
	return checkedOut.state
}

func (checkedOut *CheckedOut) Clone() (*CheckedOut, interfaces.FuncRepool) {
	dst, repool := GetCheckedOutPool().GetWithRepool()
	CheckedOutResetter.ResetWith(dst, checkedOut)
	return dst, repool
}

func (checkedOut *CheckedOut) GetExternalObjectId() domain_interfaces.ExternalObjectId {
	return checkedOut.GetSkuExternal().GetExternalObjectId()
}

func (checkedOut *CheckedOut) GetExternalState() external_state.State {
	return checkedOut.GetSkuExternal().GetExternalState()
}

func (checkedOut *CheckedOut) GetObjectId() *ids.ObjectId {
	return checkedOut.GetSkuExternal().GetObjectId()
}

func (checkedOut *CheckedOut) SetState(
	state checked_out_state.State,
) (err error) {
	checkedOut.state = state
	return err
}

func (checkedOut *CheckedOut) String() string {
	return fmt.Sprintf("%s %s", checkedOut.GetSku(), checkedOut.GetSkuExternal())
}

func (checkedOut *CheckedOut) Equals(b *CheckedOut) bool {
	return checkedOut.internal.Equals(&b.internal) && checkedOut.external.Equals(&b.external)
}

func (checkedOut *CheckedOut) GetTai() ids.Tai {
	external := checkedOut.external.GetTai()

	if external.IsZero() {
		return checkedOut.internal.GetTai()
	} else {
		return external
	}
}
