# bd Claude Code Plugin

Slash commands and skills for driving bd from inside Claude Code.

## What you get

Installing the plugin gives Claude Code:

- **Slash commands** — `/beads:ready`, `/beads:create`, `/beads:list`, `/beads:close`, etc. Each one expands into a ready-to-run `bd` invocation.
- **Skills** — richer workflow recipes under the `beads` skill namespace.
- **A task agent** (`task-agent.md`) that knows how to pick up `bd ready` work and drive it to done.

There is no MCP server. The plugin shells out to the `bd` CLI you have on your PATH.

## Install

```
/plugin marketplace add signalnine/bd
/plugin install beads
```

Then restart Claude Code.

## Prerequisite

The `bd` binary must be on your PATH. Install via the [INSTALLING.md](INSTALLING.md) guide first.

## Verify

```
/plugin list
```

Should show `beads` as installed. Try `/beads:ready` in a project that has `bd init` run.

## Uninstall

```
/plugin uninstall beads
```

## Where the plugin lives

The plugin source is in this repo under `claude-plugin/`:

- `claude-plugin/.claude-plugin/plugin.json` — manifest
- `claude-plugin/commands/*.md` — slash command definitions
- `claude-plugin/skills/beads/` — the `beads` skill
- `claude-plugin/agents/task-agent.md` — the task agent

Changes to those files ship with the next bd release.
