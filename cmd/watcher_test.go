package cmd

import "testing"

func TestShouldReload(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"README.md", "README.md", true},
		{"./README.md", "README.md", true},
		{"docs/README", "docs/README", true},
		{"docs/guide.md", "README.md", true},
		{"server.log", "README.md", false},
		{"README.md.swp", "README.md", false},
		{"README.md~", "README.md", false},
		{"4913", "README.md", false},
		{"server.log", "", false},
	}
	for _, tt := range tests {
		actual := shouldReload(tt.name, tt.filename)
		if actual != tt.expected {
			t.Errorf("shouldReload(%q, %q) = %v, want %v", tt.name, tt.filename, actual, tt.expected)
		}
	}
}
