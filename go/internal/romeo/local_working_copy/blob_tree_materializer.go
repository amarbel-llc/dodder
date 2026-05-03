package local_working_copy

import (
	"io"
	"os"
	"path/filepath"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// MaterializeBlobTree writes all blob references from a type object into a
// temporary directory tree, using each blob reference's alias as the file
// path (e.g., "filters/dodder-common.lua"). Returns the tmpdir path; cleanup
// is registered via the Repo's context.After so the tmpdir is removed when the
// context completes.
func (local *Repo) MaterializeBlobTree(
	typeObject *sku.Transacted,
) (blobTreeDir string, err error) {
	metadata := typeObject.GetMetadata()

	hasBlobRefs := false
	for range metadata.AllBlobReferences() {
		hasBlobRefs = true
		break
	}

	if !hasBlobRefs {
		return blobTreeDir, err
	}

	if blobTreeDir, err = os.MkdirTemp("", "dodder-blob-tree-*"); err != nil {
		err = errors.Wrap(err)
		return blobTreeDir, err
	}

	local.After(
		errors.MakeFuncContextFromFuncErr(
			func() error { return os.RemoveAll(blobTreeDir) },
		),
	)

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
			return blobTreeDir, err
		}
	}

	return blobTreeDir, err
}

func materializeOneBlob(
	blobStore mad_domain_interfaces.BlobStore,
	blobId mad_domain_interfaces.MarklId,
	blobTreeDir string,
	alias string,
) (err error) {
	destPath := filepath.Join(blobTreeDir, alias)

	if err = os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var reader mad_domain_interfaces.BlobReader

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
