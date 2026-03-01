package stream_index_fixed

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"math"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/_/key_bytes"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

type binaryEncoder struct {
	fieldBuffer bytes.Buffer
	field       binaryField
	ids.Sigil
}

func (encoder *binaryEncoder) writeFixedEntry(
	writer io.Writer,
	object objectWithSigil,
	overflow *overflowWriter,
) (err error) {
	encoder.fieldBuffer.Reset()

	for _, key := range fixedFieldOrder {
		encoder.field.Reset()
		encoder.field.Key = key

		if _, err = encoder.writeFieldKey(object); err != nil {
			err = errors.Wrapf(
				err,
				"Key: %q, Sku: %s",
				encoder.field.Key,
				sku.String(object.Transacted),
			)
			return err
		}
	}

	fieldBytes := encoder.fieldBuffer.Bytes()
	fieldLen := len(fieldBytes)

	var entry [EntryWidth]byte

	if fieldLen <= InlineCapacity {
		copy(entry[:], fieldBytes)
		// Zero-pad and set last byte low bit = 0 (no overflow)
		entry[EntryWidth-1] = 0x00
	} else {
		if overflow == nil {
			err = errors.Errorf(
				"entry requires overflow but no overflow writer provided (size: %d)",
				fieldLen,
			)
			return err
		}

		inlineLen := InlineCapacityOverflow
		copy(entry[:inlineLen], fieldBytes[:inlineLen])

		overflowData := fieldBytes[inlineLen:]

		var offset int32
		var length uint16

		if offset, length, err = overflow.Write(overflowData); err != nil {
			err = errors.Wrap(err)
			return err
		}

		trailerStart := EntryWidth - OverflowTrailerSize
		offsetBytes := ohio.Int32ToByteArray(offset)
		copy(entry[trailerStart:trailerStart+4], offsetBytes[:])
		lengthBytes := ohio.UInt16ToByteArray(length)
		copy(entry[trailerStart+4:trailerStart+6], lengthBytes[:])
		entry[EntryWidth-1] = 0x01 // overflow bit set
	}

	if _, err = ohio.WriteAllOrDieTrying(writer, entry[:]); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (encoder *binaryEncoder) updateSigil(
	writerAt io.WriterAt,
	sigil ids.Sigil,
	entryOffset int64,
) (err error) {
	sigil.Add(encoder.Sigil)

	// Fixed-length: no content-length prefix. Sigil is the first field.
	// Offset: key(1) + field-length(2) = 3 bytes to the content byte.
	offset := entryOffset + int64(3)

	var n int

	if n, err = writerAt.WriteAt([]byte{sigil.Byte()}, offset); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if n != 1 {
		err = catgut.MakeErrLength(1, int64(n), nil)
		return err
	}

	return err
}

func (encoder *binaryEncoder) writeFieldKey(
	object objectWithSigil,
) (n int64, err error) {
	metadata := object.GetMetadata()

	switch encoder.field.Key {
	case key_bytes.Sigil:
		sigil := object.Sigil
		sigil.Add(encoder.Sigil)

		if metadata.GetIndex().GetDormant().Bool() {
			sigil.Add(ids.SigilHidden)
		}

		if n, err = encoder.writeFieldByteReader(sigil); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.ObjectId:
		if n, err = encoder.writeFieldWriterTo(object.GetObjectId()); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.Tai:
		if n, err = encoder.writeFieldWriterTo(
			object.GetMetadata().GetTai(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.Type:
		if metadata.GetType().IsEmpty() {
			return n, err
		}

		binaryMarshaler := markl.MakeMutableLockCoder(
			object.GetMetadataMutable().GetTypeLockMutable(),
			!ids.IsBuiltin(metadata.GetType()),
		)

		if n, err = encoder.writeFieldBinaryMarshaler(
			binaryMarshaler,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.Blob:
		if n, err = encoder.writeMarklId(
			metadata.GetBlobDigest(),
			true,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.RepoPubKey:
		if n, err = encoder.writeFieldBinaryMarshaler(
			metadata.GetRepoPubKey(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.RepoSig:
		merkleId := metadata.GetObjectSig()

		if n, err = encoder.writeMarklId(
			merkleId,
			true,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.Description:
		if metadata.GetDescription().IsEmpty() {
			return n, err
		}

		if n, err = encoder.writeFieldBinaryMarshaler(
			object.GetMetadata().GetDescription(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.Tag:
		for _, tag := range quiter.SortedValues(object.AllTags()) {
			if ids.TagIsVirtual(tag) {
				continue
			}

			if tag.String() == "" {
				err = errors.ErrorWithStackf("empty tag in %q", object.GetTags())
				return n, err
			}

			var n1 int64
			n1, err = encoder.writeFieldStringer(tag)
			n += n1

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	case key_bytes.SigParentMetadataParentObjectId:
		if n, err = encoder.writeFieldMerkleId(
			metadata.GetMotherObjectSig(),
			true,
			encoder.field.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.DigestMetadataParentObjectId:
		if n, err = encoder.writeFieldMerkleId(
			metadata.GetObjectDigest(),
			true,
			encoder.field.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.DigestMetadataWithoutTai:
		if n, err = encoder.writeFieldMerkleId(
			object.GetMetadata().GetIndex().GetSelfWithoutTai(),
			true,
			encoder.field.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.CacheTagImplicit:
		tags := metadata.GetIndex().GetImplicitTags()

		for _, tag := range quiter.SortedValues(tags.All()) {
			var n1 int64
			n1, err = encoder.writeFieldBinaryMarshaler(&tag)
			n += n1

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	case key_bytes.CacheTags:
		tags := metadata.GetIndex().GetTagPaths()

		for _, tag := range tags.Paths {
			var n1 int64
			n1, err = encoder.writeFieldWriterTo(tag)
			n += n1

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	default:
		panic(fmt.Sprintf("unsupported key: %s", encoder.field.Key))
	}

	return n, err
}

func (encoder *binaryEncoder) writeMarklId(
	marklId domain_interfaces.MarklId,
	allowNull bool,
) (n int64, err error) {
	if !allowNull {
		if err = markl.AssertIdIsNotNull(marklId); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	if marklId.IsNull() {
		return n, err
	}

	if n, err = encoder.writeFieldBinaryMarshaler(
		marklId,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldWriterTo(
	wt io.WriterTo,
) (n int64, err error) {
	if _, err = wt.WriteTo(&encoder.field.Content); err != nil {
		return n, err
	}

	if n, err = encoder.field.WriteTo(&encoder.fieldBuffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldMerkleId(
	merkleId domain_interfaces.MarklId,
	allowNull bool,
	key string,
) (n int64, err error) {
	if merkleId.IsNull() {
		if !allowNull {
			err = markl.AssertIdIsNotNull(merkleId)
		}

		return n, err
	}

	if n, err = encoder.writeFieldBinaryMarshaler(merkleId); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldStringer(
	stringer interfaces.Stringer,
) (n int64, err error) {
	value := stringer.String()

	if _, err = io.WriteString(&encoder.field.Content, value); err != nil {
		err = errors.WrapExceptSentinelAsNil(err, io.EOF)
		return n, err
	}

	if n, err = encoder.field.WriteTo(&encoder.fieldBuffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldBinaryMarshaler(
	binaryMarshaler encoding.BinaryMarshaler,
) (n int64, err error) {
	var bites []byte

	if bites, err = binaryMarshaler.MarshalBinary(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if _, err = ohio.WriteAllOrDieTrying(&encoder.field.Content, bites); err != nil {
		err = errors.WrapExceptSentinelAsNil(err, io.EOF)
		return n, err
	}

	if n, err = encoder.field.WriteTo(&encoder.fieldBuffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldByteReader(
	byteReader io.ByteReader,
) (n int64, err error) {
	var bite byte

	bite, err = byteReader.ReadByte()
	if err != nil {
		return n, err
	}

	err = encoder.field.Content.WriteByte(bite)
	if err != nil {
		return n, err
	}

	if n, err = encoder.field.WriteTo(&encoder.fieldBuffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

// binaryField is the key+length+content framing used for individual fields.
type binaryField struct {
	Key           key_bytes.Binary
	ContentLength uint16
	Content       bytes.Buffer
}

func (bf *binaryField) Reset() {
	bf.Key.Reset()
	bf.ContentLength = 0
	bf.Content.Reset()
}

func (bf *binaryField) ReadFrom(r io.Reader) (n int64, err error) {
	var n1 int
	var n2 int64
	n2, err = bf.Key.ReadFrom(r)
	n += int64(n2)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	n1, bf.ContentLength, err = ohio.ReadFixedUInt16(r)
	n += int64(n1)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	bf.Content.Grow(int(bf.ContentLength))
	bf.Content.Reset()

	n2, err = io.CopyN(
		&bf.Content,
		r,
		int64(bf.ContentLength),
	)
	n += n2

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (bf *binaryField) WriteTo(w io.Writer) (n int64, err error) {
	if bf.Content.Len() > math.MaxUint16 {
		err = errors.Errorf("content length too large: %d", bf.Content.Len())
		return n, err
	}

	bf.ContentLength = uint16(bf.Content.Len())

	var n1 int
	var n2 int64
	n2, err = bf.Key.WriteTo(w)
	n += int64(n2)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	n1, err = ohio.WriteFixedUInt16(w, bf.ContentLength)
	n += int64(n1)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	n2, err = io.Copy(w, &bf.Content)
	n += n2

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}
