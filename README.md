<p align="center">
  <img src="https://signalnine.s3.amazonaws.com/disembed-revivalists.jpeg" alt="bd" width="50%" />
</p>

# bd

**Issue tracker for AI agents. 20 commands. One database. No bullshit.**

bd gives coding agents a persistent, dependency-aware task graph backed by [Dolt](https://github.com/dolthub/dolt) -- a version-controlled SQL database. Agents create issues, track blockers, close work, and keep context across sessions.

We forked [beads](https://github.com/steveyegge/beads) and deleted 78% of it.

## Install

```bash
# From source (requires Go 1.25+ and CGO)
git clone https://github.com/signalnine/bd
cd bd
make install    # builds and installs to ~/.local/bin/bd
```

## 30-second demo

```bash
cd your-project
bd init

bd create "Rewrite auth module" -p 1 -t feature
bd create "Add rate limiting" -p 2 -t task
bd dep add bd-a1b2 bd-c3d4          # c3d4 blocks a1b2

bd ready                             # shows c3d4 (unblocked)
bd update bd-c3d4 --claim            # assign to yourself
bd close bd-c3d4 "Done"
bd ready                             # now shows a1b2
```

## Commands

| | | |
|---|---|---|
| `bd init` | `bd create` | `bd list` |
| `bd show` | `bd update` | `bd edit` |
| `bd close` | `bd reopen` | `bd delete` |
| `bd dep add` | `bd dep remove` | `bd ready` |
| `bd search` | `bd comment` | `bd label` |
| `bd config` | `bd backup` | `bd version` |
| `bd dolt push` | `bd dolt pull` | |

Every command supports `--json`.

## How it works

bd stores issues in an embedded [Dolt](https://github.com/dolthub/dolt) database inside `.bd/` in your project. No server required.

- **Hash-based IDs** (`bd-a1b2`) prevent merge collisions
- **Dependency graph** tracks blockers
- **`bd ready`** returns only unblocked issues
- **Dolt sync** pushes and pulls issue state like git

```
your-project/
  .bd/
    embeddeddolt/     # Dolt database (version-controlled)
    config            # bd configuration
  src/
  ...
```

## For agents

Point your agent at bd:

```
Use `bd` for all task tracking. Run `bd ready --json` to find work.
Claim with `bd update <id> --claim`. Close with `bd close <id> "reason"`.
Create sub-tasks as you discover them.
```

bd favors non-interactive use. All output supports `--json`. Hash IDs require no coordination between agents. `bd ready` resolves the dependency graph so agents grab the next unblocked task.

## Git-free usage

bd works without git. Set `BD_DIR` to skip repo discovery:

```bash
export BD_DIR=/path/to/project/.bd
bd init --stealth
```

Works with Jujutsu, Sapling, monorepos, CI/CD, and ephemeral environments.

## Sync

bd syncs through Dolt, not git:

```bash
bd dolt push       # push issues to remote
bd dolt pull       # pull from remote
```

Configure remotes with `bd dolt remote add <name> <url>`.

## What we deleted

We cut ~165k lines from [beads](https://github.com/steveyegge/beads):

- 6 external integrations (Jira, Linear, Notion, ADO, GitLab, GitHub)
- Formula DSL
- Molecule system (templates, lifecycle, phases)
- Gate system (async coordination primitives)
- Swarm orchestration
- Doctor diagnostics (19 files)
- Server mode
- OpenTelemetry (8 modules)
- 151-method storage interface (replaced with a concrete type)
- 230+ commands

What remains: create, track, query, close, sync.

## Build

```bash
make build          # build bd binary
make test           # run tests
make install        # build + install to ~/.local/bin
make fmt            # format with gofmt
```

Requires CGO (`CGO_ENABLED=1`) for the embedded Dolt engine. macOS needs ICU4C from Homebrew; the Makefile handles it.

## License

[MIT](LICENSE)
