package doddish

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type seqTestCase struct {
	input    string
	expected [][]TokenMatcher
}

func getSeqTestCases() []seqTestCase {
	return []seqTestCase{
		{
			input: "/]",
			expected: [][]TokenMatcher{
				{
					TokenMatcherOp('/'),
				},
				{
					TokenMatcherOp(']'),
				},
			},
		},
		{
			input: "!md@blake2b256-zjt292cg6t4wtqp47jmp3akespk8lzvz69fl2nqfylcq2l5j652srdyzyt",
			expected: [][]TokenMatcher{
				{
					TokenMatcherOp('!'),
					TokenTypeIdentifier,
					TokenMatcherOp('@'),
					TokenTypeIdentifier,
				},
			},
		},
		{
			input: "-tag",
			expected: [][]TokenMatcher{
				{
					TokenMatcherOp('-'),
					TokenTypeIdentifier,
				},
			},
		},
		// {
		// 	input: ":",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, ":"),
		// 	},
		// },
		// {
		// 	input: "testing:e,t,k",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeIdentifier, "testing"),
		// 		makeTestSeq(TokenTypeOperator, ":"),
		// 		makeTestSeq(TokenTypeIdentifier, "e"),
		// 		makeTestSeq(TokenTypeOperator, ","),
		// 		makeTestSeq(TokenTypeIdentifier, "t"),
		// 		makeTestSeq(TokenTypeOperator, ","),
		// 		makeTestSeq(TokenTypeIdentifier, "k"),
		// 	},
		// },
		// {
		// 	input: "[area-personal, area-work]:etikett",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, "["),
		// 		makeTestSeq(TokenTypeIdentifier, "area-personal"),
		// 		makeTestSeq(TokenTypeOperator, ","),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(TokenTypeIdentifier, "area-work"),
		// 		makeTestSeq(TokenTypeOperator, "]"),
		// 		makeTestSeq(TokenTypeOperator, ":"),
		// 		makeTestSeq(TokenTypeIdentifier, "etikett"),
		// 	},
		// },
		// {
		// 	input: " [ uno/dos ] bez",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(TokenTypeOperator, "["),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "uno",
		// 			TokenTypeOperator, "/",
		// 			TokenTypeIdentifier, "dos",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(TokenTypeOperator, "]"),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(TokenTypeIdentifier, "bez"),
		// 	},
		// },
		// {
		// 	input: "md.type",
		// 	expected: []testSeq{
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "md",
		// 			TokenTypeOperator, ".",
		// 			TokenTypeIdentifier, "type",
		// 		),
		// 	},
		// },
		// {
		// 	input: "[md.type]",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, "["),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "md",
		// 			TokenTypeOperator, ".",
		// 			TokenTypeIdentifier, "type",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, "]"),
		// 	},
		// },
		// {
		// 	input: "[uno/dos !pdf zz-inbox]",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, "["),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "uno",
		// 			TokenTypeOperator, "/",
		// 			TokenTypeIdentifier, "dos",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeOperator, "!",
		// 			TokenTypeIdentifier, "pdf",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "zz-inbox",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, "]"),
		// 	},
		// },
		// {
		// 	input: "[uno/dos !pdf@sig zz-inbox]",
		// 	expected: []testSeq{
		// 		makeTestSeq(TokenTypeOperator, "["),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "uno",
		// 			TokenTypeOperator, "/",
		// 			TokenTypeIdentifier, "dos",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeOperator, "!",
		// 			TokenTypeIdentifier, "pdf",
		// 			TokenTypeOperator, "@",
		// 			TokenTypeIdentifier, "sig",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier, "zz-inbox",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, "]"),
		// 	},
		// },
		// {
		// 	input: `/browser/bookmark-1FuOLQOYZAsP/ "Get Help" url="https://support.\"mozilla.org/products/firefox"`,
		// 	expected: []testSeq{
		// 		makeTestSeq(
		// 			TokenTypeOperator, "/",
		// 			TokenTypeIdentifier, "browser",
		// 			TokenTypeOperator, "/",
		// 			TokenTypeIdentifier, "bookmark-1FuOLQOYZAsP",
		// 			TokenTypeOperator, "/",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeLiteral, "Get Help",
		// 		),
		// 		makeTestSeq(TokenTypeOperator, " "),
		// 		makeTestSeq(
		// 			TokenTypeIdentifier,
		// 			"url",
		// 			TokenTypeOperator,
		// 			"=",
		// 			TokenTypeLiteral,
		// 			`https://support."mozilla.org/products/firefox`,
		// 		),
		// 	},
		// },
		// typed blob ref: <@digest !type@sig (two seqs split by space)
		{
			input: "<@blake2b256-abc123",
			expected: [][]TokenMatcher{
				{
					TokenMatcherOp('<'),
					TokenMatcherOp('@'),
					TokenTypeIdentifier,
				},
			},
		},
		// type lock: !tree@sig
		{
			input: "!tree@ed25519-sig456",
			expected: [][]TokenMatcher{
				{
					TokenMatcherOp('!'),
					TokenTypeIdentifier,
					TokenMatcherOp('@'),
					TokenTypeIdentifier,
				},
			},
		},
	}
}

func TestTokenMatcherKeyValueNegated(t1 *testing.T) {
	t := ui.T{T: t1}

	// Construct a seq that matches key^=value pattern
	seq := Seq{
		{Type: TokenTypeIdentifier, Contents: []byte("status")},
		{Type: TokenTypeOperator, Contents: []byte("^")},
		{Type: TokenTypeOperator, Contents: []byte("=")},
	}

	if !seq.MatchAll(TokenMatcherKeyValueNegated...) {
		t.Errorf("expected seq to match TokenMatcherKeyValueNegated")
	}

	// Construct a seq that matches key^="value" pattern
	seqLiteral := Seq{
		{Type: TokenTypeIdentifier, Contents: []byte("status")},
		{Type: TokenTypeOperator, Contents: []byte("^")},
		{Type: TokenTypeOperator, Contents: []byte("=")},
		{Type: TokenTypeLiteral, Contents: []byte("cancelled")},
	}

	if !seqLiteral.MatchAll(TokenMatcherKeyValueNegatedLiteral...) {
		t.Errorf("expected seq to match TokenMatcherKeyValueNegatedLiteral")
	}

	// Verify non-negated doesn't match negated
	seqNonNeg := Seq{
		{Type: TokenTypeIdentifier, Contents: []byte("status")},
		{Type: TokenTypeOperator, Contents: []byte("=")},
	}

	if seqNonNeg.MatchAll(TokenMatcherKeyValueNegated...) {
		t.Errorf("expected non-negated seq not to match TokenMatcherKeyValueNegated")
	}
}

func TestSeq(t1 *testing.T) {
	t := ui.T{T: t1}

	var scanner Scanner

	for _, testCase := range getSeqTestCases() {
		reader, repool := pool.GetStringReader(testCase.input)
		defer repool()

		scanner.Reset(reader)

		var index int

		for scanner.ScanDotAllowedInIdentifiers() {
			seq := scanner.GetSeq()
			expectedTokenMatchers := testCase.expected[index]

			if !seq.MatchAll(expectedTokenMatchers...) {
				t.Errorf(
					"expected seq to match. Seq: %q, matchers: %q",
					seq,
					expectedTokenMatchers,
				)
			}

			index++
		}

		if err := scanner.Error(); err != nil {
			t.AssertNoError(err)
		}
	}
}
