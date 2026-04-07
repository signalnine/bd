//go:build cgo

package main

import (
	"os"
	"testing"

	"github.com/signalnine/bd/internal/configfile"
)

// TestNewDoltStoreFromConfig_NoMetadata verifies that newDoltStoreFromConfig
// succeeds when the beads directory has no metadata.json (fresh project).
// Regression test for GH#2988: "no database selected" error.
func TestNewDoltStoreFromConfig_NoMetadata(t *testing.T) {
	if os.Getenv("BD_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BD_TEST_EMBEDDED_DOLT=1 to run embedded dolt tests")
	}

	bdDir := t.TempDir()

	// Confirm no config exists.
	cfg, err := configfile.Load(bdDir)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config for empty dir")
	}

	// This should succeed using the default database name, not fail with
	// "no database selected".
	store, err := newDoltStoreFromConfig(t.Context(), bdDir)
	if err != nil {
		t.Fatalf("newDoltStoreFromConfig failed: %v", err)
	}
	defer store.Close()
}
