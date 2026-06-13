//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/signalnine/bd/internal/types"
)

// bdBlockedJSON runs "bd blocked --json" and parses the result into a slice of
// BlockedIssue. Retries on flock contention.
func bdBlockedJSON(t *testing.T, bd, dir string, args ...string) []types.BlockedIssue {
	t.Helper()
	fullArgs := append([]string{"blocked", "--json"}, args...)
	out, err := bdRunWithFlockRetry(t, bd, dir, fullArgs...)
	if err != nil {
		t.Fatalf("bd blocked --json failed: %v\n%s", err, out)
	}
	s := strings.TrimSpace(string(out))
	start := strings.IndexAny(s, "[{")
	if start < 0 {
		t.Fatalf("no JSON in blocked output: %s", s)
	}
	var blocked []types.BlockedIssue
	if err := json.Unmarshal([]byte(s[start:]), &blocked); err != nil {
		t.Fatalf("parse blocked JSON: %v\n%s", err, s[start:])
	}
	return blocked
}

func TestEmbeddedBlocked(t *testing.T) {
	if os.Getenv("BD_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BD_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	bd := buildEmbeddedBD(t)
	dir, _, _ := bdInit(t, bd, "--prefix", "bl")

	// ===== Empty: no blocked issues =====

	t.Run("empty_text", func(t *testing.T) {
		// An open issue with no blockers must not count as blocked.
		bdCreate(t, bd, dir, "Standalone issue", "--type", "task")
		cmd := exec.Command(bd, "blocked")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd blocked failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "No blocked issues") {
			t.Errorf("expected 'No blocked issues', got: %s", out)
		}
	})

	t.Run("empty_json", func(t *testing.T) {
		blocked := bdBlockedJSON(t, bd, dir)
		if len(blocked) != 0 {
			t.Errorf("expected empty blocked list, got %d: %+v", len(blocked), blocked)
		}
	})

	// ===== Single blocked issue with one blocker =====

	var blockerID, blockedID string
	t.Run("single", func(t *testing.T) {
		blockerID = bdCreate(t, bd, dir, "First blocker", "--type", "task").ID
		blockedID = bdCreate(t, bd, dir, "Waiting on first blocker", "--type", "task").ID
		bdDepAdd(t, bd, dir, blockedID, blockerID)

		cmd := exec.Command(bd, "blocked")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd blocked failed: %v\n%s", err, out)
		}
		s := string(out)
		if !strings.Contains(s, "Waiting on first blocker") {
			t.Errorf("expected blocked issue in output: %s", s)
		}
		if !strings.Contains(s, "Blocked by 1 open dependencies") {
			t.Errorf("expected blocker count in output: %s", s)
		}
		// The blocker itself is open and unblocked: it must not be listed.
		if strings.Contains(s, "First blocker:") {
			t.Errorf("blocker should not appear as a blocked issue: %s", s)
		}
	})

	t.Run("single_json", func(t *testing.T) {
		blocked := bdBlockedJSON(t, bd, dir)
		if len(blocked) != 1 {
			t.Fatalf("expected 1 blocked issue, got %d: %+v", len(blocked), blocked)
		}
		b := blocked[0]
		if b.ID != blockedID {
			t.Errorf("expected blocked ID %s, got %s", blockedID, b.ID)
		}
		if b.BlockedByCount != 1 {
			t.Errorf("expected blocked_by_count 1, got %d", b.BlockedByCount)
		}
		if len(b.BlockedBy) != 1 || b.BlockedBy[0] != blockerID {
			t.Errorf("expected blocked_by [%s], got %v", blockerID, b.BlockedBy)
		}
	})

	// ===== Multiple blocked issues =====

	t.Run("multiple", func(t *testing.T) {
		blocker2 := bdCreate(t, bd, dir, "Second blocker", "--type", "task").ID
		blocked2 := bdCreate(t, bd, dir, "Waiting on second blocker", "--type", "task").ID
		bdDepAdd(t, bd, dir, blocked2, blocker2)

		blocked := bdBlockedJSON(t, bd, dir)
		if len(blocked) != 2 {
			t.Fatalf("expected 2 blocked issues, got %d: %+v", len(blocked), blocked)
		}
		ids := map[string]bool{}
		for _, b := range blocked {
			ids[b.ID] = true
			if b.BlockedByCount != 1 {
				t.Errorf("issue %s: expected blocked_by_count 1, got %d", b.ID, b.BlockedByCount)
			}
		}
		if !ids[blockedID] || !ids[blocked2] {
			t.Errorf("expected both blocked issues present, got ids %v", ids)
		}
	})

	// ===== Unblocking removes from the list =====

	t.Run("closing_blocker_clears", func(t *testing.T) {
		out, err := bdRunWithFlockRetry(t, bd, dir, "close", blockerID)
		if err != nil {
			t.Fatalf("bd close %s failed: %v\n%s", blockerID, err, out)
		}
		blocked := bdBlockedJSON(t, bd, dir)
		for _, b := range blocked {
			if b.ID == blockedID {
				t.Errorf("issue %s should no longer be blocked after closing %s: %+v", blockedID, blockerID, b)
			}
		}
	})
}
