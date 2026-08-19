// backend/internal/application/query/console/inventory_detail_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"

	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
	shadom "narratives/internal/domain/shippingAddress"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type InventoryDetailQuery struct {
	invRepo              inventoryReader
	pbRepo               inventoryProductBlueprintReader
	tbRepo               inventoryTokenBlueprintReader
	shippingAddressRepo  shadom.RepositoryPort
	nameResolver         *resolver.NameResolver
	companyIDFromContext func(context.Context) string
}

func NewInventoryDetailQuery(
	invRepo inventoryReader,
	pbRepo inventoryProductBlueprintReader,
	tbRepo inventoryTokenBlueprintReader,
	shippingAddressRepo shadom.RepositoryPort,
	nameResolver *resolver.NameResolver,
	companyIDFromContext func(context.Context) string,
) *InventoryDetailQuery {
	return &InventoryDetailQuery{
		invRepo:              invRepo,
		pbRepo:               pbRepo,
		tbRepo:               tbRepo,
		shippingAddressRepo:  shippingAddressRepo,
		nameResolver:         nameResolver,
		companyIDFromContext: companyIDFromContext,
	}
}

// ============================================================
// TokenBlueprint: tbId -> Inventory Detail DTO
// - GetByID で取得した TokenBlueprint から
//   InventoryTokenBlueprintPatchDTO を組み立てる。
// - BrandName は NameResolver で解決する。
// ============================================================

func (q *InventoryDetailQuery) GetTokenBlueprintPatchByID(
	ctx context.Context,
	tokenBlueprintID string,
) (*querydto.InventoryTokenBlueprintPatchDTO, error) {
	if q == nil {
		return nil, errors.New("inventory detail query is nil")
	}

	if q.tbRepo == nil {
		return nil, errors.New("tokenBlueprint repository is not configured")
	}

	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprintId is required")
	}

	tb, err := q.tbRepo.GetByID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return nil, err
	}

	if tb == nil {
		return nil, errors.New("tokenBlueprint is nil")
	}

	brandName := ""

	if tb.BrandID != "" &&
		q.nameResolver != nil {

		brandName =
			q.nameResolver.ResolveBrandName(
				ctx,
				tb.BrandID,
			)
	}

	return buildInventoryTokenBlueprintPatchDTO(
		tb,
		brandName,
	), nil
}

// ============================================================
// Detail: inventoryId -> DTO
// ============================================================

func (q *InventoryDetailQuery) GetDetailByID(
	ctx context.Context,
	inventoryID string,
) (*querydto.InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return nil, errors.New("inventory detail query repositories are not configured")
	}

	if inventoryID == "" {
		return nil, errors.New("inventoryId is required")
	}

	inv, err := q.invRepo.GetByID(
		ctx,
		inventoryID,
	)
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

	pb, err := q.pbRepo.GetByID(
		ctx,
		pbID,
	)
	if err != nil {
		return nil, err
	}

	if q.companyIDFromContext == nil {
		return nil, errors.New("companyId context resolver is not configured")
	}

	companyID :=
		q.companyIDFromContext(
			ctx,
		)

	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	if pb.CompanyID == "" {
		return nil, errors.New("productBlueprint.companyId is empty")
	}

	if pb.CompanyID != companyID {
		return nil, invdom.ErrNotFound
	}

	productBrandName := ""

	if pb.BrandID != "" &&
		q.nameResolver != nil {

		productBrandName =
			q.nameResolver.ResolveBrandName(
				ctx,
				pb.BrandID,
			)
	}

	pbPatchPtr :=
		buildInventoryProductBlueprintPatchDTO(
			pb,
			productBrandName,
		)

	tbPatchPtr, err :=
		q.GetTokenBlueprintPatchByID(
			ctx,
			tbID,
		)
	if err != nil {
		return nil, err
	}

	if q.shippingAddressRepo == nil {
		return nil, errors.New("shippingAddress repository is not configured")
	}

	shippingAddresses, err :=
		q.shippingAddressRepo.ListByCompanyID(
			ctx,
			companyID,
		)
	if err != nil {
		return nil, err
	}

	if shippingAddresses == nil {
		shippingAddresses =
			[]shadom.ShippingAddress{}
	}

	shippingAddressOptions := make(
		[]querydto.InventoryShippingAddressDTO,
		0,
		len(shippingAddresses),
	)

	var shippingAddressPtr *querydto.InventoryShippingAddressDTO

	for _, shippingAddress := range shippingAddresses {

		option :=
			buildInventoryShippingAddressDTO(
				shippingAddress,
			)

		shippingAddressOptions =
			append(
				shippingAddressOptions,
				option,
			)

		if inv.ShippingAddressID != "" &&
			shippingAddress.ID ==
				inv.ShippingAddressID {

			selected :=
				option

			shippingAddressPtr =
				&selected
		}
	}

	if len(pb.ModelRefs) == 0 {
		return nil, errors.New("productBlueprint.modelRefs is empty (fallback via inv.Stock is abolished)")
	}

	refs := append(
		[]pbdom.ModelRef(nil),
		pb.ModelRefs...,
	)

	sort.Slice(
		refs,
		func(i, j int) bool {
			return refs[i].DisplayOrder <
				refs[j].DisplayOrder
		},
	)

	orderedModelIDs := make(
		[]string,
		0,
		len(refs),
	)

	seen :=
		map[string]struct{}{}

	for _, ref := range refs {
		modelID := ref.ModelID

		if modelID == "" {
			continue
		}

		if _, exists := seen[modelID]; exists {
			continue
		}

		seen[modelID] =
			struct{}{}

		orderedModelIDs =
			append(
				orderedModelIDs,
				modelID,
			)
	}

	if len(orderedModelIDs) == 0 {
		return nil, errors.New("productBlueprint.modelRefs has no valid modelId")
	}

	rows := make(
		[]querydto.InventoryDetailRowDTO,
		0,
		len(orderedModelIDs),
	)

	total := 0

	for _, modelID := range orderedModelIDs {
		modelStock, ok :=
			inv.Stock[modelID]

		available := 0

		if ok {
			available =
				modelStock.Accumulation -
					modelStock.ReservedCount

			if available < 0 {
				available = 0
			}
		}

		attr :=
			resolver.ModelResolved{}

		if q.nameResolver != nil {
			attr =
				q.nameResolver.ResolveModelResolved(
					ctx,
					modelID,
				)
		}

		modelNumber :=
			attr.ModelNumber

		if modelNumber == "" {
			modelNumber =
				modelID
		}

		if modelNumber == "" {
			modelNumber =
				"-"
		}

		row :=
			querydto.InventoryDetailRowDTO{
				ModelID:     modelID,
				Kind:        attr.Kind,
				ModelNumber: modelNumber,
				Stock:       available,
			}

		if attr.Kind == "alcohol" {
			row.VolumeValue =
				attr.VolumeValue

			row.VolumeUnit =
				attr.VolumeUnit
		} else {
			size :=
				attr.Size

			color :=
				attr.Color

			if size == "" {
				size =
					"-"
			}

			if color == "" {
				color =
					"-"
			}
			row.Size =
				size
			row.Color =
				color
			row.RGB =
				attr.RGB
		}

		rows =
			append(
				rows,
				row,
			)

		total +=
			available
	}

	updated :=
		inv.UpdatedAt

	if updated.IsZero() {
		updated =
			inv.CreatedAt
	}

	updatedAt := ""

	if !updated.IsZero() {
		updatedAt =
			updated.UTC().Format(
				time.RFC3339,
			)
	}

	return &querydto.InventoryDetailDTO{
		InventoryID:            inventoryID,
		TokenBlueprintID:       tbID,
		ProductBlueprintID:     pbID,
		ProductBlueprintPatch:  pbPatchPtr,
		TokenBlueprintPatch:    tbPatchPtr,
		ShippingAddressID:      inv.ShippingAddressID,
		ShippingAddress:        shippingAddressPtr,
		ShippingAddressOptions: shippingAddressOptions,
		Rows:                   rows,
		TotalStock:             total,
		UpdatedAt:              updatedAt,
	}, nil
}

