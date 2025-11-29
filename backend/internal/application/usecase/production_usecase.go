// backend/internal/application/usecase/production_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	memberdom "narratives/internal/domain/member"
	productiondom "narratives/internal/domain/production"
)

// ProductionRepo defines the minimal persistence port needed by ProductionUsecase.
type ProductionRepo interface {
	GetByID(ctx context.Context, id string) (productiondom.Production, error)
	Exists(ctx context.Context, id string) (bool, error)

	// ★ 一覧取得
	List(ctx context.Context) ([]productiondom.Production, error)

	Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
	Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
	Delete(ctx context.Context, id string) error
}

// ★ フロント用 DTO: 担当者名付き Production
type ProductionWithAssigneeName struct {
	productiondom.Production
	AssigneeName string `json:"assigneeName"`
}

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
	repo      ProductionRepo
	memberSvc *memberdom.Service
	now       func() time.Time
}

func NewProductionUsecase(repo ProductionRepo, memberSvc *memberdom.Service) *ProductionUsecase {
	return &ProductionUsecase{
		repo:      repo,
		memberSvc: memberSvc,
		now:       time.Now,
	}
}

// ============================
// Queries
// ============================

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ★ 素の一覧（必要なところ向けに残す）
func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	return u.repo.List(ctx)
}

// ★ 担当者名付き一覧（/productions 用）
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]ProductionWithAssigneeName, error) {
	list, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ProductionWithAssigneeName, 0, len(list))

	for _, p := range list {
		name := ""

		// memberSvc が DI されていて、assigneeId が空でなければ名前を解決
		if u.memberSvc != nil && strings.TrimSpace(p.AssigneeID) != "" {
			if n, err := u.memberSvc.GetNameLastFirstByID(ctx, p.AssigneeID); err == nil {
				name = n
			}
		}

		out = append(out, ProductionWithAssigneeName{
			Production:   p,
			AssigneeName: name,
		})
	}

	return out, nil
}

// ============================
// Commands
// ============================

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	// Best-effort normalization of timestamps commonly present on entities
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}
	return u.repo.Create(ctx, p)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}
	return u.repo.Save(ctx, p)
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
