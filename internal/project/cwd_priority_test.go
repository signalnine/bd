package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestFindBeadsDir_CwdPriority verifies that a .bd/ directory in cwd takes
// priority over a .bd/ directory at the git worktree root.
//
// Scenario: A "rig" subdirectory has its own .bd/ inside a git worktree
// that also has .bd/ at its root. Before this fix, step 2b
// (git.GetRepoRoot → check .bd/) fired before the cwd walk, grabbing
// the worktree root's .bd/ instead of the rig's local one.
func TestFindBeadsDir_CwdPriority(t *testing.T) {
	// Save and restore env
	origBeadsDir := os.Getenv("BD_DIR")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
	})
	os.Unsetenv("BD_DIR")

	tmpDir := t.TempDir()

	// Create a git repo (simulating the worktree root)
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .bd/ with project files (the "wrong" one)
	rootBeadsDir := filepath.Join(tmpDir, ".bd")
	if err := os.MkdirAll(rootBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "config.yaml"), []byte("issue_prefix: root\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a rig subdirectory with its own .bd/ (the "right" one)
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigBeadsDir := filepath.Join(rigDir, ".bd")
	if err := os.MkdirAll(rigBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"rig_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigBeadsDir, "config.yaml"), []byte("issue_prefix: rig\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// cd into the rig directory
	t.Chdir(rigDir)

	result := FindBdDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rigBeadsDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindBdDir() = %q, want %q (rig's .beads should win over root's)", result, rigBeadsDir)
	}
}

