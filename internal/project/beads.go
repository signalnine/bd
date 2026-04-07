// Package beads provides a minimal public API for extending bd with custom orchestration.
//
// Most extensions should use direct SQL queries against bd's database.
// This package exports only the essential types and functions needed for
// Go-based extensions that want to use bd's storage layer programmatically.
//
// For detailed guidance on extending bd, see EXTENDING.md.
package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/steveyegge/bd/internal/configfile"
	"github.com/steveyegge/bd/internal/git"
	"github.com/steveyegge/bd/internal/storage"
	"github.com/steveyegge/bd/internal/utils"
)

// CanonicalDatabaseName is the required database filename for all beads repositories
const CanonicalDatabaseName = "bd.db"

// RedirectFileName is the name of the file that redirects to another .beads directory
const RedirectFileName = "redirect"

// SourceDatabaseInfo contains the dolt_database name from a source .bd/metadata.json,
// preserved across a redirect so that the source directory's database identity is not
// lost when the redirect target has a different dolt_database.
//
// When a .bd/redirect points to a shared .beads directory that serves multiple
// databases, the source's metadata.json may specify a different dolt_database than
// the target's. This struct captures the source database name so callers can
// restore it after redirect resolution.
type SourceDatabaseInfo struct {
	// SourceDir is the original .beads directory (before redirect)
	SourceDir string
	// TargetDir is the resolved .beads directory (after redirect)
	TargetDir string
	// WasRedirected is true if a redirect was followed
	WasRedirected bool
	// SourceDatabase is dolt_database from the source metadata.json (raw field,
	// NOT the env-var-aware GetDoltDatabase()). Empty if no source metadata exists
	// or the source has no dolt_database configured.
	SourceDatabase string
}

// ResolveRedirect follows a .bd/redirect file and captures the source directory's
// dolt_database from metadata.json BEFORE following the redirect. This preserves
// the source database identity across redirects.
//
// The env var BD_DOLT_SERVER_DATABASE still takes highest priority (handled by
// GetDoltDatabase() in callers). This function only captures the raw config field
// so callers can use it as an override when the env var is not set.
//
// Returns SourceDatabaseInfo with WasRedirected=true if a redirect was followed,
// and SourceDatabase set to the source's dolt_database (if any).
func ResolveRedirect(bdDir string) SourceDatabaseInfo {
	info := SourceDatabaseInfo{
		SourceDir: bdDir,
		TargetDir: bdDir,
	}

	// Read source metadata.json directly (NOT via configfile.Load which may trigger
	// Dolt connections or recursive FollowRedirect calls causing deadlocks).
	// We only need the raw dolt_database field.
	metadataPath := filepath.Join(bdDir, "metadata.json")
	if data, err := os.ReadFile(metadataPath); err == nil { //nolint:gosec // trusted internal path
		var raw struct {
			DoltDatabase string `json:"dolt_database"`
		}
		if json.Unmarshal(data, &raw) == nil {
			info.SourceDatabase = raw.DoltDatabase
		}
	}

	// Follow redirect
	resolved := FollowRedirect(bdDir)
	if resolved != bdDir {
		info.WasRedirected = true
		info.TargetDir = resolved
	}

	return info
}

