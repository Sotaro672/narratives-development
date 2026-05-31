// backend/internal/application/query/console/list_detail_query.go
//
// 機能: ListDetailQuery の公開API（DTO組み立て）
//
// 責任:
// - DI 済み依存（ports）を保持する
// - listID を入力に listDetail.tsx 用の ListDetailDTO を生成する
// - company boundary / inventory boundary を確認し、表示可能データのみ返す
// - ListDetailDTO の priceRows / stock / model attributes を生成する
// - ProductBlueprint.GetByID から displayOrder を抽出する
//
// Firebase Storage 移行後:
// - backend は GCS signed URL / GCS object を扱わない
// - list image record は domain/list.ListImage として扱う
// - 画像URLは list image record の URL、つまり Firebase Storage getDownloadURL() を使う
package query

import (
	"context"
	"errors"
	"log"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// Ports (read-only) - list detail
// ============================================================

type ListGetter interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

type InventoryDetailGetter interface {
	GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error)
}

// ListImage を listID で取得できる port（任意）
//
// Firebase Storage 移行後:
// - domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
type ListImageLister interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
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

	// list image record 由来の Firebase Storage downloadURL を返すため（任意）
	imgLister ListImageLister
}

// ============================================================
// SINGLE ENTRYPOINT
// - ここだけを入口にして、全依存を配線する
// - optional は nil を許容する（既存DIを壊さない）
// ============================================================

type NewListDetailQueryParams struct {
	Getter       ListGetter
	NameResolver *resolver.NameResolver

	PBGetter ProductBlueprintGetter
	TBGetter TokenBlueprintGetter

	InvGetter InventoryDetailGetter
	InvRows   InventoryRowsLister

	ImgLister ListImageLister
}

func NewListDetailQuery(p NewListDetailQueryParams) *ListDetailQuery {
	return &ListDetailQuery{
		getter:       p.Getter,
		nameResolver: p.NameResolver,
		pbGetter:     p.PBGetter,
		tbGetter:     p.TBGetter,
		invGetter:    p.InvGetter,
		invRows:      p.InvRows,
		imgLister:    p.ImgLister,
	}
}

// ============================================================
// Query
// ============================================================

