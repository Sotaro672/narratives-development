// backend/internal/application/query/console/productblueprint_detail_query.go
package query

import (
	"context"
	"fmt"
	"sort"
	"strings"

	applicationport "narratives/internal/application/port"
	modeldom "narratives/internal/domain/model"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintDetailModelRepo defines the model variation read port needed by the product blueprint detail screen.
type ProductBlueprintDetailModelRepo interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error)
}

// ProductBlueprintDetailSizeRow is the apparel size state consumed directly by the detail screen.
type ProductBlueprintDetailSizeRow struct {
	ID           string
	SizeLabel    string
	Length       *int
	Width        *int
	Chest        *int
	Shoulder     *int
	SleeveLength *int
	Waist        *int
	Hip          *int
	Rise         *int
	Inseam       *int
	Thigh        *int
	HemWidth     *int
}

// ProductBlueprintDetailShippingPackage is the packaged shipping state for a model variation.
type ProductBlueprintDetailShippingPackage struct {
	WeightGrams int
	WidthMM     int
	LengthMM    int
	HeightMM    int
}

// ProductBlueprintDetailApparelModelNumber is the apparel model-number state consumed directly by the detail screen.
type ProductBlueprintDetailApparelModelNumber struct {
	Size            string
	Color           string
	Code            string
	ShippingPackage ProductBlueprintDetailShippingPackage
}

// ProductBlueprintDetailVolume is the common alcohol volume state.
type ProductBlueprintDetailVolume struct {
	Value int
	Unit  string
}

// ProductBlueprintDetailVolumeRow is the alcohol volume row consumed directly by the detail screen.
type ProductBlueprintDetailVolumeRow struct {
	ID          string
	VolumeValue int
	VolumeUnit  string
}

// ProductBlueprintDetailAlcoholModelNumber is the alcohol model-number state consumed directly by the detail screen.
type ProductBlueprintDetailAlcoholModelNumber struct {
	Kind            string
	Volume          ProductBlueprintDetailVolume
	Code            string
	ShippingPackage ProductBlueprintDetailShippingPackage
}

// ProductBlueprintDetailModelState is the screen-complete model state.
// Frontend should be able to pass this state directly to useProductBlueprintVariations.setFromUiState.
type ProductBlueprintDetailModelState struct {
	Colors              []string
	Sizes               []ProductBlueprintDetailSizeRow
	ModelNumbers        []ProductBlueprintDetailApparelModelNumber
	ColorRGBMap         map[string]string
	Volumes             []ProductBlueprintDetailVolumeRow
	AlcoholModelNumbers []ProductBlueprintDetailAlcoholModelNumber
}

// ProductBlueprintDetailResolved is the complete read model for the product blueprint detail BFF.
type ProductBlueprintDetailResolved struct {
	ProductBlueprint ProductBlueprintResolved
	ModelState       ProductBlueprintDetailModelState
}

type ProductBlueprintDetailQuery struct {
	repo                 applicationport.ProductBlueprintGetter
	modelRepo            ProductBlueprintDetailModelRepo
	managementQuery      *ProductBlueprintManagementQuery
	companyIDFromContext applicationport.CompanyIDResolver
}

func NewProductBlueprintDetailQuery(
	repo applicationport.ProductBlueprintGetter,
	modelRepo ProductBlueprintDetailModelRepo,
	managementQuery *ProductBlueprintManagementQuery,
	companyIDFromContext applicationport.CompanyIDResolver,
) *ProductBlueprintDetailQuery {
	return &ProductBlueprintDetailQuery{
		repo:                 repo,
		modelRepo:            modelRepo,
		managementQuery:      managementQuery,
		companyIDFromContext: companyIDFromContext,
	}
}

// GetByID builds the complete read model used by the product blueprint detail screen.
func (q *ProductBlueprintDetailQuery) GetByID(
	ctx context.Context,
	id string,
) (ProductBlueprintDetailResolved, error) {
	if id == "" {
		return ProductBlueprintDetailResolved{}, productbpdom.ErrInvalidID
	}

	cid := q.resolveCompanyID(ctx)
	if cid == "" {
		return ProductBlueprintDetailResolved{}, productbpdom.ErrInvalidCompanyID
	}

	if q.repo == nil {
		return ProductBlueprintDetailResolved{}, fmt.Errorf(
			"%w: product blueprint detail repository is not configured",
			productbpdom.ErrInternal,
		)
	}

	if q.managementQuery == nil {
		return ProductBlueprintDetailResolved{}, fmt.Errorf(
			"%w: product blueprint management query is not configured",
			productbpdom.ErrInternal,
		)
	}

	if q.modelRepo == nil {
		return ProductBlueprintDetailResolved{}, fmt.Errorf(
			"%w: product blueprint detail model repository is not configured",
			productbpdom.ErrInternal,
		)
	}

	pb, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return ProductBlueprintDetailResolved{}, err
	}

	if pb.CompanyID == "" || pb.CompanyID != cid {
		return ProductBlueprintDetailResolved{}, productbpdom.ErrForbidden
	}

	resolved := q.managementQuery.resolveProductBlueprint(ctx, pb)

	modelVariations, err := q.modelRepo.ListByProductBlueprintID(ctx, pb.ID)
	if err != nil {
		return ProductBlueprintDetailResolved{}, err
	}

	orderedVariations := orderModelVariationsByModelRefs(modelVariations, pb.ModelRefs)
	categoryCode := strings.Join(pb.ProductBlueprintCategoryPath, ".")

	modelState, err := buildProductBlueprintDetailModelState(orderedVariations, categoryCode)
	if err != nil {
		return ProductBlueprintDetailResolved{}, err
	}

	return ProductBlueprintDetailResolved{
		ProductBlueprint: resolved,
		ModelState:       modelState,
	}, nil
}

