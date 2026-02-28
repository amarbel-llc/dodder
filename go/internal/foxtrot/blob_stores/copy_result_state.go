package blob_stores

type CopyResultState interface {
	state()
}

//go:generate stringer -type=copyResultState
type copyResultState int

func (copyResultState) state() {}

const (
	CopyResultStateUnknown = copyResultState(iota)
	CopyResultStateSuccess
	CopyResultStateNilRemoteBlobStore
	CopyResultStateMissingLocally
	CopyResultStateExistsLocally
	CopyResultStateExistsLocallyAndRemotely
	CopyResultStateError
)
