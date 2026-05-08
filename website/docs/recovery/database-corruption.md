---
sidebar_position: 2
title: Database Corruption
description: Recover from Dolt database corruption
---

# Database Corruption Recovery

This runbook helps you recover from database corruption in Beads.

## Symptoms

- Error messages during `bd` commands
- "database is locked" errors that persist
- Missing issues that should exist
- Inconsistent database state

## Diagnosis

```bash
# Database overview and statistics
bd status

# Detect circular dependencies
bd dep cycles

# Check Dolt server health
bd dolt show
```

## Solution

**Step 1:** Stop the Dolt server
```bash
bd dolt stop
```

**Step 2:** Back up current state
```bash
cp -r .beads .beads.backup
```

**Step 3:** Restore from a known-good backup or pull from the Dolt remote
```bash
# If you have a bd backup snapshot
bd backup restore <path> --force

# Or, if the remote is the source of truth
rm -rf .beads
bd dolt pull
```

**Step 4:** Verify recovery
```bash
bd status
bd list
```

**Step 5:** Restart the Dolt server
```bash
bd dolt start
```

## Prevention

- Let the Dolt server handle synchronization
- Use `bd dolt stop` before system shutdown
- Snapshot the database with `bd backup sync` after major changes
- Push to the Dolt remote regularly (`bd dolt push`)
