package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/signalnine/bd/internal/project"
	"github.com/signalnine/bd/internal/utils"
)

// WhereResult contains information about the active beads location
type WhereResult struct {
	Path           string `json:"path"`                      // Active .beads directory path
	RedirectedFrom string `json:"redirected_from,omitempty"` // Original path if redirected
	Prefix         string `json:"prefix,omitempty"`          // Issue prefix (if detectable)
	DatabasePath   string `json:"database_path,omitempty"`   // Full path to database file
}

var whereCmd = &cobra.Command{
	Use:     "where",
	GroupID: "setup",
	Short:   "Show active bd location",
	Long: `Show the active bd database location, including redirect information.

This command is useful for debugging when using redirects, to understand
which .bd directory is actually being used.

Examples:
  bd where           # Show active bd location
  bd where --json    # Output in JSON format
`,
	Run: func(cmd *cobra.Command, args []string) {
		result := WhereResult{}

		// Find the beads directory (this follows redirects)
		bdDir := project.FindBdDir()
		if bdDir == "" {
			if jsonOutput {
				outputJSON(map[string]string{"error": "no bd directory found"})
			} else {
				fmt.Fprintln(os.Stderr, "Error: no bd directory found")
				fmt.Fprintln(os.Stderr, "Hint: "+diagHint())
			}
			os.Exit(1)
		}

		result.Path = bdDir

		// Check if we got here via redirect by looking for the original .beads directory
		// Walk up from cwd to find any .beads with a redirect file
		originalBdDir := findOriginalBdDir()
		if originalBdDir != "" && originalBdDir != bdDir {
			result.RedirectedFrom = originalBdDir
		}

		// Find the database path
		dbPath := project.FindDatabasePath()
		if dbPath != "" {
			result.DatabasePath = dbPath

			// Try to get the prefix from the database if we have a store
			if store != nil {
				ctx := rootCtx
				if prefix, err := store.GetConfig(ctx, "issue_prefix"); err == nil && prefix != "" {
					result.Prefix = prefix
				}
			}
		}

		// If we don't have the prefix from DB, try to detect it from JSONL
		if result.Prefix == "" {
			result.Prefix = detectPrefixFromDir(bdDir)
		}

		// Output results
		if jsonOutput {
			outputJSON(result)
		} else {
			fmt.Println(result.Path)
			if result.RedirectedFrom != "" {
				fmt.Printf("  (via redirect from %s)\n", result.RedirectedFrom)
			}
			if result.Prefix != "" {
				fmt.Printf("  prefix: %s\n", result.Prefix)
			}
			if result.DatabasePath != "" {
				fmt.Printf("  database: %s\n", result.DatabasePath)
			}
		}
	},
}

// findOriginalBdDir walks up from cwd looking for a .beads directory with a redirect file
// Returns the original .beads path if found, empty string otherwise
func findOriginalBdDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Canonicalize cwd to handle symlinks
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}

	// Check BD_DIR first
	if envDir := os.Getenv("BD_DIR"); envDir != "" {
		envDir = utils.CanonicalizePath(envDir)
		redirectFile := filepath.Join(envDir, project.RedirectFileName)
		if _, err := os.Stat(redirectFile); err == nil {
			return envDir
		}
		return ""
	}

	// Walk up directory tree looking for .beads with redirect
	for dir := cwd; dir != "/" && dir != "."; {
		bdDir := filepath.Join(dir, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			redirectFile := filepath.Join(bdDir, project.RedirectFileName)
			if _, err := os.Stat(redirectFile); err == nil {
				return bdDir
			}
			// Found .beads without redirect - this is the actual location
			return ""
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

// detectPrefixFromDir tries to detect the issue prefix from files in the beads directory.
// Returns empty string if prefix cannot be determined.
func detectPrefixFromDir(_ string) string {
	return ""
}

func init() {
	rootCmd.AddCommand(whereCmd)
}
