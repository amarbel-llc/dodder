package local_working_copy

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"

func (local *Repo) Reindex() {
	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))
	local.Must(errors.MakeFuncContextFromFuncErr(local.config.Reset))
	local.Must(local.GetStore().Reindex)
	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))
}
