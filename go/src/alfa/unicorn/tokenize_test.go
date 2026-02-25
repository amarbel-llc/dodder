package unicorn

import "testing"

func TestExtractUniqueComponents(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUniqueComponents(tt.lines)

			if len(got) != len(tt.expected) {
				t.Fatalf("ExtractUniqueComponents() returned %d items, want %d: %v",
					len(got), len(tt.expected), got)
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
