package trie

import "testing"

type testStringer string

func TestContains(t *testing.T) {
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
		if !sut.Contains(e) {
			t.Errorf("expected %v to contain %s", sut, e)
		}
	}

	expectedNotContains := []string{
		"3",
		"12X45",
		"1234567",
	}

	for _, e := range expectedNotContains {
		if sut.Contains(e) {
			t.Errorf("expected %v to not contain %s", sut, e)
		}
	}
}

func TestShortestUnique(t *testing.T) {
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
		if ca := sut.Abbreviate(e); ca != c {
			t.Errorf("%q: expected shorted length %q but got %q", e, c, ca)
		}
	}
}

func TestExpand(t *testing.T) {
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
		if ca := sut.Expand(a); ca != e {
			t.Errorf("%q: expected expanded %q but got %q", e, e, ca)
		}
	}
}
