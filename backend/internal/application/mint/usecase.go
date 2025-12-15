// backend/internal/application/mint/usecase.go
package mint

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"
	"time"

	dto "narratives/internal/application/mint/dto"
	qdto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	invdom "narratives/internal/domain/inventory" // ★ 追加: inventory 連携
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

	// ★ 追加: Inventory 連携（DI で注入）
	inventoryRepo invdom.RepositoryPort

	// ★ 追加: createdBy(memberId) → 氏名 を解決するため（A案）
	// 既存DIを壊さないため、Setterで後から差し込む
	nameResolver *resolver.NameResolver
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// NameResolver / InventoryRepo は任意依存（Setterで後から差し込む）とする。
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
		inventoryRepo:       nil,
		nameResolver:        nil,
	}
}

// ★ 追加: DI 側で nameResolver を後から注入できるようにする（既存constructorを壊さない）
func (u *MintUsecase) SetNameResolver(r *resolver.NameResolver) {
	if u == nil {
		return
	}
	u.nameResolver = r
}

// ★ 追加: DI 側で InventoryRepo を後から注入できるようにする
func (u *MintUsecase) SetInventoryRepo(repo invdom.RepositoryPort) {
	if u == nil {
		return
	}
	u.inventoryRepo = repo
}

// internal helper: createdBy(memberId) -> display name
// nameResolver が無い/解決できない場合は memberId を返す
func (u *MintUsecase) resolveCreatedByName(ctx context.Context, memberID string) string {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return ""
	}

	// ★ A案: nameResolver があれば ResolveMemberName を使う
	if u != nil && u.nameResolver != nil {
		if name := strings.TrimSpace(u.nameResolver.ResolveMemberName(ctx, memberID)); name != "" {
			return name
		}
	}

	// fallback
	return memberID
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// ★ NEW: POST /mint/requests/{mintRequestId}/mint 用
// - handler は tokenUC 直呼びをやめ、mintUC 経由で呼ぶ（A案）
// ============================================================

// MintFromMintRequest runs onchain mint for an existing mint request (docId = mintRequestID).
func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	mintRequestID = strings.TrimSpace(mintRequestID)
	if mintRequestID == "" {
		return nil, errors.New("mintRequestID is empty")
	}
	if u.tokenMinter == nil {
		return nil, errors.New("token minter is nil")
	}

	// actor（更新者）
	actorID := strings.TrimSpace(appusecase.MemberIDFromContext(ctx))
	if actorID == "" {
		return nil, errors.New("memberID not found in context")
	}

	// 1) まず Mint を取得（tokenBlueprintId を取り出す / 後処理に使う）
	var mintEnt *mintdom.Mint
	if u.mintRepo != nil {
		// 最優先: GetByID
		if getter, ok := any(u.mintRepo).(interface {
			GetByID(ctx context.Context, id string) (mintdom.Mint, error)
		}); ok {
			m, err := getter.GetByID(ctx, mintRequestID)
			if err == nil {
				mintEnt = &m
			}
		} else if getter, ok := any(u.mintRepo).(interface {
			Get(ctx context.Context, id string) (mintdom.Mint, error)
		}); ok {
			m, err := getter.Get(ctx, mintRequestID)
			if err == nil {
				mintEnt = &m
			}
		}
	}

	// 2) オンチェーンミント実行（TokenUsecase 実装を想定）
	result, err := u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
	if err != nil {
		return nil, err
	}

	// 3) TokenBlueprint の minted=true（未mint の場合のみ）
	if mintEnt != nil {
		tbID := strings.TrimSpace(mintEnt.TokenBlueprintID)
		if tbID != "" {
			_ = u.markTokenBlueprintMinted(ctx, tbID, actorID)
		}
	}

	// 4) （任意）mints 側 minted/mintedAt を更新（TokenUsecase 側で更新済みでも安全に無害）
	if mintEnt != nil && u.mintRepo != nil {
		if updater, ok := any(u.mintRepo).(interface {
			Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error)
		}); ok {
			now := time.Now().UTC()
			m := *mintEnt
			m.Minted = true
			m.MintedAt = &now
			_, _ = updater.Update(ctx, m)
		}
	}

	return result, nil
}

// ============================================================
// Additional API: mints を inspectionIds(docId) で取得
// ============================================================

// ListMintsByInspectionIDs は、inspectionIds（= productionIds = docId）に紐づく mints を
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

	sort.Strings(ids)

	// 最優先: mintRepo が docId 同一前提の ListByProductionID を持つ
	if lister, ok := u.mintRepo.(interface {
		ListByProductionID(ctx context.Context, productionIDs []string) (map[string]mintdom.Mint, error)
	}); ok {
		return lister.ListByProductionID(ctx, ids)
	}

	// 次点: GetByID / Get で docId を個別取得
	if getter, ok := u.mintRepo.(interface {
		GetByID(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		out := make(map[string]mintdom.Mint, len(ids))
		for _, id := range ids {
			m, err := getter.GetByID(ctx, id)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "not found") || errors.Is(err, mintdom.ErrNotFound) {
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
				if strings.Contains(strings.ToLower(err.Error()), "not found") || errors.Is(err, mintdom.ErrNotFound) {
					continue
				}
				return nil, err
			}
			out[id] = m
		}
		return out, nil
	}

	return nil, errors.New("mint repo does not support ListByProductionID/GetByID/Get")
}

