package doddish

import (
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
)

func SeqsCompare(left, right []Seq) cmp.Result {
	return seqsCompare(left, right, false)
}

func SeqCompare(left, right Seq) cmp.Result {
	return seqCompare(left, right, false)
}

func SeqComparePartial(left, right Seq) cmp.Result {
	return seqCompare(left, right, true)
}

func seqsCompare(left, right []Seq, partial bool) cmp.Result {
	return cmp.CompareUTF8(
		GetComparableSeqs(left),
		GetComparableSeqs(right),
		partial,
	)
}

func seqCompare(left, right Seq, partial bool) cmp.Result {
	return cmp.CompareUTF8(
		left.GetComparable(),
		right.GetComparable(),
		partial,
	)
}

type ComparableSeq struct {
	Tokens         []Token
	TokenIndex     int
	TokenByteIndex int
	ByteCount      int
}

func (seq Seq) GetComparable() ComparableSeq {
	var byteCount int

	for _, token := range seq {
		byteCount += len(token.Contents)
	}

	return ComparableSeq{
		Tokens:    seq,
		ByteCount: byteCount,
	}
}

func (seq ComparableSeq) Len() int {
	return seq.ByteCount
}

func (seq ComparableSeq) getRemainingTokenBytes() []byte {
	return seq.Tokens[seq.TokenIndex].Contents[seq.TokenByteIndex:]
}

func (seq ComparableSeq) DecodeRune() (char rune, width int) {
	char, width = utf8.DecodeRune(seq.getRemainingTokenBytes())
	return char, width
}

func (seq ComparableSeq) Shift(amount int) ComparableSeq {
	remainingBytes := seq.getRemainingTokenBytes()
	seq.ByteCount -= amount

	if len(remainingBytes) == amount {
		seq.TokenByteIndex = 0
		seq.TokenIndex++
	} else {
		seq.TokenByteIndex += amount
	}

	return seq
}

type ComparableSeqs struct {
	Seqs       []Seq
	SeqIndex   int
	CurrentSeq ComparableSeq
	ByteCount  int
}

func GetComparableSeqs(seqs []Seq) ComparableSeqs {
	var byteCount int

	for _, seq := range seqs {
		for _, token := range seq {
			byteCount += len(token.Contents)
		}
	}

	result := ComparableSeqs{
		Seqs:      seqs,
		SeqIndex:  0,
		ByteCount: byteCount,
	}

	// Find the first non-empty Seq
	for result.SeqIndex < len(seqs) {
		result.CurrentSeq = seqs[result.SeqIndex].GetComparable()
		if result.CurrentSeq.Len() > 0 {
			break
		}
		result.SeqIndex++
	}

	return result
}

func (seqs ComparableSeqs) Len() int {
	return seqs.ByteCount
}

func (seqs ComparableSeqs) DecodeRune() (char rune, width int) {
	return seqs.CurrentSeq.DecodeRune()
}

func (seqs ComparableSeqs) Shift(amount int) ComparableSeqs {
	seqs.ByteCount -= amount
	seqs.CurrentSeq = seqs.CurrentSeq.Shift(amount)

	for seqs.CurrentSeq.Len() == 0 && seqs.SeqIndex+1 < len(seqs.Seqs) {
		seqs.SeqIndex++
		seqs.CurrentSeq = seqs.Seqs[seqs.SeqIndex].GetComparable()
	}

	return seqs
}
