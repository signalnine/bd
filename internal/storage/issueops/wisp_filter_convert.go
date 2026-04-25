package issueops

import "github.com/signalnine/bd/internal/types"

// WispFilterToIssueFilter converts a types.WispFilter into an IssueFilter
// suitable for use with SearchIssuesInTx or searchTableInTx.
// The returned filter always has Ephemeral=true so queries are routed to the
// wisps table; callers do not need to set that flag.
func WispFilterToIssueFilter(f types.WispFilter) types.IssueFilter {
	ephemeral := true
	filter := types.IssueFilter{
		Ephemeral:     &ephemeral,
		IssueType:     f.Type,
		Status:        f.Status,
		UpdatedAfter:  f.UpdatedAfter,
		UpdatedBefore: f.UpdatedBefore,
		Limit:         f.Limit,
	}
	// When no explicit status filter is set and IncludeClosed is false,
	// exclude closed wisps. Callers that want closed wisps must opt in.
	if !f.IncludeClosed && f.Status == nil {
		filter.ExcludeStatus = []types.Status{types.StatusClosed}
	}
	return filter
}
