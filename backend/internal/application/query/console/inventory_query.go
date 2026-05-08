// backend/internal/application/query/console/inventory_query.go
package query

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Query Service (Read-model assembler)
//
// Responsibility:
// - Console 用の Inventory read-model を組み立てる（管理一覧 / 詳細）。
// - company boundary は Context の currentMember.companyId を唯一の正として扱う。
// - pbId / tbId は必ず inventory テーブル（inv.ProductBlueprintID / inv.TokenBlueprintID）から拾う。
//   - inventoryId のビルド/分解（split/parse）などの推測ロジックは一切持たない。
// ============================================================

type InventoryQuery struct {
	invRepo      inventoryReader
	pbRepo       productBlueprintIDsByCompanyReader
	pbPatchRepo  productBlueprintPatchReader
	tbPatchRepo  tokenBlueprintPatchReader
	nameResolver *resolver.NameResolver
}

func NewInventoryQueryWithTokenBlueprintPatch(
	invRepo inventoryReader,
	pbRepo productBlueprintIDsByCompanyReader,
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// ✅ currentMember.companyId -> productBlueprintIds -> inventories list
// ============================================================

func (q *InventoryQuery) ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error) {
	if q == nil || q.invRepo == nil || q.pbRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return nil, errors.New("companyId is missing in context")
	}

	pbIDs, err := q.pbRepo.ListIDsByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []querydto.InventoryManagementRowDTO{}, nil
	}

	type key struct {
		pbID     string
		tbID     string
		modelNum string
	}

	type agg struct {
		available int
		reserved  int
	}

	group := map[key]agg{}

	productNameCache := map[string]string{}
	tokenNameCache := map[string]string{}
	modelNumberCache := map[string]string{}

	for _, pbID0 := range pbIDs {
		pbID := pbID0
		if pbID == "" {
			continue
		}

		// productName cache
		if _, ok := productNameCache[pbID]; !ok {
			name := ""
			if q.nameResolver != nil {
				name = q.nameResolver.ResolveProductName(ctx, pbID)
			}
			if name == "" {
				name = pbID
			}
			productNameCache[pbID] = name
		}

		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err != nil {
			return nil, err
		}
		if len(invs) == 0 {
			continue
		}

		for _, inv := range invs {
			// ✅ TokenBlueprintID は必ず存在する前提
			tbID := inv.TokenBlueprintID

			// tokenName cache
			if _, ok := tokenNameCache[tbID]; !ok {
				name := ""
				if q.nameResolver != nil {
					name = q.nameResolver.ResolveTokenName(ctx, tbID)
				}
				if name == "" {
					name = tbID
				}
				if name == "" {
					name = "-"
				}
				tokenNameCache[tbID] = name
			}

			// ✅ inv.Stock が空でも「inventory テーブルがある限り list」する
			// - modelNumber は不明なので "-" を採用
			if len(inv.Stock) == 0 {
				k := key{pbID: pbID, tbID: tbID, modelNum: "-"}
				if _, ok := group[k]; !ok {
					group[k] = agg{available: 0, reserved: 0}
				}
				continue
			}

			for modelID0, ms := range inv.Stock {
				modelID := modelID0
				if modelID == "" {
					continue
				}

				// ✅ 管理一覧では modelNumber だけ必要
				if _, ok := modelNumberCache[modelID]; !ok {
					mn := ""
					if q.nameResolver != nil {
						attr := q.nameResolver.ResolveModelResolved(ctx, modelID)
						mn = attr.ModelNumber
					}
					if mn == "" {
						mn = modelID
					}
					if mn == "" {
						mn = "-"
					}
					modelNumberCache[modelID] = mn
				}
				modelNumber := modelNumberCache[modelID]

				// ✅ domain contract（ModelStock.Validate）前提で素直に計算
				reserved := ms.ReservedCount
				available := ms.Accumulation - reserved
				if available < 0 {
					// 契約上は起きない想定だが、表示を壊さないための保険
					log.Printf("[inventory_query][stock] WARN availableStock negative accumulation=%d reserved=%d -> clamp to 0", ms.Accumulation, reserved)
					available = 0
				}

				// ✅ 0 でも行を落とさない（要件）
				k := key{pbID: pbID, tbID: tbID, modelNum: modelNumber}
				a := group[k]
				a.available += available
				a.reserved += reserved
				group[k] = a
			}
		}
	}

	rows := make([]querydto.InventoryManagementRowDTO, 0, len(group))
	for k, a := range group {
		rows = append(rows, querydto.InventoryManagementRowDTO{
			ProductBlueprintID: k.pbID,
			ProductName:        productNameCache[k.pbID],
			TokenBlueprintID:   k.tbID,
			TokenName:          tokenNameCache[k.tbID],
			ModelNumber:        k.modelNum,
			AvailableStock:     a.available,
			ReservedCount:      a.reserved,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProductName != rows[j].ProductName {
			return rows[i].ProductName < rows[j].ProductName
		}
		if rows[i].TokenName != rows[j].TokenName {
			return rows[i].TokenName < rows[j].TokenName
		}
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		return rows[i].AvailableStock < rows[j].AvailableStock
	})

	return rows, nil
}

