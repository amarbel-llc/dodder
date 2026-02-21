package inventory_archive

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

// DeltaAlgorithm computes and applies binary deltas between blobs.
type DeltaAlgorithm interface {
	// Id returns the byte identifier written to data file delta entries.
	Id() byte

	// Compute produces a delta that transforms base into target.
	// The delta is written to the delta writer. base is a BlobReader
	// because current compression/encryption does not support seeking;
	// when BlobReader gains full ReadAtSeeker support, delta algorithms
	// can use random access for better performance.
	Compute(
		base domain_interfaces.BlobReader,
		baseSize int64,
		target io.Reader,
		delta io.Writer,
	) error

	// Apply reconstructs the original blob from a base and a delta.
	Apply(
		base domain_interfaces.BlobReader,
		baseSize int64,
		delta io.Reader,
		target io.Writer,
	) error
}

const (
	DeltaAlgorithmByteXdelta byte = 0
)

var deltaAlgorithms = map[byte]DeltaAlgorithm{}

var deltaAlgorithmNames = map[string]byte{
	"xdelta": DeltaAlgorithmByteXdelta,
}

// RegisterDeltaAlgorithm adds a DeltaAlgorithm to the registry.
func RegisterDeltaAlgorithm(alg DeltaAlgorithm) {
	deltaAlgorithms[alg.Id()] = alg
}

func DeltaAlgorithmForByte(b byte) (DeltaAlgorithm, error) {
	alg, ok := deltaAlgorithms[b]
	if !ok {
		return nil, errors.Errorf("unsupported delta algorithm byte: %d", b)
	}

	return alg, nil
}

func DeltaAlgorithmByteForName(name string) (byte, error) {
	b, ok := deltaAlgorithmNames[name]
	if !ok {
		return 0, errors.Errorf("unsupported delta algorithm name: %q", name)
	}

	return b, nil
}
