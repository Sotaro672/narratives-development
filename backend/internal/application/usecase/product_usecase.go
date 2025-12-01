package usecase

import (
	"context"
	"strings"

	productdom "narratives/internal/domain/product"
)

// ProductRepo defines the minimal persistence port needed by ProductUsecase.
type ProductRepo interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, p productdom.Product) (productdom.Product, error)
	Save(ctx context.Context, p productdom.Product) (productdom.Product, error)
	Update(ctx context.Context, id string, p productdom.Product) (productdom.Product, error)

	// ★ 追加: productionId で絞り込んだ Product 一覧
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

// ProductUsecase orchestrates product operations.
type ProductUsecase struct {
	repo ProductRepo
}

func NewProductUsecase(repo ProductRepo) *ProductUsecase {
	return &ProductUsecase{repo: repo}
}

// ==========================
// Queries
// ==========================

func (u *ProductUsecase) GetByID(ctx context.Context, id string) (productdom.Product, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ★ 追加: 同一 productionId を持つ Product を一覧取得
func (u *ProductUsecase) ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error) {
	return u.repo.ListByProductionID(ctx, strings.TrimSpace(productionID))
}

// ==========================
// Commands
// ==========================

// Create: POST 時に ID / ModelID / ProductionID / PrintedAt / PrintedBy を確定させる
func (u *ProductUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	return u.repo.Create(ctx, p)
}

// Save: 既存の互換用途として残しておく（フルアップサート）
func (u *ProductUsecase) Save(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	return u.repo.Save(ctx, p)
}

// Update:
// - ID               … URL パスの id で決定（不変）
// - ModelID          … POST 時に確定、更新不可
// - ProductionID     … POST 時に確定、更新不可
// - PrintedAt/By     … POST 時に確定、更新不可
// - InspectionResult … 更新対象
// - ConnectedToken   … 更新対象
// - InspectedAt      … 更新対象（InspectionResult の入力日時）
// - InspectedBy      … 更新対象（InspectionResult の入力者）
func (u *ProductUsecase) Update(ctx context.Context, id string, in productdom.Product) (productdom.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productdom.Product{}, productdom.ErrInvalidID
	}

	// 既存レコードを取得して、更新可能なフィールドだけ差し替える
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productdom.Product{}, err
	}

	// ---- 更新可能フィールドだけ上書き ----
	current.InspectionResult = in.InspectionResult
	current.ConnectedToken = in.ConnectedToken
	current.InspectedAt = in.InspectedAt
	current.InspectedBy = in.InspectedBy
	// ID / ModelID / ProductionID / PrintedAt / PrintedBy は current の値を維持

	// 永続化
	return u.repo.Update(ctx, id, current)
}
