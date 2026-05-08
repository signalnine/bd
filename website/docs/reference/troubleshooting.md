---
id: troubleshooting
title: Troubleshooting
sidebar_position: 4
---

# Troubleshooting

Common issues and solutions.

## Installation Issues

### `bd: command not found`

```bash
# Check if installed
which bd
go list -f {{.Target}} github.com/signalnine/bd/cmd/bd

# Add Go bin to PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Or reinstall
go install github.com/signalnine/bd/cmd/bd@latest
```

### `zsh: killed bd` on macOS

CGO/SQLite compatibility issue:

```bash
CGO_ENABLED=1 go install github.com/signalnine/bd/cmd/bd@latest
```

### Permission denied

```bash
chmod +x $(which bd)
```

## Database Issues

### Database not found

```bash
# Initialize beads
bd init --quiet

# Or specify database
bd --db .beads/beads.db list
```

### Database locked

```bash
# Stop the Dolt server if running
bd dolt stop

# Try again
bd list
```

### Corrupted database

```bash
# Pull from the Dolt remote (often the fastest fix)
bd dolt pull

# Or restore from a snapshot
bd backup restore [path] --force
```

## Dolt Server Issues

### Server not starting

```bash
# Inspect Dolt state directly
bd dolt show

# Check server logs
cat .beads/dolt/sql-server.log

# Restart the server
bd dolt stop
bd dolt start
```

### Version mismatch

After upgrading bd:

```bash
bd dolt stop
bd dolt start
```

## Sync Issues

### Changes not syncing

```bash
# Force push to Dolt remote
bd dolt push

# Confirm the remote is configured
bd dolt show
```

### Recovery from backup

```bash
# Restore from a snapshot
bd backup restore [path] --force

# Or pull from Dolt remote
bd dolt pull
```

### Merge conflicts

Inspect the conflicting working set with `bd dolt show`, resolve via Dolt SQL (`bd dolt sql`), then re-push:

```bash
bd dolt show
bd dolt sql -q "..."  # resolve as needed
bd dolt push
```

## Git Hook Issues

bd no longer installs git hooks. If you wrote your own hook scripts under `.git/hooks/` to call `bd dolt push` or similar, debug them as you would any other shell script:

```bash
# Inspect and run manually
cat .git/hooks/pre-commit
.git/hooks/pre-commit
```

## Dependency Issues

### Circular dependencies

```bash
# Detect cycles
bd dep cycles

# Remove one dependency
bd dep remove bd-A bd-B
```

### Missing dependencies

```bash
# Check orphan handling
bd config get import.orphan_handling

# Allow orphans
bd config set import.orphan_handling allow
```

## Performance Issues

### Slow queries

```bash
# Check database size
ls -lh .beads/beads.db

# Compact if large
bd admin compact --analyze
```

### High memory usage

```bash
# Reduce cache
bd config set database.cache_size 1000
```

## Getting Help

### Debug output

```bash
bd --verbose list
```

### Logs

```bash
cat .beads/dolt/sql-server.log
```

### System info

```bash
bd version
bd status --json
bd dolt show
```

### File an issue

```bash
# Include this info
bd version
bd status --json
uname -a
```

Report at: https://github.com/signalnine/bd/issues
