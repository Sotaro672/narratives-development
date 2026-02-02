// backend/internal/application/query/console/list_detail_query.go
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
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// ============================================================
// Ports (read-only) - detail
// ============================================================

type ListGetter interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

type InventoryDetailGetter interface {
	GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error)
}

// ✅ NEW: ListImage を listID で取得できる port（任意）
// - 未DIでも画面が壊れないように nil を許容する
type ListImageLister interface {
	ListByListID(ctx context.Context, listID string) ([]listimgdom.ListImage, error)
}

// ============================================================
// ListDetailQuery (listDetail.tsx)
// ============================================================

type ListDetailQuery struct {
	getter       ListGetter
	nameResolver *resolver.NameResolver

	pbGetter ProductBlueprintGetter
	tbGetter TokenBlueprintGetter

	invGetter InventoryDetailGetter
	invRows   InventoryRowsLister

	// ✅ NEW: listImage bucket の画像（= ListImage 由来のURL）を返すため
	imgLister ListImageLister

	// ✅ NEW: productBlueprintPatch から modelRef(displayOrder) を取るため（任意）
	// inventory_query.go で定義されている productBlueprintPatchReader を使う
	pbPatchRepo productBlueprintPatchReader
}

func NewListDetailQuery(getter ListGetter, nameResolver *resolver.NameResolver) *ListDetailQuery {
	return NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, nil, nil, nil)
}

func NewListDetailQueryWithBrandGetters(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
) *ListDetailQuery {
	return NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, pbGetter, tbGetter, nil)
}

func NewListDetailQueryWithBrandAndInventoryGetters(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
) *ListDetailQuery {
	return &ListDetailQuery{
		getter:       getter,
		nameResolver: nameResolver,
		pbGetter:     pbGetter,
		tbGetter:     tbGetter,
		invGetter:    invGetter,
		invRows:      nil,
		imgLister:    nil, // optional
		pbPatchRepo:  nil, // optional
	}
}

func NewListDetailQueryWithBrandInventoryAndInventoryRows(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
	invRows InventoryRowsLister,
) *ListDetailQuery {
	q := NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, pbGetter, tbGetter, invGetter)
	q.invRows = invRows
	return q
}

// ✅ NEW: listImage も注入できる ctor（既存DIを壊さないため別名で追加）
func NewListDetailQueryWithBrandInventoryRowsAndImages(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
	invRows InventoryRowsLister,
	imgLister ListImageLister,
) *ListDetailQuery {
	q := NewListDetailQueryWithBrandInventoryAndInventoryRows(getter, nameResolver, pbGetter, tbGetter, invGetter, invRows)
	q.imgLister = imgLister
	return q
}

// ✅ NEW: productBlueprintPatch も注入できる ctor（既存DIを壊さないため別名で追加）
func NewListDetailQueryWithBrandInventoryRowsImagesAndPBPatch(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
	invRows InventoryRowsLister,
	imgLister ListImageLister,
	pbPatchRepo productBlueprintPatchReader,
) *ListDetailQuery {
	q := NewListDetailQueryWithBrandInventoryRowsAndImages(getter, nameResolver, pbGetter, tbGetter, invGetter, invRows, imgLister)
	q.pbPatchRepo = pbPatchRepo
	return q
}