// TestFindDatabasePath_CwdPriority verifies FindDatabasePath (the database
// discovery path) also prefers cwd's .bd/ over the git worktree root's.
func TestFindDatabasePath_CwdPriority(t *testing.T) {
	origBeadsDir := os.Getenv("BD_DIR")
	origBeadsDB := os.Getenv("BD_DB")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
		if origBeadsDB != "" {
			os.Setenv("BD_DB", origBeadsDB)
		} else {
			os.Unsetenv("BD_DB")
		}
	})
	os.Unsetenv("BD_DIR")
	os.Unsetenv("BD_DB")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .bd/ with a dolt dir (the "wrong" one)
	rootBeadsDir := filepath.Join(tmpDir, ".bd")
	rootDoltDir := filepath.Join(rootBeadsDir, "dolt")
	if err := os.MkdirAll(rootDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig subdirectory with its own .bd/ and dolt dir
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigBeadsDir := filepath.Join(rigDir, ".bd")
	rigDoltDir := filepath.Join(rigBeadsDir, "dolt")
	if err := os.MkdirAll(rigDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"rig_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// cd into the rig directory
	t.Chdir(rigDir)

	result := FindDatabasePath()

	// The database path should be under the rig's .bd/, not the root's
	if result == "" {
		t.Fatal("FindDatabasePath() returned empty, expected rig's database path")
	}

	resultResolved, _ := filepath.EvalSymlinks(result)
	rigDirResolved, _ := filepath.EvalSymlinks(rigDir)
	rootBeadsDirResolved, _ := filepath.EvalSymlinks(rootBeadsDir)
	if !isUnder(resultResolved, rigDirResolved) {
		t.Errorf("FindDatabasePath() = %q, want path under %q (rig's .beads should win)", result, rigDirResolved)
	}
	if isUnder(resultResolved, rootBeadsDirResolved) {
		t.Errorf("FindDatabasePath() = %q, should NOT be under root's .beads %q", result, rootBeadsDir)
	}
}

// TestFindBeadsDir_CwdWithoutBeads_FallsBackToWalk verifies that when cwd
// has no .bd/, the normal walk-up behavior still works.
func TestFindBeadsDir_CwdWithoutBeads_FallsBackToWalk(t *testing.T) {
	origBeadsDir := os.Getenv("BD_DIR")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
	})
	os.Unsetenv("BD_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .bd/ only (no rig-level .bd/)
	rootBeadsDir := filepath.Join(tmpDir, ".bd")
	if err := os.MkdirAll(rootBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory WITHOUT .bd/
	subDir := filepath.Join(tmpDir, "some", "deep", "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)

	result := FindBdDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rootBeadsDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindBdDir() = %q, want %q (should fall back to root when cwd has no .bd/)", result, rootBeadsDir)
	}
}

// TestFindBeadsDir_CwdBeadsDirWithRedirect verifies that cwd's .bd/
// redirect is followed when the cwd check fires.
func TestFindBeadsDir_CwdBeadsDirWithRedirect(t *testing.T) {
	origBeadsDir := os.Getenv("BD_DIR")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
	})
	os.Unsetenv("BD_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .bd/
	rootBeadsDir := filepath.Join(tmpDir, ".bd")
	if err := os.MkdirAll(rootBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a redirect target
	targetBeadsDir := filepath.Join(tmpDir, "shared-beads")
	if err := os.MkdirAll(targetBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"shared_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig subdirectory with .bd/ that has a redirect
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigBeadsDir := filepath.Join(rigDir, ".bd")
	if err := os.MkdirAll(rigBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write redirect file pointing to the shared target
	if err := os.WriteFile(filepath.Join(rigBeadsDir, "redirect"), []byte(targetBeadsDir), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(rigDir)

	result := FindBdDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(targetBeadsDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindBdDir() = %q, want %q (cwd .bd/ redirect should be followed)", result, targetBeadsDir)
	}
}

// TestFindBeadsDir_BEADS_DIR_StillTakesPriority verifies that BD_DIR env
// var still takes priority over the cwd check.
func TestFindBeadsDir_BEADS_DIR_StillTakesPriority(t *testing.T) {
	origBeadsDir := os.Getenv("BD_DIR")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
	})

	tmpDir := t.TempDir()

	// Create an explicit BD_DIR target
	explicitBeadsDir := filepath.Join(tmpDir, "explicit-beads")
	if err := os.MkdirAll(explicitBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(explicitBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"explicit_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("BD_DIR", explicitBeadsDir)

	// Create cwd with its own .bd/
	cwdDir := filepath.Join(tmpDir, "cwd-project")
	cwdBeadsDir := filepath.Join(cwdDir, ".bd")
	if err := os.MkdirAll(cwdBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwdBeadsDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"cwd_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(cwdDir)

	result := FindBdDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(explicitBeadsDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindBdDir() = %q, want %q (BD_DIR should still take priority over cwd)", result, explicitBeadsDir)
	}
}

// TestFindBeadsDir_CwdEmptyBeadsDir_SkipsToCwdWalk verifies that when cwd
// has a .bd/ directory without any project files, it's skipped and the
// normal walk-up behavior continues.
func TestFindBeadsDir_CwdEmptyBeadsDir_SkipsToCwdWalk(t *testing.T) {
	origBeadsDir := os.Getenv("BD_DIR")
	t.Cleanup(func() {
		if origBeadsDir != "" {
			os.Setenv("BD_DIR", origBeadsDir)
		} else {
			os.Unsetenv("BD_DIR")
		}
	})
	os.Unsetenv("BD_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .bd/ with project files
	rootBeadsDir := filepath.Join(tmpDir, ".bd")
	if err := os.MkdirAll(rootBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootBeadsDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig with empty .bd/ (no project files)
	rigDir := filepath.Join(tmpDir, "empty-rig")
	rigBeadsDir := filepath.Join(rigDir, ".bd")
	if err := os.MkdirAll(rigBeadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No metadata.json, no config.yaml, no dolt/ — empty dir

	t.Chdir(rigDir)

	result := FindBdDir()

	// Should fall through to the root's .bd/ since rig's is empty
	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rootBeadsDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindBdDir() = %q, want %q (empty cwd .bd/ should be skipped)", result, rootBeadsDir)
	}
}

// isUnder returns true if child is under parent in the directory tree.
func isUnder(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// rel should not start with ".." (going up) and should not be absolute
	return !filepath.IsAbs(rel) && (rel == "." || (len(rel) >= 2 && rel[:2] != ".."))
}
