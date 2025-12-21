// backend/internal/application/query/list_query.go
package query

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strings"

	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Ports (read-only)
// ============================================================

// ListLister は lists の一覧取得（ページング付き）を行う最小契約です。
type ListLister interface {
	List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error)
}

// ListGetter は list 1件取得（detail DTO の組み立て用）
type ListGetter interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// ✅ ProductBlueprint/TokenBlueprint から brandId を引くために GetByID を注入する
type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (pbpdom.ProductBlueprint, error)
}

type TokenBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error)
}

// ============================================================
// DTO (query -> handler)
// ============================================================

// ListRowDTO はフロントの listManagement / listDetail に渡す 1 行分（最小 + 詳細補完用）。
// - 画面要件: productName, tokenName, assigneeName, status
// - 追加: inventoryId, assigneeId, pbId/tbId, brandId/brandName
type ListRowDTO struct {
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`

	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	ProductName      string `json:"productName"`
	ProductBrandID   string `json:"productBrandId"`
	ProductBrandName string `json:"productBrandName"`

	TokenName      string `json:"tokenName"`
	TokenBrandID   string `json:"tokenBrandId"`
	TokenBrandName string `json:"tokenBrandName"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Status string `json:"status"`
}

// ListCreateSeedDTO は「list新規作成画面」に必要な材料のみを返す DTO。
type ListCreateSeedDTO struct {
	InventoryID        string           `json:"inventoryId"`
	ProductBlueprintID string           `json:"productBlueprintId"`
	TokenBlueprintID   string           `json:"tokenBlueprintId"`
	ProductName        string           `json:"productName"`
	TokenName          string           `json:"tokenName"`
	Prices             map[string]int64 `json:"prices"` // modelId -> price value
}

// ============================================================
// ListQuery
// ============================================================

type ListQuery struct {
	lister       ListLister
	getter       ListGetter
	nameResolver *resolver.NameResolver

	// brand resolve用
	pbGetter ProductBlueprintGetter
	tbGetter TokenBlueprintGetter
}

// 互換 ctor（brand は出せないが既存 DI を壊さない）
func NewListQuery(lister ListLister, nameResolver *resolver.NameResolver) *ListQuery {
	return NewListQueryWithBrandGetters(lister, nameResolver, nil, nil)
}

// ✅ NEW: brandId を引く repo を注入できる ctor
func NewListQueryWithBrandGetters(
	lister ListLister,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
) *ListQuery {
	// ✅ lister が GetByID を実装していれば getter にも流用する（DI変更不要）
	var lg ListGetter
	if g, ok := any(lister).(ListGetter); ok {
		lg = g
	}

	return &ListQuery{
		lister:       lister,
		getter:       lg,
		nameResolver: nameResolver,
		pbGetter:     pbGetter,
		tbGetter:     tbGetter,
	}
}

