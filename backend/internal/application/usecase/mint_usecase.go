// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"

	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	modeldom "narratives/internal/domain/model"
	pbpdom "narratives/internal/domain/productBlueprint"
	proddom "narratives/internal/domain/production"
)

// ============================================================
// Ports (Repository interfaces for MintUsecase)
// ============================================================

// mintProductBlueprintRepo は companyId から productBlueprintId の一覧を取得したり、
// productBlueprintId から productName / Patch を解決するための最小ポート
type mintProductBlueprintRepo interface {
	// companyId に紐づく product_blueprints の ID 一覧を返す
	// 実装例: ProductBlueprintRepositoryFS.ListIDsByCompany
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// productBlueprintId から productName だけを取得するヘルパ
	// 実装例: ProductBlueprintRepositoryFS.GetProductNameByID
	GetProductNameByID(ctx context.Context, id string) (string, error)

	// ★ 追加: productBlueprintId から Patch 全体を取得するヘルパ
	// mintRequestDetail 画面の ProductBlueprintCard 表示用
	GetPatchByID(ctx context.Context, id string) (pbpdom.Patch, error)
}

// mintProductionRepo は productBlueprintId 群から productions を取得するための最小ポート
type mintProductionRepo interface {
	// 指定された productBlueprintId 群のいずれかを持つ Production をすべて返す
	// 実装例: ProductionRepositoryFS.ListByProductBlueprintID
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]proddom.Production, error)
}

// mintInspectionRepo は productionId 群から inspections を取得するための最小ポート
type mintInspectionRepo interface {
	// 指定された productionId 群に紐づく InspectionBatch をすべて返す
	// 実装例: InspectionRepositoryFS.ListByProductionID
	ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)
}

