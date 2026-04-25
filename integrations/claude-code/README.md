# Claude Code Integration for bd

Slash command for converting [Claude Code](https://docs.anthropic.com/en/docs/claude-code) plans to bd tasks.

## Prerequisites

```bash
# Install bd
curl -fsSL https://raw.githubusercontent.com/signalnine/bd/main/scripts/install.sh | bash
```

## Installation

```bash
cp commands/plan-to-bd.md ~/.claude/commands/
```

Optionally add to `~/.claude/settings.json` under `permissions.allow`:

```json
"Bash(bd:*)"
```

## /plan-to-bd

Converts a Claude Code plan file into a bd epic with tasks.

```
/plan-to-bd                    # Convert most recent plan
/plan-to-bd path/to/plan.md    # Convert specific plan
```

**What it does:**
- Parses plan structure (title, summary, phases)
- Creates an epic for the plan
- Creates tasks from each phase
- Sets up sequential dependencies
- Uses Task agent delegation for context efficiency

**Example output:**
```
Created from: peaceful-munching-spark.md

Epic: Standardize ID Generation (bd-abc)
  |- Add dependency (bd-def) - ready
  |- Create ID utility (bd-ghi) - blocked by bd-def
  \- Update schema (bd-jkl) - blocked by bd-ghi

Total: 4 tasks
Run `bd ready` to start.
```

## Related

- `bd ready` - Find unblocked work
- `bd create` - Create new issues
- For richer Claude Code integration, install the bd plugin: `/plugin marketplace add signalnine/bd && /plugin install beads`

## License

Same as bd (see repository root).
