package main

// hooks_nuclear.go provides the git hooks infrastructure needed by init_git_hooks.go
// and other files after hooks.go was deleted during nuclear simplification.
// Source: ported from cmd/bd/hooks.go in the main beads repo.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/steveyegge/bd/internal/git"
	"github.com/steveyegge/bd/internal/project"
)

const hookVersionPrefix = "# bd-hooks-version: "
const shimVersionPrefix = "# bd-shim "

// inlineHookMarker identifies inline hooks created by bd init.
const inlineHookMarker = "# bd (beads)"

// Section markers for git hooks.
const hookSectionBeginPrefix = "# --- BEGIN BEADS INTEGRATION"
const hookSectionEndPrefix = "# --- END BEADS INTEGRATION"

const hookTimeoutSeconds = 300

func hookSectionBeginLine() string {
	return fmt.Sprintf("%s v%s ---", hookSectionBeginPrefix, Version)
}

func hookSectionEndLine() string {
	return fmt.Sprintf("%s v%s ---", hookSectionEndPrefix, Version)
}

// generateHookSection returns the marked section content for a given hook name.
func generateHookSection(hookName string) string {
	return hookSectionBeginLine() + "\n" +
		"# This section is managed by beads. Do not remove these markers.\n" +
		"if command -v bd >/dev/null 2>&1; then\n" +
		"  export BD_GIT_HOOK=1\n" +
		"  _bd_timeout=${BD_HOOK_TIMEOUT:-" + fmt.Sprintf("%d", hookTimeoutSeconds) + "}\n" +
		"  if command -v timeout >/dev/null 2>&1; then\n" +
		"    timeout \"$_bd_timeout\" bd hooks run " + hookName + " \"$@\"\n" +
		"    _bd_exit=$?\n" +
		"    if [ $_bd_exit -eq 124 ]; then\n" +
		"      echo >&2 \"beads: hook '" + hookName + "' timed out after ${_bd_timeout}s -- continuing without beads\"\n" +
		"      _bd_exit=0\n" +
		"    fi\n" +
		"  else\n" +
		"    bd hooks run " + hookName + " \"$@\"\n" +
		"    _bd_exit=$?\n" +
		"  fi\n" +
		"  if [ $_bd_exit -eq 3 ]; then\n" +
		"    echo >&2 \"beads: database not initialized -- skipping hook '" + hookName + "'\"\n" +
		"    _bd_exit=0\n" +
		"  fi\n" +
		"  if [ $_bd_exit -ne 0 ]; then exit $_bd_exit; fi\n" +
		"fi\n" +
		hookSectionEndLine() + "\n"
}

// injectHookSection merges the beads section into existing hook file content.
func injectHookSection(existing, section string) string {
	return injectHookSectionWithDepth(existing, section, 0)
}

const maxInjectDepth = 5

func injectHookSectionWithDepth(existing, section string, depth int) string {
	if depth > maxInjectDepth {
		result := existing
		if !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
		return result + "\n" + section
	}

	beginIdx := strings.Index(existing, hookSectionBeginPrefix)
	endIdx := strings.Index(existing, hookSectionEndPrefix)

	if beginIdx != -1 && endIdx != -1 && beginIdx < endIdx {
		lineStart := strings.LastIndex(existing[:beginIdx], "\n")
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart++
		}
		endOfEndMarker := endIdx + len(hookSectionEndPrefix)
		restAfterPrefix := existing[endOfEndMarker:]
		if nlIdx := strings.Index(restAfterPrefix, "\n"); nlIdx != -1 {
			endOfEndMarker += nlIdx + 1
		} else {
			endOfEndMarker = len(existing)
		}
		return existing[:lineStart] + section + existing[endOfEndMarker:]
	} else if beginIdx != -1 {
		cleaned := removeOrphanedBeginBlock(existing, beginIdx)
		return injectHookSectionWithDepth(cleaned, section, depth+1)
	} else if endIdx != -1 {
		cleaned := removeMarkerLine(existing, endIdx, hookSectionEndPrefix)
		return injectHookSectionWithDepth(cleaned, section, depth+1)
	}

	result := existing
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	result += "\n" + section
	return result
}

