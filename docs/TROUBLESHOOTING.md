# Troubleshooting bd

Common issues and how to fix them. If something here is out of date, open a [GitHub issue](https://github.com/signalnine/bd/issues).

## Debug environment variables

| Variable              | What it enables                                  |
|-----------------------|--------------------------------------------------|
| `BD_DEBUG=1`          | General debug logging to stderr                  |
| `BD_DEBUG_ROUTING=1`  | Multi-repo routing decisions                     |

Example:

```bash
BD_DEBUG=1 bd ready 2> debug.log
```

## `bd: command not found`

`bd` is not on your `PATH`. If you installed with the curl script or `make install`, the binary landed in `~/.local/bin` — add that to your shell profile:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then `hash -r` or restart your shell.

## `zsh: killed bd` / crashes on macOS

Embedded Dolt needs CGO. If you built from source without the Makefile, rebuild with:

```bash
CGO_ENABLED=1 make install
```

On Apple Silicon also make sure ICU4C from Homebrew is installed (`brew install icu4c`). The Makefile handles the rest.

## Antivirus flags `bd.exe` as a trojan

This is a known industry-wide false positive for Go binaries on Windows. Our releases are code-signed (see `.goreleaser.yml`), which mitigates but does not fully eliminate the flag.

Options:

- Allow `bd.exe` in your AV.
- Submit to the vendor as a false positive.
- Build from source locally (won't trigger heuristics tied to downloaded binaries).

## `database is locked`

Another bd process is holding the embedded Dolt lock on `.bd/`. Check for stuck processes:

```bash
ps aux | grep 'bd '
```

Kill anything hung. Do NOT delete files under `.bd/embeddeddolt/` — that corrupts the database.

## `bd init` refuses to run: "already initialized"

The directory already has a `.bd/`. Options:

- Keep existing data: just run the bd commands you wanted — no re-init needed.
- Destroy and restart: `bd init --force` (interactive confirmation required).
- Non-interactive destroy: `bd init --force --destroy-token=DESTROY-<prefix>`.

Before destroying, back up with `bd backup init <path> && bd backup sync`.

## `bd ready` shows nothing but I have open issues

`bd ready` only shows unblocked open issues. An issue is blocked if any of its `blocks` or `parent-child` dependencies is still open. Use `bd blocked` to see what's held up, and `bd list --status open` for the full open list.

## Circular dependency error on `bd dep add`

bd refuses cycles in blocking dependencies. Remove one of the edges first:

```bash
bd dep cycles      # find them
bd dep remove <blocker> <blocked>
```

## Dependencies not showing up

`bd dep list <id>` defaults to outgoing dependencies. Use `bd dep list <id> --reverse` for what depends on this issue.

## Sync: `dolt push` says "no common ancestor"

The bd database's Dolt history has diverged from (or was never connected to) the remote. Either:

- Configure a fresh remote: `bd dolt remote add origin <url>`.
- On the remote side, clone from your local state first.

Full sync setup: [SYNC_SETUP.md](SYNC_SETUP.md).

## Performance: commands feel slow

bd is designed to be fast (ms per command). If you're seeing seconds, the usual cause is a very large `.bd/embeddeddolt/` database. Check:

```bash
du -sh .bd/embeddeddolt
```

If it's >100MB, you may benefit from:

- `bd dolt commit` (flush pending changes)
- `dolt gc` inside `.bd/embeddeddolt/` (manual — requires separate Dolt CLI)

Performance testing and profiling: [PERFORMANCE_TESTING.md](PERFORMANCE_TESTING.md).

## Filing a bug

Please include:

- `bd version`
- The exact command you ran
- Full error output (use `BD_DEBUG=1` to get more detail)
- OS and arch

[https://github.com/signalnine/bd/issues](https://github.com/signalnine/bd/issues)
