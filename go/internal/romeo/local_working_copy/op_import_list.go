package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func (local *Repo) ImportSeq(
	seq interfaces.SeqError[*sku.Transacted],
	importer repo.Importer,
) (err error) {
	return importer.ImportSeq(
		local,
		local,
		local,
		seq,
	)
}
