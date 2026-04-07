package main

// nuclear_stubs.go provides minimal implementations of functions and types
// whose original source files were deleted during nuclear simplification.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/signalnine/bd/internal/config"
	"github.com/signalnine/bd/internal/project"
	"github.com/signalnine/bd/internal/storage"
	"github.com/signalnine/bd/internal/storage/embeddeddolt"
	"github.com/signalnine/bd/internal/types"
	"github.com/signalnine/bd/internal/utils"
)

// ---------------------------------------------------------------------------
// feedback.go replacements
// ---------------------------------------------------------------------------

// formatFeedbackID returns "id - title" or just "id" based on output.title-length config.
func formatFeedbackID(id, title string) string {
	title = applyTitleConfig(title)
	if title == "" {
		return id
	}
	return id + " - " + title
}

// formatFeedbackIDParen returns "id (title)" for multi-ID messages.
func formatFeedbackIDParen(id, title string) string {
	title = applyTitleConfig(title)
	if title == "" {
		return id
	}
	return id + " (" + title + ")"
}

func applyTitleConfig(title string) string {
	if title == "" {
		return ""
	}
	maxLen := config.GetInt("output.title-length")
	if maxLen <= 0 {
		return "" // hide titles
	}
	if maxLen > 0 && len(title) > maxLen {
		return title[:maxLen]
	}
	return title
}

// issueTitleOrEmpty returns the title of an issue, or empty string if issue is nil.
func issueTitleOrEmpty(issue *types.Issue) string {
	if issue == nil {
		return ""
	}
	return issue.Title
}

// lookupTitle returns the title for an issue ID, or empty string on failure.
func lookupTitle(id string) string {
	if store == nil || IsExternalRef(id) {
		return ""
	}
	issue, err := store.GetIssue(rootCtx, id)
	if err != nil || issue == nil {
		return ""
	}
	return issue.Title
}

// ---------------------------------------------------------------------------
// last_touched.go replacements
// ---------------------------------------------------------------------------

const lastTouchedFile = "last-touched"