// ListRows は lists 一覧を取得し、tokenName / assigneeName / productName / brandName を解決して返します。
func (q *ListQuery) ListRows(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[ListRowDTO], error) {
	// nil ガード
	if q == nil || q.lister == nil {
		log.Printf("[ListQuery] WARN ListRows called but q or lister is nil (q=%v listerNil=%v)", q != nil, q == nil || q.lister == nil)
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	pr, err := q.lister.List(ctx, filter, sort, page)
	if err != nil {
		log.Printf("[ListQuery] ERROR lister.List failed: %v", err)
		return listdom.PageResult[ListRowDTO]{}, err
	}
	if pr.Items == nil {
		pr.Items = []listdom.List{}
	}

	// request-scope cache（同じIDの多重解決を防ぐ）
	tokenNameCache := map[string]string{}     // tbID -> tokenName
	memberNameCache := map[string]string{}    // assigneeID -> name
	productNameCache := map[string]string{}   // pbID -> productName
	brandIDCachePB := map[string]string{}     // pbID -> brandID
	brandIDCacheTB := map[string]string{}     // tbID -> brandID
	brandNameByIDCache := map[string]string{} // brandID -> brandName

	out := make([]ListRowDTO, 0, len(pr.Items))

	for _, it := range pr.Items {
		id := strings.TrimSpace(it.ID)
		invID := strings.TrimSpace(it.InventoryID)
		assigneeID := strings.TrimSpace(it.AssigneeID)

		pbID, tbID, ok := parseInventoryIDStrict(invID)

		// ------------------------------------------------------
		// productName (fallback: title)
		// ------------------------------------------------------
		productName := strings.TrimSpace(it.Title)
		if ok && pbID != "" && q.nameResolver != nil {
			if cached, ok := productNameCache[pbID]; ok {
				if cached != "" {
					productName = cached
				}
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
				productNameCache[pbID] = resolved
				if resolved != "" {
					productName = resolved
				}
			}
		}

		// ------------------------------------------------------
		// tokenName
		// ------------------------------------------------------
		tokenName := ""
		if ok && tbID != "" && q.nameResolver != nil {
			if cached, ok := tokenNameCache[tbID]; ok {
				tokenName = cached
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
				tokenNameCache[tbID] = resolved
				tokenName = resolved
			}
		}

		// ------------------------------------------------------
		// assigneeName
		// ------------------------------------------------------
		assigneeName := ""
		if assigneeID != "" && q.nameResolver != nil {
			if cached, ok := memberNameCache[assigneeID]; ok {
				assigneeName = cached
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, assigneeID))
				memberNameCache[assigneeID] = resolved
				assigneeName = resolved
			}
		}
		if assigneeName == "" {
			assigneeName = "未設定"
		}

		// ------------------------------------------------------
		// brandId (pb/tb -> brandId) then brandName (brandId -> name)
		// ------------------------------------------------------
		productBrandID := ""
		tokenBrandID := ""

		if ok && pbID != "" && q.pbGetter != nil {
			if cached, ok := brandIDCachePB[pbID]; ok {
				productBrandID = cached
			} else {
				pb, e := q.pbGetter.GetByID(ctx, pbID)
				if e == nil {
					productBrandID = strings.TrimSpace(pb.BrandID)
				}
				brandIDCachePB[pbID] = productBrandID
			}
		}
		if ok && tbID != "" && q.tbGetter != nil {
			if cached, ok := brandIDCacheTB[tbID]; ok {
				tokenBrandID = cached
			} else {
				tb, e := q.tbGetter.GetByID(ctx, tbID)
				if e == nil {
					tokenBrandID = strings.TrimSpace(tb.BrandID)
				}
				brandIDCacheTB[tbID] = tokenBrandID
			}
		}

		productBrandName := ""
		tokenBrandName := ""

		if q.nameResolver != nil {
			if productBrandID != "" {
				if cached, ok := brandNameByIDCache[productBrandID]; ok {
					productBrandName = cached
				} else {
					resolved := strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, productBrandID))
					brandNameByIDCache[productBrandID] = resolved
					productBrandName = resolved
				}
			}
			if tokenBrandID != "" {
				if cached, ok := brandNameByIDCache[tokenBrandID]; ok {
					tokenBrandName = cached
				} else {
					resolved := strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, tokenBrandID))
					brandNameByIDCache[tokenBrandID] = resolved
					tokenBrandName = resolved
				}
			}
		}

		row := ListRowDTO{
			ID:          nonEmpty(id, "(missing id)"),
			InventoryID: invID,

			ProductBlueprintID: pbID,
			TokenBlueprintID:   tbID,

			ProductName:      strings.TrimSpace(productName),
			ProductBrandID:   productBrandID,
			ProductBrandName: strings.TrimSpace(productBrandName),

			TokenName:      strings.TrimSpace(tokenName),
			TokenBrandID:   tokenBrandID,
			TokenBrandName: strings.TrimSpace(tokenBrandName),

			AssigneeID:   assigneeID,
			AssigneeName: assigneeName,

			Status: strings.TrimSpace(string(it.Status)),
		}

		out = append(out, row)
	}

	return listdom.PageResult[ListRowDTO]{
		Items:      out,
		Page:       pr.Page,
		PerPage:    pr.PerPage,
		TotalCount: pr.TotalCount,
		TotalPages: pr.TotalPages,
	}, nil
}

