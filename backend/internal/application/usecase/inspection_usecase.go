// backend/internal/application/usecase/inspection_usecase.go
package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ------------------------------------------------------------
// Ports (Repository Interfaces)
// ------------------------------------------------------------

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	// 指定 productId の inspectionResult を更新する
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result inspectiondom.InspectionResult,
	) error
}

// ★ 追加: inspectionId (= productionId 扱い) から mints を取得するための最小ポート
//   - mints テーブルが「inspectionId で複数レコードを持ちうる」前提で slice を返す
//   - 1件しか持たない設計でも、この形なら将来拡張しやすい
type InspectionMintLister interface {
	ListByInspectionID(ctx context.Context, inspectionID string) ([]mintdom.Mint, error)
}

// InspectionRepo インターフェース自体は print_usecase.go 側で定義済み。
//   type InspectionRepo interface {
//       Create(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
//       GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error)
//       Save(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
//   }

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

type InspectionUsecase struct {
	inspectionRepo InspectionRepo
	productRepo    ProductInspectionRepo

	// ★ 追加: mints を引くための repo（任意・nil 許容）
	mintRepo InspectionMintLister
}

func NewInspectionUsecase(
	inspectionRepo InspectionRepo,
	productRepo ProductInspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
	}
}

// ★ 追加: mintRepo も注入できるコンストラクタ（既存の NewInspectionUsecase は壊さない）
func NewInspectionUsecaseWithMint(
	inspectionRepo InspectionRepo,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintLister,
) *InspectionUsecase {
	u := NewInspectionUsecase(inspectionRepo, productRepo)
	u.mintRepo = mintRepo
	return u
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// ★ productionId から inspections バッチをそのまま返す
func (u *InspectionUsecase) GetBatchByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 以前はここで modelId → modelNumber を埋め込んでいたが、現在はそのロジックを削除
	return batch, nil
}

// ★ 互換用エイリアス: ListByProductionID
//
//	既存コードで「ListByProductionID」を呼んでいる箇所があっても
//	GetBatchByProductionID と同じ挙動で動くようにするためのラッパー。
func (u *InspectionUsecase) ListByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {
	return u.GetBatchByProductionID(ctx, productionID)
}

// ★ 追加: 同じ inspectionId (= productionId 扱い) を持つ mints を取得する
//
// 想定ユースケース:
// - MintRequest 一覧/詳細で、inspectionId (= productionId) をキーに mints を引く
// - mints の設計が 1 inspectionId に対して複数行を持つ場合でも対応できる
func (u *InspectionUsecase) ListMintsByInspectionID(
	ctx context.Context,
	inspectionID string,
) ([]mintdom.Mint, error) {

	if u.mintRepo == nil {
		return nil, fmt.Errorf("mintRepo is nil")
	}

	iid := strings.TrimSpace(inspectionID)
	if iid == "" {
		// inspectiondom 側に専用 Err が無いので productionId の Err を流用
		return nil, inspectiondom.ErrInvalidInspectionProductionID
	}

	mints, err := u.mintRepo.ListByInspectionID(ctx, iid)
	if err != nil {
		return nil, err
	}

	// 順序を安定化（UI/テストに優しい）
	// - CreatedAt 昇順
	// - 同時刻は ID で昇順
	sort.SliceStable(mints, func(i, j int) bool {
		if mints[i].CreatedAt.Equal(mints[j].CreatedAt) {
			return mints[i].ID < mints[j].ID
		}
		return mints[i].CreatedAt.Before(mints[j].CreatedAt)
	})

	return mints, nil
}

// ★ 任意: productionId で呼びたい場合の別名（inspectionId と同義扱い）
func (u *InspectionUsecase) ListMintsByProductionID(
	ctx context.Context,
	productionID string,
) ([]mintdom.Mint, error) {
	return u.ListMintsByInspectionID(ctx, productionID)
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// ★ inspections 内の 1 productId 分を更新する
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
	status *inspectiondom.InspectionStatus,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}
	pdID := strings.TrimSpace(productID)
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	// 1) 現在のバッチを取得（inspections/{productionId}）
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 2) 対象 productId の InspectionItem を探す
	found := false
	for i := range batch.Inspections {
		if strings.TrimSpace(batch.Inspections[i].ProductID) != pdID {
			continue
		}
		found = true

		item := &batch.Inspections[i]

		// inspectionResult の更新
		if result != nil {
			if !inspectiondom.IsValidInspectionResult(*result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		// inspectedBy の更新
		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

		// inspectedAt の更新
		if inspectedAt != nil {
			atUTC := inspectedAt.UTC()
			if atUTC.IsZero() {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
			}
			item.InspectedAt = &atUTC
		}

		break
	}

	if !found {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	// 3) status の更新（任意）
	if status != nil {
		if !inspectiondom.IsValidInspectionStatus(*status) {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	// 3.5) ★ totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 4) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 以前はここで modelNumber の埋め込みを行っていたが、現在は削除済み

	// 5) products テーブル側の inspectionResult も同期
	//    （result が指定されている場合のみ）
	if result != nil && u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}

	return updated, nil
}

// ★ 検品完了
//   - inspections 側ではドメインの Complete により notYet → notManufactured へ遷移
//   - products 側にも inspectionResult を同期（notManufactured を含む）
func (u *InspectionUsecase) CompleteInspectionForProduction(
	ctx context.Context,
	productionID string,
	by string,
	at time.Time,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	// 1) 現在のバッチを取得
	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 2) ドメイン側の Complete を利用して一括更新
	//    ここで notYet → notManufactured への遷移が行われる想定
	if err := batch.Complete(by, at); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 2.5) ★ 一括更新後に totalPassed を再集計
	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	// 3) inspections テーブル側を保存
	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	// 以前はここで modelNumber の埋め込みを行っていたが、現在は削除済み

	// 4) products テーブル側の inspectionResult も同期
	//    - Complete による notYet → notManufactured の結果も反映するため、
	//      notManufactured も含めて全て同期する
	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				continue
			}
			result := *item.InspectionResult
			if !inspectiondom.IsValidInspectionResult(result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			pid := strings.TrimSpace(item.ProductID)
			if pid == "" {
				// product ドキュメントが存在しないので同期対象外
				continue
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pid, result); err != nil {
				return inspectiondom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}
