<p align="center">
  <img src="https://signalnine.s3.amazonaws.com/disembed-revivalists.jpeg" alt="bd" width="50%" />
</p>

# bd

**issue tracker for AI agents.**

your agents need somewhere to track work that isn't a markdown file. bd is a dependency-aware task graph on top of [Dolt](https://github.com/dolthub/dolt) (version-controlled SQL). agents create issues, track blockers, close work, keep context across sessions.

forked from [beads](https://github.com/signalnine/bd). deleted 78% of it.

## install

```bash
git clone https://github.com/signalnine/bd
cd bd
make install    # builds and installs to ~/.local/bin/bd
```

requires Go 1.25+ and CGO. macOS needs ICU4C from Homebrew but the Makefile handles it.

## try it

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

## commands

| | | |
|---|---|---|
| `bd init` | `bd create` | `bd list` |
| `bd show` | `bd update` | `bd edit` |
| `bd close` | `bd reopen` | `bd delete` |
| `bd dep add` | `bd dep remove` | `bd ready` |
| `bd search` | `bd comment` | `bd label` |
| `bd config` | `bd backup` | `bd version` |
| `bd dolt push` | `bd dolt pull` | |

everything supports `--json`.

## how it works

embedded Dolt database in `.bd/` in your project. no server.

- hash-based IDs (`bd-a1b2`) -- no merge collisions
- dependency graph tracks what blocks what
- `bd ready` returns unblocked work only
- dolt push/pull syncs issue state to remotes

```
your-project/
  .bd/
    embeddeddolt/     # version-controlled SQL
    config
  src/
  ...
```

## for agents

drop this in your agent instructions:

```
Use `bd` for all task tracking. Run `bd ready --json` to find work.
Claim with `bd update <id> --claim`. Close with `bd close <id> "reason"`.
Create sub-tasks as you discover them.
```

hash IDs mean agents don't need to coordinate. `bd ready` does the dependency math. `--json` on everything.

## without git

```bash
export BD_DIR=/path/to/project/.bd
bd init --stealth
```

works with jujutsu, sapling, monorepos, CI/CD, whatever.

## sync

dolt, not git:

```bash
bd dolt push
bd dolt pull
```

`bd dolt remote add <name> <url>` to configure.

## what we cut

~165k lines from [beads](https://github.com/signalnine/bd):

- 6 tracker integrations (Jira, Linear, Notion, ADO, GitLab, GitHub)
- formula DSL
- molecule system
- gate system
- swarm orchestration
- doctor diagnostics (19 files)
- server mode
- OpenTelemetry (8 modules)
- 151-method storage interface
- 230+ commands

what's left: create, track, query, close, sync.

## build

```bash
make build          # build bd binary
make test           # run tests
make install        # build + install to ~/.local/bin
make fmt            # gofmt
```

## license

[MIT](LICENSE)
