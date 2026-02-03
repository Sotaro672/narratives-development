// backend/internal/application/query/console/list/detail/list_detail_query.go
//
// 機能: ListDetailQuery の公開API（DTO組み立て）
// 責任:
// - DI 済み依存（ports）を保持する
// - listID を入力に listDetail.tsx 用の ListDetailDTO を生成する
// - company boundary / inventory boundary を確認し、表示可能データのみ返す
package detail

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	listq "narratives/internal/application/query/console/list"
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

// ListImage を listID で取得できる port（任意）
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

	pbGetter listq.ProductBlueprintGetter
	tbGetter listq.TokenBlueprintGetter

	invGetter InventoryDetailGetter
	invRows   listq.InventoryRowsLister

	// listImage bucket の画像（= ListImage 由来のURL）を返すため（任意）
	imgLister ListImageLister

	// productBlueprintPatch から modelRef(displayOrder) を取るため（任意）
	pbPatchRepo listq.ProductBlueprintPatchReader
}

// ============================================================
// ✅ SINGLE ENTRYPOINT
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

func (q *ListDetailQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: getter is nil (wire list repo to ListDetailQuery)")
	}

	allowedSet, err := listq.AllowedInventoryIDSetFromContext(ctx, q.invRows)
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
	if !listq.InventoryAllowed(allowedSet, invID) {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	pbID, tbID, ok := listq.ParseInventoryIDStrict(invID)
	if !ok {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	// ---- names ----
	productName := ""
	tokenName := ""
	assigneeName := ""
	createdByName := ""

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
		if strings.TrimSpace(it.CreatedBy) != "" {
			createdByName = strings.TrimSpace(q.nameResolver.ResolveMemberName(ctx, it.CreatedBy))
		}
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
		CreatedAt:     it.CreatedAt.Format(time.RFC3339),

		UpdatedBy:     updatedByID,
		UpdatedByName: strings.TrimSpace(updatedByName),
		UpdatedAt:     it.UpdatedAt.Format(time.RFC3339),

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

	dto.ImageURLs = q.buildListImageURLs(ctx, strings.TrimSpace(it.ID), strings.TrimSpace(it.ImageID))
	return dto, nil
}