// FollowRedirect checks if a .beads directory contains a redirect file and follows it.
// If a redirect file exists, it returns the target .beads directory path.
// If no redirect exists or there's an error, it returns the original path unchanged.
//
// The redirect file should contain a single path (relative or absolute) to the target
// .beads directory. Relative paths are resolved from the parent directory of the
// original .beads directory (i.e., the project root).
//
// Redirect chains are not followed - only one level of redirection is supported.
// This prevents infinite loops and keeps the behavior predictable.
func FollowRedirect(bdDir string) string {
	redirectFile := filepath.Join(bdDir, RedirectFileName)
	data, err := os.ReadFile(redirectFile) //nolint:gosec // trusted internal path
	if err != nil {
		// No redirect file or can't read it - use original path
		return bdDir
	}

	// Parse the redirect target (trim whitespace and handle comments)
	target := strings.TrimSpace(string(data))

	// Skip empty lines and comments to find the actual path
	lines := strings.Split(target, "\n")
	target = ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			target = line
			break
		}
	}

	if target == "" {
		return bdDir
	}

	// Resolve relative paths from the parent of the .beads directory (project root)
	if !filepath.IsAbs(target) {
		projectRoot := filepath.Dir(bdDir)
		target = filepath.Join(projectRoot, target)
	}

	// Canonicalize the target path and prefer a stable branch worktree when the
	// redirect points at a detached snapshot checkout.
	target = canonicalizeBeadsDirPath(target)

	// Verify the target exists and is a directory
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		// Invalid redirect target - fall back to original
		fmt.Fprintf(os.Stderr, "Warning: redirect target does not exist or is not a directory: %s\n", target)
		return bdDir
	}

	// Prevent redirect chains - don't follow if target also has a redirect
	targetRedirect := filepath.Join(target, RedirectFileName)
	if _, err := os.Stat(targetRedirect); err == nil {
		fmt.Fprintf(os.Stderr, "Warning: redirect chains not allowed, ignoring redirect in %s\n", target)
	}

	if os.Getenv("BD_DEBUG_ROUTING") != "" {
		fmt.Fprintf(os.Stderr, "[routing] Followed redirect from %s -> %s\n", bdDir, target)
	}

	return target
}

func canonicalizeBeadsDirPath(bdDir string) string {
	canonical := utils.CanonicalizePath(bdDir)
	if stable := preferStableBranchWorktreeBeadsDir(canonical); stable != "" {
		return stable
	}
	return canonical
}

type worktreeInfo struct {
	Path     string
	Head     string
	Branch   string
	Detached bool
	Bare     bool
}

func preferStableBranchWorktreeBeadsDir(bdDir string) string {
	if filepath.Base(bdDir) != ".bd" {
		return ""
	}

	repoRoot := filepath.Dir(bdDir)
	if !isDetachedCommitWorktreePath(repoRoot) {
		return ""
	}

	branch, err := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch != "HEAD" {
		return ""
	}

	head, err := gitOutput(repoRoot, "rev-parse", "HEAD")
	if err != nil || head == "" {
		return ""
	}

	worktrees, err := listWorktrees(repoRoot)
	if err != nil {
		return ""
	}

	var candidates []worktreeInfo
	for _, wt := range worktrees {
		if wt.Bare || wt.Detached || wt.Branch == "" {
			continue
		}
		if wt.Head != head || utils.PathsEqual(wt.Path, repoRoot) {
			continue
		}
		candidates = append(candidates, wt)
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		iStable := !isDetachedCommitWorktreePath(candidates[i].Path)
		jStable := !isDetachedCommitWorktreePath(candidates[j].Path)
		if iStable != jStable {
			return iStable
		}
		return candidates[i].Path < candidates[j].Path
	})

	stableBeadsDir := filepath.Join(candidates[0].Path, ".bd")
	if info, err := os.Stat(stableBeadsDir); err == nil && info.IsDir() {
		return utils.CanonicalizePath(stableBeadsDir)
	}

	return ""
}

// isDetachedCommitWorktreePath checks if a path follows the megarepo convention
// of placing detached worktrees under refs/commits/<sha>.
func isDetachedCommitWorktreePath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/refs/commits/")
}

func gitOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // args are internal, not user-supplied
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func listWorktrees(repoRoot string) ([]worktreeInfo, error) {
	output, err := gitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []worktreeInfo
	var current *worktreeInfo

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &worktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		case current == nil:
			continue
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached":
			current.Detached = true
		case line == "bare":
			current.Bare = true
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// RedirectInfo contains information about a beads directory redirect.
type RedirectInfo struct {
	// IsRedirected is true if the local .beads has a redirect file
	IsRedirected bool
	// LocalDir is the local .beads directory (the one with the redirect file)
	LocalDir string
	// TargetDir is the actual .beads directory being used (after following redirect)
	TargetDir string
}

// GetRedirectInfo checks if the current beads directory is redirected.
// It searches for the local .bd/ directory and checks if it contains a redirect file.
// Returns RedirectInfo with IsRedirected=true if a redirect is active.
//
// bd-wayc3: This function now also checks the git repo's local .beads directory even when
// BD_DIR is set. This handles the case where BD_DIR is pre-set to the redirect target
// (e.g., by shell environment or tooling), but we still need to detect that a redirect exists.
func GetRedirectInfo() RedirectInfo {
	// First, always check the git repo's local .beads directory for redirects
	// This handles the case where BD_DIR is pre-set to the redirect target
	if localBdDir := findLocalBdsDirInRepo(); localBdDir != "" {
		if info := checkRedirectInDir(localBdDir); info.IsRedirected {
			return info
		}
	}

	// Fall back to original logic for non-git-repo cases
	if localBdDir := findLocalBeadsDir(); localBdDir != "" {
		return checkRedirectInDir(localBdDir)
	}

	return RedirectInfo{}
}

// checkRedirectInDir checks if a beads directory has a redirect file and returns redirect info.
// Returns RedirectInfo with IsRedirected=true if a valid redirect exists.
func checkRedirectInDir(bdDir string) RedirectInfo {
	info := RedirectInfo{LocalDir: bdDir}

	// Check if this directory has a redirect file
	redirectFile := filepath.Join(bdDir, RedirectFileName)
	if _, err := os.Stat(redirectFile); err != nil {
		// No redirect file
		return info
	}

	// There's a redirect - find the target
	targetDir := FollowRedirect(bdDir)
	if targetDir == bdDir {
		// Redirect file exists but failed to resolve (invalid target)
		return info
	}

	info.IsRedirected = true
	info.TargetDir = targetDir
	return info
}

// findLocalBdsDirInRepo finds the .beads directory relative to the git repo root.
// This ignores BD_DIR to find the "true local" .beads for redirect detection.
// bd-wayc3: Added to detect redirects even when BD_DIR is pre-set.
func findLocalBdsDirInRepo() string {
	// Get git repo root
	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		return ""
	}

	bdDir := filepath.Join(repoRoot, ".bd")
	if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
		return bdDir
	}

	return ""
}

// findLocalBeadsDir finds the local .beads directory without following redirects.
// This is used to detect if a redirect is configured.
func findLocalBeadsDir() string {
	// Check BD_DIR environment variable first
	if bdDir := os.Getenv("BD_DIR"); bdDir != "" {
		return canonicalizeBeadsDirPath(bdDir)
	}

	// For worktrees, check worktree-local redirect first (per-worktree override).
	// Returns the raw worktree .beads dir (not the resolved target) since
	// findLocalBeadsDir doesn't follow redirects — callers use FollowRedirect.
	if git.IsWorktree() {
		if root := git.GetRepoRoot(); root != "" {
			wt := filepath.Join(root, ".bd")
			// Check for redirect file first
			if _, err := os.Stat(filepath.Join(wt, "redirect")); err == nil {
				return wt
			}
			// Check for worktree's own .beads with project files (separate-DB mode)
			if info, err := os.Stat(wt); err == nil && info.IsDir() {
				if hasBeadsProjectFiles(wt) {
					return wt
				}
			}
		}
	}

	if bdDir := GetWorktreeFallbackBdDir(); bdDir != "" {
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			return bdDir
		}
	}

	// Walk up directory tree
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for dir := cwd; dir != "/" && dir != "."; {
		bdDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			return bdDir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root (works on both Unix and Windows)
			// On Unix: filepath.Dir("/") returns "/"
			// On Windows: filepath.Dir("C:\\") returns "C:\\"
			break
		}
		dir = parent
	}

	return ""
}

