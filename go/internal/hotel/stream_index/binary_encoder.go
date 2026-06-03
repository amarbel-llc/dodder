package stream_index

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"math"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/key_bytes"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type binaryEncoder struct {
	bytes.Buffer
	binaryField
	ids.Sigil
}

func (encoder *binaryEncoder) updateSigil(
	writerAt io.WriterAt,
	sigil ids.Sigil,
	offset int64,
) (err error) {
	sigil.Add(encoder.Sigil)
	// 2 uint8 + offset + 2 uint8 + Schlussel
	offset = int64(2) + offset + int64(3)

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

func (encoder *binaryEncoder) writeFormat(
	writer io.Writer,
	object objectWithSigil,
) (n int64, err error) {
	encoder.Buffer.Reset()

	for _, key := range binaryFieldOrder {
		encoder.binaryField.Reset()
		encoder.Key = key

		if _, err = encoder.writeFieldKey(object); err != nil {
			err = errors.Wrapf(
				err,
				"Key: %q, Sku: %s",
				encoder.Key,
				sku.String(object.Transacted),
			)
			return n, err
		}
	}

	encoder.binaryField.Reset()
	rawContentLength := encoder.Len()

	if rawContentLength > math.MaxUint16 {
		err = errContentLengthTooLarge
		return n, err
	}

	encoder.ContentLength = uint16(rawContentLength)

	var n1 int
	n1, err = ohio.WriteFixedUInt16(writer, encoder.ContentLength)
	n += int64(n1)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	var n2 int64
	n2, err = encoder.Buffer.WriteTo(writer)
	n += n2

	return n, err
}

func (encoder *binaryEncoder) writeFieldKey(
	object objectWithSigil,
) (n int64, err error) {
	metadata := object.GetMetadata()

	switch encoder.Key {
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

		// TODO change to false once dropping V8
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

	case key_bytes.References:
		metadataMutable := object.GetMetadataMutable()

		for refId := range metadata.AllReferencedObjects() {
			refLock := metadataMutable.GetReferencedObjectLockMutable(refId)

			if refLock.GetValue().IsNull() {
				continue
			}

			binaryMarshaler := markl.MakeMutableLockCoderValueRequired(refLock)

			var n1 int64
			if n1, err = encoder.writeFieldBinaryMarshaler(binaryMarshaler); err != nil {
				err = errors.Wrap(err)
				return n, err
			}
			n += n1
		}

	case key_bytes.ReferenceAliases:
		for refId := range metadata.AllReferencedObjects() {
			alias := metadata.GetReferenceAlias(refId)
			if alias == "" {
				continue
			}

			if _, err = io.WriteString(
				&encoder.Content,
				refId.String(),
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			if _, err = ohio.WriteAllOrDieTrying(
				&encoder.Content, []byte{0},
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			if _, err = io.WriteString(&encoder.Content, alias); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			var n1 int64
			if n1, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
				err = errors.Wrap(err)
				return n, err
			}
			n += n1
		}

	case key_bytes.BlobReferences:
		metadataMutable := object.GetMetadataMutable()

		for blobId := range metadata.AllBlobReferences() {
			bites, marshalErr := blobId.MarshalBinary()
			if marshalErr != nil {
				err = errors.Wrap(marshalErr)
				return n, err
			}

			// Write 2-byte length prefix for markl.Id bytes
			idLen := len(bites)
			if _, err = ohio.WriteFixedUInt16(
				&encoder.Content,
				uint16(idLen),
			); err != nil {
				err = errors.Wrap(err)
				return n, err
			}

			if _, err = ohio.WriteAllOrDieTrying(&encoder.Content, bites); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			// Null byte sentinel separates old format from new format with type lock
			if _, err = ohio.WriteAllOrDieTrying(
				&encoder.Content, []byte{0},
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			// Write type lock bytes with 2-byte length prefix
			typeLockMutable := metadataMutable.GetBlobReferenceTypeLockMutable(blobId)
			typeLockMarshaler := markl.MakeMutableLockCoderValueNotRequired(typeLockMutable)
			typeLockBytes, marshalErr := typeLockMarshaler.MarshalBinary()
			if marshalErr != nil {
				err = errors.Wrap(marshalErr)
				return n, err
			}

			if _, err = ohio.WriteFixedUInt16(
				&encoder.Content,
				uint16(len(typeLockBytes)),
			); err != nil {
				err = errors.Wrap(err)
				return n, err
			}

			if len(typeLockBytes) > 0 {
				if _, err = ohio.WriteAllOrDieTrying(&encoder.Content, typeLockBytes); err != nil {
					err = errors.WrapExceptSentinelAsNil(err, io.EOF)
					return n, err
				}
			}

			alias := metadata.GetBlobReferenceAlias(blobId)
			if alias != "" {
				if _, err = io.WriteString(&encoder.Content, alias); err != nil {
					err = errors.WrapExceptSentinelAsNil(err, io.EOF)
					return n, err
				}
			}

			var n1 int64
			if n1, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
				err = errors.Wrap(err)
				return n, err
			}
			n += n1
		}

	case key_bytes.SigParentMetadataParentObjectId:
		if n, err = encoder.writeFieldMerkleId(
			metadata.GetMotherObjectSig(),
			true,
			encoder.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.DigestMetadataWithoutTai:
		if n, err = encoder.writeFieldMerkleId(
			object.GetMetadata().GetIndex().GetSelfWithoutTai(),
			true,
			encoder.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.DigestMetadataParentObjectId:
		if n, err = encoder.writeFieldMerkleId(
			metadata.GetObjectDigest(),
			true,
			encoder.Key.String(),
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_bytes.ExternalObjectId:
		eoid := object.GetExternalObjectIdMutable()

		if eoid.IsEmpty() {
			return n, err
		}

		if n, err = encoder.writeFieldBinaryMarshaler(eoid); err != nil {
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

	case key_bytes.Field:
		for field := range metadata.GetIndex().GetFields() {
			if field.TypeBlobDigest.IsEmpty() {
				continue
			}

			// Wire format: key\x00value\x00typeBlobDigest (binary encoded)
			if _, err = io.WriteString(
				&encoder.Content,
				field.Key,
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			if _, err = ohio.WriteAllOrDieTrying(
				&encoder.Content, []byte{0},
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			if _, err = io.WriteString(
				&encoder.Content,
				field.Value,
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			if _, err = ohio.WriteAllOrDieTrying(
				&encoder.Content, []byte{0},
			); err != nil {
				err = errors.WrapExceptSentinelAsNil(err, io.EOF)
				return n, err
			}

			{
				digestBytes, marshalErr := markl.IdBinaryEncodingTypeData{
					MarklId: field.TypeBlobDigest,
				}.MarshalBinary()

				if marshalErr != nil {
					err = errors.Wrap(marshalErr)
					return n, err
				}

				if _, err = ohio.WriteAllOrDieTrying(
					&encoder.Content, digestBytes,
				); err != nil {
					err = errors.WrapExceptSentinelAsNil(err, io.EOF)
					return n, err
				}
			}

			var n1 int64
			if n1, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
				err = errors.Wrap(err)
				return n, err
			}
			n += n1
		}

	default:
		panic(fmt.Sprintf("unsupported key: %s", encoder.Key))
	}

	return n, err
}

func (encoder *binaryEncoder) writeMarklId(
	marklId mad_domain_interfaces.MarklId,
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
	_, err = wt.WriteTo(&encoder.Content)
	if err != nil {
		return n, err
	}

	if n, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (encoder *binaryEncoder) writeFieldMerkleId(
	merkleId mad_domain_interfaces.MarklId,
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

	if _, err = io.WriteString(&encoder.Content, value); err != nil {
		err = errors.WrapExceptSentinelAsNil(err, io.EOF)
		return n, err
	}

	if n, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
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

	if _, err = ohio.WriteAllOrDieTrying(&encoder.Content, bites); err != nil {
		err = errors.WrapExceptSentinelAsNil(err, io.EOF)
		return n, err
	}

	if n, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
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

	err = encoder.Content.WriteByte(bite)
	if err != nil {
		return n, err
	}

	if n, err = encoder.binaryField.WriteTo(&encoder.Buffer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
