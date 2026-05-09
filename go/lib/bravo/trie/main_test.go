package trie

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

type testStringer string

func TestContains(t1 *testing.T) {
	t := ui.MakeT(t1)
	sut := Make(
		"123456",
		"654321",
	)

	expectedContains := []string{
		"1",
		"12",
		"123",
		"1234",
		"12345",
		"123456",
		"654321",
		"65432",
		"6543",
		"654",
		"65",
		"6",
	}

	for _, e := range expectedContains {
		t.AssertTrue(sut.Contains(e), "expected to contain "+e)
	}

	expectedNotContains := []string{
		"3",
		"12X45",
		"1234567",
	}

	for _, e := range expectedNotContains {
		t.AssertFalse(sut.Contains(e), "expected to not contain "+e)
	}
}

func TestShortestUnique(t1 *testing.T) {
	t := ui.MakeT(t1)
	sut := Make(
		"12",
		"121",
		"127",
		"128",
		"123456",
		"654321",
	)

	expectedContains := map[string]string{
		"123":      "123",
		"123456":   "123",
		"1234567":  "1234567",
		"12345678": "1234567",
		"124":      "124",
		"2":        "2",
	}

	for e, c := range expectedContains {
		t.AssertEqualStrings(c, sut.Abbreviate(e))
	}
}

func TestExpand(t1 *testing.T) {
	t := ui.MakeT(t1)
	sut := Make(
		"12",
		"121",
		"127",
		"128",
		"123456",
		"654321",
	)

	expectedContains := map[string]string{
		"6":    "654321",
		"128":  "128",
		"123":  "123456",
		"1232": "",
	}

	for a, e := range expectedContains {
		t.AssertEqualStrings(e, sut.Expand(a))
	}
}
