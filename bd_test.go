package bd_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/steveyegge/bd"
	"github.com/steveyegge/bd/internal/testutil"
)

// testServerPort is the port of the shared test Dolt server (0 = not running).
var testServerPort int

func TestMain(m *testing.M) {
	os.Setenv("BD_TEST_MODE", "1")
	if err := testutil.EnsureDoltContainerForTestMain(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %v, skipping Dolt tests\n", err)
	} else {
		defer testutil.TerminateDoltContainer()
		testServerPort = testutil.DoltContainerPortInt()
	}

	code := m.Run()

	os.Unsetenv("BD_DOLT_PORT")
	os.Unsetenv("BD_TEST_MODE")
	os.Exit(code)
}

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dolt"); err != nil {
		t.Skip("Dolt not installed, skipping test")
	}
}

func skipIfNoDoltServer(t *testing.T) {
	t.Helper()
	if testServerPort == 0 {
		t.Skip("Test Dolt server not available, skipping test")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", testServerPort)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		t.Skipf("Dolt server not running on %s, skipping test", addr)
	}
	_ = conn.Close()
}

func TestOpen(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-dolt")

	ctx := context.Background()
	store, err := bd.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestFindDatabasePath(t *testing.T) {
	// This will return empty string in test environment without a database
	path := bd.FindDatabasePath()
	// Just verify it doesn't panic
	_ = path
}

func TestFindBeadsDir(t *testing.T) {
	// This will return empty string or a valid path
	dir := bd.FindBdDir()
	// Just verify it doesn't panic
	_ = dir
}

func TestFindAllDatabases(t *testing.T) {
	// This scans the file system, just verify it doesn't panic
	dbs := bd.FindAllDatabases()
	// Should return a slice (possibly empty)
	if dbs == nil {
		t.Error("expected non-nil slice")
	}
}

// Test that exported constants have correct values
func TestConstants(t *testing.T) {
	// Status constants
	if bd.StatusOpen != "open" {
		t.Errorf("StatusOpen = %q, want %q", bd.StatusOpen, "open")
	}
	if bd.StatusInProgress != "in_progress" {
		t.Errorf("StatusInProgress = %q, want %q", bd.StatusInProgress, "in_progress")
	}
	if bd.StatusBlocked != "blocked" {
		t.Errorf("StatusBlocked = %q, want %q", bd.StatusBlocked, "blocked")
	}
	if bd.StatusClosed != "closed" {
		t.Errorf("StatusClosed = %q, want %q", bd.StatusClosed, "closed")
	}

	// IssueType constants
	if bd.TypeBug != "bug" {
		t.Errorf("TypeBug = %q, want %q", bd.TypeBug, "bug")
	}
	if bd.TypeFeature != "feature" {
		t.Errorf("TypeFeature = %q, want %q", bd.TypeFeature, "feature")
	}
	if bd.TypeTask != "task" {
		t.Errorf("TypeTask = %q, want %q", bd.TypeTask, "task")
	}
	if bd.TypeEpic != "epic" {
		t.Errorf("TypeEpic = %q, want %q", bd.TypeEpic, "epic")
	}

	// DependencyType constants
	if bd.DepBlocks != "blocks" {
		t.Errorf("DepBlocks = %q, want %q", bd.DepBlocks, "blocks")
	}
	if bd.DepRelated != "related" {
		t.Errorf("DepRelated = %q, want %q", bd.DepRelated, "related")
	}
}

func TestPublicAPITypeAssertions(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-dolt")

	ctx := context.Background()
	store, err := bd.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	t.Run("RemoteStore", func(t *testing.T) {
		// Concrete type -- methods always available
		remotes, err := store.ListRemotes(ctx)
		if err != nil {
			t.Fatalf("ListRemotes failed: %v", err)
		}
		_ = remotes
	})

	t.Run("VersionControl", func(t *testing.T) {
		branch, err := store.CurrentBranch(ctx)
		if err != nil {
			t.Fatalf("CurrentBranch failed: %v", err)
		}
		if branch == "" {
			t.Error("expected non-empty branch name")
		}

		commit, err := store.GetCurrentCommit(ctx)
		if err != nil {
			t.Fatalf("GetCurrentCommit failed: %v", err)
		}
		if commit == "" {
			t.Error("expected non-empty commit hash")
		}

		exists, err := store.CommitExists(ctx, commit)
		if err != nil {
			t.Fatalf("CommitExists failed: %v", err)
		}
		if !exists {
			t.Errorf("CommitExists(%s) = false, want true", commit)
		}

		status, err := store.Status(ctx)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		_ = status

		logs, err := store.Log(ctx, 5)
		if err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		_ = logs
	})
}
