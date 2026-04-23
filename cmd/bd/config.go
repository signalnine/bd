package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/signalnine/bd/internal/config"
	"github.com/signalnine/bd/internal/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:     "config",
	GroupID: "setup",
	Short:   "Manage configuration settings",
	Long: `Manage configuration settings for external integrations and preferences.

Configuration is stored per-project in the bd database and is version-control-friendly.

Common namespaces:
  - jira.*            Jira integration settings
  - linear.*          Linear integration settings
  - github.*          GitHub integration settings
  - custom.*          Custom integration settings
  - status.*          Issue status configuration

Custom Status States:
  You can define custom status states for multi-step pipelines using the
  status.custom config key. Statuses should be comma-separated.

  Example:
    bd config set status.custom "awaiting_review,awaiting_testing,awaiting_docs"

  This enables issues to use statuses like 'awaiting_review' in addition to
  the built-in statuses (open, in_progress, blocked, deferred, closed).

Examples:
  bd config set jira.url "https://company.atlassian.net"
  bd config set jira.project "PROJ"
  bd config set status.custom "awaiting_review,awaiting_testing"
  bd config get jira.url
  bd config list
  bd config unset jira.url`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Check if this is a yaml-only key (startup settings like no-db, etc.)
		// These must be written to config.yaml, not SQLite, because they're read
		// before the database is opened. (GH#536)
		if config.IsYamlOnlyKey(key) {
			if err := config.SetYamlConfig(key, value); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting config: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"value":    value,
					"location": "config.yaml",
				})
			} else {
				fmt.Printf("Set %s = %s (in config.yaml)\n", key, value)
			}
			return
		}

		// bd.role is stored in git config, not SQLite (GH#1531).
		// bd doctor reads it from git config, so we write there for consistency.
		if key == "bd.role" {
			validRoles := map[string]bool{"maintainer": true, "contributor": true}
			if !validRoles[value] {
				fmt.Fprintf(os.Stderr, "Error: invalid role %q (valid values: maintainer, contributor)\n", value)
				os.Exit(1)
			}
			cmd := exec.Command("git", "config", "bd.role", value) //nolint:gosec // value is validated against allowlist above
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting bd.role in git config: %v\n", err)
				os.Exit(1)
			}
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"value":    value,
					"location": "git config",
				})
			} else {
				fmt.Printf("Set %s = %s (in git config)\n", key, value)
			}
			return
		}

		// Database-stored config requires direct mode
		if err := ensureDirectMode("config set requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx

		// Validate status.custom config before writing
		if key == "status.custom" && value != "" {
			if _, err := types.ParseCustomStatusConfig(value); err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid status.custom value: %v\n", err)
				os.Exit(1)
			}
		}

		if err := store.SetConfig(ctx, key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]string{
				"key":   key,
				"value": value,
			})
		} else {
			fmt.Printf("Set %s = %s\n", key, value)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		// Check if this is a yaml-only key (startup settings)
		// These are read from config.yaml via viper, not SQLite. (GH#536)
		if config.IsYamlOnlyKey(key) {
			value := config.GetYamlConfig(key)

			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"value":    value,
					"location": "config.yaml",
				})
			} else {
				if value == "" {
					fmt.Printf("%s (not set in config.yaml)\n", key)
				} else {
					fmt.Printf("%s\n", value)
				}
			}
			return
		}

		// bd.role is stored in git config, not SQLite (GH#1531).
		if key == "bd.role" {
			cmd := exec.Command("git", "config", "--get", "bd.role")
			output, err := cmd.Output()
			value := strings.TrimSpace(string(output))
			if err != nil {
				value = ""
			}
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"value":    value,
					"location": "git config",
				})
			} else {
				if value == "" {
					fmt.Printf("%s (not set in git config)\n", key)
				} else {
					fmt.Printf("%s\n", value)
				}
			}
			return
		}

		// Database-stored config requires direct mode
		if err := ensureDirectMode("config get requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx
		var value string
		var err error

		value, err = store.GetConfig(ctx, key)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting config: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]string{
				"key":   key,
				"value": value,
			})
		} else {
			if value == "" {
				fmt.Printf("%s (not set)\n", key)
			} else {
				fmt.Printf("%s\n", value)
			}
		}
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration",
	Run: func(cmd *cobra.Command, args []string) {
		// Config operations work in direct mode only
		if err := ensureDirectMode("config list requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx
		config, err := store.GetAllConfig(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing config: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(config)
			return
		}

		if len(config) == 0 {
			fmt.Println("No configuration set")
			return
		}

		// Sort keys for consistent output
		keys := make([]string, 0, len(config))
		for k := range config {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Println("\nConfiguration:")
		for _, k := range keys {
			fmt.Printf("  %s = %s\n", k, config[k])
		}

		// Check for config.yaml overrides that take precedence (bd-20j)
		// This helps diagnose when effective config differs from database config
		showConfigYAMLOverrides(config)
	},
}

// showConfigYAMLOverrides warns when config.yaml or env vars override database settings.
// This addresses the confusion when `bd config list` shows one value but the effective
// value used by commands is different due to higher-priority config sources.
func showConfigYAMLOverrides(dbConfig map[string]string) {
	var warnings []string

	// Check each DB config key for env var overrides
	for key, dbValue := range dbConfig {
		envKey := "BD_" + strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), ".", "_"))
		if envValue := os.Getenv(envKey); envValue != "" && envValue != dbValue {
			warnings = append(warnings, fmt.Sprintf("  %s: DB has %q, but env %s=%q takes precedence", key, dbValue, envKey, envValue))
		}
	}

	// Check for yaml-only keys set in config.yaml that aren't visible in DB output
	yamlKeys := []string{
		"no-db", "json", "actor", "identity",
		"routing.mode", "routing.default", "routing.maintainer", "routing.contributor",
		"sync.git-remote", "no-push", "no-git-ops",
		"git.author", "git.no-gpg-sign",
		"create.require-description",
		"validation.on-create", "validation.on-close", "validation.on-sync",
		"hierarchy.max-depth",
		"backup.enabled", "backup.interval", "backup.git-push", "backup.git-repo",
		"dolt.idle-timeout", "dolt.shared-server",
	}

	var yamlOverrides []string
	for _, key := range yamlKeys {
		val := config.GetYamlConfig(key)
		if val != "" && config.GetValueSource(key) == config.SourceConfigFile {
			yamlOverrides = append(yamlOverrides, fmt.Sprintf("  %s = %s", key, val))
		}
	}

	if len(yamlOverrides) > 0 {
		fmt.Println("\nAlso set in config.yaml (not shown above):")
		for _, line := range yamlOverrides {
			fmt.Println(line)
		}
	}

	if len(warnings) > 0 {
		sort.Strings(warnings)
		fmt.Println("\n⚠ Environment variable overrides detected:")
		for _, w := range warnings {
			fmt.Println(w)
		}
	}
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Delete a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		// Check if this is a yaml-only key (startup settings like backup.*, routing.*, etc.)
		// These must be removed from config.yaml, not the database. (GH#2727)
		if config.IsYamlOnlyKey(key) {
			if err := config.UnsetYamlConfig(key); err != nil {
				fmt.Fprintf(os.Stderr, "Error unsetting config: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"location": "config.yaml",
				})
			} else {
				fmt.Printf("Unset %s (in config.yaml)\n", key)
			}
			return
		}

		// bd.role is stored in git config, not the database (GH#1531).
		if key == "bd.role" {
			gitCmd := exec.Command("git", "config", "--unset", "bd.role")
			if err := gitCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error unsetting bd.role in git config: %v\n", err)
				os.Exit(1)
			}
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"key":      key,
					"location": "git config",
				})
			} else {
				fmt.Printf("Unset %s (in git config)\n", key)
			}
			return
		}

		// Database-stored config requires direct mode
		if err := ensureDirectMode("config unset requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx
		if err := store.DeleteConfig(ctx, key); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting config: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]string{
				"key": key,
			})
		} else {
			fmt.Printf("Unset %s\n", key)
		}
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate sync-related configuration",
	Long: `Validate sync-related configuration settings.

Checks:
  - federation.sovereignty is valid (T1, T2, T3, T4, or empty)
  - federation.remote, if set, has a valid URL format (http://, https://, git+)

Note: Dolt remote configuration for 'bd dolt push/pull' is managed via
'bd dolt remote add <name> <url>' and stored in the dolt_remotes SQL
table, not via the federation.remote config key.

Examples:
  bd config validate
  bd config validate --json`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Find repo root by walking up to find .bd directory
		repoPath := findBdRepoRoot(cwd)
		if repoPath == "" {
			fmt.Fprintf(os.Stderr, "Error: not in a bd repository (no .bd directory found)\n")
			os.Exit(1)
		}

		// Run sync-related validations
		allIssues := validateSyncConfig(repoPath)

		// Output results
		if jsonOutput {
			result := map[string]interface{}{
				"valid":  len(allIssues) == 0,
				"issues": allIssues,
			}
			outputJSON(result)
			return
		}

		if len(allIssues) == 0 {
			fmt.Println("All sync-related configuration is valid")
			return
		}

		fmt.Println("Configuration validation found issues:")
		for _, issue := range allIssues {
			if issue != "" {
				fmt.Printf("  - %s\n", issue)
			}
		}
		fmt.Println("\nRun 'bd config set <key> <value>' to fix configuration issues.")
		os.Exit(1)
	},
}

