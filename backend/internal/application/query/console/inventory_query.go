// backend/internal/application/query/inventory_query.go
package query

import (
	"context"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Query Service (Read-model assembler)
// ============================================================

type InventoryQuery struct {
	invRepo      inventoryReader
	pbRepo       productBlueprintIDsByCompanyReader
	pbPatchRepo  productBlueprintPatchReader
	tbPatchRepo  tokenBlueprintPatchReader // ✅ NEW: tokenBlueprint patch
	nameResolver *resolver.NameResolver
}

func NewInventoryQuery(
	invRepo inventoryReader,
	pbRepo productBlueprintIDsByCompanyReader,
	pbPatchRepo productBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  nil, // ✅ optional (backward compatible)
		nameResolver: nameResolver,
	}
}

// ✅ NEW: tokenBlueprint patch も注入できるコンストラクタ（DI でこちらを使う）
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

	type modelAttr struct {
		modelNumber string
		size        string
		color       string
		rgb         *int
	}

	group := map[key]int{}

	productNameCache := map[string]string{}
	tokenNameCache := map[string]string{}
	modelAttrCache := map[string]modelAttr{}

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
			// ✅ tbId は field を正とする（推測しない）
			tbID := strings.TrimSpace(inv.TokenBlueprintID)

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

				// ✅ Resolver でまとめて解決（管理一覧は modelNumber だけ使う）
				if _, ok := modelAttrCache[modelID]; !ok {
					attr := q.resolveModelResolved(ctx, modelID)

					mn := strings.TrimSpace(attr.ModelNumber)
					if mn == "" {
						mn = modelID
					}
					if mn == "" {
						mn = "-"
					}

					modelAttrCache[modelID] = modelAttr{
						modelNumber: mn,
						size:        strings.TrimSpace(attr.Size),
						color:       strings.TrimSpace(attr.Color),
						rgb:         attr.RGB,
					}
				}
				modelNumber := modelAttrCache[modelID].modelNumber

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
// ✅ pbId + tbId -> inventoryIds
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
			continue
		}
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

// ============================================================
// ✅ TokenBlueprint Patch: tbId -> Patch
// - Patch.BrandID -> brandName を NameResolver で解決して詰める
// - tbPatchRepo が未注入の場合は nil を返して detail を壊さない
// ============================================================

func (q *InventoryQuery) GetTokenBlueprintPatchByID(ctx context.Context, tokenBlueprintID string) (*tbdom.Patch, error) {
	if q == nil {
		return nil, errors.New("inventory query is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is required")
	}

	if q.tbPatchRepo == nil {
		log.Printf("[inventory_query][GetTokenBlueprintPatchByID] WARN tbPatchRepo is nil tbId=%q", tbID)
		return nil, nil
	}

	patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID) // value
	if err != nil {
		return nil, err
	}

	// brand name resolve（可能なら埋める）
	hasBrandID := patch.BrandID != nil && strings.TrimSpace(*patch.BrandID) != ""
	brandID := ""
	if patch.BrandID != nil {
		brandID = strings.TrimSpace(*patch.BrandID)
	}

	brandName := strings.TrimSpace(q.resolveBrandName(ctx, brandID))

	setOK := false
	if hasBrandID && brandName != "" {
		patch.BrandName = &brandName
		setOK = true
	}

	log.Printf(
		"[inventory_query][GetTokenBlueprintPatchByID] brand resolve tbId=%q hasBrandId=%t brandId=%q brandName=%q setOK=%t",
		tbID, hasBrandID, brandID, brandName, setOK,
	)

	return &patch, nil
}

// ============================================================
// ✅ Detail: inventoryId -> DTO
// - productBlueprintPatch の brandId -> brandName を NameResolver で解決して詰める
// - tokenBlueprintPatch を取得して DTO に詰める（TokenBlueprintCard 用）
// ============================================================

