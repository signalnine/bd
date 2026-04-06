package main

import (
	"context"
	"path/filepath"

	"github.com/steveyegge/bd/internal/configfile"
	"github.com/steveyegge/bd/internal/storage/embeddeddolt"
)

// isEmbeddedMode returns true -- embedded Dolt is the only mode now.
func isEmbeddedMode() bool {
	return true
}

// newDoltStore creates an embedded Dolt storage backend.
func newDoltStore(ctx context.Context, bdDir, database string, opts ...embeddeddolt.Option) (*embeddeddolt.EmbeddedDoltStore, error) {
	return embeddeddolt.New(ctx, bdDir, database, "main", opts...)
}

// acquireEmbeddedLock acquires an exclusive flock on the embeddeddolt data
// directory derived from bdDir. The caller must defer lock.Unlock().
func acquireEmbeddedLock(bdDir string) (embeddeddolt.Unlocker, error) {
	dataDir := filepath.Join(bdDir, "embeddeddolt")
	return embeddeddolt.TryLock(dataDir)
}

// newDoltStoreFromConfig creates an embedded storage backend from the beads
// directory's persisted metadata.json configuration.
func newDoltStoreFromConfig(ctx context.Context, bdDir string) (*embeddeddolt.EmbeddedDoltStore, error) {
	cfg, err := configfile.Load(bdDir)
	database := configfile.DefaultDoltDatabase
	if err == nil && cfg != nil {
		database = cfg.GetDoltDatabase()
	}
	return embeddeddolt.New(ctx, bdDir, database, "main")
}

// newReadOnlyStoreFromConfig creates a read-only storage backend from the beads
// directory's persisted metadata.json configuration.
func newReadOnlyStoreFromConfig(ctx context.Context, bdDir string) (*embeddeddolt.EmbeddedDoltStore, error) {
	// Embedded dolt is single-process so read-only is not enforced at the engine level.
	return newDoltStoreFromConfig(ctx, bdDir)
}
