package store_fs

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/charlie/filesystem_ops"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO include blobs
func (store *Store) runDiff3(
	local, base, remote *sku.FSItem,
) (merged *sku.FSItem, err error) {
	var baseReader io.Reader

	if base.FDs.Len() > 0 {
		var baseCloser io.ReadCloser

		if baseCloser, err = store.fsOps.Open(
			base.Object.GetPath(),
			filesystem_ops.OpenModeDefault,
		); err != nil {
			err = errors.Wrap(err)
			return merged, err
		}

		defer errors.DeferredCloser(&err, baseCloser)

		baseReader = baseCloser
	} else {
		baseReader = strings.NewReader("")
	}

	var currentReader io.ReadCloser

	if currentReader, err = store.fsOps.Open(
		local.Object.GetPath(),
		filesystem_ops.OpenModeDefault,
	); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	defer errors.DeferredCloser(&err, currentReader)

	var otherReader io.ReadCloser

	if otherReader, err = store.fsOps.Open(
		remote.Object.GetPath(),
		filesystem_ops.OpenModeDefault,
	); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	defer errors.DeferredCloser(&err, otherReader)

	var mergedReader io.ReadCloser
	hasConflict := false

	if mergedReader, err = store.fsOps.Merge(
		baseReader,
		currentReader,
		otherReader,
	); err != nil {
		if errors.Is(err, filesystem_ops.ErrMergeConflict) {
			hasConflict = true
			err = nil
		} else {
			err = errors.Wrap(err)
			return merged, err
		}
	}

	defer errors.DeferredCloser(&err, mergedReader)

	var tempPath string
	var tempWriter io.WriteCloser

	if tempPath, tempWriter, err = store.fsOps.CreateTemp("", "merge-*"); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	defer errors.DeferredCloser(&err, tempWriter)

	if _, err = io.Copy(tempWriter, mergedReader); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	merged = &sku.FSItem{}
	merged.ResetWith(local)
	merged.Object.Reset()
	merged.Blob.Reset()
	merged.FDs.Reset()

	if err = merged.Object.SetPath(tempPath); err != nil {
		err = errors.Wrap(err)
		return merged, err
	}

	if hasConflict {
		err = errors.Wrap(sku.MakeErrMergeConflict(merged))
	}

	return merged, err
}
