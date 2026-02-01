// backend/internal/application/query/console/inventory_query.go
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
	usecase "narratives/internal/application/usecase"

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
	tbPatchRepo  tokenBlueprintPatchReader // ✅ 必須（NewInventoryQuery は廃止）
	nameResolver *resolver.NameResolver
}

// ✅ コンストラクタはこれのみ（NewInventoryQuery は削除）
// - tokenBlueprintPatchRepo も必須注入（DI でこちらを使う）
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

	// ✅ 方針A: usecase の companyId getter を唯一の真実として利用する
	companyID := usecase.CompanyIDFromContext(ctx)
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

	// ✅ availableStock と reservedCount を両方集計する
	type agg struct {
		available int
		reserved  int
	}

	group := map[key]agg{}

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
			// ✅ TokenBlueprintID は必ず存在する前提（空ケース削除）
			tbID := strings.TrimSpace(inv.TokenBlueprintID)

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

			// ✅ inv.Stock が空でも「inventory テーブルがある限り list」する
			// - modelNumber は不明なので "-" を採用
			if len(inv.Stock) == 0 {
				k := key{pbID: pbID, tbID: tbID, modelNum: "-"}
				if _, ok := group[k]; !ok {
					group[k] = agg{available: 0, reserved: 0}
				}
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

				// ✅ availableStock/reservedCount は同一ロジックで算出
				// ✅ 0 でも行を落とさない（要件）
				_, reserved, available := modelStockNumbers(ms)

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

			// 互換: Stock は availableStock と同義
			Stock: a.available,

			// ✅ NEW: 画面へ渡す
			AvailableStock: a.available,
			ReservedCount:  a.reserved,
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
// ✅ TokenBlueprint Patch: tbId -> Patch
// - Patch.BrandID -> brandName を NameResolver で解決して詰める
// ============================================================

func (q *InventoryQuery) GetTokenBlueprintPatchByID(ctx context.Context, tokenBlueprintID string) (*tbdom.Patch, error) {
	if q == nil {
		return nil, errors.New("inventory query is nil")
	}
	if q.tbPatchRepo == nil {
		return nil, errors.New("tokenBlueprint patch repository is not configured")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is required")
	}

	patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID) // value
	if err != nil {
		return nil, err
	}

	// brand name resolve（可能なら埋める）
	brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
	brandName := strings.TrimSpace(q.resolveBrandName(ctx, brandID))

	setOK := false
	if brandID != "" && brandName != "" {
		setStringFieldAny(&patch, brandName, "BrandName", "brandName")
		setOK = true
	}

	log.Printf(
		"[inventory_query][GetTokenBlueprintPatchByID] brand resolve tbId=%q brandId=%q brandName=%q setOK=%t",
		tbID, brandID, brandName, setOK,
	)

	return &patch, nil
}

// ============================================================
// ✅ Detail: inventoryId -> DTO
// - inventory テーブルに productBlueprintId / tokenBlueprintId がある前提
// - inventoryId を分解して pbId/tbId を推測しない
// - modelRefs(displayOrder) を正として 0 在庫も rows に含める
// ============================================================