func (q *ListDetailQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: getter is nil (wire list repo to ListDetailQuery)")
	}

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListDetailQuery] ERROR company boundary (inventory_query) failed (detail): %v", err)
		return querydto.ListDetailDTO{}, err
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: listID is empty")
	}

	it, err := q.getter.GetByID(ctx, listID)
	if err != nil {
		return querydto.ListDetailDTO{}, err
	}

	invID := strings.TrimSpace(it.InventoryID)
	if !inventoryAllowed(allowedSet, invID) {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	pbID, tbID, ok := parseInventoryIDStrict(invID)
	if !ok {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	// ---- names ----
	productName := ""
	tokenName := ""
	assigneeName := ""

	createdByName := ""

	// ✅ UpdatedBy is *string in domain
	updatedByID := ""
	if it.UpdatedBy != nil {
		updatedByID = strings.TrimSpace(*it.UpdatedBy)
	}
	updatedByName := ""

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
		// ✅ createdBy は string（ポインタではない）なので ResolveMemberName を直接使う
		if strings.TrimSpace(it.CreatedBy) != "" {
			createdByName = strings.TrimSpace(q.nameResolver.ResolveMemberName(ctx, it.CreatedBy))
		}
		// ✅ updatedBy は *string（nil可）なので Resolver の *string 用ヘルパを使う
		updatedByName = strings.TrimSpace(q.nameResolver.ResolveUpdatedByName(ctx, it.UpdatedBy))
	}

	if assigneeName == "" && strings.TrimSpace(it.AssigneeID) != "" {
		assigneeName = "未設定"
	}
	if createdByName == "" && strings.TrimSpace(it.CreatedBy) != "" {
		createdByName = "未設定"
	}
	if updatedByName == "" && updatedByID != "" {
		updatedByName = "未設定"
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

	// ---- priceRows + stock(size/color/rgb) ----
	// ✅ NEW: pbPatch から modelRef(displayOrder) を引いて、priceRows に displayOrder を載せる
	priceRows, totalStock, metaLog := q.buildDetailPriceRows(ctx, it, invID, pbID)
	if metaLog != "" {
		log.Printf("[ListDetailQuery] [modelMetadata] listID=%q %s", strings.TrimSpace(it.ID), metaLog)
	}

	dto := querydto.ListDetailDTO{
		ID:          strings.TrimSpace(it.ID),
		InventoryID: invID,

		Status:   strings.TrimSpace(string(it.Status)),
		Decision: strings.TrimSpace(string(it.Status)),

		Title:       strings.TrimSpace(it.Title),
		Description: strings.TrimSpace(it.Description),

		AssigneeID:   strings.TrimSpace(it.AssigneeID),
		AssigneeName: strings.TrimSpace(assigneeName),

		CreatedBy:     strings.TrimSpace(it.CreatedBy),
		CreatedByName: strings.TrimSpace(createdByName),
		CreatedAt:     it.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),

		// ✅ UpdatedBy/UpdatedByName（UpdatedBy は domain 側が *string）
		UpdatedBy:     updatedByID,
		UpdatedByName: strings.TrimSpace(updatedByName),
		UpdatedAt:     it.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),

		ImageID: strings.TrimSpace(it.ImageID),

		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandID:   productBrandID,
		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandID:   tokenBrandID,
		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		// ✅ listImage bucket の画像URL（ListImage から組み立てる）
		ImageURLs: []string{},

		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	// ✅ NEW: listImage bucket の画像を返せるようにする（未DIなら空のまま）
	dto.ImageURLs = q.buildListImageURLs(ctx, strings.TrimSpace(it.ID), strings.TrimSpace(it.ImageID))

	return dto, nil
}

// ============================================================
// ✅ productBlueprintPatch.ModelRefs から displayOrder を引く
// - Patch は値型なので nil 比較しない（err のみ見る）
// - 0 は未設定として nil に寄せる（list_create_query.go と揃える）
// ============================================================

func (q *ListDetailQuery) buildDisplayOrderByModelID(ctx context.Context, productBlueprintID string) map[string]*int {
	out := map[string]*int{}

	if q == nil || q.pbPatchRepo == nil {
		return out
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return out
	}

	patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID)
	if err != nil {
		return out
	}

	refs := extractModelRefsFromPBPatchAny(patch)
	if len(refs) == 0 {
		return out
	}

	seen := map[string]struct{}{}
	for _, r := range refs {
		mid := strings.TrimSpace(r.modelID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}

		// 互換: 0 は未設定として nil
		var ptr *int
		if r.displayOrder != 0 {
			x := r.displayOrder
			ptr = &x
		}
		out[mid] = ptr
	}

	return out
}

type modelRefAny struct {
	modelID      string
	displayOrder int
}

