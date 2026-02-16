package object_metadata_fmt_triple_hyphen

import (
	"io"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/delim_reader"
	"code.linenisgreat.com/dodder/go/src/charlie/doddish"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
)

type textParser2 struct {
	domain_interfaces.BlobWriterFactory
	ParserContext
	hashType domain_interfaces.FormatHash
	Blob     fd.FD
}

func (parser *textParser2) ReadFrom(r io.Reader) (n int64, err error) {
	metadata := parser.GetMetadataMutable()
	objects.Resetter.Reset(metadata)

	delimReader, delimRepool := delim_reader.MakeDelimReader('\n', r)
	defer delimRepool()

	for {
		var line string

		line, err = delimReader.ReadOneString()

		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			continue
		}

		key, remainder := trimmed[0], strings.TrimSpace(trimmed[1:])

		switch doddish.Op(key) {
		case doddish.OpDescription:
			err = metadata.GetDescriptionMutable().Set(remainder)

		case doddish.OpVirtual:
			metadata.GetIndexMutable().GetCommentsMutable().Append(remainder)

		case doddish.OpTagSeparator:
			metadata.AddTagString(remainder)

		case doddish.OpType:
			err = parser.readType(metadata, remainder)

		case doddish.OpMarklId:
			err = parser.readBlobDigest(metadata, remainder)

		case doddish.OpExact:
			// TODO read object id
			err = parser.readObjectId(remainder)

		default:
			err = errors.ErrorWithStackf("unsupported entry: %q", line)
		}

		if err != nil {
			err = errors.Wrapf(
				err,
				"Line: %q, Key: %q, Value: %q",
				line,
				key,
				remainder,
			)
			return n, err
		}
	}

	return n, err
}

func (parser *textParser2) readType(
	metadata objects.MetadataMutable,
	typeString string,
) (err error) {
	if typeString == "" {
		return err
	}

	marshaler := markl.MakeMutableLockCoderValueNotRequired(metadata.GetTypeLockMutable())

	if err = marshaler.Set(ids.MakeTypeString(typeString)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (parser *textParser2) readObjectId(
	objectIdString string,
) (err error) {
	err = errors.Err405MethodNotAllowed
	return

	if objectIdString == "" {
		return err
	}

	return err
}

func (parser *textParser2) readBlobDigest(
	metadata objects.MetadataMutable,
	metadataLine string,
) (err error) {
	if metadataLine == "" {
		return err
	}

	extension := path.Ext(metadataLine)
	digest := metadataLine[:len(metadataLine)-len(extension)]

	switch {
	//@ <path>
	case files.Exists(metadataLine):
		// TODO cascade type definition
		if err = metadata.GetTypeMutable().SetType(extension); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = parser.Blob.SetWithBlobWriterFactory(
			metadataLine,
			parser.BlobWriterFactory,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	//@ <dig>.<ext>
	// case extension != "":
	// 	if err = parser.setBlobDigest(metadata, digest); err != nil {
	// 		err = errors.Wrap(err)
	// 		return err
	// 	}

	// 	if err = metadata.GetTypeMutable().Set(extension); err != nil {
	// 		err = errors.Wrap(err)
	// 		return err
	// 	}

	case extension == "":
		if err = parser.setBlobDigest(metadata, digest); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		err = errors.Errorf("unsupported blob digest or path: %q", metadataLine)
		return err
	}

	return err
}

func (parser *textParser2) setBlobDigest(
	metadata objects.MetadataMutable,
	maybeSha string,
) (err error) {
	if err = markl.SetMarklIdWithFormatBlech32(
		metadata.GetBlobDigestMutable(),
		markl.PurposeBlobDigestV1,
		maybeSha,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
