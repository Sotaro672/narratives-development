// backend/internal/application/query/console/list/detail/list_detail_query.go
//
// 機能: ListDetailQuery の公開API（DTO組み立て）
// 責任:
// - DI 済み依存（ports）を保持する
// - listID を入力に listDetail.tsx 用の ListDetailDTO を生成する
// - company boundary / inventory boundary を確認し、表示可能データのみ返す
//
// Firebase Storage 移行後:
// - backend は GCS signed URL / GCS object / domain/listImage を扱わない
// - list image record は domain/list.ListImage として扱う
// - 画像URLは list image record の URL、つまり Firebase Storage getDownloadURL() を使う
package detail

import (
	"context"
	"errors"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	listq "narratives/internal/application/query/console/list"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
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

// ListImage を listID で取得できる port（任意）
//
// Firebase Storage 移行後:
// - domain/listImage は削除
// - domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
// - ListImage.ObjectPath は Firebase Storage objectPath
type ListImageLister interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
}

// ============================================================
// ListDetailQuery (listDetail.tsx)
// ============================================================

type ListDetailQuery struct {
	getter       ListGetter
	nameResolver *resolver.NameResolver

	pbGetter listq.ProductBlueprintGetter
	tbGetter listq.TokenBlueprintGetter

	invGetter InventoryDetailGetter
	invRows   listq.InventoryRowsLister

	// list image record 由来の Firebase Storage downloadURL を返すため（任意）
	imgLister ListImageLister

	// productBlueprintPatch から modelRef(displayOrder) を取るため（任意）
	pbPatchRepo listq.ProductBlueprintPatchReader
}

// ============================================================
// SINGLE ENTRYPOINT
// - ここだけを入口にして、全依存を配線する
// - optional は nil を許容する（既存DIを壊さない）
// ============================================================

type NewListDetailQueryParams struct {
	Getter       ListGetter
	NameResolver *resolver.NameResolver

	PBGetter listq.ProductBlueprintGetter
	TBGetter listq.TokenBlueprintGetter

	InvGetter InventoryDetailGetter
	InvRows   listq.InventoryRowsLister

	ImgLister   ListImageLister
	PBPatchRepo listq.ProductBlueprintPatchReader
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
		pbPatchRepo:  p.PBPatchRepo,
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

	allowedSet, err := listq.AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
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
	if !listq.InventoryAllowed(allowedSet, invID) {
		return querydto.ListDetailDTO{}, listdom.ErrListImageNotFound
	}

	pbID, tbID, ok := listq.ParseInventoryIDStrict(invID)
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
	imageURLs := q.buildListImageURLs(ctx, it.ID, it.ImageID)

	// ---- price rows / stock ----
	//
	// list の price rows を DTO に復元する。
	// stock は inventory detail が取れる場合は inventory を優先し、
	// なければ list 側の値を使う。
	priceRows, totalStock, _ := q.buildDetailPriceRows(ctx, it, invID, pbID)

	dto := querydto.ListDetailDTO{
		ID:          it.ID,
		InventoryID: invID,

		Status:   string(it.Status),
		Decision: string(it.Status),

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

		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	return dto, nil
}

// ============================================================
// Image helpers
// ============================================================

// buildListImageURLs returns Firebase Storage downloadURLs.
//
// primaryImageID:
// - List.ImageID に保存されている primary image record docID
// - 存在する場合、その画像URLを先頭に並べる
func (q *ListDetailQuery) buildListImageURLs(
	ctx context.Context,
	listID string,
	primaryImageID string,
) []string {
	if q == nil || q.imgLister == nil || listID == "" {
		return []string{}
	}

	items, err := q.imgLister.ListByListID(ctx, listID)
	if err != nil {
		return []string{}
	}

	if len(items) == 0 {
		return []string{}
	}

	urls := make([]string, 0, len(items))
	seen := map[string]struct{}{}

	appendURL := func(u string) {
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}

	if primaryImageID != "" {
		for _, img := range items {
			if img.ID == primaryImageID {
				appendURL(img.URL)
				break
			}
		}
	}

	for _, img := range items {
		appendURL(img.URL)
	}

	return urls
}