// ============================================================
// ✅ TokenBlueprint Patch: tbId -> Patch
// - tbdom.Patch は repository_port.go の contract 通りに BrandID/BrandName が揺れない前提
// - BrandName は NameResolver で補完して返す（表示用）
// ============================================================

func (q *InventoryQuery) GetTokenBlueprintPatchByID(ctx context.Context, tokenBlueprintID string) (*tbdom.Patch, error) {
	if q == nil {
		return nil, errors.New("inventory query is nil")
	}
	if q.tbPatchRepo == nil {
		return nil, errors.New("tokenBlueprint patch repository is not configured")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is required")
	}

	patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID) // value
	if err != nil {
		return nil, err
	}

	// BrandName の補完（必要な場合のみ）
	setOK := false
	if patch.BrandID != "" && patch.BrandName == "" && q.nameResolver != nil {
		brandName := q.nameResolver.ResolveBrandName(ctx, patch.BrandID)
		if brandName != "" {
			patch.BrandName = brandName
			setOK = true
		}
	}

	log.Printf(
		"[inventory_query][GetTokenBlueprintPatchByID] brand resolve tbId=%q brandId=%q brandName=%q setOK=%t",
		tbID, patch.BrandID, patch.BrandName, setOK,
	)

	return &patch, nil
}

// ============================================================
// ✅ Detail: inventoryId -> DTO
// - pbId/tbId は inventory テーブルから拾うのみ（推測・split・build なし）
// - modelRefs(displayOrder) を正として 0 在庫も rows に含める
// ============================================================

