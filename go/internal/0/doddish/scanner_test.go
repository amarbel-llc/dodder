package doddish

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

type scannerTestCase struct {
	input    string
	expected []testSeq
}

// TODO transition to ui.TestCase framework
func getScannerTestCases() []scannerTestCase {
	return []scannerTestCase{
		{
			input: "/]",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "/"),
				makeTestSeq(TokenTypeOperator, "]"),
			},
		},
		{
			input: ":",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, ":"),
			},
		},
		{
			input: "testing:e,t,k",
			expected: []testSeq{
				makeTestSeq(TokenTypeIdentifier, "testing"),
				makeTestSeq(TokenTypeOperator, ":"),
				makeTestSeq(TokenTypeIdentifier, "e"),
				makeTestSeq(TokenTypeOperator, ","),
				makeTestSeq(TokenTypeIdentifier, "t"),
				makeTestSeq(TokenTypeOperator, ","),
				makeTestSeq(TokenTypeIdentifier, "k"),
			},
		},
		{
			input: "[area-personal, area-work]:etikett",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(TokenTypeIdentifier, "area-personal"),
				makeTestSeq(TokenTypeOperator, ","),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(TokenTypeIdentifier, "area-work"),
				makeTestSeq(TokenTypeOperator, "]"),
				makeTestSeq(TokenTypeOperator, ":"),
				makeTestSeq(TokenTypeIdentifier, "etikett"),
			},
		},
		{
			input: " [ uno/dos ] bez",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeIdentifier, "uno",
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "dos",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(TokenTypeOperator, "]"),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(TokenTypeIdentifier, "bez"),
			},
		},
		{
			input: "md.type",
			expected: []testSeq{
				makeTestSeq(
					TokenTypeIdentifier, "md",
					TokenTypeOperator, ".",
					TokenTypeIdentifier, "type",
				),
			},
		},
		{
			input: "[md.type]",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(
					TokenTypeIdentifier, "md",
					TokenTypeOperator, ".",
					TokenTypeIdentifier, "type",
				),
				makeTestSeq(TokenTypeOperator, "]"),
			},
		},
		{
			input: "[uno/dos !pdf zz-inbox]",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(
					TokenTypeIdentifier, "uno",
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "dos",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "pdf",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeIdentifier, "zz-inbox",
				),
				makeTestSeq(TokenTypeOperator, "]"),
			},
		},
		{
			input: "[uno/dos !pdf@sig zz-inbox]",
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(
					TokenTypeIdentifier, "uno",
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "dos",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "pdf",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "sig",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeIdentifier, "zz-inbox",
				),
				makeTestSeq(TokenTypeOperator, "]"),
			},
		},
		{
			input: `/browser/bookmark-1FuOLQOYZAsP/ "Get Help" url="https://support.\"mozilla.org/products/firefox"`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "browser",
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "bookmark-1FuOLQOYZAsP",
					TokenTypeOperator, "/",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeLiteral, "Get Help",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeIdentifier,
					"url",
					TokenTypeOperator,
					"=",
					TokenTypeLiteral,
					`https://support."mozilla.org/products/firefox`,
				),
			},
		},
		{
			input: `.e`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeOperator, ".",
				),
				makeTestSeq(
					TokenTypeIdentifier, "e",
				),
			},
		},
		{
			input: `-tag`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeOperator, "-",
					TokenTypeIdentifier, "tag",
				),
			},
		},
		// ^ is operatorTypeSoloSeq so status^=cancelled produces 4 seqs
		// (use prefix ^status=cancelled for negated field queries)
		{
			input: `status^=cancelled`,
			expected: []testSeq{
				makeTestSeq(TokenTypeIdentifier, "status"),
				makeTestSeq(TokenTypeOperator, "^"),
				makeTestSeq(TokenTypeOperator, "="),
				makeTestSeq(TokenTypeIdentifier, "cancelled"),
			},
		},
		// typed blob ref without alias: <@digest !type@sig
		{
			input: `<@blake2b256-abc123 !tree@ed25519-sig456`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeOperator, "<",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "blake2b256-abc123",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "tree",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "ed25519-sig456",
				),
			},
		},
		// typed blob ref with alias: alias<@digest !type@sig
		{
			input: `hero<@blake2b256-abc123 !image-png@ed25519-sig456`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeIdentifier, "hero",
					TokenTypeOperator, "<",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "blake2b256-abc123",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "image-png",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "ed25519-sig456",
				),
			},
		},
		// typed blob ref no sig: <@digest !type
		{
			input: `<@blake2b256-abc123 !tree`,
			expected: []testSeq{
				makeTestSeq(
					TokenTypeOperator, "<",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "blake2b256-abc123",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "tree",
				),
			},
		},
		// typed blob ref in box: [id <@digest !type@sig]
		{
			input: `[one/dos <@blake2b256-abc123 !tree@ed25519-sig456]`,
			expected: []testSeq{
				makeTestSeq(TokenTypeOperator, "["),
				makeTestSeq(
					TokenTypeIdentifier, "one",
					TokenTypeOperator, "/",
					TokenTypeIdentifier, "dos",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "<",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "blake2b256-abc123",
				),
				makeTestSeq(TokenTypeOperator, " "),
				makeTestSeq(
					TokenTypeOperator, "!",
					TokenTypeIdentifier, "tree",
					TokenTypeOperator, "@",
					TokenTypeIdentifier, "ed25519-sig456",
				),
				makeTestSeq(TokenTypeOperator, "]"),
			},
		},
	}
}

func TestTokenScanner(t1 *testing.T) {
	t := ui.MakeT(t1)

	var scanner Scanner

	for _, tc := range getScannerTestCases() {
		reader, repool := pool.GetStringReader(tc.input)
		defer repool()
		scanner.Reset(reader)

		actual := make([]testSeq, 0)

		for scanner.Scan() {
			t1 := scanner.GetSeq().Clone()
			actual = append(actual, makeTestSeqFromSeq(t1))
		}

		if err := scanner.Error(); err != nil {
			t.AssertNoError(err)
		}

		t.Log(tc.input, "->", actual)

		t.AssertEqual(tc.expected, actual)
	}
}
