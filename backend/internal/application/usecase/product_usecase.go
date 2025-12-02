// backend/internal/application/usecase/product_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	productdom "narratives/internal/domain/product"
)

// QR ペイロード生成時に使うベース URL
// 例: https://narratives.jp/products/{productId} のような形で利用される想定
const defaultQRBaseURL = "https://narratives.jp"

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

// ★ Inspection 用リポジトリ（print_log と同じ product ドメイン配下の集約として扱う）
type InspectionRepo interface {
	// inspections/{productionId} を新規作成
	Create(ctx context.Context, batch productdom.InspectionBatch) (productdom.InspectionBatch, error)

	// productionId から inspections を取得
	GetByProductionID(ctx context.Context, productionID string) (productdom.InspectionBatch, error)

	// 既存バッチを保存（フルアップサート想定）
	Save(ctx context.Context, batch productdom.InspectionBatch) (productdom.InspectionBatch, error)
}

// ProductUsecase orchestrates product operations.
type ProductUsecase struct {
	repo           ProductRepo
	printLogRepo   PrintLogRepo
	inspectionRepo InspectionRepo
}

func NewProductUsecase(
	repo ProductRepo,
	printLogRepo PrintLogRepo,
	inspectionRepo InspectionRepo,
) *ProductUsecase {
	return &ProductUsecase{
		repo:           repo,
		printLogRepo:   printLogRepo,
		inspectionRepo: inspectionRepo,
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

			// BuildProductQRValue は (baseURL, productID) を受け取り
			// QR コード用の文字列を返す
			payload, err := productdom.BuildProductQRValue(baseURL, productID)
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

// ★ 追加: inspections を単独で作成する
//
// POST /products/inspections 用
func (u *ProductUsecase) CreateInspectionBatchForProduction(
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

	// ProductID 一覧
	productIDs := make([]string, 0, len(products))
	for _, p := range products {
		if strings.TrimSpace(p.ID) != "" {
			productIDs = append(productIDs, strings.TrimSpace(p.ID))
		}
	}
	if len(productIDs) == 0 {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// InspectionBatch エンティティ作成（全て nil, status=inspecting）
	batch, err := productdom.NewInspectionBatch(
		pid,
		productdom.InspectionStatusInspecting,
		productIDs,
	)
	if err != nil {
		return productdom.InspectionBatch{}, err
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
func (u *ProductUsecase) CreatePrintLogForProduction(ctx context.Context, productionID string) (productdom.PrintLog, error) {
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

	// ★ ここで inspections/{productionId} 用のバッチを作成
	//   - inspectionResult / inspectedBy / inspectedAt はすべて nil で初期化
	//   - status は "inspecting" 固定で開始
	batch, err := productdom.NewInspectionBatch(
		pid,
		productdom.InspectionStatusInspecting, // enum: inspecting / completed
		productIDs,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// 先に Inspection を保存してから PrintLog を保存
	if _, err := u.inspectionRepo.Create(ctx, batch); err != nil {
		return productdom.PrintLog{}, err
	}

	// PrintLog を保存
	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	// QrPayloads を付与（文字列の配列）
	baseURL := defaultQRBaseURL
	var payloads []string
	for _, productID := range created.ProductIDs {
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		payload, err := productdom.BuildProductQRValue(baseURL, productID)
		if err != nil {
			return productdom.PrintLog{}, err
		}
		payloads = append(payloads, payload)
	}
	created.QrPayloads = payloads

	return created, nil
}

// ★ 追加: inspections 内の 1 productId 分を更新する
//
// PATCH /products/inspections 用
func (u *ProductUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *productdom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
	status *productdom.InspectionStatus,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}
	pdID := strings.TrimSpace(productID)
	if pdID == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 対象 productId の InspectionItem を探す
	found := false
	for i := range batch.Inspections {
		if strings.TrimSpace(batch.Inspections[i].ProductID) != pdID {
			continue
		}
		found = true

		item := &batch.Inspections[i]

		// inspectionResult の更新
		if result != nil {
			if !productdom.IsValidInspectionResult(*result) {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		// inspectedBy の更新
		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

		// inspectedAt の更新
		if inspectedAt != nil {
			at := inspectedAt.UTC()
			if at.IsZero() {
				return productdom.InspectionBatch{}, productdom.ErrInvalidInspectedAt
			}
			item.InspectedAt = &at
		}

		// domain 側の整合性に近づけるため、result が nil かつ inspectedBy / inspectedAt が nil の場合は
		// 「未検査状態」に戻す用途も想定できるが、今回はシンプルに「与えられたものだけ更新」に留める。
		break
	}

	if !found {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	// status の更新（任意）
	if status != nil {
		if !productdom.IsValidInspectionStatus(*status) {
			return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	// 保存（InspectionRepo.Save 側で Firestore に反映）
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return updated, nil
}

// ★ 追加: 検品完了（未検品を notManufactured にし、ステータスを completed にする）
func (u *ProductUsecase) CompleteInspectionForProduction(
	ctx context.Context,
	productionID string,
	by string,
	at time.Time,
) (productdom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	// 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	// ドメインサービス側で:
	//   - 未検品(nil / notYet) を notManufactured に変更
	//   - inspectedBy / inspectedAt を一括で設定
	//   - Status を completed に変更
	if err := batch.Complete(by, at); err != nil {
		return productdom.InspectionBatch{}, err
	}

	// 保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return updated, nil
}

// ==========================
// Commands
// ==========================

// Create: Product のみ作成する。
//
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
