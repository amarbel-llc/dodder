package local_working_copy

import (
	"io"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// MaterializeBlobTree writes all blob references from a type object into a
// temporary directory tree, using each blob reference's alias as the file
// path (e.g., "filters/dodder-common.lua"). Returns the tmpdir path and a
// cleanup function.
//
// TODO use context.AfterFunc for cleanup instead of manual defer at call sites.
func (local *Repo) MaterializeBlobTree(
	typeObject *sku.Transacted,
) (blobTreeDir string, cleanup func(), err error) {
	cleanup = func() {}

	metadata := typeObject.GetMetadata()

	hasBlobRefs := false
	for range metadata.AllBlobReferences() {
		hasBlobRefs = true
		break
	}

	if !hasBlobRefs {
		return blobTreeDir, cleanup, err
	}

	if blobTreeDir, err = os.MkdirTemp("", "dodder-blob-tree-*"); err != nil {
		err = errors.Wrap(err)
		return blobTreeDir, cleanup, err
	}

	cleanup = func() {
		os.RemoveAll(blobTreeDir)
	}

	blobStore := local.GetEnvRepo().GetDefaultBlobStore()

	for blobId := range metadata.AllBlobReferences() {
		alias := metadata.GetBlobReferenceAlias(blobId)

		if alias == "" {
			continue
		}

		if err = materializeOneBlob(
			blobStore,
			blobId,
			blobTreeDir,
			alias,
		); err != nil {
			err = errors.Wrapf(err, "blob reference alias: %q", alias)
			return blobTreeDir, cleanup, err
		}
	}

	return blobTreeDir, cleanup, err
}

func materializeOneBlob(
	blobStore domain_interfaces.BlobStore,
	blobId domain_interfaces.MarklId,
	blobTreeDir string,
	alias string,
) (err error) {
	destPath := filepath.Join(blobTreeDir, alias)

	if err = os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var reader domain_interfaces.BlobReader

	if reader, err = blobStore.MakeBlobReader(blobId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, reader)

	var file *os.File

	if file, err = os.Create(destPath); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	if _, err = io.Copy(file, reader); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
