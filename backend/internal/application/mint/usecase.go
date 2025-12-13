// backend/internal/application/mint/usecase.go
package mint

import (
	"context"
	"errors"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	modeldom "narratives/internal/domain/model"
	pbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
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
// チェーンミント起動用ポート
// ============================================================

// TokenMintPort は、MintUsecase から見た「オンチェーンミントを起動するための」ポートです。
// TokenUsecase がこのインターフェースを実装する想定です。
type TokenMintPort interface {
	MintFromMintRequest(ctx context.Context, mintID string) (*tokendom.MintResult, error)
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

	// チェーンミント実行用ポート（TokenUsecase を想定）
	tokenMinter TokenMintPort
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// DI コンテナから ProductBlueprintRepositoryFS / ProductionRepositoryFS /
// InspectionRepositoryFS / ModelRepositoryFS / TokenBlueprintRepositoryFS /
// MintRepositoryFS /（inspections 用の PassedProductLister 実装）/ brand.Service / NameResolver /
// TokenUsecase(TokenMintPort を実装) をそれぞれ満たす実装として渡してください。
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
	tokenMinter TokenMintPort,
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
		tokenMinter:         tokenMinter,
	}
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Public API
// ============================================================

// ListInspectionsForCurrentCompany は、context に埋め込まれた companyId
// （middleware.AuthMiddleware → usecase.WithCompanyID）を基点に、
func (u *MintUsecase) ListInspectionsForCurrentCompany(
	ctx context.Context,
) ([]MintInspectionView, error) {

	companyID := strings.TrimSpace(appusecase.CompanyIDFromContext(ctx))
	if companyID == "" {
		return nil, ErrCompanyIDMissing
	}

	return u.ListInspectionsByCompanyID(ctx, companyID)
}

// ListInspectionsByCompanyID は、明示的に companyId を指定して同じチェーンを実行するバリアントです。
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
		return []MintInspectionView{}, nil
	}

	// 2) productBlueprintId 群 → Production 一覧
	prods, err := u.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}
	if len(prods) == 0 {
		return []MintInspectionView{}, nil
	}

	// Production から productionId 一覧を抽出（重複除去）
	prodIDSet := make(map[string]struct{}, len(prods))
	// productionId → productBlueprintId のマップ
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

	usedPBSet := make(map[string]struct{})
	for _, pbID := range prodToPB {
		if pbID == "" {
			continue
		}
		usedPBSet[pbID] = struct{}{}
	}

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
				continue
			}

			meta := MintModelMeta{
				Size: strings.TrimSpace(mv.Size),
			}

			if mv.Color.Name != "" {
				meta.ColorName = strings.TrimSpace(mv.Color.Name)
			}
			meta.RGB = mv.Color.RGB

			modelMetaMap[modelID] = meta
		}
	}

	// 6) InspectionBatch ごとに MintInspectionView を組み立てる
	views := make([]MintInspectionView, 0, len(batches))

	for _, b := range batches {
		prodID := strings.TrimSpace(b.ProductionID)
		pbID := prodToPB[prodID]
		name := pbNameMap[pbID]

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
// Additional API: mints を inspectionIds で取得
// ============================================================

// ListMintsByInspectionIDs は、inspectionIds（= productionIds）に紐づく mints を
// inspectionId をキーにした map で返します。
func (u *MintUsecase) ListMintsByInspectionIDs(
	ctx context.Context,
	inspectionIDs []string,
) (map[string]mintdom.Mint, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	seen := make(map[string]struct{}, len(inspectionIDs))
	ids := make([]string, 0, len(inspectionIDs))

	for _, id := range inspectionIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}

	if len(ids) == 0 {
		return map[string]mintdom.Mint{}, nil
	}

	return u.mintRepo.ListByInspectionIDs(ctx, ids)
}

// ============================================================
// Additional API: ProductBlueprint Patch 解決
// ============================================================

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
// Additional API: Inspection requested 更新 + mints 作成 + チェーンミント
// ============================================================

func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
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

	memberID := strings.TrimSpace(appusecase.MemberIDFromContext(ctx))
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	brandID := strings.TrimSpace(tb.BrandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}
	if len(passedProductIDs) == 0 {
		return empty, errors.New("no passed products for this production")
	}

	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return empty, err
	}

	if scheduledBurnDate != nil {
		if s := strings.TrimSpace(*scheduledBurnDate); s != "" {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	batch, err := u.inspRepo.UpdateRequestedFlag(ctx, pid, true)
	if err != nil {
		return empty, err
	}

	savedMint, err := u.mintRepo.Create(ctx, mintEntity)
	if err != nil {
		return empty, err
	}

	if u.tokenMinter != nil {
		if _, err := u.tokenMinter.MintFromMintRequest(ctx, strings.TrimSpace(savedMint.ID)); err != nil {
			return empty, err
		}

		if err := u.markTokenBlueprintMinted(ctx, tbID, memberID); err != nil {
			return empty, err
		}
	}

	return batch, nil
}

func (u *MintUsecase) markTokenBlueprintMinted(ctx context.Context, tokenBlueprintID string, actorID string) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return errors.New("tokenBlueprint repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return errors.New("tokenBlueprintID is empty")
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return errors.New("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tb.Minted {
		return nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	_, err = u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		Minted:    &minted,
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	})
	return err
}

// ============================================================
// Helper: brandId → brandName 解決（NameResolver へ委譲）
// ============================================================

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

	companyID := strings.TrimSpace(appusecase.CompanyIDFromContext(ctx))
	if companyID == "" {
		return empty, ErrCompanyIDMissing
	}

	return u.brandSvc.ListByCompanyID(ctx, companyID, page)
}

// ============================================================
// Additional API: TokenBlueprint 一覧（brandId フィルタ）
// ============================================================

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

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}
