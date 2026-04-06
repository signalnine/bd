<p align="center">
  <img src="https://signalnine.s3.amazonaws.com/disembed-revivalists.jpeg" alt="bd" width="100%" />
</p>

# bd

**Issue tracker for AI agents. 20 commands. One database. No bullshit.**

bd gives your coding agents a persistent, dependency-aware task graph backed by [Dolt](https://github.com/dolthub/dolt) -- a version-controlled SQL database. Agents create issues, track blockers, and close work without losing context across sessions.

This is a stripped-down fork of [beads](https://github.com/steveyegge/beads). The original grew to 250 commands and 210k lines. We deleted 78% of it.

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

That's all of them:

| | | |
|---|---|---|
| `bd init` | `bd create` | `bd list` |
| `bd show` | `bd update` | `bd edit` |
| `bd close` | `bd reopen` | `bd delete` |
| `bd dep add` | `bd dep remove` | `bd ready` |
| `bd search` | `bd comment` | `bd label` |
| `bd config` | `bd backup` | `bd version` |
| `bd dolt push` | `bd dolt pull` | |

Every command supports `--json` for programmatic use.

## How it works

bd stores issues in an embedded [Dolt](https://github.com/dolthub/dolt) database (version-controlled SQL) inside a `.bd/` directory in your project. No external server needed.

- **Hash-based IDs** (`bd-a1b2`) prevent merge collisions
- **Dependency graph** tracks what blocks what
- **`bd ready`** returns only issues with zero open blockers
- **Dolt sync** pushes/pulls issue state to remotes like git

```
your-project/
  .bd/
    embeddeddolt/     # Dolt database (version-controlled)
    config            # bd configuration
  src/
  ...
```

## For agents

Point your agent at bd in its instructions:

```
Use `bd` for all task tracking. Run `bd ready --json` to find work.
Claim with `bd update <id> --claim`. Close with `bd close <id> "reason"`.
Create sub-tasks as you discover them.
```

bd is optimized for non-interactive use:
- All commands support `--json` output
- Hash IDs mean no coordination needed between agents
- `bd ready` does the dependency math so agents don't have to
- Single embedded database -- no server to manage

## Git-free usage

bd works without git. Set `BD_DIR` to skip repo discovery:

```bash
export BD_DIR=/path/to/project/.bd
bd init --stealth
```

Useful for non-git VCS (Jujutsu, Sapling), monorepos, CI/CD, or ephemeral environments.

## Sync

bd uses Dolt's native sync -- not git:

```bash
bd dolt push       # push issues to remote
bd dolt pull       # pull from remote
```

Configure remotes with `bd dolt remote add <name> <url>`.

## What we deleted

This fork removed ~165k lines from [beads](https://github.com/steveyegge/beads):

- 6 external tracker integrations (Jira, Linear, Notion, ADO, GitLab, GitHub)
- Formula DSL (a programming language for issue templates)
- Molecule system (template instantiation, lifecycle, phases)
- Gate system (async coordination primitives)
- Swarm orchestration
- Doctor diagnostics (19 files)
- Server mode (external Dolt sql-server)
- OpenTelemetry (8 modules)
- 151-method storage interface (replaced with concrete type)
- 230+ commands

What remains is the core loop: create, track, query, close, sync.

## Build

```bash
make build          # build bd binary
make test           # run tests
make install        # build + install to ~/.local/bin
make fmt            # format with gofmt
```

Requires CGO (`CGO_ENABLED=1`) for the embedded Dolt engine. On macOS, ICU4C from Homebrew is needed (the Makefile handles this automatically).

## License

[MIT](LICENSE)