func buildProductBlueprintDetailModelState(
	variations []modeldom.ModelVariation,
	categoryCode string,
) (ProductBlueprintDetailModelState, error) {
	state := ProductBlueprintDetailModelState{
		Colors:              []string{},
		Sizes:               []ProductBlueprintDetailSizeRow{},
		ModelNumbers:        []ProductBlueprintDetailApparelModelNumber{},
		ColorRGBMap:         map[string]string{},
		Volumes:             []ProductBlueprintDetailVolumeRow{},
		AlcoholModelNumbers: []ProductBlueprintDetailAlcoholModelNumber{},
	}

	colorSeen := make(map[string]struct{})
	sizeSeen := make(map[string]struct{})
	sizeOrder := make([]string, 0)
	sizeRepresentativeIDs := make(map[string]string)
	sizeMeasurements := make(map[string]modeldom.Measurements)
	volumeSeen := make(map[string]struct{})

	for _, variation := range variations {
		if variation == nil {
			continue
		}

		switch typed := variation.(type) {
		case modeldom.ApparelModelVariation:
			appendApparelVariationToModelState(
				&state,
				typed,
				colorSeen,
				sizeSeen,
				&sizeOrder,
				sizeRepresentativeIDs,
				sizeMeasurements,
			)

		case *modeldom.ApparelModelVariation:
			if typed == nil {
				continue
			}

			appendApparelVariationToModelState(
				&state,
				*typed,
				colorSeen,
				sizeSeen,
				&sizeOrder,
				sizeRepresentativeIDs,
				sizeMeasurements,
			)

		case modeldom.AlcoholModelVariation:
			appendAlcoholVariationToModelState(&state, typed, volumeSeen)

		case *modeldom.AlcoholModelVariation:
			if typed == nil {
				continue
			}

			appendAlcoholVariationToModelState(&state, *typed, volumeSeen)

		default:
			return ProductBlueprintDetailModelState{}, fmt.Errorf(
				"%w: unsupported model variation type",
				productbpdom.ErrInternal,
			)
		}
	}

	state.Sizes = make([]ProductBlueprintDetailSizeRow, 0, len(sizeOrder))
	for _, sizeLabel := range sizeOrder {
		state.Sizes = append(
			state.Sizes,
			buildProductBlueprintDetailSizeRow(
				sizeRepresentativeIDs[sizeLabel],
				sizeLabel,
				sizeMeasurements[sizeLabel],
				categoryCode,
			),
		)
	}

	return state, nil
}

func appendApparelVariationToModelState(
	state *ProductBlueprintDetailModelState,
	variation modeldom.ApparelModelVariation,
	colorSeen map[string]struct{},
	sizeSeen map[string]struct{},
	sizeOrder *[]string,
	sizeRepresentativeIDs map[string]string,
	sizeMeasurements map[string]modeldom.Measurements,
) {
	colorName := variation.Color.Name
	if colorName != "" {
		if _, exists := colorSeen[colorName]; !exists {
			colorSeen[colorName] = struct{}{}
			state.Colors = append(state.Colors, colorName)
		}

		if variation.Color.RGB >= 0 && variation.Color.RGB <= 0xFFFFFF {
			state.ColorRGBMap[colorName] = fmt.Sprintf("#%06x", variation.Color.RGB)
		}
	}

	sizeLabel := variation.Size
	if sizeLabel != "" {
		if _, exists := sizeSeen[sizeLabel]; !exists {
			sizeSeen[sizeLabel] = struct{}{}
			*sizeOrder = append(*sizeOrder, sizeLabel)
			sizeRepresentativeIDs[sizeLabel] = variation.ID
			sizeMeasurements[sizeLabel] = make(modeldom.Measurements)
		}

		merged := sizeMeasurements[sizeLabel]
		for key, value := range variation.Measurements {
			if _, exists := merged[key]; exists {
				continue
			}

			merged[key] = value
		}
	}

	state.ModelNumbers = append(
		state.ModelNumbers,
		ProductBlueprintDetailApparelModelNumber{
			Size:            variation.Size,
			Color:           variation.Color.Name,
			Code:            variation.ModelNumber,
			ShippingPackage: buildProductBlueprintDetailShippingPackage(variation.ShippingPackage),
		},
	)
}

