package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalnine/bd/internal/audit"
	"github.com/signalnine/bd/internal/config"
	"github.com/signalnine/bd/internal/types"
	"github.com/signalnine/bd/internal/ui"
	"github.com/signalnine/bd/internal/utils"
	"github.com/signalnine/bd/internal/validation"
	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:     "close [id...]",
	Aliases: []string{"done"},
	GroupID: "issues",
	Short:   "Close one or more issues",
	Long: `Close one or more issues.

If no issue ID is provided, closes the last touched issue (from most recent
create, update, show, or close operation).`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("close")

		// If no IDs provided, use last touched issue
		if len(args) == 0 {
			lastTouched := GetLastTouchedID()
			if lastTouched == "" {
				FatalErrorRespectJSON("no issue ID provided and no last touched issue")
			}
			args = []string{lastTouched}
		}
		reason, _ := cmd.Flags().GetString("reason")
		if reason == "" {
			// Check --resolution alias (Jira CLI convention)
			reason, _ = cmd.Flags().GetString("resolution")
		}
		if reason == "" {
			// Check -m alias (git commit convention)
			reason, _ = cmd.Flags().GetString("message")
		}
		if reason == "" {
			// Check --comment alias (desire-path from hq-ftpg)
			reason, _ = cmd.Flags().GetString("comment")
		}

		// Desire-path: "bd done <id> <message>" treats last positional arg as reason
		// when no reason flag was explicitly provided (hq-pe8ce)
		if reason == "" && cmd.CalledAs() == "done" && len(args) >= 2 {
			reason = args[len(args)-1]
			args = args[:len(args)-1]
		}

		if reason == "" {
			reason = "Closed"
		}

		// Validate close reason if configured
		closeValidation := config.GetString("validation.on-close")
		if closeValidation == "error" || closeValidation == "warn" {
			if err := validation.ValidateCloseReason(reason); err != nil {
				if closeValidation == "error" {
					FatalErrorRespectJSON("%v", err)
				}
				// warn mode: print warning but proceed
				fmt.Fprintf(os.Stderr, "%s %v\n", ui.RenderWarn("⚠"), err)
			}
		}

		force, _ := cmd.Flags().GetBool("force")
		suggestNext, _ := cmd.Flags().GetBool("suggest-next")

		claimNext, _ := cmd.Flags().GetBool("claim-next")

		// Get session ID from flag or environment variable
		session, _ := cmd.Flags().GetString("session")
		if session == "" {
			session = os.Getenv("CLAUDE_SESSION_ID")
		}

		ctx := rootCtx

		// --suggest-next only works with a single issue
		if suggestNext && len(args) > 1 {
			FatalErrorRespectJSON("--suggest-next only works when closing a single issue")
		}

		// Resolve partial IDs
		var resolvedIDs []string
		for _, id := range args {
			resolved, err := utils.ResolvePartialID(ctx, store, id)
			if err != nil {
				FatalErrorRespectJSON("resolving ID %s: %v", id, err)
			}
			resolvedIDs = append(resolvedIDs, resolved)
		}

		// Direct mode
		closedIssues := []*types.Issue{}
		closedCount := 0

		for _, id := range resolvedIDs {
			// Get issue for checks (nil issue is handled by validateIssueClosable)
			issue, _ := store.GetIssue(ctx, id)

			if err := validateIssueClosable(id, issue, force); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				continue
			}

			// Epic close guard: prevent closing epics with open children (mw-local-4so.5.2)
			if !force && issue != nil && issue.IssueType == types.TypeEpic {
				openChildren := countEpicOpenChildren(ctx, id)
				if openChildren > 0 {
					fmt.Fprintf(os.Stderr, "cannot close epic %s: %d open child issue(s); close children first or use --force to override\n", id, openChildren)
					continue
				}
			}

			// Check if issue has open blockers (GH#962)
			if !force {
				blocked, blockers, err := store.IsBlocked(ctx, id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error checking blockers for %s: %v\n", id, err)
					continue
				}
				if blocked && len(blockers) > 0 {
					fmt.Fprintf(os.Stderr, "cannot close %s: blocked by open issues %v (use --force to override)\n", id, blockers)
					continue
				}
			}

			if err := store.CloseIssue(ctx, id, reason, actor, session); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing %s: %v\n", id, err)
				continue
			}

			// Audit log the close (survives Dolt GC flatten)
			oldStatus := "open"
			if issue != nil {
				oldStatus = string(issue.Status)
			}
			audit.LogFieldChange(id, "status", oldStatus, "closed", actor, reason)

			closedCount++

			// Re-fetch for display
			closedIssue, _ := store.GetIssue(ctx, id)

			if jsonOutput {
				if closedIssue != nil {
					closedIssues = append(closedIssues, closedIssue)
				}
			} else {
				fmt.Printf("%s Closed %s: %s\n", ui.RenderPass("✓"), formatFeedbackID(id, issueTitleOrEmpty(issue)), reason)
			}
		}

		// Handle --suggest-next flag in direct mode
		if suggestNext && len(resolvedIDs) == 1 && closedCount > 0 {
			unblocked, err := store.GetNewlyUnblockedByClose(ctx, resolvedIDs[0])
			if err == nil && len(unblocked) > 0 {
				if jsonOutput {
					outputJSON(map[string]interface{}{
						"closed":    closedIssues,
						"unblocked": unblocked,
					})
					return
				}
				fmt.Printf("\nNewly unblocked:\n")
				for _, issue := range unblocked {
					fmt.Printf("  • %s (P%d)\n", formatFeedbackID(issue.ID, issue.Title), issue.Priority)
				}
			}
		}

		// Handle --claim-next flag
		var claimedNextIssue *types.Issue
		if claimNext && closedCount > 0 {
			readyIssues, err := store.GetReadyWork(ctx, types.WorkFilter{
				Status:     "open",
				Limit:      1,
				SortPolicy: types.SortPolicy("priority"),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not get ready issues: %v\n", err)
			} else if len(readyIssues) > 0 {
				nextIssue := readyIssues[0]
				err := store.ClaimIssue(ctx, nextIssue.ID, actor)
				if err == nil {
					claimedNextIssue = nextIssue
					if jsonOutput {
						// JSON handled below
					} else {
						fmt.Printf("%s Auto-claimed next ready issue: %s (P%d)\n", ui.RenderPass("✓"), formatFeedbackID(nextIssue.ID, nextIssue.Title), nextIssue.Priority)
					}
					SetLastTouchedID(nextIssue.ID)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: could not claim next issue %s: %v\n", nextIssue.ID, err)
				}
			} else if !jsonOutput {
				fmt.Printf("\n%s No ready issues available to claim.\n", ui.RenderWarn("✨"))
			}
		}

		if jsonOutput && len(closedIssues) > 0 {
			if claimedNextIssue != nil {
				outputJSON(map[string]interface{}{
					"closed":  closedIssues,
					"claimed": claimedNextIssue,
				})
			} else {
				outputJSON(closedIssues)
			}
		}

		// Embedded mode: flush Dolt commit. DoltStore commits
		// inline during CloseIssue so this is only needed for EmbeddedDoltStore.
		if closedCount > 0 && store != nil {
			if _, err := store.CommitPending(ctx, actor); err != nil {
				FatalErrorRespectJSON("failed to commit: %v", err)
			}
		}

		// Exit non-zero if no issues were actually closed (close guard
		// and other soft failures should surface as non-zero exit codes for scripting)
		totalAttempted := len(resolvedIDs)
		if totalAttempted > 0 && closedCount == 0 {
			os.Exit(1)
		}
	},
}

