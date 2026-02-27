package command_components_madder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type BlobError struct {
	BlobId domain_interfaces.MarklId
	Err    error
}

func PrintBlobErrors(
	envLocal env_local.Env,
	blobErrors collections_slice.Slice[BlobError],
) {
	ui.Err().Printf("blobs with errors: %d", blobErrors.Len())

	bufferedWriter, repool := pool.GetBufferedWriter(envLocal.GetErr())
	defer repool()

	defer errors.ContextMustFlush(envLocal, bufferedWriter)

	for _, errorBlob := range blobErrors {
		if errorBlob.BlobId == nil {
			bufferedWriter.WriteString("(empty blob id): ")
		} else {
			fmt.Fprintf(bufferedWriter, "%s: ", errorBlob.BlobId)
		}

		ui.CLIErrorTreeEncoder.EncodeTo(errorBlob.Err, bufferedWriter)

		bufferedWriter.WriteByte('\n')
	}
}
