package store_abbr

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type indexZettelId struct {
	readFunc func() error
	Heads    interfaces.TridexMutable
	Tails    interfaces.TridexMutable
}

func (ih *indexZettelId) Add(h *ids.ZettelId) (err error) {
	ih.Heads.Add(h.GetHead())
	ih.Tails.Add(h.GetTail())
	return err
}

func (ih *indexZettelId) Exists(parts [3]string) (err error) {
	if err = ih.readFunc(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !ih.Heads.ContainsExpansion(parts[0]) {
		err = errors.MakeErrNotFoundString(parts[0])
		return err
	}

	if !ih.Tails.ContainsExpansion(parts[2]) {
		err = errors.MakeErrNotFoundString(parts[2])
		return err
	}

	return err
}

func (ih *indexZettelId) ExpandStringString(in string) (out string, err error) {
	var h *ids.ZettelId

	if h, err = ih.ExpandString(in); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	out = h.String()

	return out, err
}

func (ih *indexZettelId) ExpandString(s string) (h *ids.ZettelId, err error) {
	if err = ih.readFunc(); err != nil {
		err = errors.Wrap(err)
		return h, err
	}

	var ha *ids.ZettelId

	if ha, err = ids.MakeZettelId(s); err != nil {
		err = errors.Wrap(err)
		return h, err
	}

	if h, err = ih.Expand(ha); err != nil {
		err = errors.Wrap(err)
		return h, err
	}

	return h, err
}

func (ih *indexZettelId) Expand(
	hAbbr *ids.ZettelId,
) (h *ids.ZettelId, err error) {
	if err = ih.readFunc(); err != nil {
		err = errors.Wrap(err)
		return h, err
	}

	head := ih.Heads.Expand(hAbbr.GetHead())
	tail := ih.Tails.Expand(hAbbr.GetTail())

	if h, err = ids.MakeZettelIdFromHeadAndTail(head, tail); err != nil {
		err = errors.Wrapf(err, "{Abbreviated: '%s'}", hAbbr)
		return h, err
	}

	return h, err
}

func (ih *indexZettelId) Abbreviate(
	id domain_interfaces.Abbreviatable,
) (v string, err error) {
	var h ids.ZettelId

	switch idt := id.(type) {
	case ids.ZettelId:
		h = idt

	case *ids.ObjectId:
		if idt.GetGenre() != genres.Zettel {
			err = genres.MakeErrUnsupportedGenre(idt)
			return v, err
		}

		if err = h.Set(idt.String()); err != nil {
			err = errors.Wrap(err)
			return v, err
		}

	default:
		err = errors.ErrorWithStackf("unsupported type %T: %q", idt, idt)
		return v, err
	}

	if err = ih.readFunc(); err != nil {
		err = errors.Wrap(err)
		return v, err
	}

	head := ih.Heads.Abbreviate(h.GetHead())
	tail := ih.Tails.Abbreviate(h.GetTail())

	if head == "" {
		v = h.String()
		return v, err
	}

	if tail == "" {
		v = h.String()
		return v, err
	}

	v = fmt.Sprintf("%s/%s", head, tail)

	return v, err
}
