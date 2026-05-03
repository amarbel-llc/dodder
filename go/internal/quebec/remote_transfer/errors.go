package remote_transfer

import (
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

var (
	ErrSkipped = newPkgError("skipped due to exclude objects option")

	ErrNeedsMerge = errors.Err409Conflict.Errorf(
		"import failed with conflicts, merging required",
	)
)

// ErrCrossPubKeyMerge occurs when merging an object whose existing copy was
// created under a different repo public key and has no local workspace checkout.
type ErrCrossPubKeyMerge struct {
	ObjectId     string
	LocalPubKey  string
	RemotePubKey string
}

func (err ErrCrossPubKeyMerge) Error() string {
	return fmt.Sprintf(
		"cannot merge object %s: local pubkey %s differs from remote pubkey %s (no local checkout exists)",
		err.ObjectId,
		err.LocalPubKey,
		err.RemotePubKey,
	)
}

func (err ErrCrossPubKeyMerge) Is(target error) bool {
	_, ok := target.(ErrCrossPubKeyMerge)
	return ok
}

func (err ErrCrossPubKeyMerge) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrCrossPubKeyMerge(err error) bool {
	return errors.Is(err, ErrCrossPubKeyMerge{})
}

// ErrDeduped occurs when an object is skipped because its metadata digest
// (excluding TAI) matches a previously imported object in this batch.
type ErrDeduped struct {
	ObjectId string
	Digest   string
}

func (err ErrDeduped) Error() string {
	return fmt.Sprintf(
		"object %s deduped (digest %s already seen in this batch)",
		err.ObjectId,
		err.Digest,
	)
}

func (err ErrDeduped) Is(target error) bool {
	_, ok := target.(ErrDeduped)
	return ok
}

func (err ErrDeduped) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrDeduped(err error) bool {
	return errors.Is(err, ErrDeduped{})
}

// ErrBloblessTypeSkipped occurs when a type definition object with no blob
// digest, no pubkey, and no signature is encountered during import.
type ErrBloblessTypeSkipped struct {
	ObjectId string
	TypeId   string
}

func (err ErrBloblessTypeSkipped) Error() string {
	return fmt.Sprintf(
		"blobless type definition skipped: object %s type %s",
		err.ObjectId,
		err.TypeId,
	)
}

func (err ErrBloblessTypeSkipped) Is(target error) bool {
	_, ok := target.(ErrBloblessTypeSkipped)
	return ok
}

func (err ErrBloblessTypeSkipped) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrBloblessTypeSkipped(err error) bool {
	return errors.Is(err, ErrBloblessTypeSkipped{})
}

// ErrObjectIdTaiCollision occurs when an object with the same ObjectId+TAI
// already exists in the index but has a different object digest.
type ErrObjectIdTaiCollision struct {
	ObjectId     string
	Tai          string
	LocalDigest  string
	RemoteDigest string
}

func (err ErrObjectIdTaiCollision) Error() string {
	return fmt.Sprintf(
		"object %s at tai %s has digest collision: local %s != remote %s",
		err.ObjectId,
		err.Tai,
		err.LocalDigest,
		err.RemoteDigest,
	)
}

func (err ErrObjectIdTaiCollision) Is(target error) bool {
	_, ok := target.(ErrObjectIdTaiCollision)
	return ok
}

func (err ErrObjectIdTaiCollision) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrObjectIdTaiCollision(err error) bool {
	return errors.Is(err, ErrObjectIdTaiCollision{})
}
