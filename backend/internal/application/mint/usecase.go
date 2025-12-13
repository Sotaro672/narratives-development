// backend/internal/application/mint/usecase.go
package mint

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	dto "narratives/internal/application/mint/dto"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

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
	// 互換のため残しているが、company -> pb -> production の探索にはもう使わない方針
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	// TokenBlueprint の minted 状態や一覧を扱うためのリポジトリ
	tbRepo tbdom.RepositoryPort

	// Brand 一覧取得用
	brandSvc *branddom.Service

	// mints テーブル用リポジトリ
	mintRepo mintdom.MintRepository

	// inspections → passed productId 一覧を取得するためのポート
	passedProductLister mintdom.PassedProductLister

	// チェーンミント実行用ポート（TokenUsecase を想定）
	tokenMinter TokenMintPort
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// ★ NameResolver は usecase に持たせず、presenter/mapper 側で扱う方針
func NewMintUsecase(
	pbRepo mintdom.MintProductBlueprintRepo,
	prodRepo mintdom.MintProductionRepo,
	inspRepo mintdom.MintInspectionRepo,
	modelRepo mintdom.MintModelRepo,
	tbRepo tbdom.RepositoryPort,
	brandSvc *branddom.Service,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
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
		tokenMinter:         tokenMinter,
	}
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Additional API: mints を inspectionIds(docId) で取得
// ============================================================

// ListMintsByInspectionIDs は、inspectionIds（= productionIds = docId）に紐づく mints を
// inspectionId をキーにした map で返します。
//
// ★ Firestore 設計上、production/inspection/mints の docId が同一であるため、
// 「where(inspectionId in ...)」ではなく「docId で Get/ループ」が最も堅牢。
// この関数は mintRepo が ListByProductionID を持つならそれを最優先します。
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

	// 順序固定（ログ/比較安定）
	sort.Strings(ids)

	// ------------------------------------------------------------
	// 最優先: mintRepo が docId 同一前提の ListByProductionID を持つ
	// ------------------------------------------------------------
	if lister, ok := u.mintRepo.(interface {
		ListByProductionID(ctx context.Context, productionIDs []string) (map[string]mintdom.Mint, error)
	}); ok {
		return lister.ListByProductionID(ctx, ids)
	}

	// ------------------------------------------------------------
	// 次点: GetByID / Get で docId を個別取得
	// ------------------------------------------------------------
	if getter, ok := u.mintRepo.(interface {
		GetByID(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		out := make(map[string]mintdom.Mint, len(ids))
		for _, id := range ids {
			m, err := getter.GetByID(ctx, id)
			if err != nil {
				// 未作成 mint は「存在しないだけ」なので握りつぶす（一覧用途）
				if strings.Contains(strings.ToLower(err.Error()), "not found") {
					continue
				}
				if errors.Is(err, mintdom.ErrNotFound) {
					continue
				}
				return nil, err
			}
			out[id] = m
		}
		return out, nil
	}

	if getter, ok := u.mintRepo.(interface {
		Get(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		out := make(map[string]mintdom.Mint, len(ids))
		for _, id := range ids {
			m, err := getter.Get(ctx, id)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "not found") {
					continue
				}
				if errors.Is(err, mintdom.ErrNotFound) {
					continue
				}
				return nil, err
			}
			out[id] = m
		}
		return out, nil
	}

	// ListByInspectionIDs は廃止方針なのでここで明示エラー
	return nil, errors.New("mint repo does not support ListByProductionID/GetByID/Get")
}

// ============================================================
// Additional API: mints(list) を inspectionIds で取得し、名前解決して DTO を組み立てる
// ============================================================

// ListMintListRowsByInspectionIDs は、inspectionIds（= productionIds）に紐づく mints を取得し、
// tokenBlueprintId → tokenName を解決して、一覧向け DTO を inspectionId をキーにした map で返します。
//
// NOTE:
//   - CreatedByName は現状 Mint.CreatedBy（memberId）をそのまま返します。
//     もし「表示名」にしたい場合は NameResolver / member.Service を注入する設計に拡張してください。
func (u *MintUsecase) ListMintListRowsByInspectionIDs(
	ctx context.Context,
	inspectionIDs []string,
) (map[string]dto.MintListRowDTO, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	// 1) mints を取得（inspectionId -> Mint）
	mintsByInspectionID, err := u.ListMintsByInspectionIDs(ctx, inspectionIDs)
	if err != nil {
		return nil, err
	}
	if len(mintsByInspectionID) == 0 {
		return map[string]dto.MintListRowDTO{}, nil
	}

	// 2) tokenBlueprintId を集めて tokenName を解決（tbRepo 経由）
	tbNameByID := map[string]string{}
	if u.tbRepo != nil {
		tbIDSet := map[string]struct{}{}
		for _, m := range mintsByInspectionID {
			tbID := strings.TrimSpace(m.TokenBlueprintID)
			if tbID == "" {
				continue
			}
			tbIDSet[tbID] = struct{}{}
		}

		tbIDs := make([]string, 0, len(tbIDSet))
		for id := range tbIDSet {
			tbIDs = append(tbIDs, id)
		}
		sort.Strings(tbIDs)

		for _, tbID := range tbIDs {
			tb, err := u.tbRepo.GetByID(ctx, tbID)
			if err != nil {
				// ここは一覧用途なので失敗しても握りつぶし（tokenName="" で返す）
				continue
			}
			tbNameByID[tbID] = strings.TrimSpace(tb.Name)
		}
	}

	// 3) DTO を組み立て（inspectionId -> MintListRowDTO）
	out := make(map[string]dto.MintListRowDTO, len(mintsByInspectionID))

	// map iteration 安定化（ログや比較がしやすい）
	keys := make([]string, 0, len(mintsByInspectionID))
	for k := range mintsByInspectionID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, inspectionID := range keys {
		m := mintsByInspectionID[inspectionID]

		iid := strings.TrimSpace(inspectionID)
		mintID := strings.TrimSpace(m.ID)
		tbID := strings.TrimSpace(m.TokenBlueprintID)

		// tokenName（tbRepo で解決済み、無ければ ""）
		tokenName := ""
		if tbID != "" {
			if n, ok := tbNameByID[tbID]; ok {
				tokenName = n
			}
		}

		// createdByName（現状は memberId をそのまま返す）
		createdByName := strings.TrimSpace(m.CreatedBy)

		// mintedAt（nil なら未mint、入れるなら RFC3339）
		var mintedAt *string
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			s := m.MintedAt.UTC().Format(time.RFC3339)
			mintedAt = &s
		}

		out[iid] = dto.MintListRowDTO{
			InspectionID:   iid,
			MintID:         mintID,
			TokenBlueprint: tbID,

			TokenName:     tokenName,
			CreatedByName: createdByName,
			MintedAt:      mintedAt,
		}
	}

	return out, nil
}

