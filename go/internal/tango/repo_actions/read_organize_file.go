package repo_actions

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/organize_text"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

type ReadOrganizeFile struct {
	*repo
}

func (c ReadOrganizeFile) RunWithPath(
	p string,
	repoId ids.RepoId,
) (ot *organize_text.Text, err error) {
	var f *os.File

	if f, err = files.Open(p); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	defer errors.DeferredCloser(&err, f)

	if ot, err = c.Run(
		f,
		organize_text.NewMetadata(repoId),
	); err != nil {
		err = errors.Wrapf(err, "Path: %q", p)
		return ot, err
	}

	return ot, err
}

func (c ReadOrganizeFile) Run(
	r io.Reader,
	om organize_text.Metadata,
) (ot *organize_text.Text, err error) {
	otFlags := organize_text.MakeFlags()
	ApplyToOrganizeOptions(c.repo, &otFlags.Options)

	o := otFlags.GetOptionsWithMetadata(
		c.GetConfig().GetPrintOptions(),
		c.SkuFormatBoxCheckedOutNoColor(),
		c.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
		om,
	)

	if ot, err = organize_text.New(o); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	if _, err = ot.ReadFrom(r); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	return ot, err
}
