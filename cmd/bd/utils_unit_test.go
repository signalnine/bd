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

// TestGetContributorsSorted and TestTruncate removed:
// getContributorsSorted and truncate were deleted in nuclear simplification.

// TestShowCleanupDeprecationHint removed: showCleanupDeprecationHint was removed in pruning.
