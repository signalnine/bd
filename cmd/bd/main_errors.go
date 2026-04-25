package main

import (
	"fmt"
	"os"
	"strings"
)

// isFreshCloneError checks if the error is due to a fresh clone scenario
// where the database exists but is missing required config (like issue_prefix).
// This happens when someone clones a repo with bd but needs to initialize.
func isFreshCloneError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for the specific migration invariant error pattern
	return strings.Contains(errStr, "post-migration validation failed") &&
		strings.Contains(errStr, "required config key missing: issue_prefix")
}

// handleFreshCloneError displays a helpful message when a fresh clone is detected
// and returns true if the error was handled (so caller should exit).
// If not a fresh clone error, returns false and does nothing.
func handleFreshCloneError(err error) bool {
	if !isFreshCloneError(err) {
		return false
	}

	fmt.Fprintf(os.Stderr, "Error: Database not initialized\n\n")
	fmt.Fprintf(os.Stderr, "This appears to be a fresh clone or an existing project whose database needs recovery.\n")
	fmt.Fprintf(os.Stderr, "\nIf this is an existing project or fresh clone, run: bd bootstrap\n")
	fmt.Fprintf(os.Stderr, "To create a brand-new database from scratch: bd init --prefix <your-prefix>\n")
	return true
}
