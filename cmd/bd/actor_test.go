package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestGetActorWithGit tests the actor resolution fallback chain.
// Priority: --actor flag > BD_ACTOR env > git config user.name > $USER > "unknown"
func TestGetActorWithGit(t *testing.T) {
	// Save original environment and actor variable
	origActor := actor
	origBdActor, bdActorSet := os.LookupEnv("BD_ACTOR")
	origUser, userSet := os.LookupEnv("USER")

	// Cleanup after test
	defer func() {
		actor = origActor
		if bdActorSet {
			os.Setenv("BD_ACTOR", origBdActor)
		} else {
			os.Unsetenv("BD_ACTOR")
		}
		if userSet {
			os.Setenv("USER", origUser)
		} else {
			os.Unsetenv("USER")
		}
	}()

	// Helper to get current git user.name (may be empty if not configured)
	getGitUserName := func() string {
		out, err := exec.Command("git", "config", "user.name").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}

	gitUserName := getGitUserName()

	tests := []struct {
		name        string
		actorFlag   string
		bdActor     string
		user        string
		expected    string
		skipIfNoGit bool // Skip if git user.name is not configured
	}{
		{
			name:      "actor flag takes priority",
			actorFlag: "flag-actor",
			bdActor:   "bd-actor",
			user:      "system-user",
			expected:  "flag-actor",
		},
		{
			name:      "BD_ACTOR takes priority when no flag",
			actorFlag: "",
			bdActor:   "bd-actor",
			user:      "system-user",
			expected:  "bd-actor",
		},
		{
			name:        "git config user.name used when no env vars",
			actorFlag:   "",
			bdActor:     "",
			user:        "system-user",
			expected:    gitUserName, // Will be git user.name if configured
			skipIfNoGit: true,
		},
		{
			name:      "USER fallback when no git config",
			actorFlag: "",
			bdActor:   "",
			user:      "fallback-user",
			expected:  "fallback-user",
			// Note: This test may fail if git user.name is configured
			// We handle this by checking the actual git config in the test
		},
		{
			name:      "unknown as final fallback",
			actorFlag: "",
			bdActor:   "",
			user:      "",
			expected:  "unknown",
			// Note: This test may get git user.name instead if configured
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require git user.name to not be configured
			if tt.skipIfNoGit && gitUserName == "" {
				t.Skip("Skipping: git config user.name is not configured")
			}

			// For tests expecting USER or unknown, skip if git user.name is configured
			// because git takes priority over USER
			if (tt.expected == tt.user || tt.expected == "unknown") && gitUserName != "" && tt.bdActor == "" && tt.actorFlag == "" {
				t.Skipf("Skipping: git config user.name (%s) takes priority over expected %s", gitUserName, tt.expected)
			}

			// Set up test environment
			actor = tt.actorFlag

			if tt.bdActor != "" {
				os.Setenv("BD_ACTOR", tt.bdActor)
			} else {
				os.Unsetenv("BD_ACTOR")
			}

			if tt.user != "" {
				os.Setenv("USER", tt.user)
			} else {
				os.Unsetenv("USER")
			}

			// Call the function
			result := getActorWithGit()

			// Check result
			if result != tt.expected {
				t.Errorf("getActorWithGit() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGetActorWithGit_PriorityOrder tests that the priority order is respected
func TestGetActorWithGit_PriorityOrder(t *testing.T) {
	// Save original state
	origActor := actor
	origBdActor, bdActorSet := os.LookupEnv("BD_ACTOR")

	defer func() {
		actor = origActor
		if bdActorSet {
			os.Setenv("BD_ACTOR", origBdActor)
		} else {
			os.Unsetenv("BD_ACTOR")
		}
	}()

	// Test: flag > BD_ACTOR
	actor = "from-flag"
	os.Setenv("BD_ACTOR", "from-bd-actor")

	result := getActorWithGit()
	if result != "from-flag" {
		t.Errorf("Expected flag to take priority, got %q", result)
	}

	// Test: BD_ACTOR when no flag
	actor = ""
	result = getActorWithGit()
	if result != "from-bd-actor" {
		t.Errorf("Expected BD_ACTOR to be used, got %q", result)
	}

	// Test: falls through when BD_ACTOR is unset
	os.Unsetenv("BD_ACTOR")
	result = getActorWithGit()
	// Should get git user.name, USER, or "unknown" -- just verify it's not empty
	if result == "" {
		t.Error("Expected non-empty result when BD_ACTOR is unset")
	}
}
