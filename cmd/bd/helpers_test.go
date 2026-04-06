package main

import (
	"testing"

	"github.com/steveyegge/beads/internal/utils"
)

func TestGetWorktreeGitDir(_ *testing.T) {
	gitDir := getWorktreeGitDir()
	// Just verify it doesn't panic and returns a string
	_ = gitDir
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bd-123", "bd"},
		{"custom-1", "custom"},
		{"TEST-999", "TEST"},
		{"no-number", "no"}, // Has hyphen, suffix not numeric, first hyphen
		{"nonumber", ""},    // No hyphen
		{"", ""},
		// Multi-part non-numeric suffixes (bd-fasa regression tests)
		{"vc-baseline-test", "vc"},
		{"vc-92cl-gate-test", "vc"},
		{"bd-multi-part-id", "bd"},
		{"prefix-a-b-c-d", "prefix"},
		// Multi-part prefixes with numeric suffixes
		{"beads-vscode-1", "beads-vscode"},
		{"alpha-beta-123", "alpha-beta"},
		{"my-project-42", "my-project"},
	}

	for _, tt := range tests {
		result := utils.ExtractIssuePrefix(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractIssuePrefix(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
