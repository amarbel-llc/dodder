package inventory_archive

import "io"

// SignatureComputer produces a fixed-length similarity signature from
// blob content. Signatures from the same computer are comparable:
// the fraction of matching positions estimates content similarity.
type SignatureComputer interface {
	SignatureLen() int
	ComputeSignature(content io.Reader) ([]uint32, error)
}
