// backend/internal/application/query/console/mint_request_detail_query.go
package query

import (
	"context"
	"errors"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	inspectiondom "narratives/internal/domain/inspection"
)

func (s *MintRequestQueryService) GetMintRequestDetail(
	ctx context.Context,
	productionID string,
) (*querydto.MintRequestDetailDTO, error) {
	if s == nil || s.productionQuery == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	pid := productionID
	if pid == "" {
		return nil, errors.New("productionId is empty")
	}

	prods, err := s.productionQuery.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		return nil, err
	}

	var prod ProductionListItemDTO
	foundProd := false

	for _, p := range prods {
		if p.ID == pid {
			prod = p
			foundProd = true
			break
		}
	}

	if !foundProd {
		return nil, errors.New("production not found")
	}

	batches, err := s.listInspectionBatchesByProductionIDs(
		ctx,
		[]string{pid},
	)
	if err != nil {
		return nil, err
	}

	var insp inspectiondom.InspectionBatch
	hasInsp := false

	for _, b := range batches {
		if b.ProductionID == pid {
			insp = b
			hasInsp = true
			break
		}
	}

	productName := prod.ProductName
	prodQty := prod.TotalQuantity

	inspectionItems := make(
		[]querydto.InspectionItemDTO,
		0,
	)

	modelMeta := make(
		map[string]querydto.MintModelMetaEntry,
	)

	if hasInsp {
		inspectionItems = make(
			[]querydto.InspectionItemDTO,
			0,
			len(insp.Inspections),
		)

		for _, it := range insp.Inspections {
			row := querydto.InspectionItemDTO{
				ProductID: it.ProductID,
				ModelID:   it.ModelID,
				InspectionResult: inspectionResultString(
					it.InspectionResult,
				),
				InspectedBy: stringPtrValue(
					it.InspectedBy,
				),
				InspectedAt: timePtrString(
					it.InspectedAt,
				),
			}

			inspectionItems = append(
				inspectionItems,
				row,
			)

			if it.ModelID == "" {
				continue
			}

			if _, exists := modelMeta[it.ModelID]; exists {
				continue
			}

			if s.productionQuery.nameResolver == nil {
				continue
			}

			resolved :=
				s.productionQuery.nameResolver.ResolveModelResolved(
					ctx,
					it.ModelID,
				)

			modelMeta[it.ModelID] =
				querydto.MintModelMetaEntry{
					ModelID:     it.ModelID,
					Kind:        resolved.Kind,
					ModelNumber: resolved.ModelNumber,

					Size:      resolved.Size,
					ColorName: resolved.Color,
					RGB:       resolved.RGB,

					Volume:     resolved.VolumeValue,
					VolumeUnit: resolved.VolumeUnit,
				}
		}
	}

	var inspSummary *querydto.InspectionSummaryDTO

	if hasInsp {
		inspSummary = &querydto.InspectionSummaryDTO{
			ProductionID: insp.ProductionID,
			Status:       string(insp.Status),
			TotalPassed:  insp.TotalPassed,
			Quantity:     prodQty,
			Inspections:  inspectionItems,
		}
	}

	out := &querydto.MintRequestDetailDTO{
		ProductBlueprintID: prod.ProductBlueprintID,
		ProductName:        productName,
		ModelMeta:          modelMeta,
		Inspection:         inspSummary,
	}

	return out, nil
}

func inspectionResultString(
	v *inspectiondom.InspectionResult,
) string {
	if v == nil {
		return ""
	}

	return string(*v)
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func timePtrString(v *time.Time) string {
	if v == nil || v.IsZero() {
		return ""
	}

	return v.UTC().Format(time.RFC3339)
}
