package store_fs

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type FileEncoder interface {
	Encode(
		checkout_options.TextFormatterOptions,
		*sku.Transacted,
		*sku.FSItem,
	) error
}

type fileEncoder struct {
	mode              int
	perm              os.FileMode
	envRepo           env_repo.Env
	inlineTypeChecker ids.InlineTypeChecker

	object_metadata_fmt_triple_hyphen.FormatterFamily
}

func MakeFileEncoder(
	envRepo env_repo.Env,
	inlineTypeChecker ids.InlineTypeChecker,
) *fileEncoder {
	blobStore := envRepo.GetDefaultBlobStore()

	return &fileEncoder{
		mode:              os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
		perm:              0o666,
		envRepo:           envRepo,
		inlineTypeChecker: inlineTypeChecker,
		FormatterFamily: object_metadata_fmt_triple_hyphen.Factory{
			EnvDir:    envRepo,
			BlobStore: blobStore,
		}.MakeFormatterFamily(),
	}
}

func (encoder *fileEncoder) openOrCreate(path string) (file *os.File, err error) {
	if file, err = files.OpenFile(path, encoder.mode, encoder.perm); err != nil {
		err = errors.Wrap(err)

		if errors.IsExist(err) {
			// err = nil
			var err2 error

			if file, err2 = files.OpenExclusiveReadOnly(path); err2 != nil {
				err = errors.Wrap(err2)
			}
		} else {
			err = errors.Wrap(err)
		}

		return file, err
	}

	return file, err
}

func (encoder *fileEncoder) EncodeObject(
	options checkout_options.TextFormatterOptions,
	object *sku.Transacted,
	objectPath string,
	blobPath string,
	lockfilePath string,
) (err error) {
	ctx := object_metadata_fmt_triple_hyphen.FormatterContext{
		EncoderContext:   object.GetSku(),
		FormatterOptions: options,
	}

	inline := encoder.inlineTypeChecker.IsInlineType(object.GetType())

	var blobReader domain_interfaces.BlobReader

	if blobReader, err = encoder.envRepo.GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobReader)

	switch {
	case blobPath != "" && objectPath != "":
		var fileBlob, fileObject *os.File

		{
			if fileBlob, err = encoder.openOrCreate(
				blobPath,
			); err != nil {
				if errors.IsExist(err) {
					var blobWriter domain_interfaces.BlobWriter

					if blobWriter, err = encoder.envRepo.GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
						err = errors.Wrap(err)
						return err
					}

					defer errors.DeferredCloser(&err, blobWriter)

					if _, err = io.Copy(blobWriter, fileBlob); err != nil {
						err = errors.Wrap(err)
						return err
					}

				} else {
					err = errors.Wrap(err)
					return err
				}
			}

			defer errors.DeferredCloser(&err, fileBlob)

			if _, err = io.Copy(fileBlob, blobReader); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if fileObject, err = encoder.openOrCreate(
			objectPath,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, fileObject)

		if _, err = encoder.BlobPath.FormatMetadata(fileObject, ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case blobPath != "":
		var fileBlob *os.File

		if fileBlob, err = encoder.openOrCreate(
			blobPath,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, fileBlob)

		if _, err = io.Copy(fileBlob, blobReader); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case objectPath != "":
		var metadataFormatter object_metadata_fmt_triple_hyphen.Formatter

		if inline {
			metadataFormatter = encoder.InlineBlob
		} else {
			metadataFormatter = encoder.MetadataOnly
		}

		var fileMetadata *os.File

		if fileMetadata, err = encoder.openOrCreate(
			objectPath,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, fileMetadata)

		if _, err = metadataFormatter.FormatMetadata(
			fileMetadata,
			ctx,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (encoder *fileEncoder) Encode(
	options checkout_options.TextFormatterOptions,
	object *sku.Transacted,
	fsItem *sku.FSItem,
) (err error) {
	return encoder.EncodeObject(
		options,
		object,
		fsItem.Object.GetPath(),
		fsItem.Blob.GetPath(),
		fsItem.Lockfile.GetPath(),
	)
}