// findDatabaseInBeadsDir searches for a database within a .beads directory.
// Checks metadata.json for the Dolt database path. For server mode, no local
// directory is required. For embedded mode, checks both the embeddeddolt/
// directory (where the embedded engine stores data) and the legacy dolt/ path.
// Returns empty string if no database is found.
func findDatabaseInBeadsDir(bdDir string, _ bool) string {
	// Check for metadata.json first (single source of truth)
	if cfg, err := configfile.Load(bdDir); err == nil && cfg != nil {
		// For Dolt server mode, database is on the server - no local directory required
		if cfg.IsDoltServerMode() {
			return cfg.DatabasePath(bdDir)
		}
		// For embedded Dolt, the engine stores data under .bd/embeddeddolt/,
		// not .bd/dolt/. Check the actual embedded data directory first.
		embeddedPath := filepath.Join(bdDir, "embeddeddolt")
		if info, err := os.Stat(embeddedPath); err == nil && info.IsDir() {
			return embeddedPath
		}
		// Fall back to configured database path (e.g. .bd/dolt/ for
		// server-mode installs or legacy setups that pre-date embeddeddolt).
		doltPath := cfg.DatabasePath(bdDir)
		if info, err := os.Stat(doltPath); err == nil && info.IsDir() {
			return doltPath
		}
	}

	// Fall back: check if embeddeddolt or dolt directory exists without metadata.json
	embeddedPath := filepath.Join(bdDir, "embeddeddolt")
	if info, err := os.Stat(embeddedPath); err == nil && info.IsDir() {
		return embeddedPath
	}
	doltPath := filepath.Join(bdDir, "dolt")
	if info, err := os.Stat(doltPath); err == nil && info.IsDir() {
		return doltPath
	}

	return ""
}

// Transaction provides atomic multi-operation support within a database transaction.
type Transaction = storage.Transaction

// FindDatabasePath discovers the bd database path using bd's standard search order:
//  1. $BD_DIR environment variable (points to .beads directory)
//  2. $BD_DB environment variable (points directly to database file, deprecated)
//  3. .bd/*.db in current directory or ancestors
//
// Redirect files are supported: if a .bd/redirect file exists, its contents
// are used as the actual .beads directory path.
//
// Returns empty string if no database is found.
func FindDatabasePath() string {
	// 1. Check BD_DIR environment variable (preferred)
	if bdDir := os.Getenv("BD_DIR"); bdDir != "" {
		// Canonicalize the path to prevent nested .beads directories
		absBeadsDir := canonicalizeBeadsDirPath(bdDir)

		// Follow redirect if present
		absBeadsDir = FollowRedirect(absBeadsDir)

		// Use helper to find database (no warnings for BD_DIR - user explicitly set it)
		if dbPath := findDatabaseInBeadsDir(absBeadsDir, false); dbPath != "" {
			return dbPath
		}

		// BD_DIR is set but no database found - this is OK for --no-db mode
		// Return empty string and let the caller handle it
	}

	// 2. Check BD_DB environment variable (deprecated but still supported)
	if envDB := os.Getenv("BD_DB"); envDB != "" {
		absDB := utils.CanonicalizePath(envDB)
		// If BD_DB points to a directory rather than a file, treat it
		// like BD_DIR to avoid filepath.Dir() resolving one level too
		// high in the caller (cmd/bd/main.go). See GH#2548.
		if info, err := os.Stat(absDB); err == nil && info.IsDir() {
			if dbPath := findDatabaseInBeadsDir(absDB, false); dbPath != "" {
				return dbPath
			}
		}
		return absDB
	}

	// 3. Search for .bd/*.db in current directory and ancestors
	if foundDB := findDatabaseInTree(); foundDB != "" {
		return utils.CanonicalizePath(foundDB)
	}

	// No fallback to ~/.bd - return empty string
	return ""
}

