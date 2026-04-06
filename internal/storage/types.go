// Package storage provides shared value types for issue storage.
package storage

import (
	"time"

	"github.com/steveyegge/bd/internal/types"
)

// HistoryEntry represents an issue at a specific point in history.
type HistoryEntry struct {
	CommitHash string       // The commit hash at this point
	Committer  string       // Who made the commit
	CommitDate time.Time    // When the commit was made
	Issue      *types.Issue // The issue state at that commit
}

// DiffEntry represents a change between two commits.
type DiffEntry struct {
	IssueID  string       // The ID of the affected issue
	DiffType string       // "added", "modified", or "removed"
	OldValue *types.Issue // State before (nil for "added")
	NewValue *types.Issue // State after (nil for "removed")
}

// Conflict represents a merge conflict.
type Conflict struct {
	IssueID     string      // The ID of the conflicting issue
	Field       string      // Which field has the conflict (empty for table-level)
	OursValue   interface{} // Value on current branch
	TheirsValue interface{} // Value on merged branch
}

// RemoteInfo describes a configured remote.
type RemoteInfo struct {
	Name string // Remote name (e.g., "town-beta")
	URL  string // Remote URL (e.g., "dolthub://org/repo")
}

// CommitInfo represents a version control commit.
type CommitInfo struct {
	Hash    string
	Author  string
	Email   string
	Date    time.Time
	Message string
}

// StatusEntry represents a changed table in the working set.
type StatusEntry struct {
	Table  string
	Status string // "new", "modified", "deleted"
}

// Status represents the current repository status.
type Status struct {
	Staged   []StatusEntry
	Unstaged []StatusEntry
}

// SyncResult contains the outcome of a Sync operation.
type SyncResult struct {
	Peer              string
	StartTime         time.Time
	EndTime           time.Time
	Fetched           bool
	Merged            bool
	Pushed            bool
	PulledCommits     int
	PushedCommits     int
	Conflicts         []Conflict
	ConflictsResolved bool
	Error             error
	PushError         error // Non-fatal push error
}

// SyncStatus describes the synchronization state with a peer.
type SyncStatus struct {
	Peer         string    // Peer name
	LastSync     time.Time // When last synced
	LocalAhead   int       // Commits ahead of peer
	LocalBehind  int       // Commits behind peer
	HasConflicts bool      // Whether there are unresolved conflicts
}

// FederationPeer represents a remote peer with authentication credentials.
type FederationPeer struct {
	Name        string     // Unique name for this peer (used as remote name)
	RemoteURL   string     // Dolt remote URL (e.g., http://host:port/org/db)
	Username    string     // SQL username for authentication
	Password    string     // Password (decrypted, not stored directly)
	Sovereignty string     // Sovereignty tier: T1, T2, T3, T4
	LastSync    *time.Time // Last successful sync time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