// ============================================================
// ProductBlueprint -> Inventory Detail DTO
// ============================================================

func buildInventoryProductBlueprintPatchDTO(
	productBlueprint pbdom.ProductBlueprint,
	brandName string,
) *querydto.InventoryProductBlueprintPatchDTO {
	modelRefs := make(
		[]querydto.InventoryProductBlueprintModelRefDTO,
		0,
		len(productBlueprint.ModelRefs),
	)

	for _, modelRef := range productBlueprint.ModelRefs {

		modelRefs =
			append(
				modelRefs,
				querydto.InventoryProductBlueprintModelRefDTO{
					ModelID: modelRef.ModelID,

					DisplayOrder: modelRef.DisplayOrder,
				},
			)
	}

	return &querydto.InventoryProductBlueprintPatchDTO{
		ProductName: productBlueprint.ProductName,
		Description: productBlueprint.Description,
		BrandID:     productBlueprint.BrandID,
		BrandName:   brandName,
		CompanyID:   productBlueprint.CompanyID,
		ProductBlueprintCategoryPath: append(
			[]string(nil),
			productBlueprint.ProductBlueprintCategoryPath...,
		),
		CategoryFields: map[string]any(
			productBlueprint.CategoryFields,
		),
		ProductIDTag: querydto.InventoryProductIDTagDTO{
			Type: string(
				productBlueprint.ProductIdTag.Type,
			),
		},
		AssigneeID: productBlueprint.AssigneeID,
		ModelRefs:  modelRefs,
	}
}

// ============================================================
// ShippingAddress -> Inventory Detail DTO
// ============================================================

func buildInventoryShippingAddressDTO(
	shippingAddress shadom.ShippingAddress,
) querydto.InventoryShippingAddressDTO {
	return querydto.InventoryShippingAddressDTO{
		ID:      shippingAddress.ID,
		ZipCode: shippingAddress.ZipCode,
		State:   shippingAddress.State,
		City:    shippingAddress.City,
		Street:  shippingAddress.Street,
		Street2: shippingAddress.Street2,
	}
}

// ============================================================
// TokenBlueprint -> Inventory Detail DTO
// ============================================================

func buildInventoryTokenBlueprintPatchDTO(
	tokenBlueprint *tbdom.TokenBlueprint,
	brandName string,
) *querydto.InventoryTokenBlueprintPatchDTO {
	if tokenBlueprint == nil {
		return nil
	}

	return &querydto.InventoryTokenBlueprintPatchDTO{
		ID:          tokenBlueprint.ID,
		TokenName:   tokenBlueprint.Name,
		Symbol:      tokenBlueprint.Symbol,
		BrandID:     tokenBlueprint.BrandID,
		BrandName:   brandName,
		CompanyID:   tokenBlueprint.CompanyID,
		Description: tokenBlueprint.Description,
		Minted:      tokenBlueprint.Minted,
		MetadataURI: tokenBlueprint.MetadataURI,
		IconURL:     tokenBlueprint.IconURL,
	}
}
