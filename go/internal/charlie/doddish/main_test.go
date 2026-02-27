package doddish

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

func TestMain(m *testing.M) {
	m.Run()
}

type testToken struct {
	TokenType
	Contents string
}

func (token testToken) String() string {
	return fmt.Sprintf("%s %s", token.TokenType, token.Contents)
}

func makeTestToken(tt TokenType, contents string) testToken {
	return testToken{
		TokenType: tt,
		Contents:  contents,
	}
}

type testSeq []testToken

func makeTestSeq(tokens ...any) (ts testSeq) {
	for i := 0; i < len(tokens); i += 2 {
		ts = append(ts,
			makeTestToken(
				tokens[i].(TokenType),
				tokens[i+1].(string),
			),
		)
	}

	return ts
}

func makeTestSeqFromSeq(seq Seq) (ts testSeq) {
	for _, t := range seq {
		ts = append(ts, testToken{
			TokenType: t.Type,
			Contents:  string(t.Contents),
		})
	}

	return ts
}

func makeSeqFromTestSeq(seq testSeq) (ts Seq) {
	for _, t := range seq {
		ts = append(ts, Token{
			Type:     t.TokenType,
			Contents: []byte(t.Contents),
		})
	}

	return ts
}

func makeSeqFromString(t *ui.T, input string) Seq {
	var scanner Scanner

	reader, repool := pool.GetStringReader(input)
	defer repool()

	scanner.Reset(reader)

	var index int

	var seq Seq

	for scanner.ScanDotAllowedInIdentifiers() {
		if index > 0 {
			t.Errorf("more than one seq in scanner")
		}

		seq = scanner.GetSeq()
		index++
	}

	if err := scanner.Error(); err != nil {
		t.AssertNoError(err)
	}

	return seq
}

func makeSeqsFromString(t *ui.T, input string) collections_slice.Slice[Seq] {
	var scanner Scanner

	reader, repool := pool.GetStringReader(input)
	defer repool()

	scanner.Reset(reader)

	var seqs collections_slice.Slice[Seq]

	for scanner.ScanDotAllowedInIdentifiers() {
		seqs.Append(scanner.GetSeq())
	}

	if err := scanner.Error(); err != nil {
		t.AssertNoError(err)
	}

	return seqs
}