// ------------------------------------------------------------
// ✅ NEW: ListDetailDTO を作る（PriceRows に size/color/rgb を埋める）
// - `dto.ListDetailDTO.PriceRows[].Size/Color/RGB` へ組み込むのが期待値
// ------------------------------------------------------------
func (q *ListQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListQuery.BuildListDetailDTO: getter is nil (wire list repo to ListQuery)")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return querydto.ListDetailDTO{}, errors.New("ListQuery.BuildListDetailDTO: listID is empty")
	}

	it, err := q.getter.GetByID(ctx, listID)
	if err != nil {
		return querydto.ListDetailDTO{}, err
	}

	invID := strings.TrimSpace(it.InventoryID)
	pbID, tbID, _ := parseInventoryIDStrict(invID)

	// ---- names ----
	productName := ""
	tokenName := ""
	assigneeName := ""

	if q.nameResolver != nil {
		if pbID != "" {
			productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
		}
		if tbID != "" {
			tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
		}
		if strings.TrimSpace(it.AssigneeID) != "" {
			assigneeName = strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, it.AssigneeID))
		}
	}
	if assigneeName == "" && strings.TrimSpace(it.AssigneeID) != "" {
		assigneeName = "未設定"
	}

	// ---- brand ----
	productBrandID := ""
	tokenBrandID := ""
	if pbID != "" && q.pbGetter != nil {
		pb, e := q.pbGetter.GetByID(ctx, pbID)
		if e == nil {
			productBrandID = strings.TrimSpace(pb.BrandID)
		}
	}
	if tbID != "" && q.tbGetter != nil {
		tb, e := q.tbGetter.GetByID(ctx, tbID)
		if e == nil {
			tokenBrandID = strings.TrimSpace(tb.BrandID)
		}
	}

	productBrandName := ""
	tokenBrandName := ""
	if q.nameResolver != nil {
		if productBrandID != "" {
			productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, productBrandID))
		}
		if tokenBrandID != "" {
			tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, tokenBrandID))
		}
	}

	// ---- priceRows (modelId -> size/color/rgb) ----
	priceRows, totalStock, metaLog := q.buildDetailPriceRows(ctx, it)

	// ✅ model metadata の取得状況が分かる最小ログ（spam しない）
	if metaLog != "" {
		log.Printf("[ListQuery] [modelMetadata] listID=%q %s", strings.TrimSpace(it.ID), metaLog)
	}

	dto := querydto.ListDetailDTO{
		ID:          strings.TrimSpace(it.ID),
		InventoryID: invID,

		Status:   strings.TrimSpace(string(it.Status)),
		Decision: strings.TrimSpace(string(it.Status)), // decision が別なら handler/usecase 側で上書きする想定

		Title:       strings.TrimSpace(it.Title),
		Description: strings.TrimSpace(it.Description),

		AssigneeID:   strings.TrimSpace(it.AssigneeID),
		AssigneeName: strings.TrimSpace(assigneeName),

		CreatedBy: strings.TrimSpace(it.CreatedBy),
		CreatedAt: it.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: it.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),

		ImageID: strings.TrimSpace(it.ImageID),

		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandID:   productBrandID,
		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandID:   tokenBrandID,
		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		ImageURLs: []string{}, // 画像URLは別層（aggregate等）で埋める想定
		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	return dto, nil
}

// buildDetailPriceRows は List の価格行から ListDetailPriceRowDTO を作り、
// modelId -> size/color/rgb を NameResolver.ResolveModelResolved で埋めます。
func (q *ListQuery) buildDetailPriceRows(ctx context.Context, it listdom.List) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rowsAny := extractPriceRowsFromList(it)
	if len(rowsAny) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

	// request-scope cache
	modelResolvedCache := map[string]resolver.ModelResolved{}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rowsAny))
	total := 0

	resolvedNonEmpty := 0
	resolvedEmpty := 0

	for _, r := range rowsAny {
		// ✅ Firestore の保存を正: Go field は ModelID / Stock / Price を想定（名揺れ吸収はしない）
		modelID := strings.TrimSpace(readStringField(r, "ModelID"))
		if modelID == "" {
			continue
		}
		stock := readIntField(r, "Stock")
		pricePtr := readIntPtrField(r, "Price")

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID: modelID,
			Stock:   stock,
			Price:   pricePtr,
		}

		// ✅ ここで組み込む（期待値）：Size/Color/RGB
		mr := q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
		dtoRow.Size = strings.TrimSpace(mr.Size)
		dtoRow.Color = strings.TrimSpace(mr.Color)
		dtoRow.RGB = mr.RGB

		if dtoRow.Size != "" || dtoRow.Color != "" || dtoRow.RGB != nil {
			resolvedNonEmpty++
		} else {
			resolvedEmpty++
		}

		out = append(out, dtoRow)
		total += stock
	}

	meta := "rows=" + itoa(len(out)) + " resolvedNonEmpty=" + itoa(resolvedNonEmpty) + " resolvedEmpty=" + itoa(resolvedEmpty)
	return out, total, meta
}

