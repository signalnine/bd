# Dolt backend

bd stores issue data in an embedded [Dolt](https://github.com/dolthub/dolt) database under `.bd/embeddeddolt/`. Dolt is a version-controlled SQL database — think git for tables. You don't need to install Dolt separately; bd statically links the engine.

## Why Dolt

- **Cell-level version control.** Merges happen per-cell, not per-line, so two agents touching different fields on the same issue don't conflict.
- **Native branching and history.** Every write is a Dolt commit. You can branch, diff, and roll back issue state like a git repo.
- **Distributed sync.** `bd dolt push` / `bd dolt pull` talk to any Dolt remote — DoltHub, another clone, a git-hosted Dolt remote.
- **No server.** bd runs the engine in-process. No daemon, no port, no setup beyond `bd init`.

## Database layout

```
.bd/
  config.yaml            # bd config (issue prefix, sync settings)
  embeddeddolt/          # Dolt working directory (don't edit by hand)
    <dbname>/
      .dolt/             # Dolt metadata, refs, commit graph
  backup/                # bd backup sync destination (optional)
  last-touched           # cache file, safe to delete
```

The database name defaults to your issue prefix with hyphens replaced by underscores. Override with `bd init --database <name>`.

## Sync with remotes

```bash
bd dolt remote add origin <url>
bd dolt push
bd dolt pull
```

Supported URL schemes:

- `https://doltremoteapi.dolthub.com/<user>/<repo>` - DoltHub (free for public repos)
- `file:///path/to/remote.dolt` - local filesystem
- `aws://...`, `gs://...`, `oci://...` - cloud object stores (see Dolt docs)
- Regular git URLs - Dolt stores its data under `refs/dolt/data`, invisible to `git clone`

See [SYNC_SETUP.md](SYNC_SETUP.md) for end-to-end setup.

## Manual Dolt access

If you want to poke at the raw tables:

```bash
cd .bd/embeddeddolt/<dbname>
# Requires the standalone dolt CLI: https://github.com/dolthub/dolt/releases
dolt sql
```

Be careful - writes from the Dolt CLI bypass bd's validation and event tracking. Read-only queries are fine.

## Common issues

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for database locks, `no common ancestor` push errors, and performance tips.