// reflection: patch.ModelRefs の型が何であっても拾えるようにする
// ✅ DisplayOrder が *int の場合も拾う（ここが重要）
func extractModelRefsFromPBPatchAny(patch any) []modelRefAny {
	rv := reflect.ValueOf(patch)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	f := rv.FieldByName("ModelRefs")
	if !f.IsValid() {
		f = rv.FieldByName("modelRefs")
	}
	if !f.IsValid() {
		return nil
	}

	// ModelRefs: *[]T or []T を許容
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil
		}
		f = f.Elem()
	}
	if f.Kind() != reflect.Slice {
		return nil
	}

	out := make([]modelRefAny, 0, f.Len())

	for i := 0; i < f.Len(); i++ {
		it := f.Index(i)
		if it.Kind() == reflect.Pointer {
			if it.IsNil() {
				continue
			}
			it = it.Elem()
		}
		if it.Kind() != reflect.Struct {
			continue
		}

		// ---- modelId ----
		mid := ""
		if mf := it.FieldByName("ModelID"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("ModelId"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("modelId"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		} else if mf := it.FieldByName("modelID"); mf.IsValid() && mf.Kind() == reflect.String {
			mid = strings.TrimSpace(mf.String())
		}

		if mid == "" {
			continue
		}

		// ---- displayOrder (int/uint/*int/*uint) ----
		ord := 0
		of := it.FieldByName("DisplayOrder")
		if !of.IsValid() {
			of = it.FieldByName("displayOrder")
		}
		if of.IsValid() && of.CanInterface() {
			// ポインタなら剥がす
			if of.Kind() == reflect.Pointer {
				if !of.IsNil() {
					ev := of.Elem()
					switch ev.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						ord = int(ev.Int())
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						ord = int(ev.Uint())
					}
				}
			} else {
				switch of.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					ord = int(of.Int())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					ord = int(of.Uint())
				}
			}
		}

		out = append(out, modelRefAny{
			modelID:      mid,
			displayOrder: ord,
		})
	}

	return out
}

func (q *ListDetailQuery) buildDetailPriceRows(ctx context.Context, it listdom.List, inventoryID string, productBlueprintID string) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rowsAny := extractPriceRowsFromList(it)
	if len(rowsAny) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

	// ✅ pbPatch から displayOrder map を用意（取れなければ空）
	displayOrderByModel := q.buildDisplayOrderByModelID(ctx, productBlueprintID)

	stockByModel := map[string]int{}
	attrByModel := map[string]resolver.ModelResolved{}
	invUsed := false
	invErrMsg := ""

	invID := strings.TrimSpace(inventoryID)
	if invID != "" && q != nil && q.invGetter != nil {
		if invDTO, err := q.invGetter.GetDetailByID(ctx, invID); err != nil {
			invErrMsg = "invErr=" + strings.TrimSpace(err.Error())
		} else if invDTO != nil {
			invUsed = true
			for _, r := range invDTO.Rows {
				mid := strings.TrimSpace(r.ModelID)
				if mid == "" {
					continue
				}
				stockByModel[mid] = r.Stock
				attrByModel[mid] = resolver.ModelResolved{
					Size:  strings.TrimSpace(r.Size),
					Color: strings.TrimSpace(r.Color),
					RGB:   r.RGB,
				}
			}
		}
	}

	modelResolvedCache := map[string]resolver.ModelResolved{}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rowsAny))
	total := 0

	resolvedNonEmpty := 0
	resolvedEmpty := 0
	stockFromInv := 0
	stockFromList := 0

	displayOrderHit := 0

	for _, r := range rowsAny {
		modelID := strings.TrimSpace(readStringField(r, "ModelID", "ModelId", "ID", "Id"))
		if modelID == "" {
			continue
		}

		pricePtr := readIntPtrField(r, "Price", "price")

		stock := 0
		if invUsed {
			if v, ok := stockByModel[modelID]; ok {
				stock = v
				stockFromInv++
			} else {
				stock = 0
				stockFromInv++
			}
		} else {
			stock = readIntField(r, "Stock", "stock")
			stockFromList++
		}

		// ✅ displayOrder（pbPatch.ModelRefs）
		var dispPtr *int
		if displayOrderByModel != nil {
			if v, ok := displayOrderByModel[modelID]; ok {
				dispPtr = v
				if v != nil {
					displayOrderHit++
				}
			}
		}

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID:      modelID,
			DisplayOrder: dispPtr,
			Stock:        stock,
			Price:        pricePtr,
			Size:         "",
			Color:        "",
			RGB:          nil,
		}

		if mr, ok := attrByModel[modelID]; ok {
			dtoRow.Size = strings.TrimSpace(mr.Size)
			dtoRow.Color = strings.TrimSpace(mr.Color)
			dtoRow.RGB = mr.RGB
		} else {
			mr := q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
			dtoRow.Size = strings.TrimSpace(mr.Size)
			dtoRow.Color = strings.TrimSpace(mr.Color)
			dtoRow.RGB = mr.RGB
		}

		if dtoRow.Size != "" || dtoRow.Color != "" || dtoRow.RGB != nil {
			resolvedNonEmpty++
		} else {
			resolvedEmpty++
		}

		out = append(out, dtoRow)
		total += stock
	}

	meta := "rows=" + itoa(len(out)) +
		" resolvedNonEmpty=" + itoa(resolvedNonEmpty) +
		" resolvedEmpty=" + itoa(resolvedEmpty) +
		" invUsed=" + bool01(invUsed) +
		" stockFromInv=" + itoa(stockFromInv) +
		" stockFromList=" + itoa(stockFromList) +
		" displayOrderHit=" + itoa(displayOrderHit)
	if invErrMsg != "" {
		meta += " " + invErrMsg
	}

	return out, total, meta
}