// resolveModelResolvedCached は modelVariationId -> size/color/rgb を解決し、cache する。
func (q *ListQuery) resolveModelResolvedCached(
	ctx context.Context,
	variationID string,
	cache map[string]resolver.ModelResolved,
) resolver.ModelResolved {
	id := strings.TrimSpace(variationID)
	if id == "" {
		return resolver.ModelResolved{}
	}
	if cache != nil {
		if v, ok := cache[id]; ok {
			return v
		}
	}

	var v resolver.ModelResolved
	if q != nil && q.nameResolver != nil {
		v = q.nameResolver.ResolveModelResolved(ctx, id)
	}

	if cache != nil {
		cache[id] = v
	}
	return v
}

// BuildCreateSeed は list新規作成画面に必要な情報を揃えて返します。
// - ここでは永続化(Create)は行いません（usecase に移譲）。
// - inventoryID は "{pbId}__{tbId}" のみ許可します（名揺れ吸収しない）。
func (q *ListQuery) BuildCreateSeed(ctx context.Context, inventoryID string, modelIDs []string) (ListCreateSeedDTO, error) {
	inventoryID = strings.TrimSpace(inventoryID)

	pbID, tbID, ok := parseInventoryIDStrict(inventoryID)
	if !ok {
		log.Printf("[ListQuery] BuildCreateSeed invalid inventoryID (expected {pbId}__{tbId}) inventoryID=%q", inventoryID)
		return ListCreateSeedDTO{}, listdom.ErrInvalidInventoryID
	}

	productName := ""
	tokenName := ""

	if q != nil && q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	// prices: modelId -> 0 (初期値)
	prices := map[string]int64{}
	for _, mid := range modelIDs {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		prices[mid] = 0
	}

	out := ListCreateSeedDTO{
		InventoryID:        inventoryID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		ProductName:        productName,
		TokenName:          tokenName,
		Prices:             prices,
	}

	return out, nil
}

// ============================================================
// helpers
// ============================================================

func nonEmpty(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func parseInventoryIDStrict(invID string) (pbID string, tbID string, ok bool) {
	invID = strings.TrimSpace(invID)
	if invID == "" {
		return "", "", false
	}
	if !strings.Contains(invID, "__") {
		return "", "", false
	}
	parts := strings.Split(invID, "__")
	if len(parts) < 2 {
		return "", "", false
	}
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

// ------------------------------------------------------------
// PriceRows extractor (reflect)
// ------------------------------------------------------------

// extractPriceRowsFromList は listdom.List から price row スライスを拾う。
// フィールドは最小限のみ許容:
// - PriceRows / Prices
func extractPriceRowsFromList(it listdom.List) []any {
	rv := reflect.ValueOf(it)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}

	if f := rv.FieldByName("PriceRows"); f.IsValid() {
		return sliceToAny(f)
	}
	if f := rv.FieldByName("Prices"); f.IsValid() {
		return sliceToAny(f)
	}
	return nil
}

func sliceToAny(v reflect.Value) []any {
	v = deref(v)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return nil
	}
	out := make([]any, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		out = append(out, v.Index(i).Interface())
	}
	return out
}

func readStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return ""
	}
	for _, fn := range fieldNames {
		f := rv.FieldByName(fn)
		f = deref(f)
		if f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

func readIntField(v any, fieldNames ...string) int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return 0
	}
	for _, fn := range fieldNames {
		f := rv.FieldByName(fn)
		f = deref(f)
		if f.IsValid() {
			if n, ok := asInt(f); ok {
				return n
			}
		}
	}
	return 0
}

func readIntPtrField(v any, fieldNames ...string) *int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}
	for _, fn := range fieldNames {
		f := rv.FieldByName(fn)
		f = deref(f)
		if f.IsValid() {
			if n, ok := asInt(f); ok {
				x := n
				return &x
			}
		}
	}
	return nil
}

func deref(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func asInt(v reflect.Value) (int, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int(v.Float()), true
	default:
		return 0, false
	}
}

// itoa: strconv を増やさないための最小実装
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
