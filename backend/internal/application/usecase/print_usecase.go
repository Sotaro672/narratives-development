// backend/internal/application/usecase/print_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	// ★ 追加: ModelNumberRepo 用
	productdom "narratives/internal/domain/product"
)

// QR コードに埋め込む公開 URL のベース
// 👉 https://narratives.jp/{productId} という形で利用
const publicQRBaseURL = "https://narratives.jp"

// ProductRepo defines the minimal persistence port needed by PrintUsecase
// to operate on Product entities.
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

// ★ Inspection 用リポジトリ（print_log と同じ product ドメイン配下の集約として扱う）
type InspectionRepo interface {
	// inspections/{productionId} を新規作成
	Create(ctx context.Context, batch productdom.InspectionBatch) (productdom.InspectionBatch, error)

	// productionId から inspections を取得
	GetByProductionID(ctx context.Context, productionID string) (productdom.InspectionBatch, error)

	// 既存バッチを保存（フルアップサート想定）
	Save(ctx context.Context, batch productdom.InspectionBatch) (productdom.InspectionBatch, error)
}

// PrintUsecase orchestrates print & inspection operations around products.
type PrintUsecase struct {
	repo           ProductRepo
	printLogRepo   PrintLogRepo
	inspectionRepo InspectionRepo

	// ★ 追加: modelId → modelNumber 解決用
	modelNumberRepo ModelNumberRepo
}

func NewPrintUsecase(
	repo ProductRepo,
	printLogRepo PrintLogRepo,
	inspectionRepo InspectionRepo,
	modelNumberRepo ModelNumberRepo,
) *PrintUsecase {
	return &PrintUsecase{
		repo:            repo,
		printLogRepo:    printLogRepo,
		inspectionRepo:  inspectionRepo,
		modelNumberRepo: modelNumberRepo,
	}
}

// ==========================
// Queries
// ==========================

func (u *PrintUsecase) GetByID(ctx context.Context, id string) (productdom.Product, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *PrintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ★ 追加: 同一 productionId を持つ Product を一覧取得
func (u *PrintUsecase) ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error) {
	return u.repo.ListByProductionID(ctx, strings.TrimSpace(productionID))
}

// ★ 追加: 同一 productionId を持つ PrintLog を一覧取得（QrPayloads 付き）
func (u *PrintUsecase) ListPrintLogsByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error) {
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

	// 2) 各 productId ごとに QR ペイロードを生成して QrPayloads に詰める
	//    👉 QR には「https://narratives.jp/{productId}」を埋め込む
	for i := range logs {
		var payloads []string
		for _, productID := range logs[i].ProductIDs {
			productID = strings.TrimSpace(productID)
			if productID == "" {
				continue
			}
			url := fmt.Sprintf("%s/%s", publicQRBaseURL, productID)
			payloads = append(payloads, url)
		}
		logs[i].QrPayloads = payloads
	}

	return logs, nil
}

