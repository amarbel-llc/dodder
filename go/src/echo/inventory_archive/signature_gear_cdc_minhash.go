package inventory_archive

import (
	"hash/fnv"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

// GearCDCMinHashComputer splits blob content into variable-length
// chunks using Gear hash CDC, hashes each chunk with FNV-1a, and
// computes a MinHash signature over the chunk hash set.
type GearCDCMinHashComputer struct {
	AvgChunkSize int
	MinChunkSize int
	MaxChunkSize int
	K            int
}

var _ SignatureComputer = &GearCDCMinHashComputer{}

func (c *GearCDCMinHashComputer) SignatureLen() int {
	return c.K
}

func (c *GearCDCMinHashComputer) ComputeSignature(
	content io.Reader,
) ([]uint32, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	chunks := GearCDCChunks(data, c.MinChunkSize, c.MaxChunkSize, c.AvgChunkSize)

	features := make([]uint32, len(chunks))
	h := fnv.New32a()

	for i, chunk := range chunks {
		h.Reset()
		h.Write(chunk)
		features[i] = h.Sum32()
	}

	return MinHashSignature(features, c.K), nil
}
