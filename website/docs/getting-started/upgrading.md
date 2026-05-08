---
id: upgrading
title: Upgrading
sidebar_position: 4
---

# Upgrading bd

How to upgrade bd and keep your projects in sync.

## Checking for Updates

```bash
# Current version
bd version
```

For release notes, see the [CHANGELOG](https://github.com/signalnine/bd/blob/main/CHANGELOG.md) on GitHub.

## Upgrading

Use the command that matches your install method.

| Install method | Platforms | Command |
|---|---|---|
| Quick install script | macOS, Linux, FreeBSD | `curl -fsSL https://raw.githubusercontent.com/signalnine/bd/main/scripts/install.sh \| bash` |
| PowerShell installer | Windows | `irm https://raw.githubusercontent.com/signalnine/bd/main/install.ps1 \| iex` |
| Homebrew | macOS, Linux | `brew upgrade beads` |
| go install | macOS, Linux, FreeBSD, Windows | `go install github.com/signalnine/bd/cmd/bd@latest` |
| npm | macOS, Linux, Windows | `npm update -g @beads/bd` |
| bun | macOS, Linux, Windows | `bun install -g --trust @beads/bd` |
| From source (Unix shell) | macOS, Linux, FreeBSD | `git pull && go build -o bd ./cmd/bd` |

### Quick install script (macOS/Linux/FreeBSD)

```bash
curl -fsSL https://raw.githubusercontent.com/signalnine/bd/main/scripts/install.sh | bash
```

### PowerShell installer (Windows)

```pwsh
irm https://raw.githubusercontent.com/signalnine/bd/main/install.ps1 | iex
```

### Homebrew

```bash
brew upgrade beads
```

### go install

```bash
go install github.com/signalnine/bd/cmd/bd@latest
```

### From Source

```bash
cd beads
git pull
go build -o bd ./cmd/bd
sudo mv bd /usr/local/bin/
```

## After Upgrading

**Important:** After upgrading, restart any long-running bd processes:

```bash
# Restart the Dolt server if you're using one
bd dolt stop && bd dolt start
```

Run `bd version` to confirm the upgrade landed and `bd status` for a database overview.

## Database Migrations

After major upgrades, check for database migrations:

```bash
# Inspect migration plan (AI agents)
bd migrate --inspect --json

# Preview migration changes
bd migrate --dry-run

# Apply migrations
bd migrate

# Migrate and clean up old files
bd migrate --cleanup --yes
```

## Troubleshooting Upgrades

### Database schema changed

```bash
bd migrate --dry-run
bd migrate
```

### Recovery after upgrade

If you need to restore from a backup:

```bash
bd init
bd backup restore [path] --force
```

Or pull from a Dolt remote:

```bash
bd dolt pull
```
