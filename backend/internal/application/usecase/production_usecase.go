// backend/internal/application/usecase/production_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	memberdom "narratives/internal/domain/member"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Ports
// ============================================================

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

// ★ フロント用 DTO: 担当者名 + プロダクト名付き Production
type ProductionWithAssigneeName struct {
	productiondom.Production
	AssigneeName         string `json:"assigneeName"`
	ProductBlueprintName string `json:"productBlueprintName"`
}

// ============================================================
// Usecase
// ============================================================

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
	repo      ProductionRepo
	memberSvc *memberdom.Service

	// ★ productBlueprintId → ProductName 解決用
	//   interface 自体は productBlueprint_usecase.go 側で定義済みの
	//   `type ProductBlueprintRepo interface {...}` をそのまま利用する。
	pbRepo ProductBlueprintRepo

	now func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
	memberSvc *memberdom.Service,
	pbRepo ProductBlueprintRepo,
) *ProductionUsecase {
	return &ProductionUsecase{
		repo:      repo,
		memberSvc: memberSvc,
		pbRepo:    pbRepo,
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

// ★ 担当者ID から表示名を解決する resolver 的メソッド
//   - memberSvc が設定されていない場合や ID が空文字の場合は "" を返す
//   - memberSvc 側（member.Service.GetNameLastFirstByID）からのエラーはそのまま返却
func (u *ProductionUsecase) ResolveAssigneeName(ctx context.Context, assigneeID string) (string, error) {
	if u.memberSvc == nil {
		// 名前解決サービスが DI されていない場合は何も表示しない前提
		return "", nil
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return "", nil
	}

	return u.memberSvc.GetNameLastFirstByID(ctx, assigneeID)
}

// ★ productBlueprintId からプロダクト名を解決する resolver 的メソッド
//   - pbRepo が設定されていない場合や ID が空文字の場合は "" を返す
func (u *ProductionUsecase) ResolveProductBlueprintName(ctx context.Context, blueprintID string) (string, error) {
	if u.pbRepo == nil {
		return "", nil
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return "", nil
	}

	pb, err := u.pbRepo.GetByID(ctx, blueprintID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pb.ProductName), nil
}

// ★ 担当者名 + プロダクト名付き一覧（/productions 用）
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]ProductionWithAssigneeName, error) {
	list, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ProductionWithAssigneeName, 0, len(list))

	for _, p := range list {
		// 担当者名
		name := ""
		if u.memberSvc != nil && strings.TrimSpace(p.AssigneeID) != "" {
			if n, err := u.memberSvc.GetNameLastFirstByID(ctx, p.AssigneeID); err == nil {
				name = n
			}
		}

		// プロダクト名
		productName := ""
		if u.pbRepo != nil && strings.TrimSpace(p.ProductBlueprintID) != "" {
			if pb, err := u.pbRepo.GetByID(ctx, p.ProductBlueprintID); err == nil {
				productName = strings.TrimSpace(pb.ProductName)
			}
		}

		out = append(out, ProductionWithAssigneeName{
			Production:           p,
			AssigneeName:         name,
			ProductBlueprintName: productName,
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
