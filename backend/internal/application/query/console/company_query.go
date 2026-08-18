package query

import (
	"context"
	"fmt"
	"time"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
)

type CompanyDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Admin    string `json:"admin"`
	IsActive bool   `json:"isActive"`

	CreatedAt     time.Time `json:"createdAt"`
	CreatedBy     string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName"`

	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedBy     string    `json:"updatedBy"`
	UpdatedByName string    `json:"updatedByName"`

	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`
}

type CompanyQuery struct {
	companyRepo companydom.Repository
	memberRepo  memberdom.Repository
}

func NewCompanyQuery(
	companyRepo companydom.Repository,
	memberRepo memberdom.Repository,
) *CompanyQuery {
	return &CompanyQuery{
		companyRepo: companyRepo,
		memberRepo:  memberRepo,
	}
}

func (q *CompanyQuery) GetByID(
	ctx context.Context,
	id string,
) (CompanyDetail, error) {
	if q == nil || q.companyRepo == nil {
		return CompanyDetail{}, fmt.Errorf("company query/repo is nil")
	}

	if id == "" {
		return CompanyDetail{}, companydom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return CompanyDetail{}, err
	}

	return CompanyDetail{
		ID:       company.ID,
		Name:     company.Name,
		Admin:    company.Admin,
		IsActive: company.IsActive,

		CreatedAt: company.CreatedAt,
		CreatedBy: company.CreatedBy,
		CreatedByName: q.resolveMemberName(
			ctx,
			company.CreatedBy,
		),

		UpdatedAt: company.UpdatedAt,
		UpdatedBy: company.UpdatedBy,
		UpdatedByName: q.resolveMemberName(
			ctx,
			company.UpdatedBy,
		),

		DeletedAt: company.DeletedAt,
		DeletedBy: company.DeletedBy,
	}, nil
}

func (q *CompanyQuery) resolveMemberName(
	ctx context.Context,
	memberUID string,
) string {
	if q == nil ||
		q.memberRepo == nil ||
		memberUID == "" {
		return ""
	}

	rec, err := q.memberRepo.GetByUID(
		ctx,
		memberUID,
	)
	if err == nil {
		return memberdom.FormatLastFirst(
			rec.Member.LastName,
			rec.Member.FirstName,
		)
	}

	// 既存データに member document ID が残っている場合の互換用。
	rec, err = q.memberRepo.GetByID(
		ctx,
		memberUID,
	)
	if err != nil {
		return ""
	}

	return memberdom.FormatLastFirst(
		rec.Member.LastName,
		rec.Member.FirstName,
	)
}
