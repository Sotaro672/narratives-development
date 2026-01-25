// backend/internal/application/query/console/list_create_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
// - pbId から: productName / brandName
// - tbId から: tokenName / brandName
//
// ✅ FIX:
// - PriceRows の母集団を「inventory.Stock」ではなく「models(variations)」にできるようにする。
//   -> 在庫が 0 / inventory.Stock に存在しない modelId でも、PriceRows に行を出せる。
//   -> stock は inventory があれば反映、無ければ 0 で返す。
//
// ✅ NOTE:
// - productBlueprintPatchReader / tokenBlueprintPatchReader / inventoryReader / getStringFieldAny / modelStockNumbers は
//   inventory_query.go 側の定義を正として「重複定義しない」
//
// ✅ IMPORTANT:
// - stock は availableStock（accumulation - reservedCount）を返す
// ============================================================

type ListCreateQuery struct {
	// inventory から stock を引くため（任意）
	invRepo inventoryReader // defined in inventory_query.go

	// ✅ models(variations) から modelId 一覧を引くため（任意）
	modelRepo modelIDsReader

	pbPatchRepo  productBlueprintPatchReader // defined in inventory_query.go
	tbPatchRepo  tokenBlueprintPatchReader   // defined in inventory_query.go
	nameResolver *resolver.NameResolver
}

// ✅ NEW: variations の ID 一覧を引く最小ポート（Firestore 実装で提供する）
type modelIDsReader interface {
	ListModelIDsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]string, error)
}

// 互換: 既存 DI を壊さない（invRepo/modelRepo なしでも DTO の基本情報は返す）
func NewListCreateQuery(
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      nil, // optional (backward compatible)
		modelRepo:    nil, // optional (backward compatible)
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// 互換: inventory reader も注入できるコンストラクタ（TotalStock を埋める）
func NewListCreateQueryWithInventory(
	invRepo inventoryReader,
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      invRepo,
		modelRepo:    nil, // optional
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// ✅ NEW: inventory + models を注入（PriceRows を「在庫に依存せず」作れる）
func NewListCreateQueryWithInventoryAndModels(
	invRepo inventoryReader,
	modelRepo modelIDsReader,
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      invRepo,
		modelRepo:    modelRepo,
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// inventoryId から ListCreateDTO を組み立てる（互換: listImage 入力無し）
// ============================================================

func (q *ListCreateQuery) GetByInventoryID(
	ctx context.Context,
	inventoryID string,
) (*querydto.ListCreateDTO, error) {
	return q.GetByInventoryIDWithListImage(ctx, inventoryID, "")
}

// inventoryId から ListCreateDTO を組み立てる（listImage 入力あり）
func (q *ListCreateQuery) GetByInventoryIDWithListImage(
	ctx context.Context,
	inventoryID string,
	listImageURL string,
) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}

	invID := strings.TrimSpace(inventoryID)
	if invID == "" {
		return nil, errors.New("inventoryId is required")
	}

	// inventoryId = "{pbId}__{tbId}" を正として parse
	pbID, tbID, ok := parseInventoryID(invID)
	if !ok || pbID == "" || tbID == "" {
		return nil, errors.New("invalid inventoryId format (expected {pbId}__{tbId})")
	}

	// ------------------------------------------------------------
	// ProductBlueprint: productName / brandName
	// ------------------------------------------------------------
	productName := ""
	productBrandName := ""

	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
	}

	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if productBrandName == "" {
				productBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if tokenBrandName == "" {
				tokenBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// ✅ PriceRows: models を母集団にして作る（stock は inventory があれば反映）
	// ------------------------------------------------------------
	priceRows, totalStock := q.buildPriceRowsByInventoryID(ctx, invID)

	dto := &querydto.ListCreateDTO{
		InventoryID:        invID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		PriceRows:  priceRows,
		TotalStock: totalStock,
	}

	dto.ListImageURL = strings.TrimSpace(listImageURL)
	return dto, nil
}

// GetByIDs assembles ListCreateDTO from pbId/tbId.
func (q *ListCreateQuery) GetByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) (*querydto.ListCreateDTO, error) {
	return q.GetByIDsWithListImage(ctx, productBlueprintID, tokenBlueprintID, "")
}

func (q *ListCreateQuery) GetByIDsWithListImage(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
	listImageURL string,
) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, errors.New("productBlueprintId and tokenBlueprintId are required")
	}

	// ------------------------------------------------------------
	// ProductBlueprint: productName / brandName
	// ------------------------------------------------------------
	productName := ""
	productBrandName := ""

	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
	}

	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if productBrandName == "" {
				productBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if tokenBrandName == "" {
				tokenBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// ✅ PriceRows: models を母集団にして作る
	// ------------------------------------------------------------
	priceRows, totalStock := q.buildPriceRowsByIDs(ctx, pbID, tbID)

	dto := &querydto.ListCreateDTO{
		InventoryID:        buildInventoryID(pbID, tbID),
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		PriceRows:  priceRows,
		TotalStock: totalStock,
	}

	dto.ListImageURL = strings.TrimSpace(listImageURL)
	return dto, nil
}

// inventoryId = "{pbId}__{tbId}"
func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	return strings.TrimSpace(productBlueprintID) + "__" + strings.TrimSpace(tokenBlueprintID)
}