// mintModelRepo は modelId(variationID) から size / color / rgb などの情報を解決するための最小ポート
type mintModelRepo interface {
	// 実装例: ModelRepositoryFS.GetModelVariationByID
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// ============================================================
// 画面向け DTO
// ============================================================

// モデル情報をフロントに渡すためのメタ情報
type MintModelMeta struct {
	Size      string `json:"size"`
	ColorName string `json:"colorName"`
	RGB       int    `json:"rgb"`
}

// MintInspectionView は Mint 管理画面向けの Inspection 表現。
// 元の InspectionBatch に加えて、productBlueprintId / productName、
// そして modelId → size/color/rgb のマップを付与して返す。
type MintInspectionView struct {
	inspectiondom.InspectionBatch

	// Production → ProductBlueprint の join 結果
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`

	// モデル情報テーブル: modelId → { size, colorName, rgb }
	ModelMeta map[string]MintModelMeta `json:"modelMeta"`
}

// ============================================================
// MintUsecase 本体
// ============================================================

type MintUsecase struct {
	pbRepo    mintProductBlueprintRepo
	prodRepo  mintProductionRepo
	inspRepo  mintInspectionRepo
	modelRepo mintModelRepo

	// brandId → brandName 解決用
	brandSvc *branddom.Service
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// DI コンテナから ProductBlueprintRepositoryFS / ProductionRepositoryFS /
// InspectionRepositoryFS / ModelRepositoryFS / brand.Service をそれぞれ満たす実装として渡してください。
func NewMintUsecase(
	pbRepo mintProductBlueprintRepo,
	prodRepo mintProductionRepo,
	inspRepo mintInspectionRepo,
	modelRepo mintModelRepo,
	brandSvc *branddom.Service,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:    pbRepo,
		prodRepo:  prodRepo,
		inspRepo:  inspRepo,
		modelRepo: modelRepo,
		brandSvc:  brandSvc,
	}
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Public API
// ============================================================

// ListInspectionsForCurrentCompany は、context に埋め込まれた companyId
// （middleware.AuthMiddleware → usecase.WithCompanyID）を基点に、
//
//  1. companyId → productBlueprintId の一覧を取得
//  2. productBlueprintId → productionId の一覧を取得
//  3. productionId → inspections の一覧を取得
//  4. productionId → productBlueprintId → productName を join
//  5. inspection 内の modelId 群 → ModelVariation を引き、modelId → {size,color,rgb} を構築
//
// という一連のチェーンを実行し、最終的な MintInspectionView の配列を返します。
func (u *MintUsecase) ListInspectionsForCurrentCompany(
	ctx context.Context,
) ([]MintInspectionView, error) {

	companyID := strings.TrimSpace(CompanyIDFromContext(ctx))
	if companyID == "" {
		return nil, ErrCompanyIDMissing
	}

	return u.ListInspectionsByCompanyID(ctx, companyID)
}

// ListInspectionsByCompanyID は、明示的に companyId を指定して
// 同じチェーンを実行するバリアントです。
func (u *MintUsecase) ListInspectionsByCompanyID(
	ctx context.Context,
	companyID string,
) ([]MintInspectionView, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, ErrCompanyIDMissing
	}

	// 1) companyId → productBlueprintId 一覧
	pbIDs, err := u.pbRepo.ListIDsByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}

	if len(pbIDs) == 0 {
		// 該当 product_blueprints が無ければ空配列を返す
		return []MintInspectionView{}, nil
	}

	// 2) productBlueprintId 群 → Production 一覧
	prods, err := u.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}

	if len(prods) == 0 {
		// 該当 Production が無ければ空配列を返す
		return []MintInspectionView{}, nil
	}

	// Production から productionId 一覧を抽出（重複除去）
	prodIDSet := make(map[string]struct{}, len(prods))
	// ついでに productionId → productBlueprintId のマップも作る
	prodToPB := make(map[string]string, len(prods))

	for _, p := range prods {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		prodIDSet[id] = struct{}{}

		pbID := strings.TrimSpace(p.ProductBlueprintID)
		if pbID != "" {
			prodToPB[id] = pbID
		}
	}

	if len(prodIDSet) == 0 {
		return []MintInspectionView{}, nil
	}

	prodIDs := make([]string, 0, len(prodIDSet))
	for id := range prodIDSet {
		prodIDs = append(prodIDs, id)
	}

	// 3) productionId 群 → InspectionBatch 一覧
	batches, err := u.inspRepo.ListByProductionID(ctx, prodIDs)
	if err != nil {
		return nil, err
	}

	if len(batches) == 0 {
		return []MintInspectionView{}, nil
	}

	// 4) productBlueprintId → productName の名前解決用マップ
	pbNameMap := make(map[string]string)

	// prods から実際に使われている productBlueprintId 群だけを抽出
	usedPBSet := make(map[string]struct{})
	for _, pbID := range prodToPB {
		if pbID == "" {
			continue
		}
		usedPBSet[pbID] = struct{}{}
	}

	for pbID := range usedPBSet {
		name, err := u.pbRepo.GetProductNameByID(ctx, pbID)
		if err != nil {
			// 個別に失敗した場合は空文字として扱い、処理は続行
			if !errors.Is(err, pbpdom.ErrNotFound) {
				// ここではログ出力などは行わず、単にスキップ
			}
			continue
		}
		pbNameMap[pbID] = strings.TrimSpace(name)
	}

	// 5) inspection 内の modelId 群を集めて、ModelVariation をまとめて取得
	modelIDSet := make(map[string]struct{})
	for _, b := range batches {
		for _, item := range b.Inspections {
			mid := strings.TrimSpace(item.ModelID)
			if mid == "" {
				continue
			}
			modelIDSet[mid] = struct{}{}
		}
	}

	modelMetaMap := make(map[string]MintModelMeta, len(modelIDSet))
	if len(modelIDSet) > 0 && u.modelRepo != nil {
		for modelID := range modelIDSet {
			mv, err := u.modelRepo.GetModelVariationByID(ctx, modelID)
			if err != nil {
				if errors.Is(err, modeldom.ErrNotFound) {
					continue
				}
				// その他のエラーもここではスキップ
				continue
			}

			meta := MintModelMeta{
				Size: strings.TrimSpace(mv.Size),
			}

			// Color 情報を追加（struct のフィールド名に合わせて調整）
			if mv.Color.Name != "" {
				meta.ColorName = strings.TrimSpace(mv.Color.Name)
			}
			meta.RGB = mv.Color.RGB

			// キーは modelId（variationID）
			modelMetaMap[modelID] = meta
		}
	}

	// 6) InspectionBatch ごとに MintInspectionView を組み立てる
	views := make([]MintInspectionView, 0, len(batches))

	for _, b := range batches {
		prodID := strings.TrimSpace(b.ProductionID)
		pbID := prodToPB[prodID]
		name := pbNameMap[pbID]

		// このバッチ内で実際に使われている modelId だけを modelMeta に入れる
		perBatchModelMeta := make(map[string]MintModelMeta)
		for _, item := range b.Inspections {
			mid := strings.TrimSpace(item.ModelID)
			if mid == "" {
				continue
			}
			if meta, ok := modelMetaMap[mid]; ok {
				perBatchModelMeta[mid] = meta
			}
		}

		view := MintInspectionView{
			InspectionBatch:    b,
			ProductBlueprintID: pbID,
			ProductName:        name,
			ModelMeta:          perBatchModelMeta,
		}
		views = append(views, view)
	}

	return views, nil
}

// ============================================================
// Additional API: ProductBlueprint Patch 解決
// ============================================================

// GetProductBlueprintPatchByID は、productBlueprintId から
// ProductBlueprint.Patch 相当の構造体を取得して返します。
// mintRequestDetail 画面の ProductBlueprintCard に渡す想定です。
func (u *MintUsecase) GetProductBlueprintPatchByID(
	ctx context.Context,
	productBlueprintID string,
) (pbpdom.Patch, error) {

	if u == nil {
		return pbpdom.Patch{}, errors.New("mint usecase is nil")
	}

	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		return pbpdom.Patch{}, errors.New("productBlueprintID is empty")
	}

	patch, err := u.pbRepo.GetPatchByID(ctx, id)
	if err != nil {
		return pbpdom.Patch{}, err
	}

	return patch, nil
}

// ============================================================
// Helper: brandId → brandName 解決
// ============================================================

// getBrandNameByID は、brandID から表示用の brandName を取得します。
// - brandSvc が設定されていない場合: 空文字 + nil を返す（ブランド名はオプション扱い）
// - brand.ErrNotFound / brand.ErrInvalidID の場合: 空文字 + nil を返す（ブランド未設定/不整合は UI では無表示）
// - その他のエラー: そのままエラーを返す
func (u *MintUsecase) getBrandNameByID(ctx context.Context, brandID string) (string, error) {
	if u == nil {
		return "", errors.New("mint usecase is nil")
	}

	if u.brandSvc == nil {
		// ブランド名表示がオプションであれば、nil エラーで空文字返却にする
		return "", nil
	}

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", nil
	}

	name, err := u.brandSvc.GetNameByID(ctx, brandID)
	if err != nil {
		// 見つからない / 無効 ID は「ブランド名なし」として扱う
		if errors.Is(err, branddom.ErrNotFound) || errors.Is(err, branddom.ErrInvalidID) {
			return "", nil
		}
		// それ以外の予期しないエラーはそのまま返す
		return "", err
	}

	return name, nil
}

// ResolveBrandNameByID は、外部（ハンドラ等）から brandID で brandName を取得する公開メソッドです。
func (u *MintUsecase) ResolveBrandNameByID(ctx context.Context, brandID string) (string, error) {
	return u.getBrandNameByID(ctx, brandID)
}
