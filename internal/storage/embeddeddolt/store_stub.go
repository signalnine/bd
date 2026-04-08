//go:build !cgo

package embeddeddolt

import (
	"context"
	"errors"
	"time"

	"github.com/signalnine/bd/internal/storage"
	"github.com/signalnine/bd/internal/types"
)

// EmbeddedDoltStore is a stub for builds without CGO.
type EmbeddedDoltStore struct {
	dataDir  string
	database string
	branch   string
}

// Option configures optional behavior for New (stub: no-op).
type Option func(*struct{})

// WithLock is a no-op in non-CGO builds.
func WithLock(_ Unlocker) Option {
	return func(*struct{}) {}
}

// New returns an error when CGO is not enabled.
func New(_ context.Context, _, _, _ string, _ ...Option) (*EmbeddedDoltStore, error) {
	return nil, errNoCGO
}

var errNoCGO = errors.New("embeddeddolt: requires CGO (build with CGO_ENABLED=1)")

// --- Issue CRUD ---

func (s *EmbeddedDoltStore) CreateIssue(_ context.Context, _ *types.Issue, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) CreateIssues(_ context.Context, _ []*types.Issue, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) CreateIssuesWithFullOptions(_ context.Context, _ []*types.Issue, _ string, _ storage.BatchCreateOptions) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetIssue(_ context.Context, _ string) (*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetIssueByExternalRef(_ context.Context, _ string) (*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) DeleteIssue(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) DeleteIssues(_ context.Context, _ []string, _, _, _ bool) (*types.DeleteIssuesResult, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) DeleteIssuesBySourceRepo(_ context.Context, _ string) (int, error) {
	return 0, errNoCGO
}

func (s *EmbeddedDoltStore) ClaimIssue(_ context.Context, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) UpdateIssue(_ context.Context, _ string, _ map[string]interface{}, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ReopenIssue(_ context.Context, _ string, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) UpdateIssueType(_ context.Context, _ string, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) CloseIssue(_ context.Context, _ string, _ string, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) UpdateIssueID(_ context.Context, _, _ string, _ *types.Issue, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) PromoteFromEphemeral(_ context.Context, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) RenameCounterPrefix(_ context.Context, _, _ string) error {
	return errNoCGO
}

// --- Queries ---

func (s *EmbeddedDoltStore) SearchIssues(_ context.Context, _ string, _ types.IssueFilter) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) ListWisps(_ context.Context, _ types.WispFilter) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetReadyWork(_ context.Context, _ types.WorkFilter) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetBlockedIssues(_ context.Context, _ types.WorkFilter) ([]*types.BlockedIssue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetEpicsEligibleForClosure(_ context.Context) ([]*types.EpicStatus, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetIssuesByLabel(_ context.Context, _ string) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetIssuesByIDs(_ context.Context, _ []string) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) IsBlocked(_ context.Context, _ string) (bool, []string, error) {
	return false, nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetNewlyUnblockedByClose(_ context.Context, _ string) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetStaleIssues(_ context.Context, _ types.StaleFilter) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetMoleculeProgress(_ context.Context, _ string) (*types.MoleculeProgressStats, error) {
	return nil, errNoCGO
}

// --- Labels ---

func (s *EmbeddedDoltStore) GetLabels(_ context.Context, _ string) ([]string, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) AddLabel(_ context.Context, _, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) RemoveLabel(_ context.Context, _, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetLabelsForIssues(_ context.Context, _ []string) (map[string][]string, error) {
	return nil, errNoCGO
}

// --- Dependencies ---

func (s *EmbeddedDoltStore) AddDependency(_ context.Context, _ *types.Dependency, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) RemoveDependency(_ context.Context, _, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetDependencies(_ context.Context, _ string) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependents(_ context.Context, _ string) ([]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependenciesWithMetadata(_ context.Context, _ string) ([]*types.IssueWithDependencyMetadata, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependentsWithMetadata(_ context.Context, _ string) ([]*types.IssueWithDependencyMetadata, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependencyTree(_ context.Context, _ string, _ int, _, _ bool) ([]*types.TreeNode, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) DetectCycles(_ context.Context) ([][]*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependencyRecords(_ context.Context, _ string) ([]*types.Dependency, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetAllDependencyRecords(_ context.Context) (map[string][]*types.Dependency, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependencyRecordsForIssues(_ context.Context, _ []string) (map[string][]*types.Dependency, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetDependencyCounts(_ context.Context, _ []string) (map[string]*types.DependencyCounts, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetBlockingInfoForIssues(_ context.Context, _ []string) (map[string][]string, map[string][]string, map[string]string, error) {
	return nil, nil, nil, errNoCGO
}

func (s *EmbeddedDoltStore) FindWispDependentsRecursive(_ context.Context, _ []string) (map[string]bool, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) RenameDependencyPrefix(_ context.Context, _, _ string) error {
	return errNoCGO
}

// --- Comments ---

func (s *EmbeddedDoltStore) AddIssueComment(_ context.Context, _, _, _ string) (*types.Comment, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetIssueComments(_ context.Context, _ string) ([]*types.Comment, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) AddComment(_ context.Context, _, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ImportIssueComment(_ context.Context, _, _, _ string, _ time.Time) (*types.Comment, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetCommentsForIssues(_ context.Context, _ []string) (map[string][]*types.Comment, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetCommentCounts(_ context.Context, _ []string) (map[string]int, error) {
	return nil, errNoCGO
}

// --- Events ---

func (s *EmbeddedDoltStore) GetEvents(_ context.Context, _ string, _ int) ([]*types.Event, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetAllEventsSince(_ context.Context, _ time.Time) ([]*types.Event, error) {
	return nil, errNoCGO
}

// --- Config / Metadata ---

func (s *EmbeddedDoltStore) SetConfig(_ context.Context, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetConfig(_ context.Context, _ string) (string, error) {
	return "", errNoCGO
}

func (s *EmbeddedDoltStore) GetAllConfig(_ context.Context) (map[string]string, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) DeleteConfig(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetMetadata(_ context.Context, _ string) (string, error) {
	return "", errNoCGO
}

func (s *EmbeddedDoltStore) SetMetadata(_ context.Context, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetInfraTypes(_ context.Context) map[string]bool {
	return nil
}

func (s *EmbeddedDoltStore) IsInfraTypeCtx(_ context.Context, _ types.IssueType) bool {
	return false
}

func (s *EmbeddedDoltStore) GetCustomStatuses(_ context.Context) ([]string, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetCustomStatusesDetailed(_ context.Context) ([]types.CustomStatus, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetCustomTypes(_ context.Context) ([]string, error) {
	return nil, errNoCGO
}

// --- Statistics ---

func (s *EmbeddedDoltStore) GetStatistics(_ context.Context) (*types.Statistics, error) {
	return nil, errNoCGO
}

// --- Federation ---

func (s *EmbeddedDoltStore) AddFederationPeer(_ context.Context, _ *storage.FederationPeer) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetFederationPeer(_ context.Context, _ string) (*storage.FederationPeer, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) ListFederationPeers(_ context.Context) ([]*storage.FederationPeer, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) RemoveFederationPeer(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Sync(_ context.Context, _ string, _ string) (*storage.SyncResult, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) SyncStatus(_ context.Context, _ string) (*storage.SyncStatus, error) {
	return nil, errNoCGO
}

// --- Version Control ---

func (s *EmbeddedDoltStore) Commit(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) AddRemote(_ context.Context, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) HasRemote(_ context.Context, _ string) (bool, error) {
	return false, errNoCGO
}

func (s *EmbeddedDoltStore) Branch(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Checkout(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) CurrentBranch(_ context.Context) (string, error) {
	return "", errNoCGO
}

func (s *EmbeddedDoltStore) DeleteBranch(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ListBranches(_ context.Context) ([]string, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) CommitExists(_ context.Context, _ string) (bool, error) {
	return false, errNoCGO
}

func (s *EmbeddedDoltStore) Status(_ context.Context) (*storage.Status, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) Log(_ context.Context, _ int) ([]storage.CommitInfo, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) Merge(_ context.Context, _ string) ([]storage.Conflict, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetConflicts(_ context.Context) ([]storage.Conflict, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) ResolveConflicts(_ context.Context, _ string, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) RemoveRemote(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ListRemotes(_ context.Context) ([]storage.RemoteInfo, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) Push(_ context.Context) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Pull(_ context.Context) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ForcePush(_ context.Context) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Fetch(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) PushTo(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) PullFrom(_ context.Context, _ string) ([]storage.Conflict, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) BackupAdd(_ context.Context, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) BackupSync(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) BackupRemove(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) BackupDatabase(_ context.Context, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) RestoreDatabase(_ context.Context, _ string, _ bool) error {
	return errNoCGO
}

// --- History / Diff ---

func (s *EmbeddedDoltStore) CommitPending(_ context.Context, _ string) (bool, error) {
	return false, errNoCGO
}

func (s *EmbeddedDoltStore) GetCurrentCommit(_ context.Context) (string, error) {
	return "", errNoCGO
}

func (s *EmbeddedDoltStore) History(_ context.Context, _ string) ([]*storage.HistoryEntry, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) AsOf(_ context.Context, _ string, _ string) (*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) Diff(_ context.Context, _, _ string) ([]*storage.DiffEntry, error) {
	return nil, errNoCGO
}

// --- Store lifecycle ---

func (s *EmbeddedDoltStore) Close() error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) IsClosed() bool {
	return true
}

func (s *EmbeddedDoltStore) Path() string {
	return ""
}

func (s *EmbeddedDoltStore) CLIDir() string {
	return ""
}

// --- Maintenance ---

func (s *EmbeddedDoltStore) DoltGC(_ context.Context) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Flatten(_ context.Context) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) Compact(_ context.Context, _, _ string, _ int, _ []string) error {
	return errNoCGO
}

// --- Compaction ---

func (s *EmbeddedDoltStore) CheckEligibility(_ context.Context, _ string, _ int) (bool, string, error) {
	return false, "", errNoCGO
}

func (s *EmbeddedDoltStore) ApplyCompaction(_ context.Context, _ string, _, _ int, _ int, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) GetTier1Candidates(_ context.Context) ([]*types.CompactionCandidate, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) GetTier2Candidates(_ context.Context) ([]*types.CompactionCandidate, error) {
	return nil, errNoCGO
}

// --- Repo Mtime ---

func (s *EmbeddedDoltStore) GetRepoMtime(_ context.Context, _ string) (int64, error) {
	return 0, errNoCGO
}

func (s *EmbeddedDoltStore) SetRepoMtime(_ context.Context, _, _ string, _ int64) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) ClearRepoMtime(_ context.Context, _ string) error {
	return errNoCGO
}

// --- Molecule ---

func (s *EmbeddedDoltStore) GetMoleculeLastActivity(_ context.Context, _ string) (*types.MoleculeLastActivity, error) {
	return nil, errNoCGO
}

// --- Transaction ---

func (s *EmbeddedDoltStore) RunInTransaction(_ context.Context, _ string, _ func(storage.Transaction) error) error {
	return errNoCGO
}

// --- Child ID ---

func (s *EmbeddedDoltStore) GetNextChildID(_ context.Context, _ string) (string, error) {
	return "", errNoCGO
}

// --- Slots ---

func (s *EmbeddedDoltStore) SlotSet(_ context.Context, _, _, _, _ string) error {
	return errNoCGO
}

func (s *EmbeddedDoltStore) SlotGet(_ context.Context, _, _ string) (string, error) {
	return "", errNoCGO
}

func (s *EmbeddedDoltStore) SlotClear(_ context.Context, _, _, _ string) error {
	return errNoCGO
}

// --- Merge Slot ---

func (s *EmbeddedDoltStore) MergeSlotCreate(_ context.Context, _ string) (*types.Issue, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) MergeSlotCheck(_ context.Context) (*storage.MergeSlotStatus, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) MergeSlotAcquire(_ context.Context, _, _ string, _ bool) (*storage.MergeSlotResult, error) {
	return nil, errNoCGO
}

func (s *EmbeddedDoltStore) MergeSlotRelease(_ context.Context, _, _ string) error {
	return errNoCGO
}
