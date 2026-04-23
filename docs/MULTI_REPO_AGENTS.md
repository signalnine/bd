# Multi-Repo Patterns for AI Agents

This guide covers multi-repo workflow patterns specifically for AI agents working with bd.

**For humans**, see [MULTI_REPO_MIGRATION.md](MULTI_REPO_MIGRATION.md) for setup details.

## Quick Reference

Agents invoke `bd` directly as a shell command. bd is embedded-only: each project keeps its own `.bd/embeddeddolt/` database, and bd routes every call to the correct database based on the current working directory. No servers, no MCP.

### Multi-Repo Config Options

Agents can configure multi-repo behavior via `bd config`:

```bash
# Auto-routing (detects role: maintainer vs contributor)
bd config set routing.mode auto
bd config set routing.maintainer "."
bd config set routing.contributor "~/.bd-planning"

# Explicit routing (always use default)
bd config set routing.mode explicit
bd config set routing.default "."

# Multi-repo aggregation (hydration)
bd config set repos.primary "."
bd config set repos.additional "~/repo1,~/repo2,~/repo3"
```

**Check current config:**
```bash
bd config get routing.mode
bd config get repos.additional
bd config list  # Shows all config
```

## Routing Behavior

### Auto-Routing (OSS Contributor Pattern)

When `routing.mode=auto`, beads detects user role and routes new issues automatically:

**Maintainer (SSH push access):**
```bash
# Git remote: git@github.com:user/repo.git
bd create "Fix bug" -p 1
# → Creates in current repo (source_repo = ".")
```

**Contributor (HTTPS or no push access):**
```bash
# Git remote: https://github.com/fork/repo.git
bd create "Fix bug" -p 1
# → Creates in planning repo (source_repo = "~/.bd-planning")
```

**Role detection priority:**
1. Explicit git config: `git config bd.role maintainer|contributor`
2. Git remote URL inspection (SSH = maintainer, HTTPS = contributor)
3. Fallback: contributor

### Explicit Override

Always available regardless of routing mode:

```bash
# Force creation in specific repo
bd create "Issue" -p 1 --repo /path/to/repo
bd create "Issue" -p 1 --repo ~/my-planning
```

### Discovered Issue Inheritance

Issues with `discovered-from` dependencies automatically inherit parent's `source_repo`:

```bash
# Parent in current repo
bd create "Implement auth" -p 1
# → Created as bd-abc (source_repo = ".")

# Discovered issue inherits parent's repo
bd create "Found bug" -p 1 --deps discovered-from:bd-abc
# → Created with source_repo = "." (same as parent)
```

**Override if needed:**
```bash
bd create "Issue" -p 1 --deps discovered-from:bd-abc --repo /different/repo
```

## Multi-Repo Hydration

Agents working in multi-repo mode see aggregated issues from multiple repositories:

```bash
# View all issues (current + additional repos)
bd ready --json
bd list --json

# Filter by source repository
bd list --json | jq '.[] | select(.source_repo == ".")'
bd list --json | jq '.[] | select(.source_repo == "~/.bd-planning")'
```

**How it works:**
1. Beads reads from all configured Dolt databases
2. Aggregates into unified view
3. Maintains `source_repo` field for provenance
4. Routes issues back to correct databases

## Common Patterns

### OSS Contributor Workflow

**Setup:** Human runs `bd init --contributor` (wizard handles config)

**Agent workflow:**
```bash
# All planning issues auto-route to separate repo
bd create "Investigate implementation" -p 1
bd create "Draft RFC" -p 2
# → Created in ~/.bd-planning (never appears in PRs)

# View all work (upstream + planning)
bd ready
bd list --json

# Complete work
bd close plan-42 --reason "Done"

# Git commit/push - no .bd/ pollution in PR ✅
```

### Team Workflow

**Setup:** Human runs `bd init --team` (wizard handles config)

**Agent workflow:**
```bash
# Shared team planning (committed to repo)
bd create "Implement feature X" -p 1
# → Created in current repo (visible to team)

# Optional: Personal experiments in separate repo
bd create "Try alternative" -p 2 --repo ~/.bd-planning-personal
# → Created in personal repo (private)

# View all
bd ready --json
```

### Multi-Phase Development

**Setup:** Multiple repos for different phases

**Agent workflow:**
```bash
# Phase 1: Planning repo
cd ~/projects/myapp-planning
bd create "Design auth" -p 1 -t epic

# Phase 2: Implementation repo (views planning + impl)
cd ~/projects/myapp-implementation
bd ready  # Shows both repos
bd create "Implement auth backend" -p 1
bd dep add impl-42 plan-10 --type blocks  # Link across repos
```

