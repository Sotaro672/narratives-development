// backend/internal/application/query/console/member_detail_query.go
package query

import (
	"context"

	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// Query
// -----------------------------------------------------------------------------

type MemberDetailQuery struct {
	repo memdom.Repository
}

func NewMemberDetailQuery(repo memdom.Repository) *MemberDetailQuery {
	return &MemberDetailQuery{
		repo: repo,
	}
}

// GetByUID は Firebase Auth UID から MemberRecord を取得します。
func (q *MemberDetailQuery) GetByUID(ctx context.Context, uid string) (MemberRecord, error) {
	if uid == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	rec, err := q.repo.GetByUID(ctx, uid)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}
