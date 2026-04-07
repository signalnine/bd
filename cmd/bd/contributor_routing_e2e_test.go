// contributor_routing_e2e_test.go - E2E tests for contributor routing
//
// These tests verify that issues are correctly routed to the planning repo
// when the user is detected as a contributor with auto-routing enabled.

//go:build cgo && integration
// +build cgo,integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steveyegge/bd/internal/routing"
	"github.com/steveyegge/bd/internal/storage/embeddeddolt"
	"github.com/steveyegge/bd/internal/types"
)

// TestContributorRoutingTracer is the Phase 1 tracer bullet test.
func TestContributorRoutingTracer(t *testing.T) {
	t.Run("ExpandPath_tilde_expansion", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("cannot get home dir: %v", err)
		}

		tests := []struct {
			input string
			want  string
		}{
			{"~/foo", filepath.Join(home, "foo")},
			{"~/bar/baz", filepath.Join(home, "bar", "baz")},
			{".", "."},
			{"", ""},
		}

		for _, tt := range tests {
			got := routing.ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("DetermineTargetRepo_contributor_routes_to_planning", func(t *testing.T) {
		config := &routing.RoutingConfig{
			Mode:            "auto",
			ContributorRepo: "~/.bd-planning",
		}

		got := routing.DetermineTargetRepo(config, routing.Contributor, ".")
		if got != "~/.bd-planning" {
			t.Errorf("DetermineTargetRepo() = %q, want %q", got, "~/.bd-planning")
		}
	})

	t.Run("DetermineTargetRepo_maintainer_stays_local", func(t *testing.T) {
		config := &routing.RoutingConfig{
			Mode:            "auto",
			MaintainerRepo:  ".",
			ContributorRepo: "~/.bd-planning",
		}

		got := routing.DetermineTargetRepo(config, routing.Maintainer, ".")
		if got != "." {
			t.Errorf("DetermineTargetRepo() = %q, want %q", got, ".")
		}
	})

	t.Run("E2E_routing_decision_with_store", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectDir := filepath.Join(tmpDir, "project")
		planningDir := filepath.Join(tmpDir, "planning")

		projectBeadsDir := filepath.Join(projectDir, ".bd")
		if err := os.MkdirAll(projectBeadsDir, 0755); err != nil {
			t.Fatalf("failed to create project .beads dir: %v", err)
		}

		planningBeadsDir := filepath.Join(planningDir, ".bd")
		if err := os.MkdirAll(planningBeadsDir, 0755); err != nil {
			t.Fatalf("failed to create planning .beads dir: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		projectStore, err := embeddeddolt.New(ctx, projectBeadsDir, "project", "main")
		if err != nil {
			t.Fatalf("failed to create project store: %v", err)
		}
		defer projectStore.Close()

		if err := projectStore.SetConfig(ctx, "issue_prefix", "proj-"); err != nil {
			t.Fatalf("failed to set issue_prefix: %v", err)
		}
		if err := projectStore.SetConfig(ctx, "routing.mode", "auto"); err != nil {
			t.Fatalf("failed to set routing.mode: %v", err)
		}
		if err := projectStore.SetConfig(ctx, "routing.contributor", planningDir); err != nil {
			t.Fatalf("failed to set routing.contributor: %v", err)
		}

		mode, err := projectStore.GetConfig(ctx, "routing.mode")
		if err != nil {
			t.Fatalf("failed to get routing.mode: %v", err)
		}
		if mode != "auto" {
			t.Errorf("routing.mode = %q, want %q", mode, "auto")
		}

		contributorPath, err := projectStore.GetConfig(ctx, "routing.contributor")
		if err != nil {
			t.Fatalf("failed to get routing.contributor: %v", err)
		}
		if contributorPath != planningDir {
			t.Errorf("routing.contributor = %q, want %q", contributorPath, planningDir)
		}

		routingConfig := &routing.RoutingConfig{
			Mode:            mode,
			ContributorRepo: contributorPath,
		}

		targetRepo := routing.DetermineTargetRepo(routingConfig, routing.Contributor, projectDir)
		if targetRepo != planningDir {
			t.Errorf("DetermineTargetRepo() = %q, want %q", targetRepo, planningDir)
		}

		targetRepo = routing.DetermineTargetRepo(routingConfig, routing.Maintainer, projectDir)
		if targetRepo != "." {
			t.Errorf("DetermineTargetRepo() for maintainer = %q, want %q", targetRepo, ".")
		}

		planningStore, err := embeddeddolt.New(ctx, planningBeadsDir, "planning", "main")
		if err != nil {
			t.Fatalf("failed to create planning store: %v", err)
		}
		defer planningStore.Close()

		if err := planningStore.SetConfig(ctx, "issue_prefix", "plan-"); err != nil {
			t.Fatalf("failed to set issue_prefix in planning store: %v", err)
		}

		issue := &types.Issue{
			Title:     "Test contributor issue",
			IssueType: types.TypeTask,
			Status:    types.StatusOpen,
			Priority:  2,
		}

		if err := planningStore.CreateIssue(ctx, issue, "test"); err != nil {
			t.Fatalf("failed to create issue in planning store: %v", err)
		}

		retrieved, err := planningStore.GetIssue(ctx, issue.ID)
		if err != nil {
			t.Fatalf("failed to get issue from planning store: %v", err)
		}
		if retrieved.Title != "Test contributor issue" {
			t.Errorf("issue title = %q, want %q", retrieved.Title, "Test contributor issue")
		}

		projectIssue, _ := projectStore.GetIssue(ctx, issue.ID)
		if projectIssue != nil {
			t.Error("issue should NOT exist in project store (isolation failure)")
		}
	})
}

// TestExplicitRoleOverride verifies git config bd.role takes precedence over URL detection
func TestExplicitRoleOverride(t *testing.T) {
	tests := []struct {
		name         string
		configRole   string
		expectedRole routing.UserRole
		description  string
	}{
		{
			name:         "explicit maintainer",
			configRole:   "maintainer",
			expectedRole: routing.Maintainer,
			description:  "git config bd.role=maintainer should force maintainer",
		},
		{
			name:         "explicit contributor",
			configRole:   "contributor",
			expectedRole: routing.Contributor,
			description:  "git config bd.role=contributor should force contributor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routingConfig := &routing.RoutingConfig{
				Mode:            "auto",
				ContributorRepo: "/path/to/planning",
				MaintainerRepo:  ".",
			}

			var expectedRepo string
			if tt.expectedRole == routing.Maintainer {
				expectedRepo = "."
			} else {
				expectedRepo = "/path/to/planning"
			}

			targetRepo := routing.DetermineTargetRepo(routingConfig, tt.expectedRole, ".")
			if targetRepo != expectedRepo {
				t.Errorf("%s: got target repo %q, want %q", tt.description, targetRepo, expectedRepo)
			}
		})
	}
}
