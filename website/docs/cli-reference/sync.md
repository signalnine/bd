---
id: sync
title: Sync & Migration
sidebar_position: 6
---

# Sync & Migration Commands

Commands for synchronizing the bd database with a Dolt remote and for managing schema migrations.

## bd dolt push

Push changes to a Dolt remote.

```bash
bd dolt push [flags]
```

**What it does:**
1. Dolt commit (snapshot current database state)
2. Push commits to Dolt remote

**Examples:**
```bash
bd dolt push
```

**When to use:**
- End of work session
- Before switching machines
- After significant changes

## bd dolt pull

Pull changes from a Dolt remote.

```bash
bd dolt pull [flags]
```

**What it does:**
1. Fetches commits from Dolt remote
2. Merges into local database

**Examples:**
```bash
bd dolt pull
```

**When to use:**
- Start of work session
- After switching machines
- Before creating new issues (to avoid duplicates)

## bd backup

Back up the bd database to a snapshot directory.

```bash
bd backup init <path>     # Initialize a backup target
bd backup sync            # Sync the working database into the backup
```

Run `bd backup --help` for the full command list and flags.

## bd migrate

Migrate database schema.

```bash
bd migrate [flags]
```

**Flags:**
```bash
--inspect    Show migration plan (for agents)
--dry-run    Preview without changes
--cleanup    Remove old files after migration
--yes        Skip confirmation
--json       JSON output
```

**Examples:**
```bash
bd migrate --inspect --json
bd migrate --dry-run
bd migrate
bd migrate --cleanup --yes
```

## Auto-Sync Behavior

### With Dolt Server Mode

When the Dolt server is running, sync is handled automatically:
- Dolt auto-commit tracks changes
- Dolt-native replication handles remote sync

Start the Dolt server with `bd dolt start`.

### Embedded Mode (No Server)

In CI/CD pipelines and ephemeral environments, no server is needed:
- Changes written directly to the database
- Must manually push to remote

```bash
bd create "CI-generated task"
bd dolt push  # Manual push needed
```

## Conflict Resolution

Dolt handles conflict resolution at the database level using its built-in
merge capabilities. When conflicts arise during `bd dolt pull`, Dolt
identifies conflicting rows and exposes them through SQL for manual
resolution.

## Best Practices

1. **Always push at session end** - `bd dolt push`
2. **Always pull at session start** - `bd dolt pull`
3. **Snapshot regularly** - `bd backup sync` after major changes
