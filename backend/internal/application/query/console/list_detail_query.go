// backend/internal/application/query/console/list_detail_query.go
//
// 機能: ListDetailQuery の公開API（DTO組み立て）
//
// 責任:
// - DI 済み依存（ports）を保持する
// - listID を入力に listDetail.tsx 用の ListDetailDTO を生成する
// - ListDetailDTO の images / priceRows / stock / model attributes を生成する
// - ListDetailDTO の transportationOption / transportationId を生成する
// - 取得済み ProductBlueprint.ModelRefs から displayOrder を抽出する
//
// Firebase Storage 移行後:
// - backend は GCS signed URL / GCS object を扱わない
// - list image record は domain/list.ListImage として扱う
// - 画像URLは list image record の URL、つまり Firebase Storage getDownloadURL() を使う
package query

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"
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

// ListImage を listID で取得できる port（任意）。
// Firebase Storage 移行後は domain/list.ListImage を使い、ListImage.URL は Firebase Storage downloadURL とする。
type ListImageLister interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
}

// ============================================================
// ListDetailQuery
// ============================================================

type ListDetailQuery struct {
	getter       ListGetter
	nameResolver *resolver.NameResolver
	memberRepo   memberdom.Repository
	pbGetter     applicationport.ProductBlueprintGetter
	tbGetter     applicationport.TokenBlueprintGetter
	invGetter    InventoryDetailGetter
	imgLister    ListImageLister
}

type NewListDetailQueryParams struct {
	Getter       ListGetter
	NameResolver *resolver.NameResolver
	MemberRepo   memberdom.Repository
	PBGetter     applicationport.ProductBlueprintGetter
	TBGetter     applicationport.TokenBlueprintGetter
	InvGetter    InventoryDetailGetter
	ImgLister    ListImageLister
}

func NewListDetailQuery(p NewListDetailQueryParams) *ListDetailQuery {
	return &ListDetailQuery{
		getter:       p.Getter,
		nameResolver: p.NameResolver,
		memberRepo:   p.MemberRepo,
		pbGetter:     p.PBGetter,
		tbGetter:     p.TBGetter,
		invGetter:    p.InvGetter,
		imgLister:    p.ImgLister,
	}
}

// ============================================================
// Query
// ============================================================