// GetLastTouchedID returns the ID of the last touched issue.
func GetLastTouchedID() string {
	bdDir := project.FindBdDir()
	if bdDir == "" {
		return ""
	}
	lastTouchedPath := filepath.Join(bdDir, lastTouchedFile)
	data, err := os.ReadFile(lastTouchedPath) // #nosec G304
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SetLastTouchedID saves the ID of the last touched issue.
func SetLastTouchedID(issueID string) {
	if issueID == "" {
		return
	}
	bdDir := project.FindBdDir()
	if bdDir == "" {
		return
	}
	lastTouchedPath := filepath.Join(bdDir, lastTouchedFile)
	_ = os.WriteFile(lastTouchedPath, []byte(issueID+"\n"), 0600)
}

// ClearLastTouched removes the last touched file.
func ClearLastTouched() {
	bdDir := project.FindBdDir()
	if bdDir == "" {
		return
	}
	lastTouchedPath := filepath.Join(bdDir, lastTouchedFile)
	_ = os.Remove(lastTouchedPath)
}

// ---------------------------------------------------------------------------
// routed.go replacements
// ---------------------------------------------------------------------------

// RoutedResult contains the result of a routed issue lookup.
type RoutedResult struct {
	Issue      *types.Issue
	Store      *embeddeddolt.EmbeddedDoltStore
	Routed     bool
	ResolvedID string
	closeFn    func()
}

// Close closes any routed storage.
func (r *RoutedResult) Close() {
	if r.closeFn != nil {
		r.closeFn()
	}
}

// isNotFoundErr returns true if the error indicates the issue was not found.
func isNotFoundErr(err error) bool {
	if errors.Is(err, storage.ErrNotFound) {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "no issue found matching") {
		return true
	}
	return false
}

// resolveAndGetIssueWithRouting resolves a partial ID and gets the issue.
func resolveAndGetIssueWithRouting(ctx context.Context, localStore *embeddeddolt.EmbeddedDoltStore, id string) (*RoutedResult, error) {
	result, err := resolveAndGetFromStore(ctx, localStore, id, false)
	if err == nil {
		return result, nil
	}
	if isNotFoundErr(err) {
		if autoResult, autoErr := resolveViaAutoRouting(ctx, localStore, id); autoErr == nil {
			return autoResult, nil
		}
	}
	return nil, err
}

func resolveAndGetFromStore(ctx context.Context, s *embeddeddolt.EmbeddedDoltStore, id string, routed bool) (*RoutedResult, error) {
	resolvedID, err := utils.ResolvePartialID(ctx, s, id)
	if err != nil {
		return nil, err
	}
	issue, err := s.GetIssue(ctx, resolvedID)
	if err != nil {
		return nil, err
	}
	return &RoutedResult{
		Issue:      issue,
		Store:      s,
		Routed:     routed,
		ResolvedID: resolvedID,
	}, nil
}

func resolveViaAutoRouting(ctx context.Context, localStore *embeddeddolt.EmbeddedDoltStore, id string) (*RoutedResult, error) {
	routedStore, routed, err := openRoutedReadStore(ctx, localStore)
	if err != nil || !routed {
		return nil, fmt.Errorf("no auto-routed store available")
	}
	result, err := resolveAndGetFromStore(ctx, routedStore, id, true)
	if err != nil {
		_ = routedStore.Close()
		return nil, err
	}
	result.closeFn = func() { _ = routedStore.Close() }
	return result, nil
}

// getIssueWithRouting gets an issue by exact ID.
func getIssueWithRouting(ctx context.Context, localStore *embeddeddolt.EmbeddedDoltStore, id string) (*RoutedResult, error) {
	issue, err := localStore.GetIssue(ctx, id)
	if err == nil {
		return &RoutedResult{
			Issue:      issue,
			Store:      localStore,
			Routed:     false,
			ResolvedID: id,
		}, nil
	}
	if isNotFoundErr(err) {
		if autoResult, autoErr := resolveViaAutoRouting(ctx, localStore, id); autoErr == nil {
			return autoResult, nil
		}
	}
	return nil, err
}

// ---------------------------------------------------------------------------
// backup_auto.go replacements
// ---------------------------------------------------------------------------

// primeHasGitRemote detects if any git remote is configured (stubbable for tests).
var primeHasGitRemote = func() bool {
	out, err := exec.Command("git", "remote").Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// maybeAutoBackup runs auto-backup if enabled and throttle interval has passed.
func maybeAutoBackup(ctx context.Context) {
	if !isBackupAutoEnabled() {
		return
	}
	if store == nil {
		return
	}
	dir, err := backupDir()
	if err != nil {
		return
	}
	state, err := loadBackupState(dir)
	if err != nil {
		return
	}
	currentCommit, err := store.GetCurrentCommit(ctx)
	if err != nil {
		return
	}
	if currentCommit == state.LastDoltCommit && state.LastDoltCommit != "" {
		return
	}
	_, _ = runBackupExport(ctx, true) // Best effort
}

// isBackupAutoEnabled returns whether backup should run.
func isBackupAutoEnabled() bool {
	if config.GetValueSource("backup.enabled") != config.SourceDefault {
		return config.GetBool("backup.enabled")
	}
	return primeHasGitRemote()
}

// ---------------------------------------------------------------------------
// compact.go / formatBytes replacement
// ---------------------------------------------------------------------------

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ---------------------------------------------------------------------------
// info.go / VersionChange replacement
// ---------------------------------------------------------------------------

// VersionChange represents agent-relevant changes for a specific version.
type VersionChange struct {
	Version     string   `json:"version"`
	Date        string   `json:"date,omitempty"`
	Summary     string   `json:"summary"`
	Changes     []string `json:"changes,omitempty"`
	AgentNotes  []string `json:"agent_notes,omitempty"`
	BreakingAPI bool     `json:"breaking_api,omitempty"`
}

// versionChanges contains agent-actionable changes for recent versions.
var versionChanges = []VersionChange{}

// isRemoteURL is an alias for isValidRemoteURL for backward compatibility.
func isRemoteURL(url string) bool {
	return isValidRemoteURL(url)
}

// ---------------------------------------------------------------------------
// direct_mode.go replacements
// ---------------------------------------------------------------------------

// ensureDirectMode makes sure the CLI is operating in direct-storage mode.
func ensureDirectMode(_ string) error {
	return ensureStoreActive()
}

// ensureStoreActive guarantees that a storage backend is initialized and tracked.
func ensureStoreActive() error {
	lockStore()
	active := isStoreActive() && getStore() != nil
	unlockStore()
	if active {
		return nil
	}

	bdDir := project.FindBdDir()
	if bdDir == "" {
		return fmt.Errorf("no bd database found.\nHint: run 'bd init' to create a database in the current directory")
	}

	s, err := newDoltStoreFromConfig(getRootContext(), bdDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w\nHint: %s", err, diagHint())
	}

	if dbPath := project.FindDatabasePath(); dbPath != "" {
		setDBPath(dbPath)
	}

	lockStore()
	setStore(s)
	setStoreActive(true)
	unlockStore()

	return nil
}

// ---------------------------------------------------------------------------
// detect_pollution.go replacements
// ---------------------------------------------------------------------------

var testPrefixPattern = regexp.MustCompile(`^(test|benchmark|sample|tmp|temp|debug|dummy)[-_\s]`)

func isTestIssue(title string) bool {
	return testPrefixPattern.MatchString(strings.ToLower(title))
}

// ---------------------------------------------------------------------------
// tips.go replacements
// ---------------------------------------------------------------------------

// maybeShowTip is a no-op stub (tips functionality removed in nuclear simplification).
func maybeShowTip(_ *embeddeddolt.EmbeddedDoltStore) {}

// ---------------------------------------------------------------------------
// diff.go / joinStrings replacement
// ---------------------------------------------------------------------------

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ---------------------------------------------------------------------------
// worktree.go replacements
// ---------------------------------------------------------------------------

func warnMultipleDatabases(_ string) {
	// Simplified stub -- no worktree detection in nuclear simplification.
}

// ---------------------------------------------------------------------------
// export_auto.go replacements
// ---------------------------------------------------------------------------

// maybeAutoExport is a no-op stub (auto-export removed in nuclear simplification).
func maybeAutoExport(_ context.Context) {}

// ---------------------------------------------------------------------------
// help_all.go replacements
// ---------------------------------------------------------------------------

var helpAllFlag bool
var helpDocFlag string
var helpListFlag bool

func registerHelpAllFlag() {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "help" {
			cmd.Flags().BoolVar(&helpAllFlag, "all", false, "Show help for all commands in a single document")
			cmd.Flags().StringVar(&helpDocFlag, "doc", "", "Generate markdown docs for a single command (use - for stdout)")
			cmd.Flags().BoolVar(&helpListFlag, "list", false, "List all available commands")
			return
		}
	}
}

// ---------------------------------------------------------------------------
// show_thread.go replacements
// ---------------------------------------------------------------------------

// showMessageThread is a stub (message thread display removed in nuclear simplification).
func showMessageThread(ctx context.Context, messageID string, jsonOutput bool) {
	fmt.Printf("Message thread for %s (not available in this build)\n", messageID)
}
