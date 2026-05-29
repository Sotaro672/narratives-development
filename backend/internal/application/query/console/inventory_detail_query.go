// backend/internal/application/query/console/inventory_detail_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"

	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type InventoryDetailQuery struct {
	invRepo      inventoryReader
	pbRepo       inventoryProductBlueprintReader
	tbRepo       inventoryTokenBlueprintReader
	nameResolver *resolver.NameResolver
}

func NewInventoryDetailQuery(
	invRepo inventoryReader,
	pbRepo inventoryProductBlueprintReader,
	tbRepo inventoryTokenBlueprintReader,
	nameResolver *resolver.NameResolver,
) *InventoryDetailQuery {
	return &InventoryDetailQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		tbRepo:       tbRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// TokenBlueprint: tbId -> Patch-compatible DTO
// - GetPatchByID は使わず、GetByID で取得した TokenBlueprint から Patch を組み立てる。
// - BrandName は NameResolver で補完する。
// ============================================================

func (q *InventoryDetailQuery) GetTokenBlueprintPatchByID(ctx context.Context, tokenBlueprintID string) (*tbdom.Patch, error) {
	if q == nil {
		return nil, errors.New("inventory detail query is nil")
	}
	if q.tbRepo == nil {
		return nil, errors.New("tokenBlueprint repository is not configured")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is required")
	}

	tb, err := q.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, errors.New("tokenBlueprint is nil")
	}

	patch := tokenBlueprintToPatch(tb)

	if patch.BrandID != "" && patch.BrandName == "" && q.nameResolver != nil {
		brandName := q.nameResolver.ResolveBrandName(ctx, patch.BrandID)
		if brandName != "" {
			patch.BrandName = brandName
		}
	}

	return &patch, nil
}

// ============================================================
// Detail: inventoryId -> DTO
// ============================================================

func (q *InventoryDetailQuery) GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory detail query repositories are not configured")
	}

	id := inventoryID
	if id == "" {
		return nil, errors.New("inventoryId is required")
	}

	inv, err := q.invRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	pbID := inv.ProductBlueprintID
	tbID := inv.TokenBlueprintID

	if pbID == "" {
		return nil, errors.New("productBlueprintId is empty in inventory")
	}
	if tbID == "" {
		return nil, errors.New("tokenBlueprintId is empty in inventory")
	}

	if q.pbRepo == nil {
		return nil, errors.New("productBlueprint repository is not configured")
	}

	pb, err := q.pbRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	pbPatch := productBlueprintToPatch(pb)

	if pbPatch.BrandID != nil &&
		*pbPatch.BrandID != "" &&
		(pbPatch.BrandName == nil || *pbPatch.BrandName == "") &&
		q.nameResolver != nil {

		brandName := q.nameResolver.ResolveBrandName(ctx, *pbPatch.BrandID)
		if brandName != "" {
			pbPatch.BrandName = &brandName
		}
	}

	pbPatchPtr := &pbPatch

	var tbPatchPtr *tbdom.Patch
	{
		p, err := q.GetTokenBlueprintPatchByID(ctx, tbID)
		if err != nil {
			tbPatchPtr = nil
		} else {
			tbPatchPtr = p
		}
	}

	if pbPatchPtr.ModelRefs == nil || len(*pbPatchPtr.ModelRefs) == 0 {
		return nil, errors.New("productBlueprint.modelRefs is empty (fallback via inv.Stock is abolished)")
	}

	refs := append([]pbdom.ModelRef(nil), (*pbPatchPtr.ModelRefs)...)

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].DisplayOrder < refs[j].DisplayOrder
	})

	orderedModelIDs := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, r := range refs {
		mid := r.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		orderedModelIDs = append(orderedModelIDs, mid)
	}

	if len(orderedModelIDs) == 0 {
		return nil, errors.New("productBlueprint.modelRefs has no valid modelId")
	}

	rows := make([]querydto.InventoryDetailRowDTO, 0, len(orderedModelIDs))
	total := 0

	for _, modelID0 := range orderedModelIDs {
		modelID := modelID0
		if modelID == "" {
			continue
		}

		ms, ok := inv.Stock[modelID]

		available := 0
		if ok {
			available = ms.Accumulation - ms.ReservedCount
			if available < 0 {
				available = 0
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, modelID)
		}

		mn := attr.ModelNumber
		if mn == "" {
			mn = modelID
		}
		if mn == "" {
			mn = "-"
		}

		row := querydto.InventoryDetailRowDTO{
			ModelID:     modelID,
			Kind:        attr.Kind,
			ModelNumber: mn,
			Stock:       available,
		}

		if attr.Kind == "alcohol" {
			row.VolumeValue = attr.VolumeValue
			row.VolumeUnit = attr.VolumeUnit
		} else {
			size := attr.Size
			color := attr.Color

			if size == "" {
				size = "-"
			}
			if color == "" {
				color = "-"
			}

			row.Size = size
			row.Color = color
			row.RGB = attr.RGB
		}

		rows = append(rows, row)
		total += available
	}

	updated := inv.UpdatedAt
	if updated.IsZero() {
		updated = inv.CreatedAt
	}
	updatedAt := ""
	if !updated.IsZero() {
		updatedAt = updated.UTC().Format(time.RFC3339)
	}

	dto := &querydto.InventoryDetailDTO{
		InventoryID:           id,
		TokenBlueprintID:      tbID,
		ProductBlueprintID:    pbID,
		ProductBlueprintPatch: pbPatchPtr,
		TokenBlueprintPatch:   tbPatchPtr,
		Rows:                  rows,
		TotalStock:            total,
		UpdatedAt:             updatedAt,
	}

	return dto, nil
}

func productBlueprintToPatch(pb pbdom.ProductBlueprint) pbdom.Patch {
	productName := pb.ProductName
	description := pb.Description
	brandID := pb.BrandID
	companyID := pb.CompanyID
	category := pb.ProductBlueprintCategory
	categoryFields := pb.CategoryFields
	productIDTag := pb.ProductIdTag
	assigneeID := pb.AssigneeID
	modelRefs := append([]pbdom.ModelRef(nil), pb.ModelRefs...)

	return pbdom.Patch{
		ProductName:              &productName,
		Description:              &description,
		BrandID:                  &brandID,
		CompanyID:                &companyID,
		ProductBlueprintCategory: &category,
		CategoryFields:           &categoryFields,
		ProductIdTag:             &productIDTag,
		AssigneeID:               &assigneeID,
		ModelRefs:                &modelRefs,
	}
}

func tokenBlueprintToPatch(tb *tbdom.TokenBlueprint) tbdom.Patch {
	if tb == nil {
		return tbdom.Patch{}
	}

	return tbdom.Patch{
		ID:          tb.ID,
		TokenName:   tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		CompanyID:   tb.CompanyID,
		Description: tb.Description,
		Minted:      tb.Minted,
		MetadataURI: tb.MetadataURI,
		IconURL:     tb.IconURL,
	}
}
