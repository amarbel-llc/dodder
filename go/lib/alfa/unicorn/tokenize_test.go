package unicorn

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestExtractUniqueComponents(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name     string
		lines    []string
		expected []string
	}{
		{
			name:     "basic extraction",
			lines:    []string{"the quick brown fox", "jumps over lazy dogs"},
			expected: []string{"brown", "dogs"},
		},
		{
			name:     "short words filtered",
			lines:    []string{"a to the end"},
			expected: []string{},
		},
		{
			name:     "punctuation filtered",
			lines:    []string{"hello, world! it's fine"},
			expected: []string{"fine"},
		},
		{
			name:     "diacritics stripped",
			lines:    []string{"café résumé naïve"},
			expected: []string{"naive"},
		},
		{
			name:     "duplicate last tokens removed",
			lines:    []string{"quick brown", "slow brown"},
			expected: []string{},
		},
		{
			name:     "single word lines",
			lines:    []string{"hello", "world"},
			expected: []string{"hello", "world"},
		},
		{
			name:     "empty input",
			lines:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			got := ExtractUniqueComponents(tt.lines)

			t.AssertEqual(tt.expected, got)
		})
	}
}