// hasBeadsProjectFiles checks if a .beads directory contains actual project files.
// Returns true if the directory contains any of:
// - metadata.json or config.yaml (project configuration)
// - Any *.db file (excluding backups and vc.db)
// - A dolt/ directory (Dolt database)
//
// Returns false for directories that only contain legacy registry files.
// This prevents FindBdDir from returning ~/.bd/ which only has registry.json.
func hasBeadsProjectFiles(bdDir string) bool {
	// Check for project configuration files
	if _, err := os.Stat(filepath.Join(bdDir, "metadata.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(bdDir, "config.yaml")); err == nil {
		return true
	}

	// Check for Dolt database directory (server mode uses dolt/, embedded uses embeddeddolt/)
	if info, err := os.Stat(filepath.Join(bdDir, "dolt")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(filepath.Join(bdDir, "embeddeddolt")); err == nil && info.IsDir() {
		return true
	}

	// Check for database files (excluding backups and vc.db)
	dbMatches, _ := filepath.Glob(filepath.Join(bdDir, "*.db"))
	for _, match := range dbMatches {
		baseName := filepath.Base(match)
		if !strings.Contains(baseName, ".backup") && baseName != "vc.db" {
			return true
		}
	}

	return false
}

// FindBdDir finds the .bd/ directory in the current directory tree.
// Returns empty string if not found.
// Stops at the git repository root to avoid finding unrelated directories.
// Validates that the directory contains actual project files.
// Redirect files are supported: if a .bd/redirect file exists, its contents
// are used as the actual .beads directory path.
// For worktrees, checks in order: worktree redirect, worktree's own .beads
// (separate-DB mode), then main repository's .beads (shared-DB fallback).
// This is useful for commands that need to detect beads projects without requiring a database.
func FindBdDir() string {
	// 1. Check BD_DIR environment variable (preferred)
	if bdDir := os.Getenv("BD_DIR"); bdDir != "" {
		absBeadsDir := canonicalizeBeadsDirPath(bdDir)

		// Follow redirect if present
		absBeadsDir = FollowRedirect(absBeadsDir)

		if info, err := os.Stat(absBeadsDir); err == nil && info.IsDir() {
			// Validate directory contains actual project files
			if hasBeadsProjectFiles(absBeadsDir) {
				return absBeadsDir
			}
		}
	}

	// 1b. Check cwd for .bd/ before git-worktree resolution.
	// This handles rigs (subdirectories with their own .bd/) inside a
	// git repo that also has .bd/. Without this, step 2b grabs the
	// git root's .bd/ and the rig's local one is never found.
	if cwd, err := os.Getwd(); err == nil {
		cwdBeadsDir := filepath.Join(cwd, ".bd")
		if info, err := os.Stat(cwdBeadsDir); err == nil && info.IsDir() {
			cwdBeadsDir = FollowRedirect(cwdBeadsDir)
			if hasBeadsProjectFiles(cwdBeadsDir) {
				return cwdBeadsDir
			}
		}
	}

	// 2. For worktrees, check worktree-local redirect first, then own .beads, then main repo
	var mainRepoRoot string
	if git.IsWorktree() {
		// 2a. Per-worktree redirect override
		if target := worktreeRedirectTarget(); target != "" {
			if info, err := os.Stat(target); err == nil && info.IsDir() {
				if hasBeadsProjectFiles(target) {
					return target
				}
			}
		}

		// 2b. Worktree's own .beads (separate-DB mode, no redirect)
		if worktreeRoot := git.GetRepoRoot(); worktreeRoot != "" {
			worktreeBeadsDir := filepath.Join(worktreeRoot, ".bd")
			if info, err := os.Stat(worktreeBeadsDir); err == nil && info.IsDir() {
				if hasBeadsProjectFiles(worktreeBeadsDir) {
					return worktreeBeadsDir
				}
			}
		}

		// 2c. Fall back to the canonical shared .beads for this worktree.
		if fallbackBeadsDir := GetWorktreeFallbackBdDir(); fallbackBeadsDir != "" {
			if info, err := os.Stat(fallbackBeadsDir); err == nil && info.IsDir() {
				fallbackBeadsDir = FollowRedirect(fallbackBeadsDir)
				if hasBeadsProjectFiles(fallbackBeadsDir) {
					return fallbackBeadsDir
				}
			}
		}

		var err error
		mainRepoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			mainRepoRoot = ""
		}
	}

	// 3. Search for .bd/ in current directory and ancestors
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Find git root to limit the search
	gitRoot := findGitRoot()
	worktreeRoot := gitRoot // save worktree-specific boundary
	if git.IsWorktree() && mainRepoRoot != "" {
		// For worktrees, extend search boundary to include main repo
		gitRoot = mainRepoRoot
	}

	for dir := cwd; dir != "/" && dir != "."; {
		bdDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			// Follow redirect if present
			bdDir = FollowRedirect(bdDir)

			// Validate directory contains actual project files
			if hasBeadsProjectFiles(bdDir) {
				return bdDir
			}
		}

		// Stop at git root to avoid finding unrelated directories
		if gitRoot != "" && dir == gitRoot {
			break
		}

		// Also stop at worktree root when it differs from main repo root
		// This prevents escaping the worktree boundary into unrelated directories
		if worktreeRoot != "" && worktreeRoot != gitRoot && dir == worktreeRoot {
			break
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root (works on both Unix and Windows)
			// On Unix: filepath.Dir("/") returns "/"
			// On Windows: filepath.Dir("C:\\") returns "C:\\"
			break
		}
		dir = parent
	}

	return ""
}

