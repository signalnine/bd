//go:build cgo

package embeddeddolt

import (
	"context"
	"database/sql"

	"github.com/steveyegge/bd/internal/storage/issueops"
	"github.com/steveyegge/bd/internal/types"
)

func (s *EmbeddedDoltStore) GetIssue(ctx context.Context, id string) (*types.Issue, error) {
	var issue *types.Issue
	err := s.withConn(ctx, false, func(tx *sql.Tx) error {
		var err error
		issue, err = issueops.GetIssueInTx(ctx, tx, id)
		return err
	})
	return issue, err
}
