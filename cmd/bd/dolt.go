package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/bd/internal/storage/doltutil"
	"golang.org/x/term"
)

var doltCmd = &cobra.Command{
	Use:     "dolt",
	GroupID: "setup",
	Short:   "Dolt database version control commands",
	Long: `Version control commands for the embedded Dolt database.

Version control:
  bd dolt commit       Commit pending changes
  bd dolt push         Push commits to Dolt remote
  bd dolt pull         Pull commits from Dolt remote

Remote management:
  bd dolt remote add <name> <url>   Add a Dolt remote
  bd dolt remote list                List configured remotes
  bd dolt remote remove <name>       Remove a Dolt remote`,
}

// isRemoteNotFoundErr checks whether the error is a Dolt "remote not found"
// error.
func isRemoteNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "remote") && strings.Contains(msg, "not found")
}

// isDivergedHistoryErr checks whether the error indicates that local and remote
// Dolt histories have diverged.
func isDivergedHistoryErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no common ancestor") ||
		strings.Contains(msg, "can't find common ancestor") ||
		strings.Contains(msg, "cannot find common ancestor")
}

// printDivergedHistoryGuidance prints recovery guidance when push/pull fails
// due to diverged local and remote histories.
func printDivergedHistoryGuidance(_ string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Local and remote Dolt histories have diverged.")
	fmt.Fprintln(os.Stderr, "This means the local database and the remote have independent commit")
	fmt.Fprintln(os.Stderr, "histories with no common merge base.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Recovery options:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  1. Keep remote, discard local (recommended if remote is authoritative):")
	fmt.Fprintln(os.Stderr, "       bd bootstrap              # re-clone from remote")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  2. Keep local, overwrite remote (if local is authoritative):")
	fmt.Fprintln(os.Stderr, "       bd dolt push --force       # force-push local history to remote")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  3. Manual recovery (re-initialize local database):")
	fmt.Fprintln(os.Stderr, "       rm -rf .bd/embeddeddolt # delete local Dolt database")
	fmt.Fprintln(os.Stderr, "       bd bootstrap              # re-clone from remote")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Tip: This usually happens when multiple agents independently initialize")
	fmt.Fprintln(os.Stderr, "databases and push to the same remote. Use 'bd bootstrap' to clone an")
	fmt.Fprintln(os.Stderr, "existing remote instead of 'bd init' to avoid divergent histories.")
}

var doltPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push commits to Dolt remote",
	Long: `Push local Dolt commits to the configured remote.

Requires a Dolt remote to be configured in the database directory.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.

Use --force to overwrite remote changes (e.g., when the remote has
uncommitted changes in its working set).`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}
		force, _ := cmd.Flags().GetBool("force")
		fmt.Println("Pushing to Dolt remote...")
		if force {
			if err := st.ForcePush(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				if isRemoteNotFoundErr(err) {
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, "No remote is configured for this database.")
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, "To set up remote sync (for backup or team sharing):")
					fmt.Fprintln(os.Stderr, "  bd dolt remote add origin <url>")
					fmt.Fprintln(os.Stderr, "  bd dolt push")
				} else if isDivergedHistoryErr(err) {
					printDivergedHistoryGuidance("push --force")
				}
				os.Exit(1)
			}
		} else {
			if err := st.Push(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				if isRemoteNotFoundErr(err) {
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, "No remote is configured for this database.")
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, "To set up remote sync (for backup or team sharing):")
					fmt.Fprintln(os.Stderr, "  bd dolt remote add origin <url>")
					fmt.Fprintln(os.Stderr, "  bd dolt push")
				} else if isDivergedHistoryErr(err) {
					printDivergedHistoryGuidance("push")
				}
				os.Exit(1)
			}
		}
		fmt.Println("Push complete.")
	},
}

var doltPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull commits from Dolt remote",
	Long: `Pull commits from the configured Dolt remote into the local database.

Requires a Dolt remote to be configured in the database directory.
For Hosted Dolt, set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD environment
variables for authentication.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}
		fmt.Println("Pulling from Dolt remote...")
		if err := st.Pull(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			if isRemoteNotFoundErr(err) {
				fmt.Fprintf(os.Stderr, "Hint: use 'bd dolt remote add <name> <url>'.\n")
			} else if isDivergedHistoryErr(err) {
				printDivergedHistoryGuidance("pull")
			}
			os.Exit(1)
		}
		fmt.Println("Pull complete.")
	},
}

var doltCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a Dolt commit from pending changes",
	Long: `Create a Dolt commit from any uncommitted changes in the working set.

This is the primary commit point for batch mode. When auto-commit is set to
"batch", changes accumulate in the working set across multiple bd commands and
are committed together here with a descriptive summary message.

Also useful before push operations that require a clean working set, or when
auto-commit was off or changes were made externally.

For more options (--stdin, custom messages), see: bd vc commit`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}
		msg, _ := cmd.Flags().GetString("message")
		if msg == "" {
			committed, err := st.CommitPending(ctx, getActor())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if !committed {
				fmt.Println("Nothing to commit.")
				return
			}
		} else {
			if err := st.Commit(ctx, msg); err != nil {
				errLower := strings.ToLower(err.Error())
				if strings.Contains(errLower, "nothing to commit") || strings.Contains(errLower, "no changes") {
					fmt.Println("Nothing to commit.")
					return
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		commandDidExplicitDoltCommit = true
		fmt.Println("Committed.")
	},
}

// confirmOverwrite prompts the user to confirm overwriting an existing remote.
func confirmOverwrite(surface, name, existingURL, newURL string) bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return true
	}
	fmt.Printf("  Remote %q already exists on %s: %s\n", name, surface, existingURL)
	fmt.Printf("  Overwrite with: %s\n", newURL)
	fmt.Print("  Overwrite? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// --- Dolt remote management commands ---

var doltRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage Dolt remotes",
	Long: `Manage Dolt remotes for push/pull replication.

