package inventory_archive

var gearTable [256]uint64

func init() {
	var state uint64 = 0x5F3759DF
	for i := range gearTable {
		state = state*6364136223846793005 + 1442695040888963407
		gearTable[i] = state
	}
}

func GearCDCChunks(
	data []byte,
	minChunkSize, maxChunkSize, avgChunkSize int,
) [][]byte {
	if len(data) == 0 {
		return nil
	}

	mask := uint64(nextPowerOfTwo(avgChunkSize) - 1)

	var chunks [][]byte
	var fp uint64

	chunkStart := 0

	for i := 0; i < len(data); i++ {
		fp = (fp << 1) + gearTable[data[i]]

		chunkLen := i - chunkStart + 1

		if chunkLen < minChunkSize {
			continue
		}

		if chunkLen >= maxChunkSize || (fp&mask) == 0 {
			chunks = append(chunks, data[chunkStart:i+1])
			chunkStart = i + 1
			fp = 0
		}
	}

	if chunkStart < len(data) {
		chunks = append(chunks, data[chunkStart:])
	}

	return chunks
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}

	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++

	return n
}
