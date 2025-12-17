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
// 返す Row は従来互換のため
// - ProductBlueprintID / ProductName / TokenName / ModelNumber / Stock
// を維持しつつ、Stock は m.Stock[modelId] の件数で算出する。
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

	type key struct {
		pbID      string
		tokenName string
		modelNum  string
	}

	group := map[key]int{}
	productNameCache := map[string]string{}

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
			tbID := strings.TrimSpace(inv.TokenBlueprintID)

			tokenName := q.resolveTokenName(ctx, tbID)
			if tokenName == "" {
				tokenName = tbID
			}
			if tokenName == "" {
				tokenName = "-"
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

				modelNumber := q.resolveModelNumber(ctx, modelID)
				if modelNumber == "" {
					modelNumber = modelID
				}
				if modelNumber == "" {
					modelNumber = "-"
				}

				stock := modelStockLen(ms) // ✅ Products/Accumulation は使わず Stock で算出
				if stock <= 0 {
					continue
				}

				k := key{pbID: pbID, tokenName: tokenName, modelNum: modelNumber}
				group[k] += stock
			}
		}
	}

	rows := make([]querydto.InventoryManagementRowDTO, 0, len(group))
	for k, stock := range group {
		rows = append(rows, querydto.InventoryManagementRowDTO{
			ProductBlueprintID: k.pbID,
			ProductName:        productNameCache[k.pbID],
			TokenName:          k.tokenName,
			ModelNumber:        k.modelNum,
			Stock:              stock,
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
