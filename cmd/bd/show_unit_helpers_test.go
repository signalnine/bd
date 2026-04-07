//go:build cgo

package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/signalnine/bd/internal/types"
)

func TestValidateIssueUpdatable(t *testing.T) {
	if err := validateIssueUpdatable("x", nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := validateIssueUpdatable("x", &types.Issue{IsTemplate: false}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := validateIssueUpdatable("bd-1", &types.Issue{IsTemplate: true}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateIssueClosable(t *testing.T) {
	if err := validateIssueClosable("x", nil, false); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := validateIssueClosable("bd-1", &types.Issue{IsTemplate: true}, false); err == nil {
		t.Fatalf("expected template close error")
	}
	if err := validateIssueClosable("bd-2", &types.Issue{Status: types.StatusPinned}, false); err == nil {
		t.Fatalf("expected pinned close error")
	}
	if err := validateIssueClosable("bd-2", &types.Issue{Status: types.StatusPinned}, true); err != nil {
		t.Fatalf("expected pinned close to succeed with force, got %v", err)
	}
}

func TestApplyLabelUpdates_SetAddRemove(t *testing.T) {
	ctx := context.Background()
	st := newTestStoreWithPrefix(t, filepath.Join(t.TempDir(), "test.db"), "test")

	issue := &types.Issue{Title: "x", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
	if err := st.CreateIssue(ctx, issue, "tester"); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	_ = st.AddLabel(ctx, issue.ID, "old1", "tester")
	_ = st.AddLabel(ctx, issue.ID, "old2", "tester")

	if err := applyLabelUpdates(ctx, st, issue.ID, "tester", []string{"a", "b"}, []string{"b", "c"}, []string{"a"}); err != nil {
		t.Fatalf("applyLabelUpdates: %v", err)
	}
	labels, _ := st.GetLabels(ctx, issue.ID)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %v", labels)
	}
	// Order is not guaranteed.
	foundB := false
	foundC := false
	for _, l := range labels {
		if l == "b" {
			foundB = true
		}
		if l == "c" {
			foundC = true
		}
		if l == "old1" || l == "old2" || l == "a" {
			t.Fatalf("unexpected label %q in %v", l, labels)
		}
	}
	if !foundB || !foundC {
		t.Fatalf("expected labels b and c, got %v", labels)
	}
}

func TestApplyLabelUpdates_AddRemoveOnly(t *testing.T) {
	ctx := context.Background()
	st := newTestStoreWithPrefix(t, filepath.Join(t.TempDir(), "test.db"), "test")
	issue := &types.Issue{Title: "x", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask}
	if err := st.CreateIssue(ctx, issue, "tester"); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	_ = st.AddLabel(ctx, issue.ID, "a", "tester")
	if err := applyLabelUpdates(ctx, st, issue.ID, "tester", nil, []string{"b"}, []string{"a"}); err != nil {
		t.Fatalf("applyLabelUpdates: %v", err)
	}
	labels, _ := st.GetLabels(ctx, issue.ID)
	if len(labels) != 1 || labels[0] != "b" {
		t.Fatalf("expected [b], got %v", labels)
	}
}

// TestFindRepliesToAndReplies_WorksWithDoltStorage removed:
// findRepliesTo and findReplies were deleted in nuclear simplification.
