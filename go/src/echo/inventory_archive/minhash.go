package inventory_archive

import (
	"math"
)

const minhashPrime = 4294967291 // largest prime < 2^32

func MinHashSignature(features []uint32, k int) []uint32 {
	sig := make([]uint32, k)
	for i := range sig {
		sig[i] = math.MaxUint32
	}

	coeffA, coeffB := minhashCoefficients(k)

	for _, f := range features {
		for i := 0; i < k; i++ {
			h := minhashUniversalHash(f, coeffA[i], coeffB[i])

			if h < sig[i] {
				sig[i] = h
			}
		}
	}

	return sig
}

func MinHashJaccard(sigA, sigB []uint32) float64 {
	if len(sigA) != len(sigB) {
		return 0
	}

	matches := 0
	nonEmpty := 0

	for i := range sigA {
		if sigA[i] == math.MaxUint32 && sigB[i] == math.MaxUint32 {
			continue
		}

		nonEmpty++

		if sigA[i] == sigB[i] {
			matches++
		}
	}

	if nonEmpty == 0 {
		return 0
	}

	return float64(matches) / float64(nonEmpty)
}

func minhashUniversalHash(x uint32, a, b uint64) uint32 {
	return uint32(((a*uint64(x) + b) % minhashPrime) & 0xFFFFFFFF)
}

func minhashCoefficients(k int) ([]uint64, []uint64) {
	coeffA := make([]uint64, k)
	coeffB := make([]uint64, k)

	state := uint64(0x5F3759DF)

	for i := 0; i < k; i++ {
		state = state*6364136223846793005 + 1442695040888963407
		coeffA[i] = (state % (minhashPrime - 1)) + 1

		state = state*6364136223846793005 + 1442695040888963407
		coeffB[i] = state % minhashPrime
	}

	return coeffA, coeffB
}
