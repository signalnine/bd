// Package beads provides a minimal public API for extending bd with custom orchestration.
//
// Most extensions should use direct SQL queries against bd's database.
// This package exports only the essential types and functions needed for
// Go-based extensions that want to use bd's storage layer programmatically.
//
// For detailed guidance on extending bd, see docs/EXTENDING.md.
package bd

import (
	"context"

	"github.com/steveyegge/bd/internal/project"
	"github.com/steveyegge/bd/internal/storage"
	"github.com/steveyegge/bd/internal/storage/embeddeddolt"
	"github.com/steveyegge/bd/internal/types"
)

// Store is the concrete embedded Dolt storage backend.
type Store = embeddeddolt.EmbeddedDoltStore

// Transaction provides atomic multi-operation support within a database transaction.
type Transaction = storage.Transaction

// VersionControlReader provides read-only version control operations.
type VersionControlReader interface {
	CurrentBranch(ctx context.Context) (string, error)
	ListBranches(ctx context.Context) ([]string, error)
	CommitExists(ctx context.Context, commitHash string) (bool, error)
	GetCurrentCommit(ctx context.Context) (string, error)
	Status(ctx context.Context) (*VCStatus, error)
	Log(ctx context.Context, limit int) ([]CommitInfo, error)
}

// Replication and version control types from internal/storage
type (
	RemoteInfo  = storage.RemoteInfo
	SyncResult  = storage.SyncResult
	SyncStatus  = storage.SyncStatus
	Conflict    = storage.Conflict
	CommitInfo  = storage.CommitInfo
	VCStatus    = storage.Status
	StatusEntry = storage.StatusEntry
)

// Open opens an embedded Dolt beads database at the given path.
func Open(ctx context.Context, bdDir string) (*Store, error) {
	return embeddeddolt.New(ctx, bdDir, "beads", "main")
}

// FindDatabasePath finds the beads database in the current directory tree
func FindDatabasePath() string {
	return project.FindDatabasePath()
}

// FindBdDir finds the .bd/ directory in the current directory tree.
// Returns empty string if not found.
func FindBdDir() string {
	return project.FindBdDir()
}

// DatabaseInfo contains information about a beads database
type DatabaseInfo = project.DatabaseInfo

// FindAllDatabases finds all beads databases in the system
func FindAllDatabases() []DatabaseInfo {
	return project.FindAllDatabases()
}

// RedirectInfo contains information about a beads directory redirect
type RedirectInfo = project.RedirectInfo

// GetRedirectInfo checks if the current beads directory is redirected.
// Returns RedirectInfo with IsRedirected=true if a redirect is active.
func GetRedirectInfo() RedirectInfo {
	return project.GetRedirectInfo()
}

// Core types from internal/types
type (
	Issue                       = types.Issue
	Status                      = types.Status
	IssueType                   = types.IssueType
	Dependency                  = types.Dependency
	DependencyType              = types.DependencyType
	Label                       = types.Label
	Comment                     = types.Comment
	Event                       = types.Event
	EventType                   = types.EventType
	BlockedIssue                = types.BlockedIssue
	TreeNode                    = types.TreeNode
	IssueFilter                 = types.IssueFilter
	WorkFilter                  = types.WorkFilter
	StaleFilter                 = types.StaleFilter
	DependencyCounts            = types.DependencyCounts
	IssueWithCounts             = types.IssueWithCounts
	IssueWithDependencyMetadata = types.IssueWithDependencyMetadata
	SortPolicy                  = types.SortPolicy
	EpicStatus                  = types.EpicStatus
	WispFilter                  = types.WispFilter
)

// Status constants
const (
	StatusOpen       = types.StatusOpen
	StatusInProgress = types.StatusInProgress
	StatusBlocked    = types.StatusBlocked
	StatusDeferred   = types.StatusDeferred
	StatusClosed     = types.StatusClosed
)

// IssueType constants
const (
	TypeBug     = types.TypeBug
	TypeFeature = types.TypeFeature
	TypeTask    = types.TypeTask
	TypeEpic    = types.TypeEpic
	TypeChore   = types.TypeChore
)

// DependencyType constants
const (
	DepBlocks            = types.DepBlocks
	DepRelated           = types.DepRelated
	DepParentChild       = types.DepParentChild
	DepDiscoveredFrom    = types.DepDiscoveredFrom
	DepConditionalBlocks = types.DepConditionalBlocks
)

// SortPolicy constants
const (
	SortPolicyHybrid   = types.SortPolicyHybrid
	SortPolicyPriority = types.SortPolicyPriority
	SortPolicyOldest   = types.SortPolicyOldest
)

// EventType constants
const (
	EventCreated           = types.EventCreated
	EventUpdated           = types.EventUpdated
	EventStatusChanged     = types.EventStatusChanged
	EventCommented         = types.EventCommented
	EventClosed            = types.EventClosed
	EventReopened          = types.EventReopened
	EventDependencyAdded   = types.EventDependencyAdded
	EventDependencyRemoved = types.EventDependencyRemoved
	EventLabelAdded        = types.EventLabelAdded
	EventLabelRemoved      = types.EventLabelRemoved
	EventCompacted         = types.EventCompacted
)
