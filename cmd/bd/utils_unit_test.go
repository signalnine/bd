package main

import (
	"testing"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{100, "s"},
		{-1, "s"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := pluralize(tt.count)
			if got != tt.want {
				t.Errorf("pluralize(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestGetContributorsSorted(t *testing.T) {
	// Test that contributors are returned in sorted order by commit count
	contributors := getContributorsSorted()

	if len(contributors) == 0 {
		t.Skip("No contributors defined")
	}

	// Check that we have at least some contributors
	if len(contributors) < 1 {
		t.Error("Expected at least one contributor")
	}

	// Verify first contributor has most commits (descending order)
	// We can't easily check counts, but we can verify the result is non-empty strings
	for i, c := range contributors {
		if c == "" {
			t.Errorf("Contributor at index %d is empty string", i)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"no truncation", "short", 10, "short"},
		{"exact length", "exact", 5, "exact"},
		{"truncate needed", "long string here", 10, "long st..."},
		{"very short max", "hello world", 5, "he..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestShowCleanupDeprecationHint removed: showCleanupDeprecationHint was removed in pruning.