// validateSyncConfig performs additional sync-related config validation
// beyond what doctor.CheckConfigValues covers.
func validateSyncConfig(repoPath string) []string {
	var issues []string

	// Load config.yaml directly from the repo path
	configPath := filepath.Join(repoPath, ".bd", "config.yaml")
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)

	// Try to read config, but don't error if it doesn't exist
	if err := v.ReadInConfig(); err != nil {
		// Config file doesn't exist or is unreadable - nothing to validate
		return issues
	}

	// Get config from yaml
	federationSov := v.GetString("federation.sovereignty")
	federationRemote := v.GetString("federation.remote")

	// Validate federation.sovereignty
	if federationSov != "" && !config.IsValidSovereignty(federationSov) {
		issues = append(issues, fmt.Sprintf("federation.sovereignty: %q is invalid (valid values: %s, or empty for no restriction)", federationSov, strings.Join(config.ValidSovereigntyTiers(), ", ")))
	}

	// federation.remote is not required: bd dolt push/pull/remote operate on
	// the SQL dolt_remotes table (managed by 'bd dolt remote add'), not this
	// config key. If it is set, only validate its format.
	if federationRemote != "" {
		if !isValidRemoteURL(federationRemote) {
			issues = append(issues, fmt.Sprintf("federation.remote: %q is not a valid remote URL (expected http://, https://, or git+ URL)", federationRemote))
		}
	}

	return issues
}

