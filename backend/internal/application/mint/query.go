// backend/internal/application/mint/query.go
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
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// internal helper: createdBy(memberId) -> display name
// nameResolver が無い/解決できない場合は memberId を返す
func (u *MintUsecase) resolveCreatedByName(ctx context.Context, memberID string) string {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return ""
	}

	if u != nil && u.nameResolver != nil {
		if name := strings.TrimSpace(u.nameResolver.ResolveMemberName(ctx, memberID)); name != "" {
			return name
		}
	}
	return memberID
}

// ============================================================
// Query: mints を inspectionIds(docId) で取得
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
				if isNotFoundErr(err) {
					continue
				}
				// ✅ 一部レコードの整合性エラーで一覧全体を 500 にしない（ログしてスキップ）
				if isInconsistentMintErr(err) {
					log.Printf("[mint_query] ListMintsByInspectionIDs skip inconsistent mint id=%q err=%v", id, err)
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
				if isNotFoundErr(err) {
					continue
				}
				// ✅ 一部レコードの整合性エラーで一覧全体を 500 にしない（ログしてスキップ）
				if isInconsistentMintErr(err) {
					log.Printf("[mint_query] ListMintsByInspectionIDs skip inconsistent mint id=%q err=%v", id, err)
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
// Query: mints(list) を inspectionIds で取得し、名前解決して DTO を組み立てる
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

	log.Printf("[mint_query] ListMintListRowsByInspectionIDs start ids=%d mints=%d nameResolver=%t",
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

	log.Printf("[mint_query] ListMintListRowsByInspectionIDs done out=%d sampleKey=%q",
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
// Query: ProductBlueprint Patch 解決
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
// Query: model variations -> modelMeta（任意）
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
// Query: Brand 一覧（current company）
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
// Query: TokenBlueprint 一覧（brandId フィルタ）
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

// ============================================================
// Query: inspection batches by production docIds
// ============================================================

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
// Query: Detail API for GET /mint/inspections/{productionId}
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