func (q *InventoryQuery) GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	id := inventoryID
	if id == "" {
		return nil, errors.New("inventoryId is required")
	}

	// ✅ inventoryId で直接取得（ここ以外で inventoryId を作らない）
	inv, err := q.invRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// ✅ inventory テーブルのカラムを正として使う（推測しない）
	pbID := inv.ProductBlueprintID
	tbID := inv.TokenBlueprintID

	if pbID == "" {
		return nil, errors.New("productBlueprintId is empty in inventory")
	}
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is empty in inventory")
	}

	// ✅ productBlueprintPatch は必須（modelRefs の唯一の真実）
	if q.pbPatchRepo == nil {
		return nil, errors.New("productBlueprint patch repository is not configured")
	}

	pbPatch, e := q.pbPatchRepo.GetPatchByID(ctx, pbID) // value
	if e != nil {
		return nil, e
	}

	// pbdom.Patch は BrandID/BrandName が *string（repository_port.go）
	setOK := false
	if pbPatch.BrandID != nil && *pbPatch.BrandID != "" && (pbPatch.BrandName == nil || *pbPatch.BrandName == "") && q.nameResolver != nil {
		brandName := q.nameResolver.ResolveBrandName(ctx, *pbPatch.BrandID)
		if brandName != "" {
			pbPatch.BrandName = &brandName
			setOK = true
		}
	}

	log.Printf(
		"[inventory_query][GetDetailByID] patch brand resolve pbId=%q brandId=%v brandName=%v setOK=%t",
		pbID, pbPatch.BrandID, pbPatch.BrandName, setOK,
	)

	pbPatchPtr := &pbPatch

	// ✅ tokenBlueprintPatch（取れない場合は省略）
	var tbPatchPtr *tbdom.Patch
	{
		p, ee := q.GetTokenBlueprintPatchByID(ctx, tbID)
		if ee != nil {
			log.Printf("[inventory_query][GetDetailByID] WARN GetTokenBlueprintPatchByID failed tbId=%q err=%v", tbID, ee)
			tbPatchPtr = nil
		} else {
			tbPatchPtr = p
		}
	}

	if pbPatchPtr.ModelRefs == nil || len(*pbPatchPtr.ModelRefs) == 0 {
		return nil, errors.New("productBlueprintPatch.modelRefs is empty (fallback via inv.Stock is abolished)")
	}

	refs := append([]pbdom.ModelRef(nil), (*pbPatchPtr.ModelRefs)...)

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].DisplayOrder < refs[j].DisplayOrder
	})

	orderedModelIDs := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, r := range refs {
		mid := r.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		orderedModelIDs = append(orderedModelIDs, mid)
	}

	if len(orderedModelIDs) == 0 {
		return nil, errors.New("productBlueprintPatch.modelRefs has no valid modelId")
	}

	rows := make([]querydto.InventoryDetailRowDTO, 0, len(orderedModelIDs))
	total := 0

	for _, modelID0 := range orderedModelIDs {
		modelID := modelID0
		if modelID == "" {
			continue
		}

		ms, ok := inv.Stock[modelID]

		available := 0
		if ok {
			// domain contract 前提で素直に計算
			available = ms.Accumulation - ms.ReservedCount
			if available < 0 {
				log.Printf("[inventory_query][stock] WARN availableStock negative accumulation=%d reserved=%d -> clamp to 0", ms.Accumulation, ms.ReservedCount)
				available = 0
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, modelID)
		}

		mn := attr.ModelNumber
		if mn == "" {
			mn = modelID
		}
		if mn == "" {
			mn = "-"
		}

		sz := attr.Size
		cl := attr.Color

		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		missingColor := attr.Color == ""
		missingRGB := attr.RGB == nil
		if missingColor || missingRGB {
			log.Printf(
				"[inventory_query][GetDetailByID] modelResolved inventoryId=%q pbId=%q tbId=%q modelId=%q mn=%q size=%q color=%q rgb=%v rgbType=%T stock=%d missing={color:%t,rgb:%t}",
				id, pbID, tbID, modelID, mn, sz, cl, attr.RGB, attr.RGB, available, missingColor, missingRGB,
			)
		}

		rows = append(rows, querydto.InventoryDetailRowDTO{
			ModelID:     modelID,
			ModelNumber: mn,
			Size:        sz,
			Color:       cl,
			RGB:         attr.RGB,
			Stock:       available,
		})

		total += available
	}

	// domain: Mint は CreatedAt/UpdatedAt が time.Time（揺れなし）
	updated := inv.UpdatedAt
	if updated.IsZero() {
		updated = inv.CreatedAt
	}
	updatedAt := ""
	if !updated.IsZero() {
		updatedAt = updated.UTC().Format(time.RFC3339)
	}

	dto := &querydto.InventoryDetailDTO{
		InventoryID:           id,
		TokenBlueprintID:      tbID,
		ProductBlueprintID:    pbID,
		ProductBlueprintPatch: pbPatchPtr,
		TokenBlueprintPatch:   tbPatchPtr,
		Rows:                  rows,
		TotalStock:            total,
		UpdatedAt:             updatedAt,
	}

	return dto, nil
}

// ============================================================
// Minimal readers (ports)
// ============================================================

type inventoryReader interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
	GetByID(ctx context.Context, inventoryID string) (invdom.Mint, error)
}

// productBlueprint.Repository の contract に合わせる（ListIDsByCompany）
type productBlueprintIDsByCompanyReader interface {
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
}

type productBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
}

// tokenBlueprint.RepositoryPort の contract（GetPatchByID）
type tokenBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}
