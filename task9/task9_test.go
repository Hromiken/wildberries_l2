package main

import (
	"testing"
)

func TestUnpackingString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"a4bc2d5e", "aaaabccddddde", false},
		{"abcd", "abcd", false},
		{"45", "", true},
		{"ab12", "abbbbbbbbbbbb", false},
		{"\\", "", true},
		{"", "", false},
		{"qwe\\4\\5", "qwe45", false},
		{"qwe\\45", "qwe44444", false},
		{"a0b3", "bbb", false},
		{"a10", "aaaaaaaaaa", false},
		{"x0y0z0", "", false},
		{"abc2", "abcc", false},
		{"a\\4b3", "a4bbb", false},
	}

	for _, tt := range tests {
		got, err := unpackingString(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("unpackingString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("unpackingString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