func removeOrphanedBeginBlock(content string, beginIdx int) string {
	lineStart := strings.LastIndex(content[:beginIdx], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	afterBegin := content[beginIdx:]
	blockEnd := len(content)
	lines := strings.SplitAfter(afterBegin, "\n")
	scanned := beginIdx
	for i, line := range lines {
		if i == 0 {
			scanned += len(line)
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blockEnd = scanned + len(line)
			break
		}
		if strings.Contains(line, hookSectionBeginPrefix) {
			blockEnd = scanned
			break
		}
		scanned += len(line)
	}
	return content[:lineStart] + content[blockEnd:]
}

func removeMarkerLine(content string, markerIdx int, markerPrefix string) string {
	lineStart := strings.LastIndex(content[:markerIdx], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	lineEnd := markerIdx + len(markerPrefix)
	restAfterPrefix := content[lineEnd:]
	if nlIdx := strings.Index(restAfterPrefix, "\n"); nlIdx != -1 {
		lineEnd += nlIdx + 1
	} else {
		lineEnd = len(content)
	}
	return content[:lineStart] + content[lineEnd:]
}

func removeHookSection(content string) (string, bool) {
	beginIdx := strings.Index(content, hookSectionBeginPrefix)
	endIdx := strings.Index(content, hookSectionEndPrefix)
	if beginIdx == -1 && endIdx == -1 {
		return content, false
	}
	if beginIdx != -1 && endIdx != -1 && beginIdx < endIdx {
		lineStart := strings.LastIndex(content[:beginIdx], "\n")
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart++
		}
		endOfSection := endIdx + len(hookSectionEndPrefix)
		restAfterPrefix := content[endOfSection:]
		if nlIdx := strings.Index(restAfterPrefix, "\n"); nlIdx != -1 {
			endOfSection += nlIdx + 1
		} else {
			endOfSection = len(content)
		}
		return content[:lineStart] + content[endOfSection:], true
	}
	// Broken markers -- remove what we can
	if beginIdx != -1 {
		cleaned := removeOrphanedBeginBlock(content, beginIdx)
		return cleaned, true
	}
	if endIdx != -1 {
		cleaned := removeMarkerLine(content, endIdx, hookSectionEndPrefix)
		return cleaned, true
	}
	return content, false
}

// HookStatus contains status information for a single git hook.
type HookStatus struct {
	Name      string
	Installed bool
	Version   string
	IsShim    bool
	Outdated  bool
}

type hookVersionInfo struct {
	Version  string
	IsShim   bool
	IsBdHook bool
}

func getHookVersion(path string) (hookVersionInfo, error) {
	// #nosec G304 -- hook path constrained to .git/hooks directory
	file, err := os.Open(path)
	if err != nil {
		return hookVersionInfo{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var content strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")
		if strings.HasPrefix(line, hookSectionBeginPrefix) {
			after := strings.TrimPrefix(line, hookSectionBeginPrefix)
			after = strings.TrimSpace(after)
			after = strings.TrimPrefix(after, "v")
			after = strings.TrimSuffix(after, "---")
			version := strings.TrimSpace(after)
			return hookVersionInfo{Version: version, IsShim: true, IsBdHook: true}, nil
		}
		if strings.HasPrefix(line, shimVersionPrefix) {
			version := strings.TrimSpace(strings.TrimPrefix(line, shimVersionPrefix))
			return hookVersionInfo{Version: version, IsShim: true, IsBdHook: true}, nil
		}
		if strings.HasPrefix(line, hookVersionPrefix) {
			version := strings.TrimSpace(strings.TrimPrefix(line, hookVersionPrefix))
			return hookVersionInfo{Version: version, IsShim: false, IsBdHook: true}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return hookVersionInfo{}, fmt.Errorf("reading hook file: %w", err)
	}
	if strings.Contains(content.String(), inlineHookMarker) {
		return hookVersionInfo{IsBdHook: true}, nil
	}
	return hookVersionInfo{}, nil
}

// CheckGitHooks checks the status of bd git hooks in .git/hooks/.
func CheckGitHooks() []HookStatus {
	hooks := []string{"pre-commit", "post-merge", "pre-push", "post-checkout", "prepare-commit-msg"}
	statuses := make([]HookStatus, 0, len(hooks))

	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		for _, hookName := range hooks {
			statuses = append(statuses, HookStatus{Name: hookName, Installed: false})
		}
		return statuses
	}

	for _, hookName := range hooks {
		status := HookStatus{Name: hookName}
		hookPath := filepath.Join(hooksDir, hookName)
		versionInfo, err := getHookVersion(hookPath)
		if err != nil {
			status.Installed = false
		} else {
			status.Installed = true
			status.Version = versionInfo.Version
			status.IsShim = versionInfo.IsShim
			if !versionInfo.IsShim && versionInfo.IsBdHook && versionInfo.Version != Version {
				status.Outdated = true
			}
		}
		statuses = append(statuses, status)
	}
	return statuses
}

//nolint:unparam // force and chain kept for CLI flag compatibility
func installHooksWithOptions(hookNames []string, force bool, shared bool, chain bool, beadsHooks bool) error {
	var hooksDir string
	if beadsHooks {
		bdDir := project.FindBdDir()
		if bdDir == "" {
			return fmt.Errorf("not in a beads workspace (no .beads directory found)")
		}
		hooksDir = filepath.Join(bdDir, "hooks")
	} else if shared {
		hooksDir = ".beads-hooks"
	} else {
		var err error
		hooksDir, err = git.GetGitHooksDir()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	if beadsHooks || shared {
		preservePreexistingHooks(hooksDir)
	}

	for _, hookName := range hookNames {
		hookPath := filepath.Join(hooksDir, hookName)
		section := generateHookSection(hookName)

		// #nosec G304 -- hook path constrained to hooks directory
		existing, readErr := os.ReadFile(hookPath)

		if readErr != nil && !os.IsNotExist(readErr) {
			return fmt.Errorf("failed to read %s: %w", hookName, readErr)
		}

		var newContent string
		if os.IsNotExist(readErr) {
			newContent = "#!/usr/bin/env sh\n" + section
		} else {
			existingStr := string(existing)
			if strings.Contains(existingStr, hookSectionBeginPrefix) {
				newContent = injectHookSection(existingStr, section)
			} else {
				versionInfo, _ := getHookVersion(hookPath)
				if versionInfo.IsBdHook {
					newContent = "#!/usr/bin/env sh\n" + section
				} else {
					newContent = injectHookSection(existingStr, section)
				}
			}
		}

		newContent = strings.ReplaceAll(newContent, "\r\n", "\n")

		// #nosec G306 -- git hooks must be executable
		if err := os.WriteFile(hookPath, []byte(newContent), 0755); err != nil {
			return fmt.Errorf("failed to write %s: %w", hookName, err)
		}
	}

	if beadsHooks {
		if err := configureBeadsHooksPath(); err != nil {
			return fmt.Errorf("failed to configure git hooks path: %w", err)
		}
	} else if shared {
		if err := configureSharedHooksPath(); err != nil {
			return fmt.Errorf("failed to configure git hooks path: %w", err)
		}
	}

	return nil
}

func preservePreexistingHooks(targetDir string) {
	currentDir, err := git.GetGitHooksDir()
	if err != nil {
		return
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return
	}
	absCurrent, err := filepath.Abs(currentDir)
	if err != nil {
		return
	}
	if absTarget == absCurrent {
		return
	}
	repoRoot := git.GetRepoRoot()
	if repoRoot != "" {
		absBeadsHooks, _ := filepath.Abs(filepath.Join(repoRoot, ".bd", "hooks"))
		absSharedHooks, _ := filepath.Abs(filepath.Join(repoRoot, ".beads-hooks"))
		if absCurrent == absBeadsHooks || absCurrent == absSharedHooks {
			return
		}
	}
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasSuffix(entry.Name(), ".sample") {
			continue
		}
		srcPath := filepath.Join(currentDir, entry.Name())
		// #nosec G304 -- hook path constrained to known hooks directories
		content, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		contentStr := string(content)
		if strings.Contains(contentStr, hookSectionBeginPrefix) || strings.Contains(contentStr, inlineHookMarker) {
			continue
		}
		dstPath := filepath.Join(targetDir, entry.Name())
		if _, err := os.Stat(dstPath); err == nil {
			continue
		}
		// #nosec G306 -- git hooks must be executable
		if err := os.WriteFile(dstPath, content, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to preserve %s hook: %v\n", entry.Name(), err)
			continue
		}
	}
}

func configureSharedHooksPath() error {
	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}
	absHooksPath := filepath.Join(repoRoot, ".beads-hooks")
	cmd := exec.Command("git", "config", "core.hooksPath", absHooksPath)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config failed: %w (output: %s)", err, string(output))
	}
	return nil
}

