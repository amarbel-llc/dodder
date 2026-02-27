package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
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
