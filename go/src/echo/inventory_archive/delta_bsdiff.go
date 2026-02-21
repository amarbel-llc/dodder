package inventory_archive

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	bsdiffpkg "github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

func init() {
	RegisterDeltaAlgorithm(&Bsdiff{})
}

// Bsdiff implements DeltaAlgorithm using the bsdiff4 binary delta algorithm.
type Bsdiff struct{}

var _ DeltaAlgorithm = &Bsdiff{}

func (b *Bsdiff) Id() byte {
	return DeltaAlgorithmByteBsdiff
}

func (b *Bsdiff) Compute(
	base domain_interfaces.BlobReader,
	baseSize int64,
	target io.Reader,
	delta io.Writer,
) error {
	// Read the full base into memory. bsdiff requires random access
	// to the base for suffix sorting. When BlobReader gains full
	// ReadAtSeeker support through compression/encryption, this can
	// be optimized.
	baseData, err := io.ReadAll(base)
	if err != nil {
		return errors.Wrap(err)
	}

	targetData, err := io.ReadAll(target)
	if err != nil {
		return errors.Wrap(err)
	}

	patch, err := bsdiffpkg.Bytes(baseData, targetData)
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err := delta.Write(patch); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (b *Bsdiff) Apply(
	base domain_interfaces.BlobReader,
	baseSize int64,
	delta io.Reader,
	target io.Writer,
) error {
	baseData, err := io.ReadAll(base)
	if err != nil {
		return errors.Wrap(err)
	}

	deltaData, err := io.ReadAll(delta)
	if err != nil {
		return errors.Wrap(err)
	}

	reconstructed, err := bspatch.Bytes(baseData, deltaData)
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err := target.Write(reconstructed); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
