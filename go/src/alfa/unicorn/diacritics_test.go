package unicorn

import "testing"

func TestStripDiacritics(t *testing.T) {
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
		t.Run(tt.input, func(t *testing.T) {
			got := StripDiacritics(tt.input)
			if got != tt.expected {
				t.Errorf("StripDiacritics(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
