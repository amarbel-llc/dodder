package doddish

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/collections_slice"
)

type Seq collections_slice.Slice[Token]

func (seq Seq) Len() int {
	return len(seq)
}

func (seq Seq) At(idx int) Token {
	return seq[idx]
}

func (seq *Seq) GetSlice() collections_slice.Slice[Token] {
	return (collections_slice.Slice[Token])(*seq)
}

func (seq *Seq) GetSliceMutable() *collections_slice.Slice[Token] {
	return (*collections_slice.Slice[Token])(seq)
}

func (seq *Seq) Add(tokenType TokenType, contents []byte) {
	seq.GetSliceMutable().Append(Token{Type: tokenType, Contents: contents})
}

func (seq Seq) StringDebug() string {
	var sb strings.Builder

	sb.WriteString("Seq{")
	for _, t := range seq {
		fmt.Fprintf(&sb, "%s:%q ", t.Type, t.Contents)
	}
	sb.WriteString("}")

	return sb.String()
}

func (seq Seq) String() string {
	var sb strings.Builder

	for _, t := range seq {
		sb.Write(t.Contents)
	}

	return sb.String()
}

func (seq Seq) Clone() (dst Seq) {
	dst = make(Seq, len(seq))

	for i := range seq {
		dst[i] = seq[i].Clone()
	}

	return dst
}

func (seq *Seq) Reset() {
	seq.GetSliceMutable().Reset()
}

func (seq Seq) GetTokenTypes() TokenTypes {
	out := make(TokenTypes, seq.Len())

	for i := range out {
		out[i] = seq.At(i).Type
	}

	return out
}

func (seq Seq) GetBinaryMarshaler() SeqBinaryCoding {
	return (SeqBinaryCoding)(seq)
}

func (seq *Seq) GetBinaryUnmarshaler() *SeqBinaryCoding {
	return (*SeqBinaryCoding)(seq)
}
