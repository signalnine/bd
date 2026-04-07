package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signalnine/bd/internal/configfile"
	"github.com/signalnine/bd/internal/project"
	"github.com/signalnine/bd/internal/storage/embeddeddolt"
	"github.com/signalnine/bd/internal/utils"
)

// openReadOnlyStoreForDBPath reopens a read-only store from an existing dbPath
// while preserving repo-local metadata such as dolt_database and the resolved
// Dolt server port. Falls back to deriving the bd directory from the dbPath
// parent when no matching metadata.json can be found.
func openReadOnlyStoreForDBPath(ctx context.Context, dbPath string) (*embeddeddolt.EmbeddedDoltStore, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("no database path available")
	}

	if bdDir := resolveBdDirForDBPath(dbPath); bdDir != "" {
		return newReadOnlyStoreFromConfig(ctx, bdDir)
	}

	// Fallback: derive bd dir from dbPath parent directory.
	return newReadOnlyStoreFromConfig(ctx, filepath.Dir(dbPath))
}

// resolveBdDirForDBPath maps a database path back to its owning .bd
// directory when metadata.json is available. This is needed for repos that use
// non-default dolt_database names or custom dolt_data_dir locations.
func resolveBdDirForDBPath(dbPath string) string {
	actualDBPath := utils.CanonicalizePath(dbPath)
	if parent := filepath.Dir(dbPath); filepath.Base(parent) == ".bd" {
		if _, err := os.Stat(filepath.Join(parent, "metadata.json")); err == nil {
			return parent
		}
	}
	if parent := filepath.Dir(actualDBPath); filepath.Base(parent) == ".bd" {
		if _, err := os.Stat(filepath.Join(parent, "metadata.json")); err == nil {
			return parent
		}
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 16)

	addCandidate := func(path string) {
		if path == "" {
			return
		}
		key := utils.NormalizePathForComparison(path)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, path)
	}

	addAncestorCandidates := func(path string) {
		for dir := path; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			addCandidate(filepath.Join(dir, ".bd"))
			if filepath.Base(dir) == ".bd" {
				addCandidate(dir)
			}
		}
	}

	if info, err := os.Stat(dbPath); err == nil && info.IsDir() {
		addCandidate(dbPath)
	}
	if info, err := os.Stat(actualDBPath); err == nil && info.IsDir() {
		addCandidate(actualDBPath)
	}

	addCandidate(filepath.Dir(dbPath))
	addCandidate(filepath.Dir(actualDBPath))
	addAncestorCandidates(filepath.Dir(dbPath))
	addAncestorCandidates(filepath.Dir(actualDBPath))

	if found := project.FindBdDir(); found != "" {
		addCandidate(found)
		addCandidate(utils.CanonicalizePath(found))
	}

	for _, bdDir := range candidates {
		cfg, err := configfile.Load(bdDir)
		if err != nil || cfg == nil {
			continue
		}
		if utils.PathsEqual(bdDir, dbPath) || utils.PathsEqual(bdDir, actualDBPath) {
			return bdDir
		}
		if utils.PathsEqual(cfg.DatabasePath(bdDir), dbPath) || utils.PathsEqual(cfg.DatabasePath(bdDir), actualDBPath) {
			return bdDir
		}
	}

	return ""
}
