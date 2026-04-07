//go:build cgo

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/steveyegge/bd/internal/config"
	"github.com/steveyegge/bd/internal/storage/embeddeddolt"
)

// testDoltServerPort is the port of the shared test Dolt server (0 = not running).
// Server-mode Dolt was removed in the nuclear simplification; this is kept for
// tests that skip when no server is available.
var testDoltServerPort int

// uniqueTestDBName generates a unique database name for test isolation.
func uniqueTestDBName(t *testing.T) string {
	t.Helper()
	h := sha256.Sum256([]byte(t.Name() + fmt.Sprintf("%d", time.Now().UnixNano())))
	return "testdb_" + hex.EncodeToString(h[:6])
}

// testIDCounter ensures unique IDs across all test runs
var testIDCounter atomic.Uint64

// doltNewMutex serializes embeddeddolt.New() calls in tests. The Dolt embedded engine's
// InitStatusVariables() has an internal race condition when called concurrently
// from multiple goroutines (writes to a shared global map without synchronization).
// Serializing store creation prevents this race while allowing tests to run their
// assertions in parallel after the store is created.
var doltNewMutex sync.Mutex

// stdioMutex serializes tests that redirect os.Stdout or os.Stderr.
// These process-global file descriptors cannot be safely redirected from
// concurrent goroutines.
//
// IMPORTANT: Any test that calls cobra's Help(), Execute(), or Print*()
// MUST NOT be parallel (no t.Parallel()), OR must serialize those calls
// under stdioMutex. Setting cmd.SetOut() is NOT sufficient because cobra's
// OutOrStdout() eagerly evaluates os.Stdout as the default argument even
// when outWriter is set -- the Go race detector catches this read.
//
// TestCobraParallelPolicyGuard in stdio_race_guard_test.go enforces this.
var stdioMutex sync.Mutex

// generateUniqueTestID creates a globally unique test ID using prefix, test name, and atomic counter.
// This prevents ID collisions when multiple tests manipulate global state.
func generateUniqueTestID(t *testing.T, prefix string, index int) string {
	t.Helper()
	counter := testIDCounter.Add(1)
	// include test name, counter, and index for uniqueness
	data := []byte(t.Name() + prefix + string(rune(counter)) + string(rune(index)))
	hash := sha256.Sum256(data)
	return prefix + "-" + hex.EncodeToString(hash[:])[:8]
}

// ptrTime returns a pointer to the given time.Time value.
func ptrTime(t time.Time) *time.Time {
	return &t
}

const windowsOS = "windows"

// initConfigForTest initializes viper config for a test and ensures cleanup.
// main.go's init() calls config.Initialize() which picks up the real .bd/config.yaml.
// TestMain resets viper, but any test calling config.Initialize() re-loads the real config.
// This helper ensures viper is reset after the test completes, preventing state pollution
// (e.g., repo config values leaking into JSONL export tests).
func initConfigForTest(t *testing.T) {
	t.Helper()
	config.ResetForTesting()
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	t.Cleanup(config.ResetForTesting)
}

// ensureTestMode is a no-op; BD_TEST_MODE is set once in TestMain.
// Previously each test set/unset the env var, which raced under t.Parallel().
func ensureTestMode(t *testing.T) {
	t.Helper()
	// BD_TEST_MODE is set in TestMain and stays set for the entire test run.
}

// ensureCleanGlobalState resets global state that may have been modified by other tests.
// Call this at the start of tests that manipulate globals directly.
func ensureCleanGlobalState(t *testing.T) {
	t.Helper()
	// Reset CommandContext so accessor functions fall back to globals
	resetCommandContext()
}

// savedGlobals holds a snapshot of package-level globals for safe restoration.
// Used by saveAndRestoreGlobals to ensure test isolation.
type savedGlobals struct {
	dbPath      string
	store       *embeddeddolt.EmbeddedDoltStore
	storeActive bool
}

