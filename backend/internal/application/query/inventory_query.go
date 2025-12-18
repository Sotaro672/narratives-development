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
// - ✅ tokenBlueprintId は inv.TokenBlueprintID が空でも inventoryId から推測する
// ============================================================

type InventoryQuery struct {
	invRepo      inventoryReader
	pbRepo       productBlueprintIDsByCompanyReader
	nameResolver *resolver.NameResolver
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
func (q *InventoryQuery) ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error) {
	if q == nil || q.invRepo == nil || q.pbRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

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
		pbID     string
		tbID     string
		modelNum string
	}

	group := map[key]int{}

	productNameCache := map[string]string{}
	tokenNameCache := map[string]string{}
	modelNumberCache := map[string]string{}

	for _, pbID := range pbIDs {
		pbID = strings.TrimSpace(pbID)
		if pbID == "" {
			continue
		}

		if _, ok := productNameCache[pbID]; !ok {
			name := q.resolveProductName(ctx, pbID)
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
			// ✅ tbId は field が空なら inventoryId から推測する
			tbID := strings.TrimSpace(inv.TokenBlueprintID)
			if tbID == "" {
				tbID = parseTokenBlueprintIDFromInventoryID(inv.ID, pbID)
			}
			tbID = strings.TrimSpace(tbID)

			// detail 遷移に必須なので、取れないものは出さない
			if tbID == "" {
				continue
			}

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

			if len(inv.Stock) == 0 {
				continue
			}

			for modelID, ms := range inv.Stock {
				modelID = strings.TrimSpace(modelID)
				if modelID == "" {
					continue
				}

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

				stock := modelStockLen(ms)
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
			TokenBlueprintID:   k.tbID,
			TokenName:          tokenNameCache[k.tbID],
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
// ✅ NEW: pbId + tbId -> inventoryIds
// ============================================================

func (q *InventoryQuery) ListInventoryIDsByProductAndToken(ctx context.Context, productBlueprintID, tokenBlueprintID string) ([]string, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, errors.New("productBlueprintId and tokenBlueprintId are required")
	}

	invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
	if err != nil {
		return nil, err
	}
	if len(invs) == 0 {
		return []string{}, nil
	}

	out := make([]string, 0, len(invs))
	seen := map[string]struct{}{}

	for _, inv := range invs {
		invID := strings.TrimSpace(inv.ID)
		if invID == "" {
			continue
		}

		gotTbID := strings.TrimSpace(inv.TokenBlueprintID)
		if gotTbID == "" {
			gotTbID = parseTokenBlueprintIDFromInventoryID(invID, pbID)
		}
		gotTbID = strings.TrimSpace(gotTbID)

		if gotTbID != tbID {
			continue
		}

		if _, ok := seen[invID]; ok {
			continue
		}
		seen[invID] = struct{}{}
		out = append(out, invID)
	}

	sort.Strings(out)
	return out, nil
}

// inventoryId = "{pbId}__{tbId}" から tbId を抜く
func parseTokenBlueprintIDFromInventoryID(inventoryID, productBlueprintID string) string {
	id := strings.TrimSpace(inventoryID)
	pb := strings.TrimSpace(productBlueprintID)
	if id == "" || pb == "" {
		return ""
	}

	prefix := pb + "__"
	if !strings.HasPrefix(id, prefix) {
		return ""
	}

	suffix := strings.TrimSpace(strings.TrimPrefix(id, prefix))
	if suffix == "" {
		return ""
	}

	// 念のため "__" が複数ある場合は最後を tbId とみなす
	parts := strings.Split(suffix, "__")
	return strings.TrimSpace(parts[len(parts)-1])
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
// ============================================================

func modelStockLen(ms invdom.ModelStock) int {
	rv := reflect.ValueOf(ms)
	if !rv.IsValid() {
		return 0
	}

	if rv.Kind() == reflect.Map {
		return rv.Len()
	}

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