## Troubleshooting

### Issues appearing in wrong repository

**Symptom:** `bd create` routes to unexpected repo

**Check:**
```bash
bd config get routing.mode
bd config get routing.maintainer
bd config get routing.contributor
bd config get routing.mode
```

**Fix:**
```bash
# Use explicit flag
bd create "Issue" -p 1 --repo .

# Or reconfigure routing
bd config set routing.mode explicit
bd config set routing.default "."
```

### Can't see issues from other repos

**Symptom:** `bd list` only shows current repo

**Check:**
```bash
bd config get repos.additional
```

**Fix:**
```bash
# Add missing repos
bd config set repos.additional "~/repo1,~/repo2"

# Force sync
bd dolt push
bd list --json
```

### Discovered issues in wrong repository

**Symptom:** Issues with `discovered-from` appear in wrong repo

**Explanation:** This is intentional - discovered issues inherit parent's `source_repo`

**Override if needed:**
```bash
bd create "Issue" -p 1 --deps discovered-from:bd-42 --repo /different/repo
```

### Planning repo polluting PRs

**Symptom:** `~/.bd-planning` changes appear in upstream PRs

**Verify:**
```bash
# Planning repo should be separate
ls -la ~/.bd-planning/.git  # Should exist

# Fork should NOT contain planning issues
cd ~/projects/fork
bd list --json | jq '.[] | select(.source_repo == "~/.bd-planning")'
# Should be empty

# Check routing
bd config get routing.contributor  # Should be ~/.bd-planning
```

### Version mismatch after upgrade

**Symptom:** Commands fail right after a `bd` upgrade

**Fix:**
```bash
bd version      # Confirm active CLI version
bd where        # Confirm active database path
```

## Best Practices for Agents

### OSS Contributors
- ✅ Planning issues auto-route to `~/.bd-planning`
- ✅ Never commit `.bd/` in PRs to upstream
- ✅ Use `bd ready` to see all work (upstream + planning)
- ❌ Don't manually override routing without good reason

### Teams
- ✅ Use `bd dolt push` to sync the shared Dolt database
- ✅ Use `bd dolt push` to ensure changes are committed/pushed
- ✅ Link related issues across repos with dependencies
- ❌ Don't delete `.bd/` - you lose all issue data

### Multi-Phase Projects
- ✅ Use clear repo names (`planning`, `impl`, `maint`)
- ✅ Link issues across phases with `blocks` dependencies
- ✅ Use `bd list --json` to filter by `source_repo`
- ❌ Don't duplicate issues across repos

### General
- ✅ Check routing config before filing issues
- ✅ Use `bd where` and `bd config list` to verify workspace state
- ✅ Run `bd dolt push` at end of session
- ❌ Don't assume routing behavior - check config

## Backward Compatibility

Multi-repo mode is fully backward compatible:

**Without multi-repo config:**
```bash
bd create "Issue" -p 1
# → Creates in local Dolt database (single-repo mode)
```

**With multi-repo config:**
```bash
bd create "Issue" -p 1
# → Auto-routed based on config
# → Old issues in local database still work
```

**Disabling multi-repo:**
```bash
bd config unset routing.mode
bd config unset repos.additional
# → Back to single-repo mode
```

## Configuration Reference

### Routing Config

```bash
# Auto-detect role (maintainer vs contributor)
bd config set routing.mode auto
bd config set routing.maintainer "."              # Where maintainer issues go
bd config set routing.contributor "~/.bd-planning"  # Where contributor issues go

# Explicit mode (always use default)
bd config set routing.mode explicit
bd config set routing.default "."                 # All issues go here

# Check settings
bd config get routing.mode
bd config get routing.maintainer
bd config get routing.contributor
bd config get routing.default
```

### Multi-Repo Hydration

```bash
# Set primary repo (optional, default is current)
bd config set repos.primary "."

# Add additional repos to aggregate
bd config set repos.additional "~/repo1,~/repo2,~/repo3"

# Check settings
bd config get repos.primary
bd config get repos.additional
```

### Verify Configuration

```bash
bd where                  # active .bd database path
bd config list --json     # full resolved configuration
```

## Related Documentation

- **[MULTI_REPO_MIGRATION.md](MULTI_REPO_MIGRATION.md)** - Complete guide for humans with interactive wizards
- **[ROUTING.md](ROUTING.md)** - Technical details of routing implementation
- **[MULTI_REPO_HYDRATION.md](MULTI_REPO_HYDRATION.md)** - Hydration layer internals
- **[AGENTS.md](../AGENTS.md)** - Main AI agent guide
