---
sidebar_position: 3
title: Merge Conflicts
description: Resolve Dolt merge conflicts
---

# Merge Conflicts Recovery

This runbook helps you resolve merge conflicts that occur during Dolt sync operations.

## Symptoms

- `bd dolt pull` fails with conflict errors
- Different issue states between clones

## Diagnosis

```bash
# Database overview
bd status

# Inspect Dolt state directly
bd dolt show
```

## Solution

**Step 1:** Back up current state
```bash
cp -r .beads .beads.backup
```

**Step 2:** Inspect the conflicting working set
```bash
bd dolt show
```

**Step 3:** Resolve the conflict in Dolt. Either accept one side wholesale or use Dolt's SQL interface (`dolt sql`) to reconcile rows manually. See the [Dolt conflict resolution docs](https://docs.dolthub.com/concepts/dolt/git/conflicts) for the full procedure.

**Step 4:** Verify state
```bash
bd list
bd status
```

**Step 5:** Push resolved state
```bash
bd dolt push
```

## Prevention

- Sync before and after work sessions using `bd dolt pull` / `bd dolt push`
- Avoid concurrent modifications from multiple clones without the Dolt server running
