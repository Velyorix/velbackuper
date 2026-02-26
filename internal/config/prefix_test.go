package config

import (
	"testing"
)

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"single segment", "backup", "backup"},
		{"trailing slash", "backup/", "backup"},
		{"leading slash", "/backup", "backup"},
		{"both slashes", "/backup/database/", "backup/database"},
		{"double slash middle", "backup//database", "backup/database"},
		{"multiple slashes", "backup///database///", "backup/database"},
		{"only slashes", "///", ""},
		{"backslashes", "backup\\database", "backup/database"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePrefix(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePrefix(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
