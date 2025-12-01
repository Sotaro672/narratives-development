// backend/internal/application/usecase/product_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	productdom "narratives/internal/domain/product"
)

// QR ペイロード生成時に使うベース URL（必要に応じて差し替えてください）
const defaultQRBaseURL = "https://example.com" // TODO: 実際のフロントエンド URL に変更する

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

// ★ 追加: 同一 productionId を持つ PrintLog を一覧取得（QrPayloads 付き）
func (u *ProductUsecase) ListPrintLogsByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error) {
	if u.printLogRepo == nil {
		return nil, fmt.Errorf("printLogRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return nil, productdom.ErrInvalidPrintLogProductionID
	}

	// 1) print_logs を取得
	logs, err := u.printLogRepo.ListByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	// 2) 各 productId ごとに QR ペイロード(JSON文字列) を生成して QrPayloads に詰める
	baseURL := defaultQRBaseURL

	for i := range logs {
		var payloads []string
		for _, productID := range logs[i].ProductIDs {
			productID = strings.TrimSpace(productID)
			if productID == "" {
				continue
			}

			// BuildProductQRValue は (productID, baseURL) を受け取り
			// QR コード用の JSON 文字列を返す想定
			payload, err := productdom.BuildProductQRValue(productID, baseURL)
			if err != nil {
				// 運用方針次第だが、ここではエラーを返して 500 に繋げる
				return nil, err
			}
			payloads = append(payloads, payload)
		}
		logs[i].QrPayloads = payloads
	}

	return logs, nil
}

// ★ 追加: 1 回の印刷分の Product 一覧から print_log を 1 件作成し、QrPayloads 付きで返す
func (u *ProductUsecase) CreatePrintLogForProduction(ctx context.Context, productionID string) (productdom.PrintLog, error) {
	if u.printLogRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("printLogRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintLogProductionID
	}

	// 該当 productionId の Product 一覧を取得
	products, err := u.repo.ListByProductionID(ctx, pid)
	if err != nil {
		return productdom.PrintLog{}, err
	}
	if len(products) == 0 {
		return productdom.PrintLog{}, fmt.Errorf("no products found for productionId=%s", pid)
	}

	// ProductID 一覧
	productIDs := make([]string, 0, len(products))
	for _, p := range products {
		if strings.TrimSpace(p.ID) != "" {
			productIDs = append(productIDs, strings.TrimSpace(p.ID))
		}
	}
	if len(productIDs) == 0 {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintLogProductIDs
	}

	// printedAt を決定
	// Product 側の PrintedAt があればそれを採用、なければ現在時刻
	var printedAt time.Time
	for _, p := range products {
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			printedAt = p.PrintedAt.UTC()
			break
		}
	}
	if printedAt.IsZero() {
		printedAt = time.Now().UTC()
	}

	// PrintLog エンティティ作成
	// ※ printedBy フィールドはドメイン構造体には残っているが、
	//   Firestore には保存していない（printLogToDoc から削除済み）。
	logID := fmt.Sprintf("%s-%d", pid, printedAt.UnixNano())
	log, err := productdom.NewPrintLog(
		logID,
		pid,
		productIDs,
		"system", // 互換用のダミー値。永続化はされない方針。
		printedAt,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// 保存
	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// QrPayloads を付与（JSON文字列の配列）
	baseURL := defaultQRBaseURL
	var payloads []string
	for _, productID := range created.ProductIDs {
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		payload, err := productdom.BuildProductQRValue(productID, baseURL)
		if err != nil {
			return productdom.PrintLog{}, err
		}
		payloads = append(payloads, payload)
	}
	created.QrPayloads = payloads

	return created, nil
}

// ==========================
// Commands
// ==========================

// Create: Product のみ作成する。
// 以前の仕様（Create のたびに 1 件ずつ print_log を作成）は廃止し、
// 「1 回の印刷バッチでまとめて PrintLog を作る」ために
// CreatePrintLogForProduction を別途呼び出す方式に変更。
func (u *ProductUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	created, err := u.repo.Create(ctx, p)
	if err != nil {
		return productdom.Product{}, err
	}
	return created, nil
}

// Save: 既存の互換用途として残しておく（フルアップサート）
func (u *ProductUsecase) Save(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	return u.repo.Save(ctx, p)
}

// Update:
// - ID               … URL パスの id で決定（不変）
// - ModelID          … POST 時に確定、更新不可
// - ProductionID     … POST 時に確定、更新不可
// - PrintedAt        … POST 時に確定、更新不可
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
	// ID / ModelID / ProductionID / PrintedAt は current の値を維持

	// 永続化
	return u.repo.Update(ctx, id, current)
}
