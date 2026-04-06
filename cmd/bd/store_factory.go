package main

import (
	"context"
	"path/filepath"

	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/storage/embeddeddolt"
)

// isEmbeddedMode returns true -- embedded Dolt is the only mode now.
func isEmbeddedMode() bool {
	return true
}

// newDoltStore creates an embedded Dolt storage backend.
func newDoltStore(ctx context.Context, beadsDir, database string, opts ...embeddeddolt.Option) (*embeddeddolt.EmbeddedDoltStore, error) {
	return embeddeddolt.New(ctx, beadsDir, database, "main", opts...)
}

// acquireEmbeddedLock acquires an exclusive flock on the embeddeddolt data
// directory derived from beadsDir. The caller must defer lock.Unlock().
func acquireEmbeddedLock(beadsDir string) (embeddeddolt.Unlocker, error) {
	dataDir := filepath.Join(beadsDir, "embeddeddolt")
	return embeddeddolt.TryLock(dataDir)
}

// newDoltStoreFromConfig creates an embedded storage backend from the beads
// directory's persisted metadata.json configuration.
func newDoltStoreFromConfig(ctx context.Context, beadsDir string) (*embeddeddolt.EmbeddedDoltStore, error) {
	cfg, err := configfile.Load(beadsDir)
	database := configfile.DefaultDoltDatabase
	if err == nil && cfg != nil {
		database = cfg.GetDoltDatabase()
	}
	return embeddeddolt.New(ctx, beadsDir, database, "main")
}

// newReadOnlyStoreFromConfig creates a read-only storage backend from the beads
// directory's persisted metadata.json configuration.
func newReadOnlyStoreFromConfig(ctx context.Context, beadsDir string) (*embeddeddolt.EmbeddedDoltStore, error) {
	// Embedded dolt is single-process so read-only is not enforced at the engine level.
	return newDoltStoreFromConfig(ctx, beadsDir)
}
