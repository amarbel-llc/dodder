package object_fmt_digest

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/delta/key_strings"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type format struct {
	purpose string
	keys    []keyType
}

func (format format) writeMetadataTo(
	writer io.Writer,
	context FormatterContext,
) (n int64, err error) {
	var n1 int64

	for _, key := range format.keys {
		n1, err = format.writeMetadataKeyTo(writer, context, key)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}

func (format format) writeMetadataKeyTo(
	writer io.Writer,
	context FormatterContext,
	key keyType,
) (n int64, err error) {
	return format.writeMetadataKeyStringTo(writer, context, key)
}

func (format format) writeMetadataKeyStringTo(
	writer io.Writer,
	context FormatterContext,
	key *catgut.String,
) (n int64, err error) {
	metadata := context.GetMetadata()

	var n1 int

	switch key {
	case key_strings.Blob:
		n1, err = format.writeMarklIdKeyIfNotNull(
			writer,
			key,
			metadata.GetBlobDigest(),
		)

		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_strings.Description:
		lines := strings.Split(metadata.GetDescription().String(), "\n")

		for _, line := range lines {
			if line == "" {
				continue
			}

			n1, err = ohio.WriteKeySpaceValueNewlineString(
				writer,
				key.String(),
				line,
			)
			n += int64(n1)

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	case key_strings.Tag:
		tags := metadata.GetTags()

		if tags == nil {
			break
		}

		var sortedValues []ids.TagStruct

		func() {
			defer func() {
				_ = recover()
			}()

			sortedValues = quiter.SortedValues(tags.All())
		}()

		for _, tag := range sortedValues {
			if tag.IsVirtual() {
				continue
			}

			n1, err = ohio.WriteKeySpaceValueNewlineString(
				writer,
				key.String(),
				tag.String(),
			)
			n += int64(n1)

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	case key_strings.ObjectId:
		n1, err = ohio.WriteKeySpaceValueNewlineString(
			writer,
			key_strings.Genre.String(),
			context.GetObjectId().GetGenre().String(),
		)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n1, err = ohio.WriteKeySpaceValueNewlineString(
			writer,
			key.String(),
			context.GetObjectId().String(),
		)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_strings.ZZRepoPub:
		n1, err = format.writeMarklIdKeyIfNotNull(
			writer,
			key,
			metadata.GetRepoPubKey(),
		)

		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_strings.ZZSigMother:
		n1, err = format.writeMarklIdKeyIfNotNull(
			writer,
			key,
			metadata.GetMotherObjectSig(),
		)

		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_strings.Tai:
		n1, err = ohio.WriteKeySpaceValueNewlineString(
			writer,
			key_strings.Tai.String(),
			metadata.GetTai().String(),
		)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	case key_strings.Type:
		if !metadata.GetType().IsEmpty() {
			n1, err = ohio.WriteKeySpaceValueNewlineString(
				writer,
				key.String(),
				metadata.GetType().String(),
			)
			n += int64(n1)

			if err != nil {
				err = errors.Wrap(err)
				return n, err
			}
		}

	case key_strings.TypeLock:
		typeLock := metadata.GetTypeLock()
		typeLockMarshaler := markl.MakeLockCoderValueNotRequired(typeLock)

		n1, err = ohio.WriteKeySpaceValueNewlineString(
			writer,
			key.String(),
			typeLockMarshaler.String(),
		)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

	default:
		err = errors.Errorf("unsupported key: %s", key)
		return n, err
	}

	return n, err
}

func (format format) writeMarklIdKey(
	writer io.Writer,
	key *catgut.String,
	id domain_interfaces.MarklId,
) (n int, err error) {
	if err = markl.AssertIdIsNotNull(id); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n, err = ohio.WriteKeySpaceValueNewlineString(
		writer,
		key.String(),
		markl.FormatBytesAsHex(id),
	)
	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (format format) writeMarklIdKeyIfNotNull(
	writer io.Writer,
	key *catgut.String,
	id domain_interfaces.MarklId,
) (n int, err error) {
	if id.IsNull() {
		return n, err
	}

	return format.writeMarklIdKey(writer, key, id)
}

func (format format) writeMetadata(
	writer io.Writer,
	context FormatterContext,
) (blobDigest domain_interfaces.MarklId, err error) {
	if context.GetMetadata().GetTai().IsEmpty() {
		err = ErrEmptyTai
		return blobDigest, err
	}

	hash, hashRepool := markl.FormatHashSha256.Get()
	defer hashRepool()

	marklWriter, writerRepool := markl_io.MakeWriterWithRepool(
		hash,
		writer,
	)
	defer writerRepool()

	_, err = format.writeMetadataTo(marklWriter, context)
	if err != nil {
		err = errors.Wrap(err)
		return blobDigest, err
	}

	blobDigest = marklWriter.GetMarklId()

	return blobDigest, err
}

// func GetDigestForContextDebug(
// 	format Format,
// 	context FormatterContext,
// ) (digest interfaces.MarklId, err error) {
// 	var sb strings.Builder
// 	writer, repool := markl_io.MakeWriterWithRepool(
// 		markl.FormatHashSha256.Get(),
// 		&sb,
// 	)
// 	defer repool()

// 	_, err = format.WriteMetadataTo(writer, context)
// 	if err != nil {
// 		err = errors.Wrap(err)
// 		return digest, err
// 	}

// 	digest = writer.GetMarklId()

// 	value := sb.String()

// 	ui.Debug().Printf("%q -> %s", value, markl.FormatBytesAsHex(digest))

// 	return digest, err
// }
