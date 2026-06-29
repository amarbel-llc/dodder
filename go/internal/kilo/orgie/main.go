// Package orgie implements the organize-text format: a hierarchical,
// tag-headed text view of tagged/typed objects that round-trips through an
// editor (or, since #7, the MCP organize_plan/organize_commit tools) to apply
// bulk tag/description/move changes.
//
// orgie-extract: this package is a candidate for extraction into a standalone
// amarbel-llc/orgie module — a general tool for applying structured,
// hierarchical mutations to tagged/typed objects. Two follow-ups shape that
// boundary: (#3) a structured (JSON / box-format) alternative to the text
// round-trip below, and object-signature drift detection between plan and
// commit (signatures embedded in the text/JSON let a commit detect that the
// underlying objects changed since the plan, and fail or merge). The SKU
// coupling lives only at the leaf (obj wrapper) and the GetSkus/addToSet
// mutation; the Assignment tree itself is object-agnostic.
package orgie

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/format"
)

type Text struct {
	Options
	*Assignment // TODO make not embedded
}

func New(options Options) (ot *Text, err error) {
	if !options.wasMade {
		panic("options not initialized")
	}

	ot, err = options.Make()

	return ot, err
}

func (t *Text) Refine() (err error) {
	if !t.Options.wasMade {
		panic("options not initialized")
	}

	if err = t.Options.refiner().Refine(t.Assignment); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

type metadataReader struct {
	*Text
	reader
}

func (mr *metadataReader) ReadFrom(r io.Reader) (n int64, err error) {
	if n, err = mr.Metadata.ReadFrom(r); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	ocs := mr.OptionComments

	for _, oc := range ocs {
		if ocwa, ok := oc.(OptionCommentWithApply); ok {
			if err = ocwa.ApplyToReader(mr.Options, &mr.reader); err != nil {
				err = errors.Wrapf(err, "OptionComment: %s", oc)
				return n, err
			}
		}
	}
	return n, err
}

// orgie-extract: ReadFrom/WriteTo are the text-format seam. #3 adds a
// structured (JSON / box-format) alternative alongside this hyphence text
// round-trip; object signatures (for plan↔commit drift detection) ride in the
// same metadata header these read/write.
func (t *Text) ReadFrom(r io.Reader) (n int64, err error) {
	if !t.Options.wasMade {
		panic("options not initialized")
	}

	r1 := metadataReader{
		Text: t,
		reader: reader{
			options: t.Options,
		},
	}

	mr := hyphence.Reader{
		Metadata: &r1,
		Blob:     &r1.reader,
	}

	if n, err = mr.ReadFrom(r); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	t.Assignment = r1.root

	return n, err
}

func (ot Text) WriteTo(out io.Writer) (n int64, err error) {
	if !ot.Options.wasMade {
		panic("options not initialized")
	}

	lw := format.NewLineWriter()

	omit := ot.HasMetadataContent()

	aw := writer{
		ObjectFactory:        ot.ObjectFactory,
		LineWriter:           lw,
		maxDepth:             ot.MaxDepth(),
		Metadata:             ot.AsMetadata(),
		OmitLeadingEmptyLine: omit,
		options:              ot.Options,
	}

	ocs := ot.OptionComments

	for _, oc := range ocs {
		if ocwa, ok := oc.(OptionCommentWithApply); ok {
			if err = ocwa.ApplyToWriter(ot.Options, &aw); err != nil {
				err = errors.Wrapf(err, "OptionComment: %s", oc)
				return n, err
			}
		}
	}

	if err = aw.write(ot.Assignment); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	mw := hyphence.Writer{
		Blob: lw,
	}

	mw.Metadata = ot.Metadata

	if n, err = mw.WriteTo(out); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