func (q *ListDetailQuery) BuildListDetailDTO(
	ctx context.Context,
	listID string,
) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: getter is nil (wire list repo to ListDetailQuery)")
	}

	allowedSet, err := AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListDetailQuery] ERROR company boundary (inventory_query) failed (detail): %v", err)
		return querydto.ListDetailDTO{}, err
	}

	if listID == "" {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: listID is empty")
	}

	it, err := q.getter.GetByID(ctx, listID)
	if err != nil {
		return querydto.ListDetailDTO{}, err
	}

	invID := it.InventoryID
	if !InventoryAllowed(allowedSet, invID) {
		return querydto.ListDetailDTO{}, listdom.ErrListImageNotFound
	}

	pbID, tbID, ok := ParseInventoryIDStrict(invID)
	if !ok {
		return querydto.ListDetailDTO{}, listdom.ErrListImageNotFound
	}

	// ---- names ----
	productName := ""
	tokenName := ""
	assigneeName := ""
	createdByName := ""

	updatedByID := ""
	if it.UpdatedBy != nil {
		updatedByID = *it.UpdatedBy
	}
	updatedByName := ""

	if q.nameResolver != nil {
		if pbID != "" {
			productName = q.nameResolver.ResolveProductName(ctx, pbID)
		}
		if tbID != "" {
			tokenName = q.nameResolver.ResolveTokenName(ctx, tbID)
		}
		if it.AssigneeID != "" {
			assigneeName = q.nameResolver.ResolveAssigneeName(ctx, it.AssigneeID)
		}
		if it.CreatedBy != "" {
			createdByName = q.nameResolver.ResolveMemberName(ctx, it.CreatedBy)
		}
		if updatedByID != "" {
			updatedByName = q.nameResolver.ResolveUpdatedByName(ctx, it.UpdatedBy)
		}
	}

	if assigneeName == "" && it.AssigneeID != "" {
		assigneeName = "未設定"
	}
	if createdByName == "" && it.CreatedBy != "" {
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
			productBrandID = pb.BrandID
		}
	}

	if tbID != "" && q.tbGetter != nil {
		tb, e := q.tbGetter.GetByID(ctx, tbID)
		if e == nil && tb != nil {
			tokenBrandID = tb.BrandID
		}
	}

	productBrandName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		if productBrandID != "" {
			productBrandName = q.nameResolver.ResolveBrandName(ctx, productBrandID)
		}
		if tokenBrandID != "" {
			tokenBrandName = q.nameResolver.ResolveBrandName(ctx, tokenBrandID)
		}
	}

	// ---- timestamps ----
	createdAt := ""
	if !it.CreatedAt.IsZero() {
		createdAt = it.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if it.UpdatedAt != nil && !it.UpdatedAt.IsZero() {
		updatedAt = it.UpdatedAt.Format(time.RFC3339)
	}

	// ---- images ----
	//
	// Firebase Storage 移行後:
	// - List.ImageID は primary image record の docID
	// - 実際の画像URLは /lists/{listId}/images/{imageId} record の URL
	// - URL は Firebase Storage getDownloadURL()
	// - frontend の削除差分計算用に id/url/displayOrder を返す
	images := q.buildListImages(ctx, it.ID, it.ImageID)
	imageURLs := buildImageURLsFromImages(images)

	// ---- price rows / stock ----
	//
	// list の price rows を DTO に復元する。
	// stock は inventory detail が取れる場合は inventory を優先し、
	// なければ stock=0 とする。
	priceRows, totalStock, priceRowsMeta := q.buildDetailPriceRows(ctx, it, invID, pbID)
	if priceRowsMeta != "" {
		log.Printf("[ListDetailQuery] priceRows listID=%q %s", listID, priceRowsMeta)
	}

	dto := querydto.ListDetailDTO{
		ID:          it.ID,
		InventoryID: invID,

		Status:   string(it.Status),
		Decision: string(it.Status), // DTO互換維持

		Title:       it.Title,
		Description: it.Description,

		AssigneeID:   it.AssigneeID,
		AssigneeName: assigneeName,

		CreatedBy:     it.CreatedBy,
		CreatedByName: createdByName,
		CreatedAt:     createdAt,

		UpdatedBy:     updatedByID,
		UpdatedByName: updatedByName,
		UpdatedAt:     updatedAt,

		// Policy:
		// - ImageID は URL ではなく primary imageId
		ImageID: it.ImageID,

		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandID:   productBrandID,
		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandID:   tokenBrandID,
		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		ImageURLs: imageURLs,
		Images:    images,

		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	return dto, nil
}

// ============================================================
// Image helpers
// ============================================================

// buildListImages returns image records for the list detail DTO.
//
// primaryImageID:
// - List.ImageID に保存されている primary image record docID
// - 存在する場合、その画像を先頭に並べる
func (q *ListDetailQuery) buildListImages(
	ctx context.Context,
	listID string,
	primaryImageID string,
) []querydto.ListImageDTO {
	if q == nil || q.imgLister == nil || listID == "" {
		return []querydto.ListImageDTO{}
	}

	items, err := q.imgLister.ListByListID(ctx, listID)
	if err != nil {
		log.Printf("[ListDetailQuery] WARN list images failed listID=%q err=%v", listID, err)
		return []querydto.ListImageDTO{}
	}

	if len(items) == 0 {
		return []querydto.ListImageDTO{}
	}

	ordered := make([]listdom.ListImage, 0, len(items))
	used := make(map[string]struct{}, len(items))

	if primaryImageID != "" {
		for _, img := range items {
			if img.ID == primaryImageID {
				ordered = append(ordered, img)
				used[img.ID] = struct{}{}
				break
			}
		}
	}

	for _, img := range items {
		if img.ID != "" {
			if _, ok := used[img.ID]; ok {
				continue
			}
		}

		ordered = append(ordered, img)
		if img.ID != "" {
			used[img.ID] = struct{}{}
		}
	}

	out := make([]querydto.ListImageDTO, 0, len(ordered))

	for index, img := range ordered {
		if img.URL == "" {
			continue
		}

		out = append(out, querydto.ListImageDTO{
			ID:           img.ID,
			ImageID:      img.ID,
			URL:          img.URL,
			DisplayOrder: index,
		})
	}

	return out
}

func buildImageURLsFromImages(images []querydto.ListImageDTO) []string {
	if len(images) == 0 {
		return []string{}
	}

	urls := make([]string, 0, len(images))
	seen := map[string]struct{}{}

	for _, img := range images {
		if img.URL == "" {
			continue
		}
		if _, ok := seen[img.URL]; ok {
			continue
		}

		seen[img.URL] = struct{}{}
		urls = append(urls, img.URL)
	}

	return urls
}

// ============================================================
// Price row helpers
// ============================================================

// buildDetailPriceRows builds ListDetailDTO price rows.
//
// 責任:
// - listdom.List の価格行を抽出し、DTO(ListDetailPriceRowDTO)へ変換する
// - 在庫情報は InventoryDetailGetter から優先的に取得し、なければ stock=0 とする
// - displayOrder は ProductBlueprint.GetByID(ModelRefs) から付与する（取得できない場合は nil）
// - model 情報は resolver.ModelResolved を使って解決する
//   - apparel: kind / modelNumber / size / color / rgb
//   - alcohol: kind / modelNumber / volumeValue / volumeUnit
func (q *ListDetailQuery) buildDetailPriceRows(
	ctx context.Context,
	it listdom.List,
	inventoryID string,
	productBlueprintID string,
) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rows := it.Prices
	if len(rows) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

	displayOrderByModel := q.buildDisplayOrderByModelID(ctx, productBlueprintID)

	stockByModel := map[string]int{}
	attrByModel := map[string]resolver.ModelResolved{}
	invUsed := false
	invErrMsg := ""

	invID := inventoryID
	if invID != "" && q != nil && q.invGetter != nil {
		if invDTO, err := q.invGetter.GetDetailByID(ctx, invID); err != nil {
			invErrMsg = "invErr=" + err.Error()
		} else if invDTO != nil {
			invUsed = true
			for _, r := range invDTO.Rows {
				mid := r.ModelID
				if mid == "" {
					continue
				}

				stockByModel[mid] = r.Stock
				attrByModel[mid] = resolver.ModelResolved{
					Kind:        r.Kind,
					ModelNumber: r.ModelNumber,

					Size:  r.Size,
					Color: r.Color,
					RGB:   r.RGB,

					VolumeValue: r.VolumeValue,
					VolumeUnit:  r.VolumeUnit,
				}
			}
		}
	}

	modelResolvedCache := map[string]resolver.ModelResolved{}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rows))
	total := 0

	resolvedNonEmpty := 0
	resolvedEmpty := 0
	stockFromInv := 0
	stockFromList := 0
	displayOrderHit := 0

	for _, r := range rows {
		modelID := r.ModelID
		if modelID == "" {
			continue
		}

		price := r.Price
		pricePtr := &price

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
			// ListPriceRow の正規形は ModelID / Price のみ。
			// Stock は list ではなく inventory 側から解決する。
			stock = 0
			stockFromList++
		}

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
		}

		mr, ok := attrByModel[modelID]
		if !ok {
			mr = q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
		}

		applyModelResolvedToListDetailPriceRow(&dtoRow, modelID, mr)

		if dtoRow.ModelNumber != "" ||
			dtoRow.Size != "" ||
			dtoRow.Color != "" ||
			dtoRow.RGB != nil ||
			dtoRow.VolumeValue != nil ||
			dtoRow.VolumeUnit != "" {
			resolvedNonEmpty++
		} else {
			resolvedEmpty++
		}

		out = append(out, dtoRow)
		total += stock
	}

	meta := "rows=" + Itoa(len(out)) +
		" resolvedNonEmpty=" + Itoa(resolvedNonEmpty) +
		" resolvedEmpty=" + Itoa(resolvedEmpty) +
		" invUsed=" + Bool01(invUsed) +
		" stockFromInv=" + Itoa(stockFromInv) +
		" stockFromList=" + Itoa(stockFromList) +
		" displayOrderHit=" + Itoa(displayOrderHit)
	if invErrMsg != "" {
		meta += " " + invErrMsg
	}

	return out, total, meta
}

