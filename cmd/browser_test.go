package cmd

import (
	"errors"
	"testing"
)

type MockFileReader struct {
	content string
	err     error
}

func (m MockFileReader) ReadFile(path string) (string, error) {
	return m.content, m.err
}

func TestIsContainWSL(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{
			name:     "WSL Data",
			data:     "Linux version 4.19.128-microsoft-standard (WSL2)",
			expected: true,
		},
		{
			name:     "Non-WSL Data",
			data:     "Linux version 4.15.0-72-generic",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContainWSL(tt.data)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsWSL(t *testing.T) {
	tests := []struct {
		name     string
		reader   FileReader
		expected bool
	}{
		{
			name:     "WSL Data",
			reader:   MockFileReader{content: "Linux version 4.19.128-microsoft-standard (WSL2)"},
			expected: true,
		},
		{
			name:     "Non-WSL Data",
			reader:   MockFileReader{content: "Linux version 4.15.0-72-generic"},
			expected: false,
		},
		{
			name:     "Read error",
			reader:   MockFileReader{err: errors.New("read error")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWSLWithReader(tt.reader)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