func (q *ListDetailQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: getter is nil (wire list repo to ListDetailQuery)")
	}

	if listID == "" {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: listID is empty")
	}

	it, err := q.getter.GetByID(ctx, listID)
	if err != nil {
		return querydto.ListDetailDTO{}, err
	}

	invID := it.InventoryID
	pbID, tbID, ok := ParseInventoryIDStrict(invID)
	if !ok {
		return querydto.ListDetailDTO{}, listdom.ErrListImageNotFound
	}

	productName := ""
	productBrandID := ""
	displayOrderByModel := map[string]*int{}

	if pbID != "" && q.pbGetter != nil {
		pb, err := q.pbGetter.GetByID(ctx, pbID)
		if err == nil {
			productName = pb.ProductName
			productBrandID = pb.BrandID
			displayOrderByModel = buildDisplayOrderByModelID(pb)
		}
	}

	tokenName := ""
	assigneeName := ""
	createdByName := ""
	updatedByName := ""
	updatedByID := ""

	if it.UpdatedBy != nil {
		updatedByID = *it.UpdatedBy
	}

	if q.nameResolver != nil {
		if productName == "" && pbID != "" && q.pbGetter == nil {
			productName = q.nameResolver.ResolveProductName(ctx, pbID)
		}

		if tbID != "" {
			tokenName = q.nameResolver.ResolveTokenName(ctx, tbID)
		}

		if it.CreatedBy != "" {
			createdByName = q.nameResolver.ResolveMemberName(ctx, it.CreatedBy)
		}

		if updatedByID != "" {
			updatedByName = q.nameResolver.ResolveUpdatedByName(ctx, it.UpdatedBy)
		}
	}

	if it.AssigneeID != "" && q.memberRepo != nil {
		rec, err := q.memberRepo.GetByID(ctx, it.AssigneeID)
		if err == nil {
			assigneeName = memberdom.FormatLastFirst(
				rec.Member.LastName,
				rec.Member.FirstName,
			)
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

	tokenBrandID := ""
	if tbID != "" && q.tbGetter != nil {
		tb, err := q.tbGetter.GetByID(ctx, tbID)
		if err == nil && tb != nil {
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

	createdAt := ""
	if !it.CreatedAt.IsZero() {
		createdAt = it.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if it.UpdatedAt != nil && !it.UpdatedAt.IsZero() {
		updatedAt = it.UpdatedAt.Format(time.RFC3339)
	}

	images := q.buildImages(ctx, it.ID, it.ImageID)
	priceRows := q.buildDetailPriceRows(ctx, it, invID, displayOrderByModel)

	return querydto.ListDetailDTO{
		ID:          it.ID,
		InventoryID: invID,

		Status:      string(it.Status),
		Title:       it.Title,
		Description: it.Description,

		AssigneeID:   it.AssigneeID,
		AssigneeName: assigneeName,

		TransportationOption: string(it.TransportationOption),
		TransportationID:     it.TransportationID,

		CreatedBy:     it.CreatedBy,
		CreatedByName: createdByName,
		CreatedAt:     createdAt,

		UpdatedBy:     updatedByID,
		UpdatedByName: updatedByName,
		UpdatedAt:     updatedAt,

		ProductBlueprintID: pbID,
		ProductBrandID:     productBrandID,
		ProductBrandName:   productBrandName,
		ProductName:        productName,

		TokenBlueprintID: tbID,
		TokenBrandID:     tokenBrandID,
		TokenBrandName:   tokenBrandName,
		TokenName:        tokenName,

		PrimaryImageID: it.ImageID,
		Images:         images,

		PriceRows: priceRows,
	}, nil
}

// ============================================================
// Image helpers
// ============================================================

// buildImages は list detail BFF 用の画像一覧を返す。
// List.ImageID が primary image ID のため、該当画像を先頭へ移動する。
// frontend は id / imageId や images / imageUrls の alias を持たず、このDTOを正とする。
func (q *ListDetailQuery) buildImages(ctx context.Context, listID string, primaryImageID string) []querydto.ListDetailImageDTO {
	if q == nil || q.imgLister == nil || listID == "" {
		return []querydto.ListDetailImageDTO{}
	}

	items, err := q.imgLister.ListByListID(ctx, listID)
	if err != nil || len(items) == 0 {
		return []querydto.ListDetailImageDTO{}
	}

	ordered := make([]listdom.ListImage, 0, len(items))
	usedIDs := make(map[string]struct{}, len(items))

	if primaryImageID != "" {
		for _, img := range items {
			if img.ID != primaryImageID {
				continue
			}

			ordered = append(ordered, img)
			if img.ID != "" {
				usedIDs[img.ID] = struct{}{}
			}
			break
		}
	}

	for _, img := range items {
		if img.ID != "" {
			if _, ok := usedIDs[img.ID]; ok {
				continue
			}
		}

		ordered = append(ordered, img)
		if img.ID != "" {
			usedIDs[img.ID] = struct{}{}
		}
	}

	out := make([]querydto.ListDetailImageDTO, 0, len(ordered))
	seenURLs := make(map[string]struct{}, len(ordered))

	for _, img := range ordered {
		if img.ID == "" || img.URL == "" {
			continue
		}

		if _, ok := seenURLs[img.URL]; ok {
			continue
		}

		seenURLs[img.URL] = struct{}{}
		out = append(out, querydto.ListDetailImageDTO{
			ID:           img.ID,
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		})
	}

	return out
}

// ============================================================
// Price row helpers
// ============================================================

// buildDetailPriceRows builds ListDetailDTO price rows.
// - listdom.List の価格行を抽出する
// - 在庫情報は InventoryDetailGetter から取得する
// - displayOrder は ProductBlueprint.ModelRefs から付与する
// - model 情報は resolver.ModelResolved を使って解決する
func (q *ListDetailQuery) buildDetailPriceRows(
	ctx context.Context,
	it listdom.List,
	inventoryID string,
	displayOrderByModel map[string]*int,
) []querydto.ListDetailPriceRowDTO {
	rows := it.Prices
	if len(rows) == 0 {
		return []querydto.ListDetailPriceRowDTO{}
	}

	stockByModel := map[string]int{}
	attrByModel := map[string]resolver.ModelResolved{}
	invUsed := false

	if inventoryID != "" && q != nil && q.invGetter != nil {
		invDTO, err := q.invGetter.GetDetailByID(ctx, inventoryID)
		if err == nil && invDTO != nil {
			invUsed = true

			for _, row := range invDTO.Rows {
				modelID := row.ModelID
				if modelID == "" {
					continue
				}

				stockByModel[modelID] = row.Stock
				attrByModel[modelID] = resolver.ModelResolved{
					Kind:        row.Kind,
					ModelNumber: row.ModelNumber,
					Size:        row.Size,
					Color:       row.Color,
					RGB:         row.RGB,
					VolumeValue: row.VolumeValue,
					VolumeUnit:  row.VolumeUnit,
				}
			}
		}
	}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rows))

	for _, row := range rows {
		modelID := row.ModelID
		if modelID == "" {
			continue
		}

		price := row.Price
		stock := 0
		if invUsed {
			stock = stockByModel[modelID]
		}

		var displayOrder *int
		if value, ok := displayOrderByModel[modelID]; ok {
			displayOrder = value
		}

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID:      modelID,
			DisplayOrder: displayOrder,
			Stock:        stock,
			Price:        &price,
		}

		modelResolved, ok := attrByModel[modelID]
		if !ok && q != nil && q.nameResolver != nil {
			modelResolved = q.nameResolver.ResolveModelResolved(ctx, modelID)
		}

		applyModelResolvedToListDetailPriceRow(&dtoRow, modelID, modelResolved)
		out = append(out, dtoRow)
	}

	return out
}

func applyModelResolvedToListDetailPriceRow(row *querydto.ListDetailPriceRowDTO, modelID string, model resolver.ModelResolved) {
	if row == nil {
		return
	}

	modelNumber := model.ModelNumber
	if modelNumber == "" {
		modelNumber = modelID
	}

	if modelNumber == "" {
		modelNumber = "-"
	}

	row.Kind = model.Kind
	row.ModelNumber = modelNumber

	if model.Kind == "alcohol" {
		row.VolumeValue = model.VolumeValue
		row.VolumeUnit = model.VolumeUnit
		return
	}

	size := model.Size
	color := model.Color

	if size == "" {
		size = "-"
	}

	if color == "" {
		color = "-"
	}

	row.Size = size
	row.Color = color
	row.RGB = model.RGB
}

// ============================================================
// Display order helpers
// ============================================================

func buildDisplayOrderByModelID(pb pbdom.ProductBlueprint) map[string]*int {
	out := map[string]*int{}
	if len(pb.ModelRefs) == 0 {
		return out
	}

	seen := map[string]struct{}{}

	for _, ref := range pb.ModelRefs {
		modelID := ref.ModelID
		if modelID == "" {
			continue
		}

		if _, ok := seen[modelID]; ok {
			continue
		}

		seen[modelID] = struct{}{}

		var displayOrder *int
		if ref.DisplayOrder != 0 {
			value := ref.DisplayOrder
			displayOrder = &value
		}

		out[modelID] = displayOrder
	}

	return out
}