Subcommands:
  add <name> <url>   Add a new remote
  list               List all configured remotes
  remove <name>      Remove a remote`,
}

var doltRemoteAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a Dolt remote",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}
		name, url := args[0], args[1]

		// Check existing remotes
		sqlRemotes, _ := st.ListRemotes(ctx)
		var sqlURL string
		for _, r := range sqlRemotes {
			if r.Name == name {
				sqlURL = r.URL
				break
			}
		}

		// Prompt for overwrite if remote already exists
		if sqlURL != "" && sqlURL != url {
			if !confirmOverwrite("database", name, sqlURL, url) {
				fmt.Println("Canceled.")
				return
			}
			if err := st.RemoveRemote(ctx, name); err != nil {
				fmt.Fprintf(os.Stderr, "Error removing existing remote: %v\n", err)
				os.Exit(1)
			}
		}

		// Add remote
		if sqlURL != url {
			if err := st.AddRemote(ctx, name, url); err != nil {
				if jsonOutput {
					outputJSONError(err, "remote_add_failed")
				} else {
					fmt.Fprintf(os.Stderr, "Error adding remote: %v\n", err)
				}
				os.Exit(1)
			}
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"name": name,
				"url":  url,
			})
		} else {
			fmt.Printf("Added remote %q -> %s\n", name, url)
		}
	},
}

var doltRemoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured Dolt remotes",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}

		sqlRemotes, sqlErr := st.ListRemotes(ctx)
		if sqlErr != nil {
			if jsonOutput {
				outputJSONError(sqlErr, "remote_list_failed")
			} else {
				fmt.Fprintf(os.Stderr, "Error listing remotes: %v\n", sqlErr)
			}
			os.Exit(1)
		}

		if jsonOutput {
			type remoteEntry struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			}
			var entries []remoteEntry
			for _, r := range sqlRemotes {
				entries = append(entries, remoteEntry{Name: r.Name, URL: r.URL})
			}
			outputJSON(entries)
			return
		}

		if len(sqlRemotes) == 0 {
			fmt.Println("No remotes configured.")
			return
		}

		for _, r := range sqlRemotes {
			fmt.Printf("%-20s %s\n", r.Name, r.URL)
		}
	},
}

var doltRemoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a Dolt remote",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		st := getStore()
		if st == nil {
			fmt.Fprintf(os.Stderr, "Error: no store available\n")
			os.Exit(1)
		}
		name := args[0]

		if err := st.RemoveRemote(ctx, name); err != nil {
			if jsonOutput {
				outputJSONError(err, "remote_remove_failed")
			} else {
				fmt.Fprintf(os.Stderr, "Error removing remote: %v\n", err)
			}
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"name":    name,
				"removed": true,
			})
		} else {
			fmt.Printf("Removed remote %q\n", name)
		}
	},
}

// isTimeoutError checks if an error is a context deadline exceeded or timeout.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func init() {
	doltPushCmd.Flags().Bool("force", false, "Force push (overwrite remote changes)")
	doltCommitCmd.Flags().StringP("message", "m", "", "Commit message (default: auto-generated)")
	doltRemoteRemoveCmd.Flags().Bool("force", false, "Force remove even when URLs conflict")
	doltRemoteCmd.AddCommand(doltRemoteAddCmd)
	doltRemoteCmd.AddCommand(doltRemoteListCmd)
	doltRemoteCmd.AddCommand(doltRemoteRemoveCmd)
	doltCmd.AddCommand(doltCommitCmd)
	doltCmd.AddCommand(doltPushCmd)
	doltCmd.AddCommand(doltPullCmd)
	doltCmd.AddCommand(doltRemoteCmd)
	rootCmd.AddCommand(doltCmd)
}

func selectedDoltBeadsDir() string {
	bdDir := selectedNoDBBeadsDir()
	if bdDir == "" {
		return ""
	}
	prepareSelectedNoDBContext(bdDir)
	return bdDir
}

// extractSSHHost extracts the hostname from an SSH URL for connectivity testing.
func extractSSHHost(url string) string {
	url = strings.TrimPrefix(url, "git+ssh://")
	url = strings.TrimPrefix(url, "ssh://")
	if idx := strings.Index(url, "@"); idx >= 0 {
		url = url[idx+1:]
	}
	if idx := strings.Index(url, ":"); idx >= 0 && !strings.Contains(url[:idx], "/") {
		return url[:idx]
	}
	if idx := strings.Index(url, "/"); idx >= 0 {
		return url[:idx]
	}
	return url
}

// testSSHConnectivity tests if an SSH host is reachable on port 22.
func testSSHConnectivity(host string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "22"), 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// httpURLToTCPAddr extracts a TCP dial address (host:port) from an HTTP(S) URL.
func httpURLToTCPAddr(rawURL string) string {
	host := rawURL
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	defaultPort := "443"
	if strings.HasPrefix(rawURL, "http://") {
		defaultPort = "80"
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		return net.JoinHostPort(h, p)
	}
	h := strings.TrimPrefix(host, "[")
	h = strings.TrimSuffix(h, "]")
	return net.JoinHostPort(h, defaultPort)
}

// testHTTPConnectivity tests if an HTTP(S) URL is reachable via TCP.
func testHTTPConnectivity(rawURL string) bool {
	addr := httpURLToTCPAddr(rawURL)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Ensure doltutil import is used (referenced by other files that use doltutil.IsSSHURL etc.)
var _ = doltutil.IsSSHURL