func parseInventoryID(inventoryID string) (pbID string, tbID string, ok bool) {
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return "", "", false
	}
	parts := strings.SplitN(id, "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

// ============================================================
// internal: build PriceRows
// - 母集団: modelRepo(=models/variations) があればそれを正とする
// - stock: inventory が取れれば picked.Stock[modelId] を反映、無ければ 0
// - 重要: stock==0 でも行を出す（価格入力のため）
//
// ✅ stock は availableStock（accumulation - reservedCount）
// ============================================================

func (q *ListCreateQuery) buildPriceRowsByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) ([]querydto.ListCreatePriceRowDTO, int) {
	if q == nil {
		return nil, 0
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, 0
	}

	// 1) 母集団 modelIds（modelRepo があれば優先）
	modelIDs := q.listModelIDs(ctx, pbID)

	// 2) inventory を拾えれば stock 参照に使う（拾えなくても PriceRows は返す）
	var picked *invdom.Mint
	if q.invRepo != nil {
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err == nil && len(invs) > 0 {
			wantInvID := buildInventoryID(pbID, tbID)

			for i := range invs {
				if strings.TrimSpace(invs[i].ID) == wantInvID {
					picked = &invs[i]
					break
				}
			}
			if picked == nil {
				for i := range invs {
					if strings.TrimSpace(invs[i].TokenBlueprintID) == tbID {
						picked = &invs[i]
						break
					}
				}
			}
		}
	}

	// fallback: modelRepo が無い/空で、inventory.Stock があればキーで回す
	if len(modelIDs) == 0 && picked != nil && picked.Stock != nil {
		for mid := range picked.Stock {
			mid = strings.TrimSpace(mid)
			if mid != "" {
				modelIDs = append(modelIDs, mid)
			}
		}
		sort.Strings(modelIDs)
	}

	if len(modelIDs) == 0 {
		return nil, 0
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(modelIDs))
	total := 0

	for _, mid0 := range modelIDs {
		mid := strings.TrimSpace(mid0)
		if mid == "" {
			continue
		}

		stock := 0
		if picked != nil && picked.Stock != nil {
			if ms, ok := picked.Stock[mid]; ok {
				// ✅ availableStock を採用（accumulation - reservedCount）
				_, _, available := modelStockNumbers(ms) // defined in inventory_query.go
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		sz := strings.TrimSpace(attr.Size)
		cl := strings.TrimSpace(attr.Color)
		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		rows = append(rows, querydto.ListCreatePriceRowDTO{
			ModelID: mid,
			Stock:   stock, // ✅ 0 でも出す（availableStock）
			Size:    sz,
			Color:   cl,
			RGB:     attr.RGB,
			Price:   nil,
		})

		total += stock
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		return rows[i].ModelID < rows[j].ModelID
	})

	return rows, total
}

func (q *ListCreateQuery) buildPriceRowsByInventoryID(
	ctx context.Context,
	inventoryID string,
) ([]querydto.ListCreatePriceRowDTO, int) {
	if q == nil {
		return nil, 0
	}

	invID := strings.TrimSpace(inventoryID)
	if invID == "" {
		return nil, 0
	}

	pbID, tbID, ok := parseInventoryID(invID)
	if !ok || pbID == "" || tbID == "" {
		return nil, 0
	}

	// 1) 母集団 modelIds（modelRepo があれば優先）
	modelIDs := q.listModelIDs(ctx, pbID)

	// 2) inventory を拾えれば stock 参照に使う（拾えなくても PriceRows は返す）
	var picked *invdom.Mint
	if q.invRepo != nil {
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err == nil && len(invs) > 0 {
			for i := range invs {
				if strings.TrimSpace(invs[i].ID) == invID {
					picked = &invs[i]
					break
				}
			}
			if picked == nil {
				for i := range invs {
					if strings.TrimSpace(invs[i].TokenBlueprintID) == tbID {
						picked = &invs[i]
						break
					}
				}
			}
		}
	}

	// fallback: modelRepo が無い/空で、inventory.Stock があればキーで回す
	if len(modelIDs) == 0 && picked != nil && picked.Stock != nil {
		for mid := range picked.Stock {
			mid = strings.TrimSpace(mid)
			if mid != "" {
				modelIDs = append(modelIDs, mid)
			}
		}
		sort.Strings(modelIDs)
	}

	if len(modelIDs) == 0 {
		return nil, 0
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(modelIDs))
	total := 0

	for _, mid0 := range modelIDs {
		mid := strings.TrimSpace(mid0)
		if mid == "" {
			continue
		}

		stock := 0
		if picked != nil && picked.Stock != nil {
			if ms, ok := picked.Stock[mid]; ok {
				// ✅ availableStock を採用（accumulation - reservedCount）
				_, _, available := modelStockNumbers(ms) // defined in inventory_query.go
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		sz := strings.TrimSpace(attr.Size)
		cl := strings.TrimSpace(attr.Color)
		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		rows = append(rows, querydto.ListCreatePriceRowDTO{
			ModelID: mid,
			Stock:   stock, // ✅ 0 でも出す（availableStock）
			Size:    sz,
			Color:   cl,
			RGB:     attr.RGB,
			Price:   nil,
		})

		total += stock
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		return rows[i].ModelID < rows[j].ModelID
	})

	return rows, total
}

func (q *ListCreateQuery) listModelIDs(ctx context.Context, productBlueprintID string) []string {
	if q == nil || q.modelRepo == nil {
		return nil
	}
	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return nil
	}

	ids, err := q.modelRepo.ListModelIDsByProductBlueprintID(ctx, pbID)
	if err != nil || len(ids) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, x := range ids {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