func init() {
	closeCmd.Flags().StringP("reason", "r", "", "Reason for closing")
	closeCmd.Flags().String("resolution", "", "Alias for --reason (Jira CLI convention)")
	_ = closeCmd.Flags().MarkHidden("resolution") // Hidden alias for agent/CLI ergonomics
	closeCmd.Flags().StringP("message", "m", "", "Alias for --reason (git commit convention)")
	_ = closeCmd.Flags().MarkHidden("message") // Hidden alias for agent/CLI ergonomics
	closeCmd.Flags().String("comment", "", "Alias for --reason")
	_ = closeCmd.Flags().MarkHidden("comment") // Hidden alias for agent/CLI ergonomics
	closeCmd.Flags().BoolP("force", "f", false, "Force close pinned issues")
	closeCmd.Flags().Bool("suggest-next", false, "Show newly unblocked issues after closing")
	closeCmd.Flags().Bool("claim-next", false, "Automatically claim the next highest priority available issue")
	closeCmd.Flags().String("session", "", "Claude Code session ID (or set CLAUDE_SESSION_ID env var)")
	closeCmd.ValidArgsFunction = issueIDCompletion
	rootCmd.AddCommand(closeCmd)
}

// countEpicOpenChildren returns the number of open (non-closed) children for an epic.
// Uses GetDependentsWithMetadata to find parent-child relationships.
func countEpicOpenChildren(ctx context.Context, epicID string) int {
	dependents, err := store.GetDependentsWithMetadata(ctx, epicID)
	if err != nil {
		return 0
	}
	count := 0
	for _, dep := range dependents {
		if dep.DependencyType == types.DepParentChild && dep.Issue.Status != types.StatusClosed {
			count++
		}
	}
	return count
}