// ListMintListRowsByProductionIDs は、ProductionUsecase 等で取得した productionIds をそのまま渡すための薄いラッパです。
// productionIds == inspectionIds == docId の設計前提。
func (u *MintUsecase) ListMintListRowsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]dto.MintListRowDTO, error) {
	return u.ListMintListRowsByInspectionIDs(ctx, productionIDs)
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
	if u.pbRepo == nil {
		return pbpdom.Patch{}, errors.New("productBlueprint repo is nil")
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
// Additional API: Inspection へ mintId を記録 + mints 作成 + チェーンミント
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

	// （任意）仕様固定：scheduledBurnDate は "YYYY-MM-DD" を UTC の日付として解釈する
	if scheduledBurnDate != nil {
		if s := strings.TrimSpace(*scheduledBurnDate); s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	// 1) mints に作成（ID を確定させる）
	savedMint, err := u.mintRepo.Create(ctx, mintEntity)
	if err != nil {
		return empty, err
	}

	mid := strings.TrimSpace(savedMint.ID)
	if mid == "" {
		return empty, errors.New("saved mintID is empty")
	}

	// 2) inspections に mintId を記録（requested の代替）
	batch, err := u.inspRepo.UpdateMintID(ctx, pid, &mid)
	if err != nil {
		return empty, err
	}

	// 3) オンチェーンミント（任意）
	if u.tokenMinter != nil {
		if _, err := u.tokenMinter.MintFromMintRequest(ctx, mid); err != nil {
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

// ListInspectionBatchesByProductionIDs fetches inspection batches by production docIds.
// New flow: ProductionUsecase (or frontend) prepares productionIds and passes them here.
func (u *MintUsecase) ListInspectionBatchesByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
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
		return []inspectiondom.InspectionBatch{}, nil
	}

	sort.Strings(ids)

	return u.inspRepo.ListByProductionID(ctx, ids)
}
