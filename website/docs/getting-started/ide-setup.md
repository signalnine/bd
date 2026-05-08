---
id: ide-setup
title: IDE Setup
sidebar_position: 3
---

# IDE Setup for AI Agents

Configure your IDE for beads integration. There is no setup wizard — every editor has its own native config file, so wire up bd by editing those directly.

## Claude Code

Add hooks to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd ready"}
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd dolt push"}
        ]
      }
    ]
  }
}
```

**How it works:**

1. SessionStart runs `bd ready`, surfacing unblocked work to the agent.
2. PreCompact runs `bd dolt push`, flushing the database to the Dolt remote before the context window compacts.
3. You use `bd` CLI commands directly during the session.

See [Claude Code Integration](/integrations/claude-code) for the full guide.

## Cursor IDE

Create `.cursor/rules/beads.mdc` with rules that reference bd commands:

```markdown
---
description: Beads issue tracker conventions
---

This project uses **bd** (beads) for issue tracking. Start a session with
`bd ready` and use `bd <command> --help` for command syntax.

Key commands:
- `bd ready` -- find unblocked work
- `bd create "Title" -t task -p 2` -- create an issue
- `bd update <id> --claim` -- claim work atomically
- `bd close <id> --reason "..."` -- finish work
- `bd dolt push` -- push to the Dolt remote at session end
```

## Aider

Create or edit `.aider.conf.yml`:

```yaml
# Disable auto-commits so bd manages issue lifecycle
auto-commits: false
```

Pipe `bd ready` or `bd show <id>` into aider as the initial message:

```bash
bd ready --json | aider --message-file -
```

See [Aider Integration](/integrations/aider).

## GitHub Copilot

For VS Code with GitHub Copilot, use the MCP server:

```bash
# Install MCP server
uv tool install beads-mcp
```

Create `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "beads": {
      "command": "beads-mcp"
    }
  }
}
```

**For all projects:** add to VS Code user-level MCP config:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/Code/User/mcp.json` |
| Linux | `~/.config/Code/User/mcp.json` |
| Windows | `%APPDATA%\Code\User\mcp.json` |

```json
{
  "servers": {
    "beads": {
      "command": "beads-mcp",
      "args": []
    }
  }
}
```

Initialize beads and reload VS Code:

```bash
bd init --quiet
```

See [GitHub Copilot Integration](/integrations/github-copilot) for the full guide.

## MCP Server (Alternative)

For MCP-only environments (Claude Desktop, no shell access):

```bash
pip install beads-mcp
```

Add to Claude Desktop config:

```json
{
  "mcpServers": {
    "beads": {
      "command": "beads-mcp"
    }
  }
}
```

**Trade-offs:**
- Works in MCP-only environments
- Higher context overhead (10-50k tokens for tool schemas)
- Additional latency from MCP protocol

See [MCP Server](/integrations/mcp-server).

## Verifying Your Setup

```bash
bd version
bd status      # Database overview and statistics
bd ready       # The same command your SessionStart hook runs
bd dolt push   # The same command your PreCompact hook runs
```

If `bd ready` and `bd dolt push` work cleanly from the command line, your hooks will work cleanly too.
