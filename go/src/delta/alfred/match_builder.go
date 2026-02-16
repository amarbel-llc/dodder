package alfred

import (
	"bytes"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/charlie/doddish"
)

var poolMatchBuilder = pool.Make(
	NewMatchBuilder,
	func(matchBuilder *MatchBuilder) {
		matchBuilder.Buffer.Reset()
	},
)

func GetPoolMatchBuilder() interfaces.PoolPtr[MatchBuilder, *MatchBuilder] {
	return poolMatchBuilder
}

type MatchBuilder struct {
	bytes.Buffer
}

func NewMatchBuilder() *MatchBuilder {
	return &MatchBuilder{}
}

var sliceBytesUnderscore = []byte("_")

func (matchBuilder *MatchBuilder) AddMatchBytes(s []byte) {
	s1 := bytes.SplitSeq(s, sliceBytesUnderscore)

	for s2 := range s1 {
		matchBuilder.Write(s2)
		matchBuilder.WriteRune(' ')
	}
}

func (matchBuilder *MatchBuilder) AddMatchSeq(seq doddish.Seq) {
	for _, token := range seq {
		matchBuilder.Write(token.Contents)
		matchBuilder.WriteString(" ")
	}
}

func (matchBuilder *MatchBuilder) AddMatch(s string) {
	s1 := strings.SplitSeq(s, "_")

	for s2 := range s1 {
		matchBuilder.WriteString(s2)
		matchBuilder.WriteString(" ")
	}
}

func (matchBuilder *MatchBuilder) AddMatches(values ...string) {
	for _, value := range values {
		matchBuilder.AddMatch(value)
	}
}

func (matchBuilder *MatchBuilder) Bytes() []byte {
	return matchBuilder.Buffer.Bytes()
}