func applyModelResolvedToListDetailPriceRow(
	row *querydto.ListDetailPriceRowDTO,
	modelID string,
	mr resolver.ModelResolved,
) {
	if row == nil {
		return
	}

	mn := mr.ModelNumber
	if mn == "" {
		mn = modelID
	}
	if mn == "" {
		mn = "-"
	}

	row.Kind = mr.Kind
	row.ModelNumber = mn

	if mr.Kind == "alcohol" {
		row.VolumeValue = mr.VolumeValue
		row.VolumeUnit = mr.VolumeUnit
		return
	}

	sz := mr.Size
	cl := mr.Color

	if sz == "" {
		sz = "-"
	}
	if cl == "" {
		cl = "-"
	}

	row.Size = sz
	row.Color = cl
	row.RGB = mr.RGB
}

func (q *ListDetailQuery) resolveModelResolvedCached(
	ctx context.Context,
	variationID string,
	cache map[string]resolver.ModelResolved,
) resolver.ModelResolved {
	id := variationID
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
// Display order helpers
// ============================================================

// buildDisplayOrderByModelID extracts displayOrder from ProductBlueprint.
//
// 責任:
// - ProductBlueprint.ModelRefs の正規構造から modelID/displayOrder を読み取る
// - modelID -> *displayOrder の辞書を生成する
// - displayOrder=0 は未設定として nil にする
func (q *ListDetailQuery) buildDisplayOrderByModelID(
	ctx context.Context,
	productBlueprintID string,
) map[string]*int {
	out := map[string]*int{}

	if q == nil || q.pbGetter == nil {
		return out
	}

	pbID := productBlueprintID
	if pbID == "" {
		return out
	}

	pb, err := q.pbGetter.GetByID(ctx, pbID)
	if err != nil {
		return out
	}

	if len(pb.ModelRefs) == 0 {
		return out
	}

	seen := map[string]struct{}{}
	for _, r := range pb.ModelRefs {
		mid := r.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}

		var ptr *int
		if r.DisplayOrder != 0 {
			x := r.DisplayOrder
			ptr = &x
		}

		out[mid] = ptr
	}

	return out
}
