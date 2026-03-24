package object_metadata_fmt_hyphence

import (
	"io"
	"path"
	"strconv"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/doddish"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/delim_reader"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
			if isBlobReference(remainder) {
				err = parser.readBlobReference(metadata, remainder)
			} else if isContainedReference(remainder) {
				refStr := strings.Replace(remainder, " < ", " = ", 1)
				err = parser.readReference(metadata, refStr)
			} else {
				metadata.AddTagString(remainder)
			}

		case doddish.OpType:
			err = parser.readType(metadata, remainder)

		case doddish.OpMarklId:
			err = parser.readBlobDigest(metadata, remainder)

		case doddish.OpReference:
			err = parser.readReference(metadata, remainder)

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

// isContainedReference uses the doddish scanner to distinguish references
// from tags under the unified `-` prefix. References contain a zettel ID
// (identifier/identifier), while tags are simple identifiers.
func isContainedReference(value string) bool {
	seq, scanErr := doddish.ScanExactlyOneSeqWithDotAllowedInIdenfierFromString(value)
	if scanErr != nil {
		// Multiple sequences (e.g., alias = ref) → reference with alias
		return true
	}

	hasPathSep, _, _, _ := seq.PartitionFavoringLeft(
		doddish.TokenMatcherOp(doddish.OpPathSeparator),
	)

	return hasPathSep
}

// isBlobReference detects blob reference patterns:
// - @digest (without alias)
// - alias < @digest (with alias)
func isBlobReference(value string) bool {
	if strings.HasPrefix(value, "@") {
		return true
	}

	if found := strings.Contains(value, " < @"); found {
		return true
	}

	return false
}

func (parser *textParser2) readBlobReference(
	metadata objects.MetadataMutable,
	refString string,
) (err error) {
	if refString == "" {
		return err
	}

	var alias string
	var blobRefPortion string

	if before, after, ok := strings.Cut(refString, " < "); ok {
		alias = strings.TrimSpace(before)

		if len(alias) >= 2 && alias[0] == '"' && alias[len(alias)-1] == '"' {
			if unquoted, err := strconv.Unquote(alias); err == nil {
				alias = unquoted
			}
		}

		blobRefPortion = strings.TrimSpace(after)
	} else {
		blobRefPortion = refString
	}

	reader, repool := pool.GetStringReader(blobRefPortion)
	defer repool()

	scanner := doddish.MakeScanner(reader)

	// First seq: @digest
	if !scanner.ScanDotAllowedInIdentifiers() {
		err = errors.Errorf("expected @digest in blob reference: %q", refString)
		return err
	}

	seq := scanner.GetSeq()

	if !seq.MatchAll(doddish.TokenMatcherBlobDigest...) {
		err = errors.Errorf("expected @digest, got %q in blob reference: %q", seq, refString)
		return err
	}

	var blobId markl.Id

	if err = blobId.Set(seq.At(1).String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Scan optional type lock: space then !type or !type@sig
	var typeLock markl.Lock[ids.SeqId, *ids.SeqId]

	if scanner.ScanDotAllowedInIdentifiers() {
		spaceSeq := scanner.GetSeq()

		// Skip space seq
		if spaceSeq.MatchAll(doddish.TokenMatcherOp(' ')) {
			if scanner.ScanDotAllowedInIdentifiers() {
				typeSeq := scanner.GetSeq()

				switch {
				case typeSeq.MatchAll(doddish.TokenMatcherTypeLock...):
					marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

					if err = marshaler.Set(typeSeq.String()); err != nil {
						err = errors.Wrapf(err, "blob reference type lock: %q", refString)
						return err
					}

				case typeSeq.MatchAll(doddish.TokenMatcherType...):
					if err = typeLock.GetKeyMutable().Set(typeSeq.String()); err != nil {
						err = errors.Wrapf(err, "blob reference type: %q", refString)
						return err
					}

				default:
					err = errors.Errorf("expected !type or !type@sig, got %q in blob reference: %q", typeSeq, refString)
					return err
				}
			}
		}
	}

	metadata.AddBlobReference(blobId, typeLock)

	if alias != "" {
		if err = metadata.SetBlobReferenceAlias(blobId, alias); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (parser *textParser2) readType(
	metadata objects.MetadataMutable,
	typeString string,
) (err error) {
	if typeString == "" {
		return err
	}

	// Support old format where blob paths were written with `!` instead of `@`
	if strings.Contains(typeString, "/") {
		return parser.readBlobDigest(metadata, typeString)
	}

	marshaler := markl.MakeMutableLockCoderValueNotRequired(metadata.GetTypeLockMutable())

	if err = marshaler.Set(ids.MakeTypeString(typeString)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (parser *textParser2) readReference(
	metadata objects.MetadataMutable,
	refString string,
) (err error) {
	if refString == "" {
		return err
	}

	var alias string
	objectRefString := refString

	if before, after, ok := strings.Cut(refString, " = "); ok {
		alias = strings.TrimSpace(before)
		objectRefString = strings.TrimSpace(after)

		if len(alias) >= 2 && alias[0] == '"' && alias[len(alias)-1] == '"' {
			if unquoted, err := strconv.Unquote(alias); err == nil {
				alias = unquoted
			}
		}
	}

	var refId ids.SeqId

	objectIdStr := objectRefString
	if before, _, ok := strings.Cut(objectRefString, "@"); ok {
		objectIdStr = before
	}

	if err = refId.Set(objectIdStr); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = metadata.AddReference(refId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if strings.Contains(objectRefString, "@") {
		refLock := metadata.GetReferencedObjectLockMutable(refId)
		marshaler := markl.MakeMutableLockCoderValueNotRequired(refLock)

		if err = marshaler.Set(objectRefString); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if alias != "" {
		if err = metadata.SetReferenceAlias(refId, alias); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (parser *textParser2) readObjectId(
	objectIdString string,
) (err error) {
	err = errors.Err405MethodNotAllowed
	return
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