// ============================================================
// Additional API: mints(list) を inspectionIds で取得し、名前解決して DTO を組み立てる
// ============================================================

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

	mintsByInspectionID, err := u.ListMintsByInspectionIDs(ctx, inspectionIDs)
	if err != nil {
		return nil, err
	}
	if len(mintsByInspectionID) == 0 {
		return map[string]dto.MintListRowDTO{}, nil
	}

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
				continue
			}
			tbNameByID[tbID] = strings.TrimSpace(tb.Name)
		}
	}

	out := make(map[string]dto.MintListRowDTO, len(mintsByInspectionID))
	keys := make([]string, 0, len(mintsByInspectionID))
	for k := range mintsByInspectionID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	log.Printf("[mint_usecase] ListMintListRowsByInspectionIDs start ids=%d mints=%d nameResolver=%t",
		len(inspectionIDs), len(keys), u.nameResolver != nil,
	)

	for _, inspectionID := range keys {
		m := mintsByInspectionID[inspectionID]

		iid := strings.TrimSpace(inspectionID)
		mintID := strings.TrimSpace(m.ID)
		tbID := strings.TrimSpace(m.TokenBlueprintID)

		tokenName := ""
		if tbID != "" {
			if n, ok := tbNameByID[tbID]; ok {
				tokenName = n
			}
		}

		createdByName := u.resolveCreatedByName(ctx, m.CreatedBy)

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

	log.Printf("[mint_usecase] ListMintListRowsByInspectionIDs done out=%d sampleKey=%q",
		len(out),
		func() string {
			if len(keys) == 0 {
				return ""
			}
			return keys[0]
		}(),
	)

	return out, nil
}

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
// ★ NEW: model variations -> modelMeta（任意）
// ============================================================

type modelMetaLister interface {
	ListModelMetaByIDs(ctx context.Context, modelIDs []string) (map[string]qdto.MintModelMetaEntry, error)
}

type modelMetaGetter interface {
	GetModelMetaByID(ctx context.Context, modelID string) (*qdto.MintModelMetaEntry, error)
}

func (u *MintUsecase) resolveModelMetaByModelIDs(
	ctx context.Context,
	modelIDs []string,
) (map[string]qdto.MintModelMetaEntry, error) {

	if u == nil {
		return map[string]qdto.MintModelMetaEntry{}, nil
	}
	if u.modelRepo == nil {
		return map[string]qdto.MintModelMetaEntry{}, nil
	}

	seen := map[string]struct{}{}
	ids := make([]string, 0, len(modelIDs))
	for _, id := range modelIDs {
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
		return map[string]qdto.MintModelMetaEntry{}, nil
	}
	sort.Strings(ids)

	if l, ok := any(u.modelRepo).(modelMetaLister); ok {
		m, err := l.ListModelMetaByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return map[string]qdto.MintModelMetaEntry{}, nil
		}
		for k, v := range m {
			if strings.TrimSpace(v.ModelID) == "" {
				v.ModelID = strings.TrimSpace(k)
				m[k] = v
			}
		}
		return m, nil
	}

	if g, ok := any(u.modelRepo).(modelMetaGetter); ok {
		out := make(map[string]qdto.MintModelMetaEntry, len(ids))
		for _, id := range ids {
			ent, err := g.GetModelMetaByID(ctx, id)
			if err != nil {
				continue
			}
			if ent == nil {
				continue
			}
			v := *ent
			if strings.TrimSpace(v.ModelID) == "" {
				v.ModelID = id
			}
			out[id] = v
		}
		return out, nil
	}

	return map[string]qdto.MintModelMetaEntry{}, nil
}

func (u *MintUsecase) ResolveModelMetaFromInspectionBatch(
	ctx context.Context,
	batch inspectiondom.InspectionBatch,
) (map[string]qdto.MintModelMetaEntry, error) {

	modelIDs := make([]string, 0, len(batch.Inspections))
	for _, it := range batch.Inspections {
		modelIDs = append(modelIDs, strings.TrimSpace(it.ModelID))
	}

	return u.resolveModelMetaByModelIDs(ctx, modelIDs)
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

	savedMint, err := u.mintRepo.Create(ctx, mintEntity)
	if err != nil {
		return empty, err
	}

	mid := strings.TrimSpace(savedMint.ID)
	if mid == "" {
		return empty, errors.New("saved mintID is empty")
	}

	batch, err := u.inspRepo.UpdateMintID(ctx, pid, &mid)
	if err != nil {
		return empty, err
	}

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

// ============================================================
// ★ NEW: Detail API for GET /mint/inspections/{productionId}
// ============================================================

func (u *MintUsecase) GetMintRequestDetail(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {

	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	batches, err := u.ListInspectionBatchesByProductionIDs(ctx, []string{pid})
	if err != nil {
		return empty, err
	}
	if len(batches) == 0 {
		return empty, inspectiondom.ErrNotFound
	}

	for _, b := range batches {
		if strings.TrimSpace(b.ProductionID) == pid {
			return b, nil
		}
	}

	return batches[0], nil
}