func (q *ListDetailQuery) resolveModelResolvedCached(
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

// ============================================================
// ✅ listImage bucket の画像URLを組み立てる
// ============================================================

type listImageURLRow struct {
	id        string
	url       string
	order     int
	createdAt time.Time
}

func (q *ListDetailQuery) buildListImageURLs(ctx context.Context, listID string, primaryImageID string) []string {
	if q == nil || q.imgLister == nil {
		return []string{}
	}

	lid := strings.TrimSpace(listID)
	if lid == "" {
		return []string{}
	}

	items, err := q.imgLister.ListByListID(ctx, lid)
	if err != nil || len(items) == 0 {
		return []string{}
	}

	rows := make([]listImageURLRow, 0, len(items))

	for _, it := range items {
		// ID
		id := strings.TrimSpace(readStringFieldAny(it, "ID", "Id", "ImageID", "ImageId"))
		// URL fields (optional)
		u := strings.TrimSpace(readStringFieldAny(it, "PublicURL", "PublicUrl", "URL", "Url", "SignedURL", "SignedUrl"))
		// bucket/objectPath fields
		b := strings.TrimSpace(readStringFieldAny(it, "Bucket", "bucket"))
		op := strings.TrimLeft(strings.TrimSpace(readStringFieldAny(it, "ObjectPath", "objectPath", "Path", "path")), "/")

		if u == "" && op != "" {
			if b == "" {
				// usecase の契約に合わせて、bucket が空なら DefaultBucket を採用
				b = strings.TrimSpace(listimgdom.DefaultBucket)
			}
			if b != "" {
				u = "https://storage.googleapis.com/" + b + "/" + op
			}
		}

		if u == "" {
			continue
		}

		order := readIntFieldAny(it, "DisplayOrder", "displayOrder", "Order", "order")
		ca := readTimeFieldAny(it, "CreatedAt", "createdAt")

		rows = append(rows, listImageURLRow{
			id:        id,
			url:       u,
			order:     order,
			createdAt: ca,
		})
	}

	if len(rows) == 0 {
		return []string{}
	}

	// sort: displayOrder asc -> createdAt asc -> id
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].order != rows[j].order {
			return rows[i].order < rows[j].order
		}
		if !rows[i].createdAt.Equal(rows[j].createdAt) {
			// zero time は後ろへ
			if rows[i].createdAt.IsZero() && !rows[j].createdAt.IsZero() {
				return false
			}
			if !rows[i].createdAt.IsZero() && rows[j].createdAt.IsZero() {
				return true
			}
			return rows[i].createdAt.Before(rows[j].createdAt)
		}
		return rows[i].id < rows[j].id
	})

	// dedupe by url (keep order)
	seen := map[string]bool{}
	out := make([]string, 0, len(rows))
	primaryURL := ""
	wantPrimary := strings.TrimSpace(primaryImageID)

	for _, r := range rows {
		u := strings.TrimSpace(r.url)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true

		if wantPrimary != "" && strings.TrimSpace(r.id) == wantPrimary && primaryURL == "" {
			primaryURL = u
			continue
		}
		out = append(out, u)
	}

	// primary を先頭に
	if primaryURL != "" {
		return append([]string{primaryURL}, out...)
	}

	return out
}

// --- reflection helpers (ListImage のフィールド名差分に強くする) ---

func readStringFieldAny(v any, names ...string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range names {
		f := rv.FieldByName(name)
		if f.IsValid() && f.CanInterface() && f.Kind() == reflect.String {
			s := strings.TrimSpace(f.Interface().(string))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func readIntFieldAny(v any, names ...string) int {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return 0
	}

	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(f.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(f.Uint())
		}
	}
	return 0
}

func readTimeFieldAny(v any, names ...string) time.Time {
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

	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		if t, ok := f.Interface().(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