// ★ 追加: inspections を単独で作成する
//
// POST /products/inspections 用
func (u *PrintUsecase) CreateInspectionBatchForProduction(
	ctx context.Context,
	productionID string,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	// 対象 productionId の Product 一覧を取得
	products, err := u.repo.ListByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}
	if len(products) == 0 {
		return productdom.InspectionBatch{}, fmt.Errorf("no products found for productionId=%s", pid)
	}

	// ProductID 一覧 + productId -> modelId マップ
	productIDs := make([]string, 0, len(products))
	modelIDByProductID := make(map[string]string, len(products))
	for _, p := range products {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		productIDs = append(productIDs, id)
		modelIDByProductID[id] = strings.TrimSpace(p.ModelID) // ★ Product の ModelID を保持
	}
	if len(productIDs) == 0 {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// ★ modelId → modelNumber のキャッシュを構築（ModelNumberRepo があれば）
	modelNumberByModelID := map[string]string{}
	if u.modelNumberRepo != nil {
		for _, mid := range modelIDByProductID {
			mid = strings.TrimSpace(mid)
			if mid == "" {
				continue
			}
			if _, exists := modelNumberByModelID[mid]; exists {
				continue
			}
			mv, err := u.modelNumberRepo.GetModelVariationByID(ctx, mid)
			if err != nil {
				continue
			}
			mn := strings.TrimSpace(mv.ModelNumber)
			if mn != "" {
				modelNumberByModelID[mid] = mn
			}
		}
	}

	// InspectionBatch エンティティ作成（全て notYet, status=inspecting）
	// quantity / totalPassed / requestedBy / requestedAt / mintedAt / tokenBlueprintId
	// は NewInspectionBatch 側で初期化される
	batch, err := productdom.NewInspectionBatch(
		pid,
		productdom.InspectionStatusInspecting,
		productIDs,
	)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// ★ InspectionItem に modelId / modelNumber を埋め込む
	for i := range batch.Inspections {
		pid := batch.Inspections[i].ProductID
		if mid, ok := modelIDByProductID[pid]; ok {
			mid = strings.TrimSpace(mid)
			batch.Inspections[i].ModelID = mid

			if mn, ok := modelNumberByModelID[mid]; ok && mn != "" {
				mnCopy := mn
				batch.Inspections[i].ModelNumber = &mnCopy
			}
		}
	}

	created, err := u.inspectionRepo.Create(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return created, nil
}

// ★ 追加: 1 回の印刷分の Product 一覧から print_log を 1 件作成し、
//
//	同じタイミングで inspections を 1 件作成する。
func (u *PrintUsecase) CreatePrintLogForProduction(ctx context.Context, productionID string) (productdom.PrintLog, error) {
	if u.printLogRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("printLogRepo is nil")
	}
	if u.inspectionRepo == nil {
		// print_log と inspection はセットで作る前提なので、nil は構成エラー扱い
		return productdom.PrintLog{}, fmt.Errorf("inspectionRepo is nil")
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

	// ProductID 一覧 + productId -> modelId マップ
	productIDs := make([]string, 0, len(products))
	modelIDByProductID := make(map[string]string, len(products))
	for _, p := range products {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		productIDs = append(productIDs, id)
		modelIDByProductID[id] = strings.TrimSpace(p.ModelID)
	}
	if len(productIDs) == 0 {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintLogProductIDs
	}

	// ★ modelId → modelNumber のキャッシュを構築
	modelNumberByModelID := map[string]string{}
	if u.modelNumberRepo != nil {
		for _, mid := range modelIDByProductID {
			mid = strings.TrimSpace(mid)
			if mid == "" {
				continue
			}
			if _, exists := modelNumberByModelID[mid]; exists {
				continue
			}
			mv, err := u.modelNumberRepo.GetModelVariationByID(ctx, mid)
			if err != nil {
				continue
			}
			mn := strings.TrimSpace(mv.ModelNumber)
			if mn != "" {
				modelNumberByModelID[mid] = mn
			}
		}
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

	// ★ ここで inspections/{productionId} 用のバッチを作成
	//   - inspectionResult / inspectedBy / inspectedAt はすべて notYet / nil で初期化
	//   - status は "inspecting" 固定で開始
	batch, err := productdom.NewInspectionBatch(
		pid,
		productdom.InspectionStatusInspecting, // enum: inspecting / completed
		productIDs,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// ★ InspectionItem に modelId / modelNumber を埋め込む
	for i := range batch.Inspections {
		pid := batch.Inspections[i].ProductID
		if mid, ok := modelIDByProductID[pid]; ok {
			mid = strings.TrimSpace(mid)
			batch.Inspections[i].ModelID = mid

			if mn, ok := modelNumberByModelID[mid]; ok && mn != "" {
				mnCopy := mn
				batch.Inspections[i].ModelNumber = &mnCopy
			}
		}
	}
	// quantity / totalPassed / requestedBy / requestedAt / mintedAt / tokenBlueprintId は
	// NewInspectionBatch 側の初期値のまま

	// 先に Inspection を保存してから PrintLog を保存
	if _, err := u.inspectionRepo.Create(ctx, batch); err != nil {
		return productdom.PrintLog{}, err
	}

	// PrintLog を保存
	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// QrPayloads を付与（https://narratives.jp/{productId} を埋め込む）
	var payloads []string
	for _, productID := range created.ProductIDs {
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		url := fmt.Sprintf("%s/%s", publicQRBaseURL, productID)
		payloads = append(payloads, url)
	}
	created.QrPayloads = payloads

	return created, nil
}

// ==========================
// Commands
// ==========================

// Create: Product のみ作成する。
//
// 以前の仕様（Create のたびに 1 件ずつ print_log を作成）は廃止し、
// 「1 回の印刷バッチでまとめて PrintLog を作る」ために
// CreatePrintLogForProduction を別途呼び出す方式に変更。
func (u *PrintUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	created, err := u.repo.Create(ctx, p)
	if err != nil {
		return productdom.Product{}, err
	}
	return created, nil
}

// Save: 既存の互換用途として残しておく（フルアップサート）
func (u *PrintUsecase) Save(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	return u.repo.Save(ctx, p)
}

// Update:
//
// - ID               … URL パスの id で決定（不変）
// - ModelID          … POST 時に確定、更新不可
// - ProductionID     … POST 時に確定、更新不可
// - PrintedAt        … POST 時に確定、更新不可
// - InspectionResult … 更新対象
// - ConnectedToken   … 更新対象
// - InspectedAt      … 更新対象（InspectionResult の入力日時）
// - InspectedBy      … 更新対象（InspectionResult の入力者）
func (u *PrintUsecase) Update(ctx context.Context, id string, in productdom.Product) (productdom.Product, error) {
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
