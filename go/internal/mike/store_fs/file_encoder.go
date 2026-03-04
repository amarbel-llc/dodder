package store_fs

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/filesystem_ops"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type FileEncoder interface {
	Encode(
		checkout_options.TextFormatterOptions,
		*sku.Transacted,
		*sku.FSItem,
	) error
}

type fileEncoder struct {
	fsOps             filesystem_ops.V0
	envRepo           env_repo.Env
	inlineTypeChecker ids.InlineTypeChecker

	object_metadata_fmt_triple_hyphen.FormatterFamily
}

func MakeFileEncoder(
	fsOps filesystem_ops.V0,
	envRepo env_repo.Env,
	inlineTypeChecker ids.InlineTypeChecker,
) *fileEncoder {
	blobStore := envRepo.GetDefaultBlobStore()

	return &fileEncoder{
		fsOps:             fsOps,
		envRepo:           envRepo,
		inlineTypeChecker: inlineTypeChecker,
		FormatterFamily: object_metadata_fmt_triple_hyphen.Factory{
			EnvDir:    envRepo,
			BlobStore: blobStore,
		}.MakeFormatterFamily(),
	}
}

func (encoder *fileEncoder) openOrCreate(path string) (file io.WriteCloser, err error) {
	if file, err = encoder.fsOps.Create(path, filesystem_ops.CreateModeTruncate); err != nil {
		err = errors.Wrap(err)
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
		var fileBlob, fileObject io.WriteCloser

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

					var existingBlob io.ReadCloser

					if existingBlob, err = encoder.fsOps.Open(
						blobPath,
						filesystem_ops.OpenModeExclusive,
					); err != nil {
						err = errors.Wrap(err)
						return err
					}

					defer errors.DeferredCloser(&err, existingBlob)

					if _, err = io.Copy(blobWriter, existingBlob); err != nil {
						err = errors.Wrap(err)
						return err
					}

				} else {
					err = errors.Wrap(err)
					return err
				}
			}

			if fileBlob != nil {
				defer errors.DeferredCloser(&err, fileBlob)

				if _, err = io.Copy(fileBlob, blobReader); err != nil {
					err = errors.Wrap(err)
					return err
				}
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
		var fileBlob io.WriteCloser

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

		var fileMetadata io.WriteCloser

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
