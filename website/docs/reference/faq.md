---
id: faq
title: FAQ
sidebar_position: 5
---

# Frequently Asked Questions

## General

### Why beads instead of GitHub Issues or Jira?

Beads was designed specifically for AI-supervised coding workflows:
- **Hash-based IDs** prevent collisions with concurrent agents
- **Dolt-backed storage** enables branch-based workflows
- **Dependency-aware** ready queue for automated work selection
- **Formula system** for declarative workflow templates

### What does "beads" stand for?

Nothing specific - it's a metaphor for linked work items (like beads on a string).

### Is beads production-ready?

Yes, beads is used in production for AI-assisted development. The API is stable with semantic versioning.

## Architecture

### Why Dolt instead of plain SQLite?

- **Dolt** provides a version-controlled SQL database with built-in replication
- Git-like branching, diffing, and merging at the database level
- No need for a separate sync format -- Dolt handles it natively

### Why hash-based IDs instead of sequential?

Sequential IDs (`#1`, `#2`) break when:
- Multiple agents create issues simultaneously
- Different branches have independent numbering
- Forks diverge and merge

Hash-based IDs are globally unique without coordination.

### How does the Dolt server work?

Beads uses Dolt server mode for concurrent access:
- Transaction isolation for multiple agents
- SQL-based queries for performance
- Automatic retry on conflicts

In CI/CD or single-agent environments, beads uses embedded mode automatically (no server required).

## Usage

### How do I sync issues?

```bash
# Push changes to Dolt remote:
bd dolt push

# Pull changes from Dolt remote:
bd dolt pull
```

### How do I handle merge conflicts?

Dolt handles merge conflicts at the database level. If conflicts arise during pull, inspect with `bd dolt show`, resolve via `bd dolt sql`, then push:
```bash
bd dolt show
bd dolt sql -q "..."  # resolve as needed
bd dolt push
```

See [Merge Conflicts Recovery](/recovery/merge-conflicts) for the full procedure.

### Can multiple agents work on the same repo?

Yes! That's what beads was designed for:
- Hash IDs prevent collisions
- Pin work to specific agents
- Track who's working on what

### How do I use beads in CI/CD?

```bash
# Just run commands directly — beads uses embedded mode in CI
bd list
```

## Workflows

### What are formulas?

Declarative workflow templates in TOML or JSON. Pour them to create molecules (instances).

### What are gates?

Async coordination primitives:
- Human gates wait for approval
- Timer gates wait for duration
- GitHub gates wait for CI/PR events

### What's the difference between molecules and wisps?

- **Molecules** persist in `.beads/` and sync with git
- **Wisps** are ephemeral in `.beads-wisp/` and don't sync

## Integration

### Should I use CLI or MCP?

**Use CLI + hooks** when shell is available (Claude Code, Cursor, etc.):
- Lower context overhead (~1-2k vs 10-50k tokens)
- Faster execution
- Universal across editors

**Use MCP** when CLI unavailable (Claude Desktop).

### How do I integrate with my editor?

There is no setup wizard — wire bd into your editor's native config file.
See [IDE Setup](/getting-started/ide-setup) for per-editor recipes.

### Can beads import from GitHub Issues?

Yes — see the [GitHub integration documentation](https://github.com/signalnine/bd/blob/main/docs/GITHUB_IMPORT.md) for details.

## Troubleshooting

### Why is the Dolt server not starting?

```bash
# Check Dolt status directly
bd dolt show

# Check server logs
cat .beads/dolt/sql-server.log

# Restart the server
bd dolt stop
bd dolt start
```

### Why aren't my changes syncing?

```bash
# Force push to Dolt remote
bd dolt push

# Confirm remote configuration
bd dolt show
```

### How do I report a bug?

1. Check existing issues: https://github.com/signalnine/bd/issues
2. Include: `bd version`, `bd status --json`, reproduction steps
3. File at: https://github.com/signalnine/bd/issues/new
