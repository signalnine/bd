package main

import (
	"fmt"
	"strings"

	"github.com/signalnine/bd/internal/types"
	"github.com/signalnine/bd/internal/ui"
	"github.com/signalnine/bd/internal/utils"
	"github.com/spf13/cobra"
)

var commentsCmd = &cobra.Command{
	Use:     "comments <issue-id>",
	GroupID: "issues",
	Short:   "List comments on an issue",
	Long: `List all comments on an issue. To add a comment, use 'bd comment <id> "text"'.

Examples:
  bd comments bd-123
  bd comments bd-123 --json
  bd comments bd-123 --local-time`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		localTime, _ := cmd.Flags().GetBool("local-time")
		issueID := args[0]

		if err := ensureStoreActive(); err != nil {
			FatalErrorRespectJSON("getting comments: %v", err)
		}
		ctx := rootCtx
		fullID, err := utils.ResolvePartialID(ctx, store, issueID)
		if err != nil {
			FatalErrorRespectJSON("resolving %s: %v", issueID, err)
		}
		issueID = fullID

		comments, err := store.GetIssueComments(ctx, issueID)
		if err != nil {
			FatalErrorRespectJSON("getting comments: %v", err)
		}

		// Normalize nil to empty slice for consistent JSON output
		if comments == nil {
			comments = make([]*types.Comment, 0)
		}

		if jsonOutput {
			outputJSON(comments)
			return
		}

		// Human-readable output
		if len(comments) == 0 {
			fmt.Printf("No comments on %s\n", issueID)
			return
		}

		fmt.Printf("\nComments on %s:\n\n", issueID)
		for _, comment := range comments {
			ts := comment.CreatedAt
			if localTime {
				ts = ts.Local()
			}
			fmt.Printf("[%s] at %s\n", comment.Author, ts.Format("2006-01-02 15:04"))
			rendered := ui.RenderMarkdown(comment.Text)
			// TrimRight removes trailing newlines that Glamour adds, preventing extra blank lines
			for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				fmt.Printf("  %s\n", line)
			}
			fmt.Println()
		}
	},
}

func init() {
	commentsCmd.Flags().Bool("local-time", false, "Show timestamps in local time instead of UTC")
	commentsCmd.ValidArgsFunction = issueIDCompletion
	rootCmd.AddCommand(commentsCmd)
}

func isUnknownOperationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "unknown operation")
}