func (q *InventoryQuery) GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return nil, errors.New("inventoryId is required")
	}

	// ✅ inventoryId で直接取得
	inv, err := q.invRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// ✅ inventory テーブルのカラムを正として使う（推測しない）
	pbID := strings.TrimSpace(inv.ProductBlueprintID)
	tbID := strings.TrimSpace(inv.TokenBlueprintID)

	if pbID == "" {
		return nil, errors.New("productBlueprintId is empty in inventory")
	}
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is empty in inventory")
	}

	// ✅ productBlueprintPatch（取れない場合は省略）
	var pbPatchPtr *pbdom.Patch
	if q.pbPatchRepo != nil {
		pbPatch, e := q.pbPatchRepo.GetPatchByID(ctx, pbID) // value
		if e == nil {
			brandID := strings.TrimSpace(getStringFieldAny(pbPatch, "BrandID", "BrandId", "brandId"))
			brandName := strings.TrimSpace(q.resolveBrandName(ctx, brandID))

			setOK := false
			if brandID != "" && brandName != "" {
				setStringFieldAny(&pbPatch, brandName, "BrandName", "brandName")
				setOK = true
			}

			log.Printf(
				"[inventory_query][GetDetailByID] patch brand resolve pbId=%q brandId=%q brandName=%q setOK=%t",
				pbID, brandID, brandName, setOK,
			)

			pbPatchPtr = &pbPatch
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

	// ============================================================
	// ✅ rows: modelRefs(displayOrder) を正として 0 在庫も含める
	// ============================================================

	rows := make([]querydto.InventoryDetailRowDTO, 0, len(inv.Stock))
	total := 0

	// 1) modelRefs が取れればそれを正とする
	var orderedModelIDs []string
	if pbPatchPtr != nil && pbPatchPtr.ModelRefs != nil && len(*pbPatchPtr.ModelRefs) > 0 {
		// Patch の実体を壊さないようにコピーしてからソート
		refs := append([]pbdom.ModelRef(nil), (*pbPatchPtr.ModelRefs)...)

		// displayOrder 昇順にソート
		sort.Slice(refs, func(i, j int) bool {
			return refs[i].DisplayOrder < refs[j].DisplayOrder
		})

		orderedModelIDs = make([]string, 0, len(refs))
		seen := map[string]struct{}{}
		for _, r := range refs {
			mid := strings.TrimSpace(r.ModelID)
			if mid == "" {
				continue
			}
			// 念のため重複排除
			if _, ok := seen[mid]; ok {
				continue
			}
			seen[mid] = struct{}{}
			orderedModelIDs = append(orderedModelIDs, mid)
		}
	}

	// 2) modelRefs が無い/取れない場合だけ fallback（従来通り inv.Stock）
	if len(orderedModelIDs) == 0 {
		orderedModelIDs = make([]string, 0, len(inv.Stock))
		for modelID := range inv.Stock {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			orderedModelIDs = append(orderedModelIDs, modelID)
		}
		// map iteration の非決定性を排除
		sort.Strings(orderedModelIDs)
	}

	// 3) orderedModelIDs を回して rows を作る（Stock に無い modelId は stock=0）
	for _, modelID := range orderedModelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}

		ms, ok := inv.Stock[modelID]

		available := 0
		if ok {
			// ✅ 0 でも行を落とさない
			_, _, available = modelStockNumbers(ms)
		} else {
			available = 0 // ✅ これで 0 行が必ず出る
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
				id, pbID, tbID, modelID, mn, sz, cl, attr.RGB, attr.RGB, available, missingColor, missingRGB,
			)
		}

		rows = append(rows, querydto.InventoryDetailRowDTO{
			ModelID:     modelID,
			ModelNumber: mn,
			Size:        sz,
			Color:       cl,
			RGB:         attr.RGB,
			Stock:       available, // ✅ 0 も入る
		})

		total += available
	}

	// ✅ 並び順は modelRefs(displayOrder) を尊重するため、ここでは rows をソートしない

	updated := pickTimeFromStruct(inv, "UpdatedAt")
	if updated.IsZero() {
		updated = pickTimeFromStruct(inv, "CreatedAt")
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
		TokenBlueprintPatch:   tbPatchPtr, // ✅ TokenBlueprintCard 用
		Rows:                  rows,
		TotalStock:            total, // availableStock 合計（0 行も含めるが加算は 0）
		UpdatedAt:             updatedAt,
	}

	return dto, nil
}

// ============================================================
// reflect helpers
// ============================================================

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
	// 一覧用途
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
	// 詳細用途（inventoryId で直接取得）
	GetByID(ctx context.Context, inventoryID string) (invdom.Mint, error)
}

type productBlueprintIDsByCompanyReader interface {
	ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error)
}

// ✅ detail 用に PB Patch を引ける最小ポート
type productBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
}

// ✅ detail 用に TokenBlueprint Patch を引ける最小ポート
type tokenBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

// ============================================================
// Stock helpers
// ============================================================

// modelStockNumbers は UI 表示用の在庫数を安定して算出する。
// 方針:
//   - accumulation は ms.Accumulation を正とし、0 の場合のみ len(Products) をフォールバック
//   - reservedCount は「availableStock 計算に使った値」と一致させる（= max(stored, sum(ReservedByOrder))）
//   - available は accumulation - reservedCount（負なら 0 に丸めて WARN）
func modelStockNumbers(ms invdom.ModelStock) (accumulation int, reservedCount int, available int) {
	// accumulation
	accumulation = ms.Accumulation
	if accumulation == 0 && len(ms.Products) > 0 {
		accumulation = len(ms.Products)
	}

	// reservedCount（ドメイン更新の値を基本にしつつ、表示は過大在庫を防ぐ）
	reservedStored := ms.ReservedCount

	sum := 0
	for _, q := range ms.ReservedByOrder {
		sum += q
	}

	// ✅ availableStock 計算に使う reservedCount（DTO に渡す reservedCount もこれ）
	reservedCount = reservedStored
	if sum > reservedCount {
		reservedCount = sum
	}

	if reservedStored != sum {
		log.Printf(
			"[inventory_query][stock] WARN reservedCount mismatch stored=%d sum(ReservedByOrder)=%d orders=%d accumulation=%d -> use reservedCount=%d for display",
			reservedStored, sum, len(ms.ReservedByOrder), accumulation, reservedCount,
		)
	}

	available = accumulation - reservedCount
	if available < 0 {
		log.Printf(
			"[inventory_query][stock] WARN availableStock negative accumulation=%d reservedCount=%d (stored=%d sum=%d) -> clamp to 0",
			accumulation, reservedCount, reservedStored, sum,
		)
		available = 0
	}

	return accumulation, reservedCount, available
}

// ============================================================
// Patch field helpers (string / *string 揺れ吸収)
// ============================================================

func setStringFieldAny(target any, value string, names ...string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false
	}
	rv = rv.Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return false
	}

	for _, n := range names {
		f := rv.FieldByName(n)
		if !f.IsValid() || !f.CanSet() {
			continue
		}

		switch f.Kind() {
		case reflect.String:
			f.SetString(value)
			return true

		case reflect.Pointer:
			if f.Type().Elem().Kind() == reflect.String {
				s := value
				f.Set(reflect.ValueOf(&s))
				return true
			}
		}
	}

	return false
}
