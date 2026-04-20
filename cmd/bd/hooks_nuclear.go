package main

// Cleanup helpers for legacy git hook shims installed by bd <= v1.0.2.
// Those versions wrote pre-commit / post-merge / pre-push scripts that called
// `bd hooks run <name>`, a subcommand that no longer exists, which breaks
// every git commit in affected repos. bd init now calls uninstallLegacyGitHooks
// to remove those shims and unset core.hooksPath when it points at a
// bd-managed directory.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signalnine/bd/internal/git"
)

const hookVersionPrefix = "# bd-hooks-version: "
const shimVersionPrefix = "# bd-shim "
const inlineHookMarker = "# bd"
const hookSectionBeginPrefix = "# --- BEGIN BD INTEGRATION"
const hookSectionEndPrefix = "# --- END BD INTEGRATION"

var legacyHookNames = []string{"pre-commit", "post-merge", "pre-push", "post-checkout", "prepare-commit-msg"}

// uninstallLegacyGitHooks removes bd-managed git hook shims and unsets
// core.hooksPath when it points at .bd/hooks or .bd-hooks. Idempotent.
func uninstallLegacyGitHooks() error {
	hooksDir, err := git.GetGitHooksDir()
	if err != nil {
		return err
	}
	for _, hookName := range legacyHookNames {
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
		if isBdManagedHook(hookPath) {
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
	return resetHooksPathIfBdManaged()
}

func isBdManagedHook(path string) bool {
	// #nosec G304 -- hook path constrained to .git/hooks directory
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var content strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")
		if strings.HasPrefix(line, hookSectionBeginPrefix) ||
			strings.HasPrefix(line, shimVersionPrefix) ||
			strings.HasPrefix(line, hookVersionPrefix) {
			return true
		}
	}
	return strings.Contains(content.String(), inlineHookMarker)
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
	if beginIdx != -1 {
		return removeOrphanedBeginBlock(content, beginIdx), true
	}
	return removeMarkerLine(content, endIdx, hookSectionEndPrefix), true
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

func resetHooksPathIfBdManaged() error {
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
	absBdHooks := filepath.Join(repoRoot, ".bd", "hooks")
	absSharedHooks := filepath.Join(repoRoot, ".bd-hooks")
	if hooksPath == ".bd/hooks" || hooksPath == ".bd-hooks" ||
		hooksPath == absBdHooks || hooksPath == absSharedHooks {
		cmd = exec.Command("git", "config", "--unset", "core.hooksPath")
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config --unset core.hooksPath failed: %w (output: %s)", err, string(output))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// doctor stubs retained for init.go (doctor proper was removed in the
// nuclear simplification; init.go still references runInitDiagnostics).
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

func runInitDiagnostics(_ string) doctorResult {
	return doctorResult{OverallOK: true}
}