func appendAlcoholVariationToModelState(
	state *ProductBlueprintDetailModelState,
	variation modeldom.AlcoholModelVariation,
	volumeSeen map[string]struct{},
) {
	volume := ProductBlueprintDetailVolume{
		Value: variation.Volume.Value,
		Unit:  variation.Volume.Unit,
	}

	volumeKey := fmt.Sprintf("%d\x00%s", volume.Value, volume.Unit)
	if _, exists := volumeSeen[volumeKey]; !exists {
		volumeSeen[volumeKey] = struct{}{}
		state.Volumes = append(
			state.Volumes,
			ProductBlueprintDetailVolumeRow{
				ID:          variation.ID,
				VolumeValue: volume.Value,
				VolumeUnit:  volume.Unit,
			},
		)
	}

	state.AlcoholModelNumbers = append(
		state.AlcoholModelNumbers,
		ProductBlueprintDetailAlcoholModelNumber{
			Kind:            string(modeldom.ModelVariationKindAlcohol),
			Volume:          volume,
			Code:            variation.ModelNumber,
			ShippingPackage: buildProductBlueprintDetailShippingPackage(variation.ShippingPackage),
		},
	)
}

func buildProductBlueprintDetailShippingPackage(
	shippingPackage modeldom.ShippingPackage,
) ProductBlueprintDetailShippingPackage {
	return ProductBlueprintDetailShippingPackage{
		WeightGrams: shippingPackage.WeightGrams,
		WidthMM:     shippingPackage.WidthMM,
		LengthMM:    shippingPackage.LengthMM,
		HeightMM:    shippingPackage.HeightMM,
	}
}

func buildProductBlueprintDetailSizeRow(
	id string,
	sizeLabel string,
	measurements modeldom.Measurements,
	categoryCode string,
) ProductBlueprintDetailSizeRow {
	row := ProductBlueprintDetailSizeRow{
		ID:        id,
		SizeLabel: sizeLabel,
	}

	switch categoryCode {
	case "apparel.tops":
		row.Length = measurementPointer(measurements, "着丈")
		row.Width = measurementPointer(measurements, "身幅")
		row.Chest = measurementPointer(measurements, "胸囲")
		row.Shoulder = measurementPointer(measurements, "肩幅")
		row.SleeveLength = measurementPointer(measurements, "袖丈")

	case "apparel.bottoms":
		row.Waist = measurementPointer(measurements, "ウエスト")
		row.Hip = measurementPointer(measurements, "ヒップ")
		row.Rise = measurementPointer(measurements, "股上")
		row.Inseam = measurementPointer(measurements, "股下")
		row.Thigh = measurementPointer(measurements, "わたり幅")
		row.HemWidth = measurementPointer(measurements, "裾幅")

	case "apparel.dress":
		row.Length = measurementPointer(measurements, "着丈")
		row.Width = measurementPointer(measurements, "身幅")
		row.Chest = measurementPointer(measurements, "胸囲")
		row.Shoulder = measurementPointer(measurements, "肩幅")
		row.SleeveLength = measurementPointer(measurements, "袖丈")
		row.Waist = measurementPointer(measurements, "ウエスト")
		row.Hip = measurementPointer(measurements, "ヒップ")
	}

	return row
}

func measurementPointer(measurements modeldom.Measurements, key string) *int {
	if measurements == nil {
		return nil
	}

	value, exists := measurements[key]
	if !exists {
		return nil
	}

	return &value
}

func orderModelVariationsByModelRefs(
	variations []modeldom.ModelVariation,
	modelRefs []productbpdom.ModelRef,
) []modeldom.ModelVariation {
	if len(variations) == 0 {
		return []modeldom.ModelVariation{}
	}

	if len(modelRefs) == 0 {
		return append([]modeldom.ModelVariation(nil), variations...)
	}

	refs := append([]productbpdom.ModelRef(nil), modelRefs...)
	sort.SliceStable(refs, func(i, j int) bool {
		return refs[i].DisplayOrder < refs[j].DisplayOrder
	})

	byID := make(map[string]modeldom.ModelVariation, len(variations))
	for _, variation := range variations {
		if variation == nil {
			continue
		}

		id := variation.GetID()
		if id == "" {
			continue
		}

		if _, exists := byID[id]; exists {
			continue
		}

		byID[id] = variation
	}

	ordered := make([]modeldom.ModelVariation, 0, len(variations))
	used := make(map[string]struct{}, len(variations))

	for _, ref := range refs {
		if ref.ModelID == "" {
			continue
		}

		variation, exists := byID[ref.ModelID]
		if !exists {
			continue
		}

		if _, exists := used[ref.ModelID]; exists {
			continue
		}

		used[ref.ModelID] = struct{}{}
		ordered = append(ordered, variation)
	}

	for _, variation := range variations {
		if variation == nil {
			continue
		}

		id := variation.GetID()
		if id == "" {
			continue
		}

		if _, exists := used[id]; exists {
			continue
		}

		used[id] = struct{}{}
		ordered = append(ordered, variation)
	}

	return ordered
}

func (q *ProductBlueprintDetailQuery) resolveCompanyID(ctx context.Context) string {
	if q.companyIDFromContext == nil {
		return ""
	}

	return q.companyIDFromContext(ctx)
}
