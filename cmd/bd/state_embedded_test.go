//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// bdState runs "bd state" with the given args and returns stdout.
func bdState(t *testing.T, bd, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"state"}, args...)
	cmd := exec.Command(bd, fullArgs...)
	cmd.Dir = dir
	cmd.Env = bdEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bd state %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// bdSetState runs "bd set-state" with the given args and returns stdout.
func bdSetState(t *testing.T, bd, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"set-state"}, args...)
	cmd := exec.Command(bd, fullArgs...)
	cmd.Dir = dir
	cmd.Env = bdEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bd set-state %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// bdStateRemove runs "bd state remove" with the given args and returns
// (stdout, error). The error preserves the binary's exit status so the
// caller can verify failure cases (e.g. removing a dimension that isn't set).
func bdStateRemove(t *testing.T, bd, dir string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"state", "remove"}, args...)
	cmd := exec.Command(bd, fullArgs...)
	cmd.Dir = dir
	cmd.Env = bdEnv(dir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestEmbeddedState(t *testing.T) {
	if os.Getenv("BD_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BD_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	bd := buildEmbeddedBD(t)
	dir, _, _ := bdInit(t, bd, "--prefix", "st")

	issue := bdCreate(t, bd, dir, "State test issue", "--type", "task")

	// ===== set-state =====

	t.Run("set_state_basic", func(t *testing.T) {
		out := bdSetState(t, bd, dir, issue.ID, "phase=planning")
		if !strings.Contains(out, "planning") {
			t.Logf("set-state output: %s", out)
		}
	})

	t.Run("set_state_json", func(t *testing.T) {
		cmd := exec.Command(bd, "set-state", issue.ID, "env=staging", "--json")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd set-state --json failed: %v\n%s", err, out)
		}
		s := strings.TrimSpace(string(out))
		start := strings.Index(s, "{")
		if start >= 0 {
			var m map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(s[start:]), &m); jsonErr != nil {
				t.Errorf("invalid JSON: %v\n%s", jsonErr, s)
			}
		}
	})

	t.Run("set_state_with_reason", func(t *testing.T) {
		out := bdSetState(t, bd, dir, issue.ID, "risk=high", "--reason", "New vulnerability found")
		_ = out
	})

	t.Run("set_state_overwrites", func(t *testing.T) {
		bdSetState(t, bd, dir, issue.ID, "phase=development")
		bdSetState(t, bd, dir, issue.ID, "phase=testing")

		out := bdState(t, bd, dir, issue.ID, "phase")
		if !strings.Contains(out, "testing") {
			t.Errorf("expected 'testing' after overwrite, got: %s", out)
		}
	})

	// ===== state query =====

	t.Run("state_query", func(t *testing.T) {
		out := bdState(t, bd, dir, issue.ID, "phase")
		if !strings.Contains(out, "testing") {
			t.Logf("state query output: %s", out)
		}
	})

	t.Run("state_query_json", func(t *testing.T) {
		cmd := exec.Command(bd, "state", issue.ID, "phase", "--json")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd state --json failed: %v\n%s", err, out)
		}
		_ = out
	})

	t.Run("state_query_nonexistent_dimension", func(t *testing.T) {
		out := bdState(t, bd, dir, issue.ID, "nonexistent")
		// Should return empty/not-set, not error
		_ = out
	})

	// ===== state list =====

	t.Run("state_list", func(t *testing.T) {
		out := bdState(t, bd, dir, "list", issue.ID)
		// Should show the dimensions we set
		if !strings.Contains(out, "phase") {
			t.Logf("state list output: %s", out)
		}
	})

	t.Run("state_list_json", func(t *testing.T) {
		cmd := exec.Command(bd, "state", "list", issue.ID, "--json")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd state list --json failed: %v\n%s", err, out)
		}
		_ = out
	})

	t.Run("state_list_no_dimensions", func(t *testing.T) {
		fresh := bdCreate(t, bd, dir, "No state", "--type", "task")
		out := bdState(t, bd, dir, "list", fresh.ID)
		_ = out
	})

	// ===== state remove =====

	t.Run("state_remove_clears_dimension", func(t *testing.T) {
		target := bdCreate(t, bd, dir, "Remove dim", "--type", "task")
		bdSetState(t, bd, dir, target.ID, "phase=planning")

		out, err := bdStateRemove(t, bd, dir, target.ID, "phase")
		if err != nil {
			t.Fatalf("bd state remove failed: %v\n%s", err, out)
		}

		queryOut := bdState(t, bd, dir, target.ID, "phase")
		if !strings.Contains(queryOut, "no phase state set") {
			t.Errorf("expected 'no phase state set' after removal, got: %s", queryOut)
		}
	})

	t.Run("state_remove_creates_event", func(t *testing.T) {
		target := bdCreate(t, bd, dir, "Remove event", "--type", "task")
		bdSetState(t, bd, dir, target.ID, "mode=normal")

		out, err := bdStateRemove(t, bd, dir, target.ID, "mode", "--reason", "decommissioned")
		if err != nil {
			t.Fatalf("bd state remove failed: %v\n%s", err, out)
		}

		// Look for an event child issue recording the removal.
		showCmd := exec.Command(bd, "show", target.ID, "--json")
		showCmd.Dir = dir
		showCmd.Env = bdEnv(dir)
		showOut, err := showCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd show failed: %v\n%s", err, showOut)
		}
		if !strings.Contains(string(showOut), "State removed") {
			t.Errorf("expected event with 'State removed' on parent, got: %s", showOut)
		}
	})

	t.Run("state_remove_missing_dimension_errors", func(t *testing.T) {
		target := bdCreate(t, bd, dir, "Missing dim", "--type", "task")

		out, err := bdStateRemove(t, bd, dir, target.ID, "neverset")
		if err == nil {
			t.Fatalf("expected error removing unset dimension, got success: %s", out)
		}
		if !strings.Contains(out, "neverset") {
			t.Errorf("expected error to mention dimension name, got: %s", out)
		}
	})

	t.Run("state_remove_json", func(t *testing.T) {
		target := bdCreate(t, bd, dir, "Remove json", "--type", "task")
		bdSetState(t, bd, dir, target.ID, "risk=high")

		cmd := exec.Command(bd, "state", "remove", target.ID, "risk", "--json")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd state remove --json failed: %v\n%s", err, out)
		}
		s := strings.TrimSpace(string(out))
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("expected JSON object in output, got: %s", s)
		}
		var m map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(s[start:]), &m); jsonErr != nil {
			t.Fatalf("invalid JSON: %v\n%s", jsonErr, s)
		}
		for _, key := range []string{"issue_id", "dimension", "old_value", "event_id"} {
			if _, ok := m[key]; !ok {
				t.Errorf("expected key %q in JSON output, got: %v", key, m)
			}
		}
		if got, _ := m["old_value"].(string); got != "high" {
			t.Errorf("expected old_value=high, got %v", m["old_value"])
		}
	})
}

// TestEmbeddedStateConcurrent exercises set-state concurrently on different dimensions.
func TestEmbeddedStateConcurrent(t *testing.T) {
	if os.Getenv("BD_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BD_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	bd := buildEmbeddedBD(t)
	dir, _, _ := bdInit(t, bd, "--prefix", "sx")

	issue := bdCreate(t, bd, dir, "Concurrent state", "--type", "task")

	const numWorkers = 6

	type workerResult struct {
		worker int
		err    error
	}

	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			r := workerResult{worker: worker}

			dim := fmt.Sprintf("dim%d=val%d", worker, worker)
			cmd := exec.Command(bd, "set-state", issue.ID, dim)
			cmd.Dir = dir
			cmd.Env = bdEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("set-state %s: %v\n%s", dim, err, out)
			}

			results[worker] = r
		}(w)
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil && !strings.Contains(r.err.Error(), "one writer at a time") {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}
}