// DatabaseInfo contains information about a discovered beads database
type DatabaseInfo struct {
	Path       string // Full path to the .db file
	BdDir   string // Parent .beads directory
	IssueCount int    // Number of issues (-1 if unknown)
}

// findGitRoot returns the root directory of the current git repository,
// or empty string if not in a git repository. Used to limit directory
// tree walking to within the current git repo.
//
// This function delegates to git.GetRepoRoot() which is worktree-aware
// and handles Windows path normalization.
func findGitRoot() string {
	return git.GetRepoRoot()
}

// GetWorktreeFallbackBdDir returns the canonical shared .beads location for
// the current git worktree when no local redirect or worktree-local .beads is present.
func GetWorktreeFallbackBdDir() string {
	if !git.IsWorktree() {
		return ""
	}

	commonDir, err := git.GetGitCommonDir()
	if err != nil || commonDir == "" {
		return ""
	}

	commonDir = utils.CanonicalizePath(commonDir)
	if filepath.Base(commonDir) == ".git" {
		return filepath.Join(filepath.Dir(commonDir), ".bd")
	}

	return filepath.Join(commonDir, ".bd")
}

// worktreeRedirectTarget returns the resolved redirect target for the current
// worktree's .bd/redirect file, or empty string if not in a worktree or no
// redirect exists. This centralizes the per-worktree redirect override logic
// used by findLocalBeadsDir, FindBdDir, and findDatabaseInTree.
func worktreeRedirectTarget() string {
	if !git.IsWorktree() {
		return ""
	}
	worktreeRoot := git.GetRepoRoot()
	if worktreeRoot == "" {
		return ""
	}
	worktreeBeadsDir := filepath.Join(worktreeRoot, ".bd")
	redirectFile := filepath.Join(worktreeBeadsDir, "redirect")
	if _, err := os.Stat(redirectFile); err != nil {
		return ""
	}
	target := FollowRedirect(worktreeBeadsDir)
	if target == worktreeBeadsDir {
		// Redirect file exists but FollowRedirect returned the original path
		// (empty/invalid content). Return the raw .beads dir so callers that
		// only need to know a redirect *exists* (findLocalBeadsDir) still work.
		return worktreeBeadsDir
	}
	return target
}