func (q *InventoryQuery) GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return nil, errors.New("inventoryId is required")
	}

	// inventoryId は "{pbId}__{tbId}" 前提
	pbID := parseProductBlueprintIDFromInventoryID(id)
	if pbID == "" {
		return nil, errors.New("invalid inventoryId format (expected {pbId}__{tbId})")
	}

	invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	var inv *invdom.Mint
	for i := range invs {
		if strings.TrimSpace(invs[i].ID) == id {
			inv = &invs[i]
			break
		}
	}
	if inv == nil {
		return nil, invdom.ErrNotFound
	}

	tbID := strings.TrimSpace(inv.TokenBlueprintID)
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is empty in inventory")
	}

	// ✅ productBlueprintPatch（取れない場合は省略）
	var pbPatchPtr *pbdom.Patch
	if q.pbPatchRepo != nil {
		pbPatch, e := q.pbPatchRepo.GetPatchByID(ctx, pbID) // value
		if e == nil {
			// ✅ ここが重要：Patch の BrandID/BrandName が「*string」前提で直接扱う
			hasBrandID := pbPatch.BrandID != nil && strings.TrimSpace(*pbPatch.BrandID) != ""
			brandID := ""
			if pbPatch.BrandID != nil {
				brandID = strings.TrimSpace(*pbPatch.BrandID)
			}

			brandName := strings.TrimSpace(q.resolveBrandName(ctx, brandID))

			setOK := false
			if hasBrandID && brandName != "" {
				// Patch に BrandName フィールド（*string）がある前提
				pbPatch.BrandName = &brandName
				setOK = true
			}

			log.Printf(
				"[inventory_query][GetDetailByID] patch brand resolve pbId=%q hasBrandId=%t brandId=%q brandName=%q setOK=%t",
				pbID, hasBrandID, brandID, brandName, setOK,
			)

			pbPatchPtr = &pbPatch // ✅ *Patch に合わせてアドレスを渡す
		} else {
			log.Printf("[inventory_query][GetDetailByID] WARN GetPatchByID failed pbId=%q err=%v", pbID, e)
			pbPatchPtr = nil
		}
	}

	// ✅ tokenBlueprintPatch（取れない場合は省略）
	var tbPatchPtr *tbdom.Patch
	{
		p, e := q.GetTokenBlueprintPatchByID(ctx, tbID)
		if e != nil {
			log.Printf("[inventory_query][GetDetailByID] WARN GetTokenBlueprintPatchByID failed tbId=%q err=%v", tbID, e)
			tbPatchPtr = nil
		} else {
			tbPatchPtr = p
		}
	}

	// rows: modelId ごとの productIds を count + attributes 解決
	rows := make([]querydto.InventoryDetailRowDTO, 0, len(inv.Stock))
	total := 0

	for modelID, ms := range inv.Stock {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}

		cnt := modelStockLen(ms)
		if cnt <= 0 {
			continue
		}

		attr := q.resolveModelResolved(ctx, modelID)

		mn := strings.TrimSpace(attr.ModelNumber)
		if mn == "" {
			mn = modelID
		}
		if mn == "" {
			mn = "-"
		}

		sz := strings.TrimSpace(attr.Size)
		cl := strings.TrimSpace(attr.Color)

		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		// 追跡用（欠落時だけ）
		missingColor := strings.TrimSpace(attr.Color) == ""
		missingRGB := attr.RGB == nil
		if missingColor || missingRGB {
			log.Printf(
				"[inventory_query][GetDetailByID] modelResolved inventoryId=%q pbId=%q tbId=%q modelId=%q mn=%q size=%q color=%q rgb=%v rgbType=%T stock=%d missing={color:%t,rgb:%t}",
				id, pbID, tbID, modelID, mn, sz, cl, attr.RGB, attr.RGB, cnt, missingColor, missingRGB,
			)
		}

		rows = append(rows, querydto.InventoryDetailRowDTO{
			ModelID:     modelID,
			ModelNumber: mn,
			Size:        sz,
			Color:       cl,
			RGB:         attr.RGB,
			Stock:       cnt,
		})

		total += cnt
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		return rows[i].Stock < rows[j].Stock
	})

	updated := pickTimeFromStruct(*inv, "UpdatedAt")
	if updated.IsZero() {
		updated = pickTimeFromStruct(*inv, "CreatedAt")
	}
	updatedAt := ""
	if !updated.IsZero() {
		updatedAt = updated.UTC().Format(time.RFC3339)
	}

	dto := &querydto.InventoryDetailDTO{
		InventoryID:           id,
		TokenBlueprintID:      tbID,
		ProductBlueprintID:    pbID,
		ProductBlueprintPatch: pbPatchPtr, // ✅ *Patch（nil なら omitempty で出ない）
		TokenBlueprintPatch:   tbPatchPtr, // ✅ NEW: TokenBlueprintCard 用
		Rows:                  rows,
		TotalStock:            total,
		UpdatedAt:             updatedAt,
	}

	return dto, nil
}

// inventoryId = "{pbId}__{tbId}" から pbId を抜く
func parseProductBlueprintIDFromInventoryID(inventoryID string) string {
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return ""
	}
	parts := strings.Split(id, "__")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

// reflect で time.Time フィールドを安全に抜く（無ければ zero）
func pickTimeFromStruct(v any, fieldName string) time.Time {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return time.Time{}
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return time.Time{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return time.Time{}
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		return time.Time{}
	}

	if f.Type() == reflect.TypeOf(time.Time{}) {
		if t, ok := f.Interface().(time.Time); ok {
			return t
		}
	}
	if f.Kind() == reflect.Pointer && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
		if f.IsNil() {
			return time.Time{}
		}
		if t, ok := f.Elem().Interface().(time.Time); ok {
			return t
		}
	}

	return time.Time{}
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

func (q *InventoryQuery) resolveProductName(ctx context.Context, productBlueprintID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, productBlueprintID))
}

func (q *InventoryQuery) resolveBrandName(ctx context.Context, brandID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	id := strings.TrimSpace(brandID)
	if id == "" {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, id))
}

// ✅ modelId から modelNumber/size/color/rgb をまとめて解決
func (q *InventoryQuery) resolveModelResolved(ctx context.Context, modelVariationID string) resolver.ModelResolved {
	if q == nil || q.nameResolver == nil {
		return resolver.ModelResolved{}
	}
	id := strings.TrimSpace(modelVariationID)
	if id == "" {
		return resolver.ModelResolved{}
	}
	return q.nameResolver.ResolveModelResolved(ctx, id)
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

// ✅ detail 用に PB Patch を引ける最小ポート
type productBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
}

// ✅ NEW: detail 用に TokenBlueprint Patch を引ける最小ポート
type tokenBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
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

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}

	if rv.Kind() == reflect.Struct {
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)

			if f.Kind() == reflect.Map && f.Type().Key().Kind() == reflect.String {
				return f.Len()
			}
			if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
				return f.Len()
			}
		}
	}

	return 0
}