func configureBeadsHooksPath() error {
	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}
	absHooksPath := filepath.Join(repoRoot, ".bd", "hooks")
	cmd := exec.Command("git", "config", "core.hooksPath", absHooksPath)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config failed: %w (output: %s)", err, string(output))
	}
	return nil
}

func uninstallHooks() error {
	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		return err
	}
	hookNames := []string{"pre-commit", "post-merge", "pre-push", "post-checkout", "prepare-commit-msg"}
	for _, hookName := range hookNames {
		hookPath := filepath.Join(hooksDir, hookName)
		// #nosec G304 -- hook path constrained to .git/hooks directory
		content, err := os.ReadFile(hookPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read %s: %w", hookName, err)
		}
		newContent, found := removeHookSection(string(content))
		if found {
			remaining := strings.TrimSpace(newContent)
			if remaining == "" || remaining == "#!/usr/bin/env sh" || remaining == "#!/bin/sh" {
				if err := os.Remove(hookPath); err != nil {
					return fmt.Errorf("failed to remove %s: %w", hookName, err)
				}
			} else {
				// #nosec G306 -- git hooks must be executable
				if err := os.WriteFile(hookPath, []byte(newContent), 0755); err != nil {
					return fmt.Errorf("failed to write %s: %w", hookName, err)
				}
			}
			continue
		}
		versionInfo, verr := getHookVersion(hookPath)
		if verr == nil && versionInfo.IsBdHook {
			if err := os.Remove(hookPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", hookName, err)
			}
			backupPath := hookPath + ".backup"
			if _, err := os.Stat(backupPath); err == nil {
				if err := os.Rename(backupPath, hookPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to restore backup for %s: %v\n", hookName, err)
				}
			}
		}
	}
	if err := resetHooksPathIfBeadsManaged(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to reset core.hooksPath: %v\n", err)
	}
	return nil
}

