package organize_text

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/format"
	"code.linenisgreat.com/dodder/go/internal/golf/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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

	mr := triple_hyphen_io.Reader{
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

	mw := triple_hyphen_io.Writer{
		Blob: lw,
	}

	mw.Metadata = ot.Metadata

	if n, err = mw.WriteTo(out); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
