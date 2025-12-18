// backend/internal/application/query/inventory_query.go
package query

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"

	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// Query Service (Read-model assembler)
// - ✅ currentMember.companyId -> productBlueprintIds -> inventoryId(docId)
// - ❌ inventory.Mint の legacy fields (ModelID/Products/Accumulation) は参照しない
// - ✅ Stock(map[modelId]ModelStock) から model 別に在庫数を集計する
// ============================================================

type InventoryQuery struct {
	invRepo      inventoryReader
	pbRepo       productBlueprintIDsByCompanyReader // companyId -> productBlueprintIds
	nameResolver *resolver.NameResolver             // tokenName / modelNumber / productName 解決
}

func NewInventoryQuery(
	invRepo inventoryReader,
	pbRepo productBlueprintIDsByCompanyReader,
	nameResolver *resolver.NameResolver,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// ✅ currentMember.companyId -> productBlueprintIds -> inventories list
// ============================================================
//
// 返す Row は（管理一覧）として
// - ProductBlueprintID / ProductName / TokenBlueprintID / TokenName / ModelNumber / Stock
// を返す。
// ✅ Stock は m.Stock[modelId] の件数で算出する。
func (q *InventoryQuery) ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error) {
	if q == nil || q.invRepo == nil || q.pbRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	// NOTE: companyIDFromContext は package query 内で 1箇所だけ定義してください（重複禁止）
	companyID := companyIDFromContext(ctx)
	if strings.TrimSpace(companyID) == "" {
		return nil, errors.New("companyId is missing in context")
	}

	pbIDs, err := q.pbRepo.ListIDsByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []querydto.InventoryManagementRowDTO{}, nil
	}

	// ✅ 集計キーは tokenName ではなく tokenBlueprintId を正とする（detail遷移に必須）
	type key struct {
		pbID     string
		tbID     string
		modelNum string
	}

	// key -> stock sum
	group := map[key]int{}

	// caches
	productNameCache := map[string]string{}
	tokenNameCache := map[string]string{}
	modelNumberCache := map[string]string{}

	for _, pbID := range pbIDs {
		pbID = strings.TrimSpace(pbID)
		if pbID == "" {
			continue
		}

		// product name cache
		if _, ok := productNameCache[pbID]; !ok {
			name := q.resolveProductName(ctx, pbID)
			if name == "" {
				name = pbID
			}
			productNameCache[pbID] = name
		}

		// inventories (docId = inventoryId) を pbID で取得
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err != nil {
			return nil, err
		}
		if len(invs) == 0 {
			continue
		}

		for _, inv := range invs {
			// ✅ tokenBlueprintId は inv.TokenBlueprintID が空でも
			// inventoryId(=docId) の後半から復元する
			tbID := strings.TrimSpace(inv.TokenBlueprintID)
			if tbID == "" {
				_, parsedTbID, ok := parseInventoryID(strings.TrimSpace(inv.ID))
				if ok {
					tbID = parsedTbID
				}
			}

			// ✅ 方針A: tokenBlueprintId が無い在庫は detail へ遷移できないので一覧にも出さない
			if tbID == "" {
				continue
			}

			// token name cache（表示用）
			if _, ok := tokenNameCache[tbID]; !ok {
				name := q.resolveTokenName(ctx, tbID)
				if name == "" {
					name = tbID
				}
				if name == "" {
					name = "-"
				}
				tokenNameCache[tbID] = name
			}

			// Stock から model 別に集計
			if len(inv.Stock) == 0 {
				// 在庫ゼロでも行を出したいならここで出す（現状はスキップ）
				continue
			}

			for modelID, ms := range inv.Stock {
				modelID = strings.TrimSpace(modelID)
				if modelID == "" {
					continue
				}

				// model number cache
				if _, ok := modelNumberCache[modelID]; !ok {
					mn := q.resolveModelNumber(ctx, modelID)
					if mn == "" {
						mn = modelID
					}
					if mn == "" {
						mn = "-"
					}
					modelNumberCache[modelID] = mn
				}
				modelNumber := modelNumberCache[modelID]

				stock := modelStockLen(ms) // ✅ Products/Accumulation は使わず Stock で算出
				if stock <= 0 {
					continue
				}

				k := key{pbID: pbID, tbID: tbID, modelNum: modelNumber}
				group[k] += stock
			}
		}
	}

	rows := make([]querydto.InventoryManagementRowDTO, 0, len(group))
	for k, stock := range group {
		rows = append(rows, querydto.InventoryManagementRowDTO{
			ProductBlueprintID: k.pbID,
			ProductName:        productNameCache[k.pbID],

			// ✅ ここが今回の本命（フロントが detail URL を作るのに必要）
			TokenBlueprintID: k.tbID,

			TokenName:   tokenNameCache[k.tbID],
			ModelNumber: k.modelNum,
			Stock:       stock,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProductName != rows[j].ProductName {
			return rows[i].ProductName < rows[j].ProductName
		}
		// tokenName より tbId のほうが安定。表示順は tokenName でもOK
		if rows[i].TokenName != rows[j].TokenName {
			return rows[i].TokenName < rows[j].TokenName
		}
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		return rows[i].Stock < rows[j].Stock
	})

	return rows, nil
}

// ============================================================
// helpers (NameResolver)
// ============================================================

func (q *InventoryQuery) resolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tokenBlueprintID))
}

func (q *InventoryQuery) resolveModelNumber(ctx context.Context, modelVariationID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveModelNumber(ctx, modelVariationID))
}

func (q *InventoryQuery) resolveProductName(ctx context.Context, productBlueprintID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, productBlueprintID))
}

// ============================================================
// Minimal readers (ports)
// ============================================================

type inventoryReader interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
}

type productBlueprintIDsByCompanyReader interface {
	ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error)
}

// ============================================================
// ID helpers
// - inventoryId := "{productBlueprintId}__{tokenBlueprintId}"
//   → 後半から tokenBlueprintId を復元できる
// ============================================================

func parseInventoryID(inventoryID string) (pbID, tbID string, ok bool) {
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return "", "", false
	}

	const sep = "__"
	i := strings.LastIndex(id, sep)
	if i <= 0 || i+len(sep) >= len(id) {
		return "", "", false
	}

	pbID = strings.TrimSpace(id[:i])
	tbID = strings.TrimSpace(id[i+len(sep):])
	if pbID == "" || tbID == "" {
		return "", "", false
	}
	return pbID, tbID, true
}

// ============================================================
// Stock helpers
// - invdom.ModelStock が map alias でも struct でも安全に len を取る
// ============================================================

func modelStockLen(ms invdom.ModelStock) int {
	rv := reflect.ValueOf(ms)
	if !rv.IsValid() {
		return 0
	}

	// map alias
	if rv.Kind() == reflect.Map {
		return rv.Len()
	}

	// struct 内に map[string]bool フィールドがあるケース
	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)
			if f.Kind() != reflect.Map {
				continue
			}
			if f.Type().Key().Kind() != reflect.String || f.Type().Elem().Kind() != reflect.Bool {
				continue
			}
			return f.Len()
		}
	}

	return 0
}
