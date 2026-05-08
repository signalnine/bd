# Beads Skill Maintenance Guide

## Architecture Decisions

ADRs in `adr/` document key decisions. These are NOT loaded during skill invocation—they're reference material for maintainers making changes.

| ADR | Decision |
|-----|----------|
| [ADR-0001](adr/0001-bd-prime-as-source-of-truth.md) | (Superseded) `bd prime` was removed in the simplification sweep. |

## Key Principle: DRY via `--help`

**NEVER duplicate CLI documentation in SKILL.md or resources.**

- `bd <command> --help` is the single source of truth for command syntax.
- Auto-updates with each bd release.

**SKILL.md should only contain:**
- Decision frameworks (bd vs TodoWrite)
- Prerequisites (install verification)
- Resource index (progressive disclosure)
- Pointer to `--help`

## Keeping the Skill Updated

### When bd releases new version:

1. **Check for new features**: `bd --help` for new commands
2. **Update SKILL.md frontmatter**: `version: "X.Y.Z"`
3. **Add resources for conceptual features** (workflow patterns, recovery, etc.)
4. **Don't add CLI reference** — let `--help` carry it.

### What belongs in resources:

| Content Type | Belongs in Resources? | Why |
|--------------|----------------------|-----|
| Conceptual frameworks | ✅ Yes | --help doesn't explain "when to use" |
| Decision trees | ✅ Yes | Cognitive guidance, not CLI reference |
| Advanced patterns | ✅ Yes | Depth beyond --help |
| CLI command syntax | ❌ No | Use `bd <cmd> --help` |
| Workflow checklists | ✅ Yes | Session protocol, recovery flow |

### Resource update checklist:

```
[ ] Check if --help now covers this content
[ ] If yes, remove from resources (avoid duplication)
[ ] If no, update resource for new bd version
[ ] Update version compatibility in README.md
```

## File Roles

| File | Purpose | When to Update |
|------|---------|----------------|
| SKILL.md | Entry point, resource index | New features, version bumps |
| README.md | Human docs, installation | Structure changes |
| CLAUDE.md | This file, maintenance guide | Architecture changes |
| adr/*.md | Decision records | When making architectural decisions |
| resources/*.md | Deep-dive guides | New conceptual content |

## Testing Changes

After skill updates:

```bash
# Verify SKILL.md is within token budget
wc -w claude-plugin/skills/beads/SKILL.md  # Target: 400-600 words

# Verify links resolve
# (Manual check: ensure all resource links in SKILL.md exist)

# Sanity check the canonical session-start command
bd ready
```

## Attribution

Resources adapted from other sources should include attribution header:

```markdown
# Resource Title

> Adapted from [source]
```
