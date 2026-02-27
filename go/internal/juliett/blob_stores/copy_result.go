package blob_stores

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/india/env_dir"
)

type CopyResult struct {
	BlobId       domain_interfaces.MarklId // may not be nil
	bytesWritten int64
	state        copyResultState
	err          error
}

func (copyResult CopyResult) String() string {
	var stringsBuilder strings.Builder

	stringsBuilder.WriteString("CopyResult{")

	if copyResult.BlobId == nil {
		stringsBuilder.WriteString("BlobId: nil, ")
	} else {
		fmt.Fprintf(&stringsBuilder, "BlobId: %s, ", copyResult.BlobId.StringWithFormat())
	}

	fmt.Fprintf(&stringsBuilder, "BytesWritten: %d, ", copyResult.bytesWritten)
	fmt.Fprintf(&stringsBuilder, "State: %s, ", copyResult.state)

	if copyResult.err == nil {
		stringsBuilder.WriteString("Error: nil")
	} else {
		fmt.Fprintf(&stringsBuilder, "Error: %s", copyResult.err)
	}

	stringsBuilder.WriteString("}")

	return stringsBuilder.String()
}

func (copyResult *CopyResult) SetError(err error) {
	copyResult.setErrorAfterCopy(-1, err)
}

func (copyResult *CopyResult) setErrorAfterCopy(n int64, err error) {
	copyResult.bytesWritten = n

	if env_dir.IsErrBlobAlreadyExists(err) {
		copyResult.SetBlobExistsLocallyAndRemotely()
	} else if markl.IsErrNull(err) {
		copyResult.SetBlobExistsLocallyAndRemotely()
	} else if env_dir.IsErrBlobMissing(err) {
		copyResult.SetBlobMissingLocally()
	} else if err != nil {
		copyResult.state = CopyResultStateError
		copyResult.err = err
	} else {
		copyResult.state = CopyResultStateSuccess
	}
}

func (copyResult CopyResult) GetBytesWrittenAndState() (int64, CopyResultState) {
	return copyResult.bytesWritten, copyResult.state
}

func (copyResult CopyResult) IsError() bool {
	return copyResult.state == CopyResultStateError
}

func (copyResult CopyResult) GetError() error {
	return copyResult.err
}

func (copyResult CopyResult) IsMissing() bool {
	return copyResult.state == CopyResultStateNilRemoteBlobStore
}

func (copyResult CopyResult) Exists() bool {
	switch copyResult.state {
	case CopyResultStateExistsLocally,
		CopyResultStateExistsLocallyAndRemotely:
		return true

	default:
		return false
	}
}

func (copyResult *CopyResult) SetBlobMissingLocally() {
	copyResult.bytesWritten = -1
	copyResult.state = CopyResultStateMissingLocally
}

func (copyResult *CopyResult) SetBlobExistsLocally() {
	copyResult.bytesWritten = -1
	copyResult.state = CopyResultStateExistsLocally
}

func (copyResult *CopyResult) SetBlobExistsLocallyAndRemotely() {
	copyResult.bytesWritten = -1
	copyResult.state = CopyResultStateExistsLocallyAndRemotely
}