// findDatabaseInTree walks up the directory tree looking for .bd/*.db
// Stops at the git repository root to avoid finding unrelated databases.
// For worktrees, searches the main repository root first, then falls back to worktree.
// Prefers config.json, falls back to bd.db, and warns if multiple .db files exist.
// Redirect files are supported: if a .bd/redirect file exists, its contents
// are used as the actual .beads directory path.
func findDatabaseInTree() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Resolve symlinks in working directory to ensure consistent path handling
	// This prevents issues when repos are accessed via symlinks (e.g. /Users/user/Code -> /Users/user/Documents/Code)
	if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolvedDir
	}

	// Check cwd first — a rig subdirectory with its own .bd/ takes
	// priority over the git root's .bd/ (same fix as FindBdDir step 1b).
	{
		cwdBeadsDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(cwdBeadsDir); err == nil && info.IsDir() {
			cwdBeadsDir = FollowRedirect(cwdBeadsDir)
			if dbPath := findDatabaseInBeadsDir(cwdBeadsDir, true); dbPath != "" {
				return dbPath
			}
		}
	}

	// Check if we're in a git worktree
	var mainRepoRoot string
	if git.IsWorktree() {
		// Per-worktree redirect override
		if target := worktreeRedirectTarget(); target != "" {
			if dbPath := findDatabaseInBeadsDir(target, true); dbPath != "" {
				return dbPath
			}
		}

		// Worktree's own .beads (separate-DB mode, no redirect)
		if worktreeRoot := git.GetRepoRoot(); worktreeRoot != "" {
			worktreeBeadsDir := filepath.Join(worktreeRoot, ".bd")
			if info, err := os.Stat(worktreeBeadsDir); err == nil && info.IsDir() {
				if dbPath := findDatabaseInBeadsDir(worktreeBeadsDir, true); dbPath != "" {
					return dbPath
				}
			}
		}

		// Fall back: search the canonical shared .beads for this worktree.
		if fallbackBeadsDir := GetWorktreeFallbackBdDir(); fallbackBeadsDir != "" {
			if info, err := os.Stat(fallbackBeadsDir); err == nil && info.IsDir() {
				fallbackBeadsDir = FollowRedirect(fallbackBeadsDir)
				if dbPath := findDatabaseInBeadsDir(fallbackBeadsDir, true); dbPath != "" {
					return dbPath
				}
			}
		}
		var err error
		mainRepoRoot, err = git.GetMainRepoRoot()
		if err != nil {
			mainRepoRoot = ""
		}
		// If not found in main repo, fall back to worktree search below
	}

	// Find git root to limit the search
	gitRoot := findGitRoot()
	if git.IsWorktree() && mainRepoRoot != "" {
		// For worktrees, extend search boundary to include main repo
		gitRoot = mainRepoRoot
	}

	// Walk up directory tree (regular repository or worktree fallback)
	for {
		bdDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			// Follow redirect if present
			bdDir = FollowRedirect(bdDir)

			// Use helper to find database (with warnings for auto-discovery)
			if dbPath := findDatabaseInBeadsDir(bdDir, true); dbPath != "" {
				return dbPath
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}

		// Stop at git root to avoid finding unrelated databases
		if gitRoot != "" && dir == gitRoot {
			break
		}

		dir = parent
	}

	return ""
}

// FindAllDatabases scans the directory hierarchy for the closest .beads directory.
// Returns a slice with at most one DatabaseInfo - the closest database to CWD.
// Stops searching upward as soon as a .beads directory is found,
// because in multi-workspace setups, nested .beads directories
// are intentional and separate - parent directories are out of scope.
// Redirect files are supported: if a .bd/redirect file exists, its contents
// are used as the actual .beads directory path.
func FindAllDatabases() []DatabaseInfo {
	databases := []DatabaseInfo{} // Initialize to empty slice, never return nil
	seen := make(map[string]bool) // Track canonical paths to avoid duplicates

	dir, err := os.Getwd()
	if err != nil {
		return databases
	}

	// Find git root to limit the search
	gitRoot := findGitRoot()

	// Walk up directory tree
	for {
		bdDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			// Follow redirect if present
			bdDir = FollowRedirect(bdDir)

			// Look for database: dolt directory first, then legacy *.db files
			dbPath := ""
			doltDir := filepath.Join(bdDir, "dolt")
			if dInfo, dErr := os.Stat(doltDir); dErr == nil && dInfo.IsDir() {
				dbPath = doltDir
			} else {
				// Legacy: check for *.db files (pre-migration)
				matches, err := filepath.Glob(filepath.Join(bdDir, "*.db"))
				if err == nil && len(matches) > 0 {
					dbPath = matches[0]
				}
			}

			if dbPath != "" {
				// Resolve symlinks to get canonical path for deduplication
				canonicalPath := dbPath
				if resolved, err := filepath.EvalSymlinks(dbPath); err == nil {
					canonicalPath = resolved
				}

				// Skip if we've already seen this database (via symlink or other path)
				if seen[canonicalPath] {
					// Move up one directory
					parent := filepath.Dir(dir)
					if parent == dir {
						break
					}
					dir = parent
					continue
				}
				seen[canonicalPath] = true

				databases = append(databases, DatabaseInfo{
					Path:       dbPath,
					BdDir:   bdDir,
					IssueCount: -1,
				})

				// Stop searching upward - the closest .beads is the one to use
				break
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}

		// Stop at git root to avoid finding unrelated databases
		if gitRoot != "" && dir == gitRoot {
			break
		}

		dir = parent
	}

	return databases
}
