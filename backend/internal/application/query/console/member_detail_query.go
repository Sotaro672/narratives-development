// backend/internal/application/query/console/member_detail_query.go
package query

import (
	"context"
	"errors"

	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var ErrMemberForbidden = errors.New("member forbidden")

// -----------------------------------------------------------------------------
// Query
// -----------------------------------------------------------------------------

type MemberDetailQuery struct {
	repo memdom.Repository
}

func NewMemberDetailQuery(repo memdom.Repository) *MemberDetailQuery {
	return &MemberDetailQuery{repo: repo}
}

// GetByUID は Firebase Auth UID から MemberRecord を取得します。
// GET /members/{uid} は Firebase UID 専用 endpoint として維持します。
func (q *MemberDetailQuery) GetByUID(ctx context.Context, uid string, companyID ...string) (MemberRecord, error) {
	if uid == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	rec, err := q.repo.GetByUID(ctx, uid)
	if err != nil {
		return MemberRecord{}, err
	}

	if len(companyID) > 0 && companyID[0] != "" && rec.Member.CompanyID != companyID[0] {
		return MemberRecord{}, ErrMemberForbidden
	}

	return MemberRecord{DocID: rec.DocID, Member: rec.Member}, nil
}

// GetByID は Firestore Member document ID から MemberRecord を取得します。
// GET /members/by-id/{memberId} から利用し、Firebase UIDとして解釈しません。
func (q *MemberDetailQuery) GetByID(ctx context.Context, memberID string, companyID ...string) (MemberRecord, error) {
	if memberID == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	rec, err := q.repo.GetByID(ctx, memberID)
	if err != nil {
		return MemberRecord{}, err
	}

	if rec.DocID == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	if len(companyID) > 0 && companyID[0] != "" && rec.Member.CompanyID != companyID[0] {
		return MemberRecord{}, ErrMemberForbidden
	}

	return MemberRecord{DocID: rec.DocID, Member: rec.Member}, nil
}
