//go:build cgo

package embeddeddolt_test

import (
	"strings"
	"testing"

	"github.com/signalnine/bd/internal/types"
)

// TestRemoveLabel covers the RemoveLabel code path. BUG-cr6:
// `bd label remove` on a nonexistent label used to report success because the
// DELETE statement silently affected 0 rows and an EventLabelRemoved was still
// recorded. RemoveLabel must return an error when no label row matches, and
// no spurious "label_removed" event should be written.
func TestRemoveLabel(t *testing.T) {
	skipUnlessEmbeddedDolt(t)

	t.Run("removes_existing_label", func(t *testing.T) {
		te := newTestEnv(t, "rl")
		ctx := t.Context()

		issue := &types.Issue{ID: "rl-a", Title: "A", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
		if err := te.store.CreateIssue(ctx, issue, "tester"); err != nil {
			t.Fatalf("CreateIssue: %v", err)
		}
		if err := te.store.AddLabel(ctx, "rl-a", "bug", "tester"); err != nil {
			t.Fatalf("AddLabel: %v", err)
		}
		if err := te.store.RemoveLabel(ctx, "rl-a", "bug", "tester"); err != nil {
			t.Fatalf("RemoveLabel of existing label: %v", err)
		}
		te.assertLabelCount(t, ctx, "labels", "rl-a", 0)
		// One add + one remove event expected.
		te.assertEventCount(t, ctx, "events", "rl-a", string(types.EventLabelAdded), 1)
		te.assertEventCount(t, ctx, "events", "rl-a", string(types.EventLabelRemoved), 1)
	})

	t.Run("nonexistent_label_errors", func(t *testing.T) {
		te := newTestEnv(t, "nl")
		ctx := t.Context()

		issue := &types.Issue{ID: "nl-a", Title: "A", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
		if err := te.store.CreateIssue(ctx, issue, "tester"); err != nil {
			t.Fatalf("CreateIssue: %v", err)
		}
		// Issue has no labels at all.
		err := te.store.RemoveLabel(ctx, "nl-a", "ghost", "tester")
		if err == nil {
			t.Fatal("expected error for RemoveLabel on nonexistent label, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
		// No spurious label_removed event should be recorded.
		te.assertEventCount(t, ctx, "events", "nl-a", string(types.EventLabelRemoved), 0)
	})

	t.Run("nonexistent_label_with_other_labels_errors", func(t *testing.T) {
		te := newTestEnv(t, "ol")
		ctx := t.Context()

		issue := &types.Issue{ID: "ol-a", Title: "A", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
		if err := te.store.CreateIssue(ctx, issue, "tester"); err != nil {
			t.Fatalf("CreateIssue: %v", err)
		}
		if err := te.store.AddLabel(ctx, "ol-a", "real", "tester"); err != nil {
			t.Fatalf("AddLabel: %v", err)
		}
		err := te.store.RemoveLabel(ctx, "ol-a", "ghost", "tester")
		if err == nil {
			t.Fatal("expected error for RemoveLabel on nonexistent label, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
		// The real label should still be present.
		te.assertLabelCount(t, ctx, "labels", "ol-a", 1)
		// Only the add event was real; no remove event for the ghost.
		te.assertEventCount(t, ctx, "events", "ol-a", string(types.EventLabelRemoved), 0)
	})

	t.Run("double_remove_errors", func(t *testing.T) {
		te := newTestEnv(t, "dl")
		ctx := t.Context()

		issue := &types.Issue{ID: "dl-a", Title: "A", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
		if err := te.store.CreateIssue(ctx, issue, "tester"); err != nil {
			t.Fatalf("CreateIssue: %v", err)
		}
		if err := te.store.AddLabel(ctx, "dl-a", "bug", "tester"); err != nil {
			t.Fatalf("AddLabel: %v", err)
		}
		if err := te.store.RemoveLabel(ctx, "dl-a", "bug", "tester"); err != nil {
			t.Fatalf("first RemoveLabel: %v", err)
		}
		// Second remove: label already gone, must error.
		err := te.store.RemoveLabel(ctx, "dl-a", "bug", "tester")
		if err == nil {
			t.Fatal("expected error on second RemoveLabel, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
		// One add event, exactly one remove event (not two).
		te.assertEventCount(t, ctx, "events", "dl-a", string(types.EventLabelRemoved), 1)
	})
}
