package ids

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

type (
	Abbr struct {
		BlobId   domain_interfaces.Abbreviator
		ZettelId domain_interfaces.Abbreviator
	}
)

func DontExpandString(v string) (string, error) {
	return v, nil
}

func DontAbbreviateString[VPtr interfaces.Stringer](k VPtr) (string, error) {
	return k.String(), nil
}

func (a Abbr) ExpanderFor(g genres.Genre) domain_interfaces.FuncExpandString {
	switch g {
	case genres.Zettel:
		return a.ZettelId.Expand

		// TODO add repo abbreviation
	case genres.Tag, genres.Type, genres.Repo:
		return DontExpandString

	default:
		return nil
	}
}

func (a Abbr) AbbreviateZettelIdOnly(
	in *ObjectId,
) (err error) {
	if in.GetGenre() != genres.Zettel || IsVirtual(in) {
		return err
	}

	var getAbbr domain_interfaces.FuncAbbreviateString

	var h ZettelId

	if err = h.Set(in.String()); err != nil {
		err = nil
		return err
	}

	getAbbr = a.ZettelId.Abbreviate

	var abbr string

	if abbr, err = getAbbr(h); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = SetWithGenre(in, abbr, h); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (a Abbr) ExpandZettelIdOnly(
	in *ObjectId,
) (err error) {
	if in.GetGenre() != genres.Zettel || a.ZettelId.Expand == nil {
		return err
	}

	var h ZettelId

	if err = h.Set(in.String()); err != nil {
		err = nil
		return err
	}

	var ex string

	if ex, err = a.ZettelId.Expand(h.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = SetWithGenre(in, ex, h); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (a Abbr) AbbreviateObjectId(
	in *ObjectId,
	out *ObjectId,
) (err error) {
	var getAbbr domain_interfaces.FuncAbbreviateString

	switch in.GetGenre() {
	case genres.Zettel:
		getAbbr = a.ZettelId.Abbreviate

	case genres.Tag, genres.Type, genres.Repo:
		getAbbr = DontAbbreviateString

	case genres.Config:
		out.ResetWithObjectId(in)
		return err

	default:
		err = errors.ErrorWithStackf("unsupported object id: %q, %T", in, in)
		return err
	}

	var abbr string

	if abbr, err = getAbbr(in); err != nil {
		err = nil
		out.ResetWithObjectId(in)
		// err = errors.Wrap(err)
		return err
	}

	if err = SetWithGenre(out, abbr, in); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
