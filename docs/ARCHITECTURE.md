# Architecture

This document describes bd's overall architecture - the data model, sync mechanism, and how components fit together. For internal implementation details (FlushManager, Blocked Cache), see [INTERNALS.md](INTERNALS.md).

## The Two-Layer Data Model

bd's core design enables a distributed, Dolt-powered issue tracker that feels like a centralized database. The architecture has two synchronized layers:

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Layer                                 │
│                                                                  │
│  bd create, list, update, close, ready, show, dep, sync, ...    │
│  - Cobra commands in cmd/bd/                                     │
│  - All commands support --json for programmatic use              │
│  - Direct DB access (server mode via dolt sql-server)            │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               v
┌─────────────────────────────────────────────────────────────────┐
│                      Dolt Database                               │
│                      (.bd/dolt/)                               │
│                                                                  │
│  - Version-controlled SQL database with cell-level merge         │
│  - Server mode via dolt sql-server (multi-writer capable)        │
│  - Fast queries, indexes, foreign keys                           │
│  - Issues, dependencies, labels, comments, events                │
│  - Automatic Dolt commits on every write                         │
│  - Native push/pull to Dolt remotes                              │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                        Dolt push/pull
                    (or federation peer sync)
                               │
                               v
┌─────────────────────────────────────────────────────────────────┐
│                     Remote (Dolt or Git)                          │
│                                                                  │
│  - Dolt remotes (DoltHub, S3, GCS, filesystem)                   │
│  - All collaborators share the same issue database               │
│  - Cell-level merge for conflict resolution                      │
│  - Protected branch support via separate sync branch             │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Design?

**Dolt for versioned SQL:** Queries complete in milliseconds with full SQL support. Dolt adds native version control — every write is automatically committed to Dolt history, providing a complete audit trail. Cell-level merge resolves conflicts automatically.

**Dolt for distribution:** Native push/pull to Dolt remotes (DoltHub, S3, GCS). No special sync server needed. Issues travel with your code. Offline work just works.

**Backup:** Use `bd backup init` / `bd backup sync` to push Dolt-native backups (preserving full commit history) to a filesystem path or DoltHub, and `bd backup restore` to restore from them.

## Write Path

When you create or modify an issue:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Command   │───▶│   Dolt Write    │───▶│  Dolt Commit    │
│   (bd create)   │    │  (immediate)    │    │  (automatic)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

1. **Command executes:** `bd create "New feature"` writes to Dolt immediately
2. **Dolt commit:** Every write is automatically committed to Dolt history
3. **Sync:** Use `bd dolt push` to share changes with Dolt remotes

Key implementation:
- Dolt storage: `internal/storage/dolt/`
- Export (for portability): `cmd/bd/export.go`

## Read Path

All queries run directly against the local Dolt database:

```
┌─────────────────┐    ┌─────────────────┐
│   CLI Query     │───▶│   Dolt Query    │
│   (bd ready)    │    │   (SQL)         │
└─────────────────┘    └─────────────────┘
```

1. **Query:** Commands read from fast local Dolt database via SQL
2. **Sync:** Use `bd dolt pull` to fetch updates from Dolt remotes

Key implementation:
- Backup restore: `cmd/bd/backup_restore.go`
- Issue bootstrap/migration: `cmd/bd/init.go`
- Dolt storage: `internal/storage/dolt/`

## Hash-Based Collision Prevention

The key insight that enables distributed operation: **content-based hashing for deduplication**.

### The Problem

Sequential IDs (bd-1, bd-2, bd-3) cause collisions when multiple agents create issues concurrently:

```
Branch A: bd create "Add OAuth"   → bd-10
Branch B: bd create "Add Stripe"  → bd-10 (collision!)
```

### The Solution

Hash-based IDs derived from random UUIDs ensure uniqueness:

```
Branch A: bd create "Add OAuth"   → bd-a1b2
Branch B: bd create "Add Stripe"  → bd-f14c (no collision)
```

### How It Works

1. **Issue creation:** Generate random UUID, derive short hash as ID
2. **Progressive scaling:** IDs start at 4 chars, grow to 5-6 chars as database grows
3. **Content hashing:** Each issue has a content hash for change detection
4. **Merge logic:** Same ID + different content = update, same ID + same content = skip

```
┌─────────────────────────────────────────────────────────────────┐
│                        Merge Logic                               │
│             (used by Dolt pull and init --from-jsonl)            │
│                                                                  │
│  For each issue in incoming data:                                │
│    1. Compute content hash                                       │
│    2. Look up existing issue by ID                               │
│    3. Compare hashes:                                            │
│       - Same hash → skip (already present)                       │
│       - Different hash → update (newer version)                  │
│       - No match → create (new issue)                            │
└─────────────────────────────────────────────────────────────────┘
```

This eliminates the need for central coordination while ensuring all machines converge to the same state.

See [COLLISION_MATH.md](COLLISION_MATH.md) for birthday paradox calculations on hash length vs collision probability.

## Server Architecture

Each workspace can run its own Dolt server for multi-writer access:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Dolt Server Mode                              │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐                             │
│  │ RPC Server  │    │ dolt        │                             │
│  │             │    │ sql-server  │                             │
│  └─────────────┘    └─────────────┘                             │
│         │                  │                                     │
│         └──────────────────┘                                     │
│                            │                                     │
│                            v                                     │
│                   ┌─────────────┐                                │
│                   │    Dolt     │                                │
│                   │   Database  │                                │
│                   └─────────────┘                                │
└─────────────────────────────────────────────────────────────────┘

     CLI commands ───SQL───▶ dolt sql-server ───▶ Database
                              or
     CLI commands ───SQL───▶ Database (embedded mode)
