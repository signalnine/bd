package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/steveyegge/beads/internal/beads"
)

// isGitRepo checks if the current working directory is in a git repository.
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// isBareGitRepo checks if the current git repository is bare.
func isBareGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// gitRemoteGetURL returns the URL for a named git remote.
func gitRemoteGetURL(remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// gitLsRemoteHasRef checks if a remote has a specific ref.
func gitLsRemoteHasRef(remote, ref string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "ls-remote", remote, ref)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// gitURLToDoltRemote converts a git remote URL to dolt's remote format.
func gitURLToDoltRemote(url string) string {
	if strings.HasPrefix(url, "git+") {
		return url
	}
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return "git+" + url
	}
	if strings.HasPrefix(url, "ssh://") {
		return "git+" + url
	}
	if idx := strings.Index(url, ":"); idx > 0 && !strings.Contains(url[:idx], "/") {
		return "git+ssh://" + url[:idx] + "/" + url[idx+1:]
	}
	return "git+" + url
}

// gitHasUpstream checks if the current branch has an upstream configured.
func gitHasUpstream() bool {
	rc, err := beads.GetRepoContext()
	if err != nil {
		return false
	}
	ctx := context.Background()
	branchCmd := rc.GitCmd(ctx, "symbolic-ref", "--short", "HEAD")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return false
	}
	branch := strings.TrimSpace(string(branchOutput))
	return gitBranchHasUpstream(branch)
}

// gitHasAnyRemotes returns true if the git repository has any remotes configured.
func gitHasAnyRemotes() bool {
	rc, err := beads.GetRepoContext()
	if err != nil {
		return false
	}
	ctx := context.Background()
	remoteCmd := rc.GitCmd(ctx, "remote")
	output, err := remoteCmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// gitBranchHasUpstream checks if a specific branch has an upstream configured.
func gitBranchHasUpstream(branch string) bool {
	rc, err := beads.GetRepoContext()
	if err != nil {
		return false
	}
	ctx := context.Background()
	remoteCmd := rc.GitCmd(ctx, "config", "--get", fmt.Sprintf("branch.%s.remote", branch)) //nolint:gosec
	mergeCmd := rc.GitCmd(ctx, "config", "--get", fmt.Sprintf("branch.%s.merge", branch))   //nolint:gosec
	remoteErr := remoteCmd.Run()
	mergeErr := mergeCmd.Run()
	return remoteErr == nil && mergeErr == nil
}
