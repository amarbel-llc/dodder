package blob_stores

import (
	"io/fs"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func localAllBlobs(
	basePath string,
	defaultHashType markl.FormatHash,
) interfaces.SeqError[domain_interfaces.MarklId] {
	return func(yield func(domain_interfaces.MarklId, error) bool) {
		id, repool := defaultHashType.GetBlobId()
		defer repool()

		if basePath == "" {
			yield(nil, errors.Errorf("empty base path"))
			return
		}

		{
			var err error

			if basePath, err = files.Readlink(basePath); err != nil {
				yield(nil, errors.Wrap(err))
				return
			}
		}

		if err := filepath.WalkDir(
			basePath,
			func(path string, dirEntry fs.DirEntry, in error) (err error) {
				if in != nil {
					err = errors.Wrapf(in, "BasePath: %q", basePath)
					return err
				}

				if path == basePath {
					return err
				}

				if dirEntry.IsDir() {
					return err
				}

				if err = markl.SetHexStringFromAbsolutePath(id, path, basePath); err != nil {
					if !yield(nil, errors.Wrap(err)) {
						if dirEntry.IsDir() {
							err = filepath.SkipDir
						} else {
							err = nil
						}

						return err
					}

					return err
				}

				if id.IsNull() {
					return err
				}

				if !yield(id, nil) {
					err = filepath.SkipAll
					return err
				}

				return err
			},
		); err != nil {
			if !yield(nil, errors.Wrap(err)) {
				return
			}
		}
	}
}

func localAllBlobsMultihash(
	basePath string,
) interfaces.SeqError[domain_interfaces.MarklId] {
	return func(yield func(domain_interfaces.MarklId, error) bool) {
		dirEntries, err := files.DirEntries(basePath)
		if err != nil {
			yield(nil, errors.Wrap(err))
			return
		}

		for _, dirEntry := range dirEntries {
			if !dirEntry.IsDir() {
				continue
			}

			hashTypeId := dirEntry.Name()

			if hashTypeId == "." {
				continue
			}

			hashType, err := markl.GetFormatHashOrError(hashTypeId)
			if err != nil {
				if !yield(nil, errors.Wrap(err)) {
					return
				}

				continue
			}

			seq := localAllBlobs(
				filepath.Join(basePath, dirEntry.Name()),
				hashType,
			)

			for id, err := range seq {
				if !yield(id, err) {
					return
				}
			}
		}
	}
}
