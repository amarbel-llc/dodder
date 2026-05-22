package repo_actions

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
)

type ReadOrganizeFile struct {
	*repo
}

func (c ReadOrganizeFile) RunWithPath(
	p string,
	repoId ids.RepoId,
) (ot *orgie.Text, err error) {
	var f *os.File

	if f, err = files.Open(p); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	defer errors.DeferredCloser(&err, f)

	if ot, err = c.Run(
		f,
		orgie.NewMetadata(repoId),
	); err != nil {
		err = errors.Wrapf(err, "Path: %q", p)
		return ot, err
	}

	return ot, err
}

func (c ReadOrganizeFile) Run(
	r io.Reader,
	om orgie.Metadata,
) (ot *orgie.Text, err error) {
	otFlags := orgie.MakeFlags()
	ApplyToOrganizeOptions(c.repo, &otFlags.Options)

	o := otFlags.GetOptionsWithMetadata(
		c.GetConfig().GetPrintOptions(),
		c.SkuFormatBoxCheckedOutNoColor(),
		c.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
		om,
	)

	if ot, err = orgie.New(o); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	if _, err = ot.ReadFrom(r); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	return ot, err
}
