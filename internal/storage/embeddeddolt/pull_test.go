//go:build cgo

package embeddeddolt_test

import (
	"path/filepath"
	"testing"
)

// TestPull_InitCreatedStore is a regression test for pull on a database
// created via `bd init` + `bd dolt remote add`. That path never writes
// dolt's branch-upstream config (repo_state.json "branches" stays empty),
// so a pull that names only the remote fails with "you asked to pull from
// the remote 'origin', but did not specify a branch". Pull must pass the
// branch explicitly, as Push already does.
func TestPull_InitCreatedStore(t *testing.T) {
	ctx := t.Context()
	env := newTestEnv(t, "pullrt")
	remoteURL := "file://" + filepath.ToSlash(filepath.Join(t.TempDir(), "remote"))

	if err := env.store.AddRemote(ctx, "origin", remoteURL); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}
	if err := env.store.Push(ctx); err != nil {
		t.Fatalf("Push: %v", err)
	}

	if err := env.store.Pull(ctx); err != nil {
		t.Fatalf("Pull on init-created store: %v", err)
	}
}
