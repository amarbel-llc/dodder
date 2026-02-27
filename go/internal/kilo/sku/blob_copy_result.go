package sku

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/golf/fd"
	"code.linenisgreat.com/dodder/go/internal/juliett/blob_stores"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type BlobCopyResult struct {
	ObjectOrNil *Transacted // may be nil
	blob_stores.CopyResult
}

func (copyResult BlobCopyResult) String() string {
	var stringsBuilder strings.Builder

	stringsBuilder.WriteString("BlobCopyResult{")

	if copyResult.ObjectOrNil == nil {
		stringsBuilder.WriteString("ObjectId: nil, ")
	} else {
		fmt.Fprintf(&stringsBuilder, "ObjectId: %s, ", copyResult.ObjectOrNil.GetObjectId())
	}

	stringsBuilder.WriteString(copyResult.CopyResult.String())

	stringsBuilder.WriteString("}")

	return stringsBuilder.String()
}

func MakeBlobCopierDelegate(
	fd fd.Std,
	includeAlreadyExists bool,
) func(BlobCopyResult) error {
	if !includeAlreadyExists {
		return func(result BlobCopyResult) error {
			if result.BlobId == nil {
				err := errors.Errorf("empty blob id")
				return err
			}

			bytesWritten, state := result.GetBytesWrittenAndState()

			switch state {
			case blob_stores.CopyResultStateExistsLocallyAndRemotely,
				blob_stores.CopyResultStateExistsLocally:
				return nil

			case blob_stores.CopyResultStateError:
				return fd.Printf(
					"Blob %s copy failed: %s",
					result.BlobId,
					result.GetError(),
				)

			case blob_stores.CopyResultStateNilRemoteBlobStore:
				return fd.Printf(
					"Blob %s: unable to copy, nil remote blob store",
					result.BlobId,
				)

			case blob_stores.CopyResultStateSuccess:
				// TODO rename to "Blob %s succeeded"
				return fd.Printf(
					"copied Blob %s (%s)",
					result.BlobId,
					ui.GetHumanBytesStringOrError(bytesWritten),
				)

			default:
				return fd.Printf(
					"Blob %s (%x) copy failure: %s",
					result.BlobId,
					result.BlobId.GetBytes(),
					state,
				)
			}
		}
	} else {
		return func(result BlobCopyResult) error {
			if result.BlobId == nil {
				err := errors.Errorf("empty blob id")
				return err
			}

			bytesWritten, state := result.GetBytesWrittenAndState()

			switch state {
			case blob_stores.CopyResultStateExistsLocallyAndRemotely, blob_stores.CopyResultStateExistsLocally:
				return fd.Printf(
					"Blob %s: already exists",
					result.BlobId,
				)

			case blob_stores.CopyResultStateNilRemoteBlobStore:
				return fd.Printf(
					"Blob %s: unable to copy, nil remote blob store",
					result.BlobId,
				)

			case blob_stores.CopyResultStateError:
				return fd.Printf(
					"Blob %s: %s",
					result.BlobId,
					result.GetError(),
				)

			case blob_stores.CopyResultStateSuccess:
				// TODO rename to "Blob %s succeeded"
				return fd.Printf(
					"copied Blob %s (%s)",
					result.BlobId,
					ui.GetHumanBytesStringOrError(bytesWritten),
				)

			default:
				return fd.Printf(
					"Blob %s (%x) copy failure: %s",
					result.BlobId,
					result.BlobId.GetBytes(),
					state,
				)
			}
		}
	}
}
