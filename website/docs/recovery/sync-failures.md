---
sidebar_position: 5
title: Sync Failures
description: Recover from Dolt sync failures
---

# Sync Failures Recovery

This runbook helps you recover from Dolt sync failures.

## Symptoms

- `bd dolt push` or `bd dolt pull` hangs or times out
- Network-related error messages
- "failed to push" or "failed to pull" errors
- Dolt server not responding

## Diagnosis

```bash
# Inspect Dolt state directly
bd dolt show

# View Dolt server logs
tail -50 .beads/dolt/sql-server.log
```

## Solution

**Step 1:** Stop the Dolt server
```bash
bd dolt stop
```

**Step 2:** Check for lock files
```bash
ls -la .beads/*.lock
# Remove stale locks if Dolt server is definitely stopped
rm -f .beads/*.lock
```

**Step 3:** Back up before any destructive recovery
```bash
cp -r .beads .beads.backup
```

**Step 4:** Restart the Dolt server
```bash
bd dolt start
```

**Step 5:** Verify sync works
```bash
bd dolt push
bd dolt show
```

If the database itself is corrupted, restore from a snapshot
(`bd backup restore <path> --force`) or pull fresh from the remote
(`rm -rf .beads && bd dolt pull`).

## Common Causes

| Cause | Solution |
|-------|----------|
| Network timeout | Retry with better connection |
| Stale lock file | Remove lock after stopping Dolt server |
| Corrupted state | Back up, then `bd backup restore` or fresh `bd dolt pull` |
| Merge conflicts | See [Merge Conflicts](/recovery/merge-conflicts) |

## Prevention

- Ensure stable network before sync
- Let sync complete before closing terminal
- Use `bd dolt stop` before system shutdown
- Snapshot regularly with `bd backup sync`
