package unicorn

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestStripDiacritics(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		input    string
		expected string
	}{
		{"café", "cafe"},
		{"naïve", "naive"},
		{"Ångström", "Angstrom"},
		{"hello", "hello"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.input), func(t *ui.T) {
			got := StripDiacritics(tt.input)
			t.AssertEqualStrings(tt.expected, got)
		})
	}
}