func resetHooksPathIfBeadsManaged() error {
	repoRoot := git.GetRepoRoot()
	if repoRoot == "" {
		return nil
	}
	cmd := exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	hooksPath := strings.TrimSpace(string(out))
	absBeadsHooks := filepath.Join(repoRoot, ".bd", "hooks")
	absSharedHooks := filepath.Join(repoRoot, ".beads-hooks")
	if hooksPath == ".bd/hooks" || hooksPath == ".beads-hooks" ||
		hooksPath == absBeadsHooks || hooksPath == absSharedHooks {
		cmd = exec.Command("git", "config", "--unset", "core.hooksPath")
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config --unset core.hooksPath failed: %w (output: %s)", err, string(output))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// doctor types and functions needed by init.go
// ---------------------------------------------------------------------------

const (
	statusOK      = "ok"
	statusWarning = "warning"
	statusError   = "error"
)

type doctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	Fix      string `json:"fix,omitempty"`
	Category string `json:"category,omitempty"`
}

type doctorResult struct {
	Path      string        `json:"path"`
	Checks    []doctorCheck `json:"checks"`
	OverallOK bool          `json:"overall_ok"`
}

// runInitDiagnostics is a stub that returns an empty (clean) result.
// (Doctor functionality removed in nuclear simplification.)
func runInitDiagnostics(_ string) doctorResult {
	return doctorResult{OverallOK: true}
}