// isValidRemoteURL validates remote URL formats for sync configuration.
func isValidRemoteURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "git+")
}

// findBdRepoRoot walks up from the given path to find the repo root (containing .bd)
func findBdRepoRoot(startPath string) string {
	path := startPath
	for {
		bdDir := filepath.Join(path, ".bd")
		if info, err := os.Stat(bdDir); err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}

var configSetManyCmd = &cobra.Command{
	Use:   "set-many <key=value>...",
	Short: "Set multiple configuration values in one operation",
	Long: `Set multiple configuration values at once with a single auto-commit and auto-push.

Each argument must be in key=value format. All values are validated before
any writes occur. This is faster and less noisy than separate 'bd config set'
calls, especially in CI.

Examples:
  bd config set-many ado.state_map.open=New ado.state_map.closed=Closed
  bd config set-many jira.url=https://example.atlassian.net jira.project=PROJ`,
	Args: cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		// Phase 1: Parse all key=value pairs
		type kvPair struct {
			key, value string
		}
		pairs := make([]kvPair, 0, len(args))
		for _, arg := range args {
			idx := strings.Index(arg, "=")
			if idx <= 0 {
				fmt.Fprintf(os.Stderr, "Error: invalid argument %q (expected key=value format)\n", arg)
				os.Exit(1)
			}
			pairs = append(pairs, kvPair{key: arg[:idx], value: arg[idx+1:]})
		}

		// Phase 2: Validate all pairs before writing any
		for _, p := range pairs {
			if p.key == "bd.role" {
				validRoles := map[string]bool{"maintainer": true, "contributor": true}
				if !validRoles[p.value] {
					fmt.Fprintf(os.Stderr, "Error: invalid role %q (valid values: maintainer, contributor)\n", p.value)
					os.Exit(1)
				}
			}
			if p.key == "status.custom" && p.value != "" {
				if _, err := types.ParseCustomStatusConfig(p.value); err != nil {
					fmt.Fprintf(os.Stderr, "Error: invalid status.custom value: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Phase 3: Separate into categories
		var yamlPairs, gitPairs, dbPairs []kvPair
		for _, p := range pairs {
			if config.IsYamlOnlyKey(p.key) {
				yamlPairs = append(yamlPairs, p)
			} else if p.key == "bd.role" {
				gitPairs = append(gitPairs, p)
			} else {
				dbPairs = append(dbPairs, p)
			}
		}

		// Phase 4: Write yaml-only keys
		for _, p := range yamlPairs {
			if err := config.SetYamlConfig(p.key, p.value); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting config %s: %v\n", p.key, err)
				os.Exit(1)
			}
		}

		// Phase 5: Write git config keys
		for _, p := range gitPairs {
			cmd := exec.Command("git", "config", "bd.role", p.value) //nolint:gosec // value is validated against allowlist above
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting %s in git config: %v\n", p.key, err)
				os.Exit(1)
			}
		}

		// Phase 6: Write DB keys in batch
		if len(dbPairs) > 0 {
			if err := ensureDirectMode("config set-many requires direct database access"); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			ctx := rootCtx
			for _, p := range dbPairs {
				if err := store.SetConfig(ctx, p.key, p.value); err != nil {
					fmt.Fprintf(os.Stderr, "Error setting config %s: %v\n", p.key, err)
					os.Exit(1)
				}
			}
		}

		// Phase 7: Output results
		if jsonOutput {
			results := make([]map[string]string, 0, len(pairs))
			for _, p := range pairs {
				location := "database"
				if config.IsYamlOnlyKey(p.key) {
					location = "config.yaml"
				} else if p.key == "bd.role" {
					location = "git config"
				}
				results = append(results, map[string]string{
					"key":      p.key,
					"value":    p.value,
					"location": location,
				})
			}
			outputJSON(results)
		} else {
			for _, p := range pairs {
				location := ""
				if config.IsYamlOnlyKey(p.key) {
					location = " (in config.yaml)"
				} else if p.key == "bd.role" {
					location = " (in git config)"
				}
				fmt.Printf("Set %s = %s%s\n", p.key, p.value, location)
			}
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configSetManyCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
