package local_working_copy

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

func (local *Repo) Reindex() {
	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))
	local.Must(errors.MakeFuncContextFromFuncErr(local.config.Reset))
	local.Must(local.GetStore().Reindex)
	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))
}
