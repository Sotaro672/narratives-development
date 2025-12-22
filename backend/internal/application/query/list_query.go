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

// ✅ inventoryId から modelId ごとの stock を count して返す（InventoryQuery.GetDetailByID を利用）
type InventoryDetailGetter interface {
	GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error)
}

// ✅ NEW: InventoryQuery の結果を ListQuery に渡すための最小ポート
// - companyId は ctx から取る（InventoryQuery と同じ）
type InventoryRowsLister interface {
	ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error)
}

// ============================================================
// DTO (query -> handler)
// ============================================================

// ListRowDTO はフロントの listManagement / listDetail に渡す 1 行分（最小 + 詳細補完用）。
// - 画面要件: title, productName, tokenName, assigneeName, status
// - 追加: inventoryId, assigneeId, pbId/tbId, brandId/brandName
type ListRowDTO struct {
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`

	// ✅ NEW: 一覧に title を返す（フロントの最左列用）
	Title string `json:"title"`

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

	// ✅ stock resolve用（inventoryId -> count）
	invGetter InventoryDetailGetter

	// ✅ NEW: inventory_query の結果で「この company が持つ inventoryId set」を作る
	invRows InventoryRowsLister
}

// 互換 ctor（brand/stock/invRows は出せないが既存 DI を壊さない）
func NewListQuery(lister ListLister, nameResolver *resolver.NameResolver) *ListQuery {
	return NewListQueryWithBrandAndInventoryGetters(lister, nameResolver, nil, nil, nil)
}

// 互換 ctor（brand は出せるが stock/invRows は出せない）
func NewListQueryWithBrandGetters(
	lister ListLister,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
) *ListQuery {
	return NewListQueryWithBrandAndInventoryGetters(lister, nameResolver, pbGetter, tbGetter, nil)
}

// 互換 ctor（brand + stock は出せるが invRows(company boundary) は出せない）
func NewListQueryWithBrandAndInventoryGetters(
	lister ListLister,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
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
		invGetter:    invGetter,
		invRows:      nil, // ✅ optional (backward compatible)
	}
}

// ✅ NEW: brand + stock + inventoryQuery(company boundary) まで注入できる ctor
func NewListQueryWithBrandInventoryAndInventoryRows(
	lister ListLister,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
	invRows InventoryRowsLister,
) *ListQuery {
	q := NewListQueryWithBrandAndInventoryGetters(lister, nameResolver, pbGetter, tbGetter, invGetter)
	q.invRows = invRows
	return q
}

// ============================================================
// ✅ company boundary helpers (inventory_query 経由)
// ============================================================

// allowedInventoryIDSetFromContext は inventory_query の結果を使って
// 「currentMember.companyId が持つ inventoryId (= {pbId}__{tbId}) set」を作る。
func (q *ListQuery) allowedInventoryIDSetFromContext(ctx context.Context) (map[string]struct{}, error) {
	if q == nil {
		return nil, errors.New("ListQuery is nil")
	}
	if q.invRows == nil {
		// ✅ 漏洩防止：company 境界が無いならエラーにする
		return nil, errors.New("ListQuery.invRows is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := q.invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, r := range rows {
		pbID := strings.TrimSpace(r.ProductBlueprintID)
		tbID := strings.TrimSpace(r.TokenBlueprintID)
		if pbID == "" || tbID == "" {
			continue
		}
		invID := pbID + "__" + tbID
		set[invID] = struct{}{}
	}
	return set, nil
}

func inventoryAllowed(set map[string]struct{}, inventoryID string) bool {
	if len(set) == 0 {
		return false
	}
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return false
	}
	_, ok := set[id]
	return ok
}

func normalizePage(p listdom.Page) listdom.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	return p
}

func totalPages(totalCount int, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if totalCount <= 0 {
		return 0
	}
	return (totalCount + perPage - 1) / perPage
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ListRows は lists 一覧を取得し、tokenName / assigneeName / productName / brandName を解決して返します。
// ✅ 追加: inventory_query の結果にある inventoryId のみに絞り込み（multi-tenant 保護）
//
// ⚠️ 重要:
// lists 側に companyId が無いので、DB側で絞れません。
// そのため「元の List のページング」→「後段で除外」だと、ページが空になる（他社listが混ざる）問題が起きます。
// ここでは “全ページ走査 → 許可分だけで再ページング” して、UIのページングが壊れないようにしています。
func (q *ListQuery) ListRows(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[ListRowDTO], error) {
	page = normalizePage(page)

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

	allowedSet, err := q.allowedInventoryIDSetFromContext(ctx)
	if err != nil {
		log.Printf("[ListQuery] ERROR company boundary (inventory_query) failed: %v", err)
		return listdom.PageResult[ListRowDTO]{}, err
	}
	if len(allowedSet) == 0 {
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	// request-scope cache（同じIDの多重解決を防ぐ）
	tokenNameCache := map[string]string{}     // tbID -> tokenName
	memberNameCache := map[string]string{}    // assigneeID -> name
	productNameCache := map[string]string{}   // pbID -> productName
	brandIDCachePB := map[string]string{}     // pbID -> brandID
	brandIDCacheTB := map[string]string{}     // tbID -> brandID
	brandNameByIDCache := map[string]string{} // brandID -> brandName

	// ✅ 全ページ走査して “allowed のみ” を集計（＝この company の list 全件）
	allowedAll := make([]ListRowDTO, 0, page.PerPage)

	// 走査上限（異常データ/DoS対策）。通常運用ではまず当たりません。
	const maxScanPages = 500

	srcPage := 1
	for {
		if srcPage > maxScanPages {
			log.Printf("[ListQuery] WARN scan page limit reached (max=%d). results may be truncated.", maxScanPages)
			break
		}

		pr, e := q.lister.List(ctx, filter, sort, listdom.Page{Number: srcPage, PerPage: page.PerPage})
		if e != nil {
			log.Printf("[ListQuery] ERROR lister.List failed (scan page=%d): %v", srcPage, e)
			return listdom.PageResult[ListRowDTO]{}, e
		}
		if pr.Items == nil {
			pr.Items = []listdom.List{}
		}

		for _, it := range pr.Items {
			id := strings.TrimSpace(it.ID)
			invID := strings.TrimSpace(it.InventoryID)

			// ✅ company boundary：inventory_query の set に無いものは除外（漏洩防止）
			if !inventoryAllowed(allowedSet, invID) {
				continue
			}

			assigneeID := strings.TrimSpace(it.AssigneeID)

			pbID, tbID, ok := parseInventoryIDStrict(invID)
			if !ok {
				// inventoryId が壊れている list は表示しない（安全側）
				continue
			}

			// ------------------------------------------------------
			// ✅ title（一覧の最左列で使う）
			// ------------------------------------------------------
			title := strings.TrimSpace(it.Title)

			// ------------------------------------------------------
			// productName (fallback: title)
			// ------------------------------------------------------
			productName := title
			if pbID != "" && q.nameResolver != nil {
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
			if tbID != "" && q.nameResolver != nil {
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

			if pbID != "" && q.pbGetter != nil {
				if cached, ok := brandIDCachePB[pbID]; ok {
					productBrandID = cached
				} else {
					pb, ee := q.pbGetter.GetByID(ctx, pbID)
					if ee == nil {
						productBrandID = strings.TrimSpace(pb.BrandID)
					}
					brandIDCachePB[pbID] = productBrandID
				}
			}
			if tbID != "" && q.tbGetter != nil {
				if cached, ok := brandIDCacheTB[tbID]; ok {
					tokenBrandID = cached
				} else {
					tb, ee := q.tbGetter.GetByID(ctx, tbID)
					if ee == nil {
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

			allowedAll = append(allowedAll, ListRowDTO{
				ID:          nonEmpty(id, "(missing id)"),
				InventoryID: invID,

				// ✅ NEW: title を返す
				Title: title,

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
			})
		}

		// ---- scan end condition ----
		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else {
			// TotalPages が返らない実装の場合は「1ページの件数 < perPage」で終了
			if len(pr.Items) < page.PerPage {
				break
			}
		}

		srcPage++
	}

	// ✅ allowedAll を “allowed側のページング” に変換
	totalCount := len(allowedAll)
	tp := totalPages(totalCount, page.PerPage)

	start := (page.Number - 1) * page.PerPage
	if start < 0 {
		start = 0
	}
	if start >= totalCount {
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: totalCount,
			TotalPages: tp,
		}, nil
	}
	end := minInt(start+page.PerPage, totalCount)

	return listdom.PageResult[ListRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

// ------------------------------------------------------------
// ✅ ListDetailDTO を作る（PriceRows に size/color/rgb を埋める）
// - 在庫(stock)は inventoryId から数える（lists に保存しない前提）
// ✅ 追加: inventory_query の結果にある inventoryId のみ許可（他社は NotFound）
// ------------------------------------------------------------
func (q *ListQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListQuery.BuildListDetailDTO: getter is nil (wire list repo to ListQuery)")
	}

	allowedSet, err := q.allowedInventoryIDSetFromContext(ctx)
	if err != nil {
		log.Printf("[ListQuery] ERROR company boundary (inventory_query) failed (detail): %v", err)
		return querydto.ListDetailDTO{}, err
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

	// ---- priceRows (modelId -> price) + stock(size/color/rgb) ----
	priceRows, totalStock, metaLog := q.buildDetailPriceRows(ctx, it, invID)
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

		ImageURLs: []string{},
		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	return dto, nil
}

// BuildCreateSeed は list新規作成画面に必要な情報を揃えて返します。
// - ここでは永続化(Create)は行いません（usecase に移譲）。
// - inventoryID は "{pbId}__{tbId}" のみ許可します（名揺れ吸収しない）。
// ✅ 追加: inventory_query の結果にある inventoryId のみ許可（他社は NotFound）
func (q *ListQuery) BuildCreateSeed(ctx context.Context, inventoryID string, modelIDs []string) (ListCreateSeedDTO, error) {
	allowedSet, err := q.allowedInventoryIDSetFromContext(ctx)
	if err != nil {
		log.Printf("[ListQuery] ERROR company boundary (inventory_query) failed (seed): %v", err)
		return ListCreateSeedDTO{}, err
	}

	inventoryID = strings.TrimSpace(inventoryID)
	if !inventoryAllowed(allowedSet, inventoryID) {
		return ListCreateSeedDTO{}, listdom.ErrNotFound
	}

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

	prices := map[string]int64{}
	for _, mid := range modelIDs {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		prices[mid] = 0
	}

	return ListCreateSeedDTO{
		InventoryID:        inventoryID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		ProductName:        productName,
		TokenName:          tokenName,
		Prices:             prices,
	}, nil
}

// ------------------------------------------------------------
// priceRows build (元のまま)
// ------------------------------------------------------------

func (q *ListQuery) buildDetailPriceRows(ctx context.Context, it listdom.List, inventoryID string) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rowsAny := extractPriceRowsFromList(it)
	if len(rowsAny) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

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

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID: modelID,
			Stock:   stock,
			Price:   pricePtr,
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
		" stockFromList=" + itoa(stockFromList)
	if invErrMsg != "" {
		meta += " " + invErrMsg
	}

	return out, total, meta
}

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

func extractPriceRowsFromList(it listdom.List) []any {
	rv := reflect.ValueOf(it)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}

	if f := rv.FieldByName("PriceRows"); f.IsValid() {
		if out := sliceToAny(f); len(out) > 0 {
			return out
		}
	}

	if f := rv.FieldByName("Prices"); f.IsValid() {
		if out := sliceToAny(f); len(out) > 0 {
			return out
		}
		if out := mapPricesToAnyRows(f); len(out) > 0 {
			return out
		}
	}

	return nil
}

func mapPricesToAnyRows(v reflect.Value) []any {
	v = deref(v)
	if !v.IsValid() || v.Kind() != reflect.Map {
		return nil
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil
	}

	out := make([]any, 0, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		val := iter.Value()

		modelID := ""
		if k.IsValid() && k.Kind() == reflect.String {
			modelID = strings.TrimSpace(k.String())
		}
		if modelID == "" {
			continue
		}

		priceInt := 0
		if n, ok := asInt(deref(val)); ok {
			priceInt = n
		}

		out = append(out, map[string]any{
			"ModelID": modelID,
			"Price":   priceInt,
		})
	}

	return out
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
	if !rv.IsValid() {
		return ""
	}

	if rv.Kind() == reflect.Struct {
		for _, fn := range fieldNames {
			f := rv.FieldByName(fn)
			f = deref(f)
			if f.IsValid() && f.Kind() == reflect.String {
				return f.String()
			}
		}
		return ""
	}

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() && mv.Kind() == reflect.String {
				return mv.String()
			}
		}
		return ""
	}

	return ""
}

func readIntField(v any, fieldNames ...string) int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() {
		return 0
	}

	if rv.Kind() == reflect.Struct {
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

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() {
				if n, ok := asInt(mv); ok {
					return n
				}
			}
		}
		return 0
	}

	return 0
}

func readIntPtrField(v any, fieldNames ...string) *int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() {
		return nil
	}

	if rv.Kind() == reflect.Struct {
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

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() {
				if n, ok := asInt(mv); ok {
					x := n
					return &x
				}
			}
		}
		return nil
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

func bool01(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

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
