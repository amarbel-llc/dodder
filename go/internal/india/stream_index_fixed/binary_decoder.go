package stream_index_fixed

import (
	"bytes"
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/key_bytes"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/tag_paths"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func makeBinaryDecoder(s ids.Sigil) binaryDecoder {
	return binaryDecoder{
		queryGroup: sku.MakePrimitiveQueryGroupWithSigils(s),
		sigil:      s,
	}
}

func makeBinaryDecoderWithQueryGroup(
	query sku.PrimitiveQueryGroup,
	sigil ids.Sigil,
) binaryDecoder {
	if query == nil {
		query = sku.MakePrimitiveQueryGroup()
	}

	if !query.HasHidden() {
		sigil.Add(ids.SigilHidden)
	}

	return binaryDecoder{
		queryGroup: query,
		sigil:      sigil,
	}
}

type binaryDecoder struct {
	field      binaryField
	sigil      ids.Sigil
	queryGroup sku.PrimitiveQueryGroup
}

// readFixedEntry reads a single fixed-length entry at a known offset using
// ReaderAt. Used for probe-based random access reads.
func (decoder *binaryDecoder) readFixedEntry(
	readerAt io.ReaderAt,
	overflowReaderAt *overflowReaderAt,
	entryOffset int64,
	object *sku.Transacted,
) (err error) {
	decoder.field.Reset()

	var entry [EntryWidth]byte

	if _, err = readerAt.ReadAt(entry[:], entryOffset); err != nil {
		if err == io.EOF {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	fieldData, err := decoder.resolveFieldData(entry[:], overflowReaderAt)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	return decoder.parseFields(fieldData, object)
}

// readFixedEntryFromStream reads a single fixed-length entry from a sequential
// reader. Returns the sigil for early filtering.
func (decoder *binaryDecoder) readFixedEntryFromStream(
	reader io.Reader,
	overflowReaderAt *overflowReaderAt,
	object *objectWithCursorAndSigil,
) (n int64, err error) {
	decoder.field.Reset()

	var entry [EntryWidth]byte
	var n1 int

	n1, err = ohio.ReadAllOrDieTrying(reader, entry[:])
	n = int64(n1)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	fieldData, err := decoder.resolveFieldData(entry[:], overflowReaderAt)
	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = decoder.parseFieldsWithSigil(fieldData, object); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

// readFixedEntryMatchSigil reads entries sequentially, skipping those that
// don't match the query's sigil filter.
func (decoder *binaryDecoder) readFixedEntryMatchSigil(
	reader io.Reader,
	overflowReaderAt *overflowReaderAt,
	object *objectWithCursorAndSigil,
) (n int64, err error) {
	for {
		var entry [EntryWidth]byte
		var n1 int

		n1, err = ohio.ReadAllOrDieTrying(reader, entry[:])
		n += int64(n1)

		if err != nil {
			err = errors.WrapExceptSentinel(err, io.EOF)
			return n, err
		}

		// Quick sigil check at byte 3 (key(1) + length(2) = offset 3)
		// before resolving overflow.
		sigilByte := entry[3]
		var entrySigil ids.Sigil
		entrySigil.SetByte(sigilByte)

		fieldData, err1 := decoder.resolveFieldData(entry[:], overflowReaderAt)
		if err1 != nil {
			err = errors.Wrap(err1)
			return n, err
		}

		if err = decoder.parseFieldsWithSigil(fieldData, object); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		genre := genres.Must(object.Transacted)
		query, ok := decoder.queryGroup.Get(genre)

		if ok {
			sigil := query.GetSigil()

			wantsHidden := sigil.IncludesHidden()
			wantsHistory := sigil.IncludesHistory()
			isLatest := object.Contains(ids.SigilLatest)
			isHidden := object.Contains(ids.SigilHidden)

			if (wantsHistory && wantsHidden) ||
				(wantsHidden && isLatest) ||
				(wantsHistory && !isHidden) ||
				(isLatest && !isHidden) {
				return n, err
			}

			if query.ContainsObjectId(object.GetObjectId()) &&
				(sigil.ContainsOneOf(ids.SigilHistory) ||
					object.ContainsOneOf(ids.SigilLatest)) {
				return n, err
			}
		}

		// No match — reset the object and try the next entry.
		sku.TransactedResetter.Reset(object.Transacted)
	}
}

func (decoder *binaryDecoder) resolveFieldData(
	entry []byte,
	overflowReaderAt *overflowReaderAt,
) ([]byte, error) {
	lastByte := entry[EntryWidth-1]
	hasOverflow := lastByte&0x01 != 0

	if !hasOverflow {
		// Find effective length by scanning for zero key byte.
		return entry[:InlineCapacity], nil
	}

	trailerStart := EntryWidth - OverflowTrailerSize
	var offsetBytes [4]byte
	copy(offsetBytes[:], entry[trailerStart:trailerStart+4])
	offset := ohio.ByteArrayToInt32(offsetBytes)

	var lengthBytes [2]byte
	copy(lengthBytes[:], entry[trailerStart+4:trailerStart+6])
	length := ohio.ByteArrayToUInt16(lengthBytes)

	inlineData := entry[:InlineCapacityOverflow]

	if overflowReaderAt == nil {
		return nil, errors.Errorf("entry has overflow but no overflow reader available")
	}

	overflowData, err := overflowReaderAt.ReadAt(offset, length)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	combined := make([]byte, len(inlineData)+len(overflowData))
	copy(combined, inlineData)
	copy(combined[len(inlineData):], overflowData)

	return combined, nil
}

func (decoder *binaryDecoder) parseFields(
	data []byte,
	object *sku.Transacted,
) (err error) {
	buf := bytes.NewBuffer(data)

	for buf.Len() > 0 {
		nextByte, peekErr := buf.ReadByte()
		if peekErr != nil {
			break
		}

		if nextByte == 0 {
			break
		}

		if err = buf.UnreadByte(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if _, err = decoder.field.ReadFrom(buf); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if decoder.field.Key == key_bytes.Sigil {
			var s ids.Sigil

			if _, err = s.ReadFrom(&decoder.field.Content); err != nil {
				err = errors.Wrap(err)
				return err
			}

			object.SetDormant(s.IncludesHidden())
			continue
		}

		if err = decoder.readFieldKey(object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (decoder *binaryDecoder) parseFieldsWithSigil(
	data []byte,
	object *objectWithCursorAndSigil,
) (err error) {
	buf := bytes.NewBuffer(data)

	for buf.Len() > 0 {
		nextByte, peekErr := buf.ReadByte()
		if peekErr != nil {
			break
		}

		if nextByte == 0 {
			break
		}

		if err = buf.UnreadByte(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if _, err = decoder.field.ReadFrom(buf); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if decoder.field.Key == key_bytes.Sigil {
			if _, err = object.Sigil.ReadFrom(&decoder.field.Content); err != nil {
				err = errors.Wrap(err)
				return err
			}

			object.SetDormant(object.IncludesHidden())
			continue
		}

		if err = decoder.readFieldKey(object.Transacted); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (decoder *binaryDecoder) readFieldKey(
	object *sku.Transacted,
) (err error) {
	metadata := object.GetMetadataMutable()

	switch decoder.field.Key {
	case key_bytes.ObjectId:
		if _, err = object.GetObjectIdMutable().ReadFrom(&decoder.field.Content); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.Tai:
		if _, err = metadata.GetTaiMutable().ReadFrom(
			&decoder.field.Content,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.Type:
		marshaler := markl.MakeMutableLockCoderValueRequired(
			metadata.GetTypeLockMutable(),
		)

		if err = marshaler.UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.Blob:
		if err = metadata.GetBlobDigestMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.RepoPubKey:
		if err = metadata.GetRepoPubKeyMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.RepoSig:
		if err = metadata.GetObjectSigMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.Description:
		if err = metadata.GetDescriptionMutable().Set(
			decoder.field.Content.String(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.Tag:
		var tag ids.TagStruct

		if err = tag.Set(decoder.field.Content.String()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = object.AddTagPtrFast(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.SigParentMetadataParentObjectId:
		if err = metadata.GetMotherObjectSigMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.DigestMetadataParentObjectId:
		if err = metadata.GetObjectDigestMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.DigestMetadataWithoutTai:
		if err = metadata.GetIndexMutable().GetSelfWithoutTaiMutable().UnmarshalBinary(
			decoder.field.Content.Bytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case key_bytes.CacheTagImplicit:
		var tag ids.TagStruct

		if err = tag.Set(decoder.field.Content.String()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		metadata.GetIndexMutable().AddTagsImplicitPtr(tag)

	case key_bytes.CacheTags:
		var tag tag_paths.PathWithType

		if _, err = tag.ReadFrom(&decoder.field.Content); err != nil {
			err = errors.WrapExceptSentinel(err, io.EOF)
			return err
		}

		metadata.GetIndexMutable().GetTagPaths().AddPath(&tag)

	default:
		ui.Log().Printf("skipping unknown key: %s", decoder.field.Key)
	}

	return err
}
