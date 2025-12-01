package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// ★ PrintLog 用リポジトリ
type PrintLogRepo interface {
	Create(ctx context.Context, log productdom.PrintLog) (productdom.PrintLog, error)

	// ★ 追加: productionId で絞り込んだ PrintLog 一覧
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error)
}

// ProductUsecase orchestrates product operations.
type ProductUsecase struct {
	repo         ProductRepo
	printLogRepo PrintLogRepo
}

func NewProductUsecase(repo ProductRepo, printLogRepo PrintLogRepo) *ProductUsecase {
	return &ProductUsecase{
		repo:         repo,
		printLogRepo: printLogRepo,
	}
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

// ★ 追加: 同一 productionId を持つ PrintLog を一覧取得
//
//	（この中で BuildProductQRValue による QR ペイロード付与を行う想定）
func (u *ProductUsecase) ListPrintLogsByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error) {
	if u.printLogRepo == nil {
		return nil, fmt.Errorf("printLogRepo is nil")
	}
	return u.printLogRepo.ListByProductionID(ctx, strings.TrimSpace(productionID))
}

// ★ 追加: 1回の印刷バッチに対してまとめて 1 件の print_log を作成する
//
//   - productionID: 対象の生産計画ID
//   - printedAt   : フロント側で各 Product 作成時に付与した printedAt（全Product共通）
//
// 手順:
//  1. repo.ListByProductionID で該当 productionId の Product 一覧を取得
//  2. PrintedAt が指定値と一致する Product だけを抽出
//  3. その productId 一覧 + PrintedBy + PrintedAt で 1 件の PrintLog を作成
func (u *ProductUsecase) CreatePrintLogBatch(
	ctx context.Context,
	productionID string,
	printedAt time.Time,
) (productdom.PrintLog, error) {

	if u.printLogRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("printLogRepo is nil")
	}

	id := strings.TrimSpace(productionID)
	if id == "" {
		return productdom.PrintLog{}, productdom.ErrInvalidProductionID
	}
	if printedAt.IsZero() {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintedAt
	}
	printedAtUTC := printedAt.UTC()

	// 1. 該当 productionId の Product 一覧を取得
	products, err := u.repo.ListByProductionID(ctx, id)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// 2. PrintedAt が一致する Product だけを抽出
	var productIDs []string
	printedBy := ""

	for _, p := range products {
		if p.PrintedAt == nil {
			continue
		}
		if !p.PrintedAt.UTC().Equal(printedAtUTC) {
			continue
		}

		productIDs = append(productIDs, p.ID)

		// PrintedBy は 1 件目から拝借（全て同じ UID の想定）
		if printedBy == "" && p.PrintedBy != nil {
			if pb := strings.TrimSpace(*p.PrintedBy); pb != "" {
				printedBy = pb
			}
		}
	}

	if len(productIDs) == 0 {
		return productdom.PrintLog{}, fmt.Errorf(
			"no products found for productionId=%s and printedAt=%s",
			id, printedAtUTC.Format(time.RFC3339Nano),
		)
	}
	if printedBy == "" {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintedBy
	}

	// 3. 1件分の PrintLog を作成
	logID := fmt.Sprintf("%s-%d", id, printedAtUTC.UnixNano())

	log, err := productdom.NewPrintLog(
		logID,
		id,
		productIDs,
		printedBy,
		printedAtUTC,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	return created, nil
}

// ==========================
// Commands
// ==========================

// Create: POST 時に ID / ModelID / ProductionID / PrintedAt / PrintedBy を確定させる。
//
// 以前はここで Product ごとに print_log を 1 件ずつ作成していたが、
// 「1回の印刷バッチで印刷された product 一覧をまとめて 1件の print_log にする」
// 仕様に変更したため、ここでは **print_log は作成しない**。
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
