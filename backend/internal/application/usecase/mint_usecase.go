// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	modeldom "narratives/internal/domain/model"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

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
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	// TokenBlueprint の minted 状態や一覧を扱うためのリポジトリ
	tbRepo tbdom.RepositoryPort

	// brandId → brandName 解決や Brand 一覧取得用
	brandSvc *branddom.Service

	// mints テーブル用リポジトリ
	mintRepo mintdom.MintRepository
	// inspections → passed productId 一覧を取得するためのポート
	passedProductLister mintdom.PassedProductLister

	// 各種「名前解決」を集約する NameResolver
	nameResolver *resolver.NameResolver
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// DI コンテナから ProductBlueprintRepositoryFS / ProductionRepositoryFS /
// InspectionRepositoryFS / ModelRepositoryFS / TokenBlueprintRepositoryFS /
// MintRepositoryFS /（inspections 用の PassedProductLister 実装）/ brand.Service / NameResolver
// をそれぞれ満たす実装として渡してください。
func NewMintUsecase(
	pbRepo mintdom.MintProductBlueprintRepo,
	prodRepo mintdom.MintProductionRepo,
	inspRepo mintdom.MintInspectionRepo,
	modelRepo mintdom.MintModelRepo,
	tbRepo tbdom.RepositoryPort,
	brandSvc *branddom.Service,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	nameResolver *resolver.NameResolver,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:              pbRepo,
		prodRepo:            prodRepo,
		inspRepo:            inspRepo,
		modelRepo:           modelRepo,
		tbRepo:              tbRepo,
		brandSvc:            brandSvc,
		mintRepo:            mintRepo,
		passedProductLister: passedProductLister,
		nameResolver:        nameResolver,
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

	// NameResolver に「商品名解決」を委譲
	if u.nameResolver != nil {
		for pbID := range usedPBSet {
			name := strings.TrimSpace(u.nameResolver.ResolveProductName(ctx, pbID))
			if name != "" {
				pbNameMap[pbID] = name
			}
		}
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
// Additional API: Inspection requested 更新 + mints 作成
// ============================================================
//
// ミント申請ボタン押下時に、
//
//   - inspections 側: 該当 Production の InspectionBatch.requested を true に更新
//   - mints 側    : brandId / tokenBlueprintId / passedProductIDs / createdAt / createdBy /
//     scheduledBurnDate（任意）/ minted=false を 1 レコード作成
//
// という 2 つの処理を行います。
func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string, // HTML date input の "YYYY-MM-DD" 形式（nil or 空文字なら未指定）
) (inspectiondom.InspectionBatch, error) {

	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return empty, errors.New("inspection repo is nil")
	}
	if u.mintRepo == nil {
		return empty, errors.New("mint repo is nil")
	}
	if u.passedProductLister == nil {
		return empty, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return empty, errors.New("tokenBlueprintID is empty")
	}

	// requestedBy 相当は currentMember（= ミント申請ボタンを押下したユーザー）
	// → inspections には保存せず、mints.createdBy に責務を移譲する
	memberID := strings.TrimSpace(MemberIDFromContext(ctx))
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	// 1) TokenBlueprint から brandId を解決
	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	brandID := strings.TrimSpace(tb.BrandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty on tokenBlueprint")
	}

	// 2) inspections テーブルから inspectionResult: passed の productId 一覧を取得
	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}
	if len(passedProductIDs) == 0 {
		return empty, errors.New("no passed products for this production")
	}

	// 3) Mint エンティティ生成（minted=false / mintedAt=nil で作成）
	mintEntity, err := mintdom.NewMint(
		"",
		brandID,
		tbID,
		passedProductIDs,
		memberID, // createdBy 相当
		now,      // createdAt 相当
	)
	if err != nil {
		return empty, err
	}

	// 3-1) ScheduledBurnDate を文字列からパースして設定（任意）
	if scheduledBurnDate != nil {
		if s := strings.TrimSpace(*scheduledBurnDate); s != "" {
			// フロントからは "2006-01-02" 形式で来る想定
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	// 4) InspectionBatch 側の requested フラグを更新（true にする）
	batch, err := u.inspRepo.UpdateRequestedFlag(ctx, pid, true)
	if err != nil {
		return empty, err
	}

	// 5) mints テーブルへ保存（ScheduledBurnDate を含む）
	if _, err := u.mintRepo.Create(ctx, mintEntity); err != nil {
		return empty, err
	}

	return batch, nil
}

// ============================================================
// Helper: brandId → brandName 解決（NameResolver へ委譲）
// ============================================================

// ResolveBrandNameByID は、外部（ハンドラ等）から brandID で brandName を取得する公開メソッド。
// 実体の名前解決は NameResolver に委譲し、ここではエラーではなく空文字／文字列を返す。
func (u *MintUsecase) ResolveBrandNameByID(ctx context.Context, brandID string) (string, error) {
	if u == nil {
		return "", errors.New("mint usecase is nil")
	}
	if u.nameResolver == nil {
		return "", errors.New("name resolver is nil")
	}

	name := strings.TrimSpace(u.nameResolver.ResolveBrandName(ctx, brandID))
	return name, nil
}

// ============================================================
// Additional API: Brand 一覧（current company）
// ============================================================

// ListBrandsForCurrentCompany は、context から companyId を取り出し、
// brand.Service の ListByCompanyID を呼び出して同じ companyId を持つ Brand 一覧を返します。
// Mint 画面でのブランドフィルタ等に利用する想定です。
func (u *MintUsecase) ListBrandsForCurrentCompany(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {

	var empty branddom.PageResult[branddom.Brand]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.brandSvc == nil {
		return empty, errors.New("brand service is nil")
	}

	companyID := strings.TrimSpace(CompanyIDFromContext(ctx))
	if companyID == "" {
		return empty, ErrCompanyIDMissing
	}

	// 一覧取得は brand.Service の責務のまま
	return u.brandSvc.ListByCompanyID(ctx, companyID, page)
}

// ============================================================
// Additional API: TokenBlueprint 一覧（brandId フィルタ）
// ============================================================

// ListTokenBlueprintsByBrand は、指定された brandID に紐づく
// TokenBlueprint 一覧を返します。
// ドメインヘルパー tbdom.ListByBrandID を利用して BrandIDs フィルタを適用します。
func (u *MintUsecase) ListTokenBlueprintsByBrand(
	ctx context.Context,
	brandID string,
	page tbdom.Page,
) (tbdom.PageResult, error) {

	var empty tbdom.PageResult

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty")
	}

	// tokenBlueprint パッケージ側のフィルター関数を使用
	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}
