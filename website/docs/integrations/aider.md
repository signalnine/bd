---
id: aider
title: Aider
sidebar_position: 3
---

# Aider Integration

How to use beads with Aider.

## Setup

Create or edit `.aider.conf.yml` in your project:

```yaml
# Disable auto-commits so bd can manage issue lifecycle
auto-commits: false
```

Aider doesn't load bd context automatically — pipe `bd ready` or `bd show <id>` into the chat as needed (see Workflow below).

## Workflow

### Start Session

```bash
# See what's ready, then start aider
bd ready
aider

# Or pipe a specific issue's details into aider as the initial message
bd show bd-42 --json | aider --message-file -
```

### During Work

Use bd commands alongside aider:

```bash
# In another terminal or after exiting aider
bd create "Found bug during work" --deps discovered-from:bd-42 --json
bd update bd-42 --claim
bd ready
```

### End Session

```bash
bd dolt push
```

## Best Practices

1. **Keep issues visible** - Pipe `bd ready` or `bd show <id>` into aider when starting
2. **Push regularly** - Run `bd dolt push` after significant changes
3. **Use discovered-from** - Track issues found during work
4. **Document context** - Include descriptions in issues

## Example Workflow

```bash
# 1. Check ready work
bd ready

# 2. Start aider with issue context
aider --message "Working on bd-42: Fix auth bug"

# 3. Work in aider...

# 4. Create discovered issues
bd create "Found related bug" --deps discovered-from:bd-42 --json

# 5. Complete and push
bd close bd-42 --reason "Fixed"
bd dolt push
```

## Troubleshooting

### Config not loading

```bash
# Check config exists
cat .aider.conf.yml
```

### Issues not visible

```bash
# Pipe ready work or a specific issue into aider's first message
bd ready --json | aider --message-file -
bd show bd-42 --json | aider --message-file -
```

## See Also

- [Claude Code](/integrations/claude-code)
- [IDE Setup](/getting-started/ide-setup)
