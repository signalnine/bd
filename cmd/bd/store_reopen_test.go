//go:build cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/steveyegge/bd/internal/configfile"
	"github.com/steveyegge/bd/internal/storage/embeddeddolt"
	"github.com/steveyegge/bd/internal/types"
	"github.com/steveyegge/bd/internal/utils"
)

func TestWithStorage_ReopensUsingMetadata(t *testing.T) {
	ctx := context.Background()
	testDBPath := filepath.Join(t.TempDir(), "dolt")
	newTestStoreIsolatedDB(t, testDBPath, "cfg")

	var gotPrefix string
	err := withStorage(ctx, nil, testDBPath, func(s *embeddeddolt.EmbeddedDoltStore) error {
		var err error
		gotPrefix, err = s.GetConfig(ctx, "issue_prefix")
		return err
	})
	if err != nil {
		t.Fatalf("withStorage() error = %v", err)
	}
	if gotPrefix != "cfg" {
		t.Fatalf("issue_prefix = %q, want %q", gotPrefix, "cfg")
	}
}

func TestResolveBeadsDirForDBPath_UsesRawBeadsDirForSymlinkedDBPath(t *testing.T) {
	repoDir := t.TempDir()
	bdDir := filepath.Join(repoDir, ".bd")
	actualDBPath := filepath.Join(repoDir, "external-dolt")
	linkDBPath := filepath.Join(bdDir, "dolt")

	if err := os.MkdirAll(bdDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	if err := os.MkdirAll(actualDBPath, 0o755); err != nil {
		t.Fatalf("mkdir external dolt dir: %v", err)
	}
	if err := os.Symlink(actualDBPath, linkDBPath); err != nil {
		t.Fatalf("symlink db path: %v", err)
	}

	cfg := &configfile.Config{
		Database: "dolt",
		Backend:  configfile.BackendDolt,
	}
	if err := cfg.Save(bdDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	if got := resolveBeadsDirForDBPath(linkDBPath); !utils.PathsEqual(got, bdDir) {
		t.Fatalf("resolveBeadsDirForDBPath(%q) = %q, want %q", linkDBPath, got, bdDir)
	}
}

func TestIssueIDCompletion_UsesMetadataWhenStoreNil(t *testing.T) {
	originalStore := store
	originalDBPath := dbPath
	originalRootCtx := rootCtx
	defer func() {
		store = originalStore
		dbPath = originalDBPath
		rootCtx = originalRootCtx
	}()

	ctx := context.Background()
	rootCtx = ctx

	testDBPath := filepath.Join(t.TempDir(), "dolt")
	testStore := newTestStoreIsolatedDB(t, testDBPath, "cfg")
	if err := testStore.CreateIssue(ctx, &types.Issue{
		ID:        "cfg-abc1",
		Title:     "Completion target",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}, "test"); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	store = nil
	dbPath = testDBPath

	completions, directive := issueIDCompletion(&cobra.Command{}, nil, "cfg-a")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %d, want %d", directive, cobra.ShellCompDirectiveNoFileComp)
	}
	if len(completions) != 1 {
		t.Fatalf("len(completions) = %d, want 1 (%v)", len(completions), completions)
	}
	if len(completions[0]) < len("cfg-abc1") || completions[0][:len("cfg-abc1")] != "cfg-abc1" {
		t.Fatalf("completion = %q, want prefix %q", completions[0], "cfg-abc1")
	}
}

func TestResolveCommandBeadsDir_NoCWDFallbackForExplicitPath(t *testing.T) {
	// Set up project A with metadata so FindBdDir() discovers it from CWD.
	projectA := t.TempDir()
	beadsDirA := filepath.Join(projectA, ".bd")
	if err := os.MkdirAll(filepath.Join(beadsDirA, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir beads dir A: %v", err)
	}
	cfgA := &configfile.Config{
		Database:     "dolt",
		Backend:      configfile.BackendDolt,
		DoltDatabase: "project_a_db",
	}
	if err := cfgA.Save(beadsDirA); err != nil {
		t.Fatalf("save metadata A: %v", err)
	}

	// Project B: .bd/dolt exists but metadata.json is missing.
	// This triggers the bug: filepath.Dir(dbPath) gives the correct
	// .beads dir but configfile.Load returns nil, so the old code falls
	// through to FindBdDir() which discovers project A instead.
	projectB := t.TempDir()
	beadsDirB := filepath.Join(projectB, ".bd")
	if err := os.MkdirAll(filepath.Join(beadsDirB, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir beads dir B: %v", err)
	}

	// CWD is inside project A so FindBdDir() discovers A
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectA); err != nil {
		t.Fatalf("chdir to project A: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Simulate --db pointing to project B's database path
	dbPathB := filepath.Join(beadsDirB, "dolt")
	got := resolveCommandBeadsDir(dbPathB)

	// Must resolve to project B's .beads, NOT project A's.
	// The old code falls back to FindBdDir() and returns beadsDirA.
	if !utils.PathsEqual(got, beadsDirB) {
		t.Fatalf("resolveCommandBeadsDir(%q) = %q, want %q", dbPathB, got, beadsDirB)
	}
}