// saveAndRestoreGlobals snapshots all commonly-mutated package-level globals
// and registers a t.Cleanup() to restore them when the test completes.
func saveAndRestoreGlobals(t *testing.T) *savedGlobals {
	t.Helper()
	saved := &savedGlobals{
		dbPath:      dbPath,
		store:       store,
		storeActive: storeActive,
	}
	t.Cleanup(func() {
		dbPath = saved.dbPath
		store = saved.store
		storeMutex.Lock()
		storeActive = saved.storeActive
		storeMutex.Unlock()
	})
	return saved
}

// newTestStore creates an embedded dolt store with issue_prefix configured.
func newTestStore(t *testing.T, dbPath string) *embeddeddolt.EmbeddedDoltStore {
	t.Helper()
	return newTestStoreWithPrefix(t, dbPath, "test")
}

// newTestStoreIsolatedDB creates an embedded dolt store with its own dedicated database.
func newTestStoreIsolatedDB(t *testing.T, dbPath string, prefix string) *embeddeddolt.EmbeddedDoltStore {
	t.Helper()
	return newTestStoreWithPrefix(t, dbPath, prefix)
}

// newTestStoreWithPrefix creates an embedded dolt store with custom issue_prefix configured.
func newTestStoreWithPrefix(t *testing.T, dbPath string, prefix string) *embeddeddolt.EmbeddedDoltStore {
	t.Helper()

	ensureTestMode(t)

	ctx := context.Background()

	// Derive bdDir from dbPath (parent directory)
	bdDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(bdDir, 0755); err != nil {
		t.Fatalf("Failed to create bdDir: %v", err)
	}

	database := uniqueTestDBName(t)

	doltNewMutex.Lock()
	s, err := embeddeddolt.New(ctx, bdDir, database, "main")
	doltNewMutex.Unlock()
	if err != nil {
		t.Fatalf("Failed to create embedded dolt store: %v", err)
	}

	if err := s.SetConfig(ctx, "issue_prefix", prefix); err != nil {
		s.Close()
		t.Fatalf("Failed to set issue_prefix: %v", err)
	}
	if err := s.SetConfig(ctx, "types.custom", "molecule,gate,convoy,merge-request,slot,agent,role,rig,event,message"); err != nil {
		s.Close()
		t.Fatalf("Failed to set types.custom: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})
	return s
}

// openExistingTestDB reopens an existing embedded dolt store for verification in tests.
func openExistingTestDB(t *testing.T, dbPath string) (*embeddeddolt.EmbeddedDoltStore, error) {
	t.Helper()
	doltNewMutex.Lock()
	defer doltNewMutex.Unlock()
	ctx := context.Background()
	bdDir := filepath.Dir(dbPath)
	database := uniqueTestDBName(t)
	return embeddeddolt.New(ctx, bdDir, database, "main")
}

// writeTestMetadata is a no-op stub; server-mode Dolt was removed.
// Kept for backward compatibility with tests that call it.
func writeTestMetadata(t *testing.T, dbPath string, database string) {
	t.Helper()
	// No-op: server-mode metadata.json is no longer needed.
}

// dropTestDatabase is a no-op stub; server-mode Dolt was removed.
func dropTestDatabase(dbName string, port int) {
	// No-op: server-mode database cleanup is no longer needed.
}

// runCommandInDir runs a command in the specified directory
func runCommandInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = testEnvNoPrompt()
	return cmd.Run()
}

// runCommandInDirWithOutput runs a command in the specified directory and returns its output
func runCommandInDirWithOutput(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = testEnvNoPrompt()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// testEnvNoPrompt returns the current environment with git auth prompts
// suppressed. Prevents ksshaskpass/SSH_ASKPASS popups during tests that
// configure fake git remotes (e.g. github.com/test/repo.git).
func testEnvNoPrompt() []string {
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0", "SSH_ASKPASS=", "GIT_ASKPASS=")
	return env
}

// captureStderr captures stderr output from fn and returns it as a string.
// Uses stdioMutex to prevent races with concurrent os.Stderr redirection.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	stdioMutex.Lock()
	defer stdioMutex.Unlock()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()
	_ = w.Close()
	os.Stderr = old
	<-done
	_ = r.Close()

	return buf.String()
}