```

**Server mode:**
- Connects to `dolt sql-server` (multi-writer, high-concurrency)
- PID file at `.bd/dolt-server.pid`
- Logs at `.bd/dolt-server.log`
- **Shared server mode** (opt-in): all projects share a single Dolt server at
  `~/.bd/shared-server/` instead of per-project servers. Enable via
  `dolt.shared-server: true` in config.yaml or `BEADS_DOLT_SHARED_SERVER=1`.

**Embedded mode:**
- Direct database access (single-writer, no server process)

**Communication:**
- Protocol defined in `internal/rpc/protocol.go`
- Used by Dolt server mode for multi-writer access

## Data Types

Core types in `internal/types/types.go`:

| Type | Description | Key Fields |
|------|-------------|------------|
| **Issue** | Work item | ID, Title, Description, Status, Priority, Type |
| **Dependency** | Relationship | FromID, ToID, Type (blocks/related/parent-child/discovered-from) |
| **Label** | Tag | Name, Color, Description |
| **Comment** | Discussion | IssueID, Author, Content, Timestamp |
| **Event** | Audit trail | IssueID, Type, Data, Timestamp |

### Dependency Types

| Type | Semantic | Affects `bd ready`? |
|------|----------|---------------------|
| `blocks` | Issue X must close before Y starts | Yes |
| `parent-child` | Hierarchical (epic/subtask) | Yes (children blocked if parent blocked) |
| `related` | Soft link for reference | No |
| `discovered-from` | Found during work on parent | No |

### Status Flow

```
open ──▶ in_progress ──▶ closed
  │                        │
  └────────────────────────┘
         (reopen)
```

### Issue Schema

Each issue in the Dolt database has the following fields. Fields marked with `(optional)` use `omitempty` and are excluded when empty/zero.

**Core Identification:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (e.g., `bd-a1b2`) |

**Issue Content:**

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Issue title (required) |
| `description` | string | Detailed description (optional) |
| `design` | string | Design notes (optional) |
| `acceptance_criteria` | string | Acceptance criteria (optional) |
| `notes` | string | Additional notes (optional) |

**Status & Workflow:**

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Current status: `open`, `in_progress`, `blocked`, `deferred`, `closed`, `tombstone`, `pinned`, `hooked` (optional, defaults to `open`) |
| `priority` | int | Priority 0-4 where 0=critical, 4=backlog |
| `issue_type` | string | Type: `bug`, `feature`, `task`, `epic`, `chore`, `message`, `merge-request`, `molecule`, `gate`, `agent`, `role`, `convoy` (optional, defaults to `task`) |

**Assignment:**

| Field | Type | Description |
|-------|------|-------------|
| `assignee` | string | Assigned user/agent (optional) |
| `estimated_minutes` | int | Time estimate in minutes (optional) |

**Timestamps:**

| Field | Type | Description |
|-------|------|-------------|
| `created_at` | RFC3339 | When issue was created |
| `created_by` | string | Who created the issue (optional) |
| `updated_at` | RFC3339 | Last modification time |
| `closed_at` | RFC3339 | When issue was closed (optional, set when status=closed) |
| `close_reason` | string | Reason provided when closing (optional) |

**External Integration:**

| Field | Type | Description |
|-------|------|-------------|
| `external_ref` | string | External reference (e.g., `gh-9`, `jira-ABC`) (optional) |

**Relational Data:**

| Field | Type | Description |
|-------|------|-------------|
| `labels` | []string | Tags attached to the issue (optional) |
| `dependencies` | []Dependency | Relationships to other issues (optional) |
| `comments` | []Comment | Discussion comments (optional) |

**Tombstone Fields (soft-delete):**

| Field | Type | Description |
|-------|------|-------------|
| `deleted_at` | RFC3339 | When deleted (optional, set when status=tombstone) |
| `deleted_by` | string | Who deleted (optional) |
| `delete_reason` | string | Why deleted (optional) |
| `original_type` | string | Issue type before deletion (optional) |

**Note:** Fields with `json:"-"` tags (like `content_hash`, `source_repo`, `id_prefix`) are internal and not included in exports.

## Directory Structure

```
.bd/
├── dolt/             # Dolt database, sql-server.pid, sql-server.log (gitignored)
├── metadata.json     # Backend config (local, gitignored)
└── config.yaml       # Project config (optional)
```

## Key Code Paths

| Area | Files |
|------|-------|
| CLI entry | `cmd/bd/main.go` |
| Storage interface | `internal/storage/storage.go` |
| Dolt implementation | `internal/storage/dolt/` |
| RPC protocol | `internal/rpc/protocol.go`, `server_*.go` |
| Export logic (portability) | `cmd/bd/export.go` |
| Backup restore | `cmd/bd/backup_restore.go` |
| Issue bootstrap/migration | `cmd/bd/init.go` |

## Related Documentation

- [INTERNALS.md](INTERNALS.md) - FlushManager, Blocked Cache implementation details
- [ADVANCED.md](ADVANCED.md) - Advanced features and configuration
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Recovery procedures and common issues
- [FAQ.md](FAQ.md) - Common questions about the architecture
- [COLLISION_MATH.md](COLLISION_MATH.md) - Hash collision probability analysis
