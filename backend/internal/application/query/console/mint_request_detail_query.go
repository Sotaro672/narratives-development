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

	batches, err := s.listInspectionBatchesByProductionIDs(ctx, []string{pid})
	if err != nil {
		return nil, err
	}

	type inspectionItemLite struct {
		ProductID        string `json:"productId,omitempty"`
		ModelID          string `json:"modelId"`
		InspectionResult string `json:"inspectionResult"`
		RGB              *int   `json:"rgb,omitempty"`
		Size             string `json:"size,omitempty"`
		Color            string `json:"color,omitempty"`
		ModelNumber      string `json:"modelNumber,omitempty"`
		InspectedBy      string `json:"inspectedBy,omitempty"`
		InspectedAt      string `json:"inspectedAt,omitempty"`
	}

	type inspectionBatchLite struct {
		ProductionID  string               `json:"productionId"`
		Status        string               `json:"status"`
		TotalPassed   int                  `json:"totalPassed"`
		TotalQuantity int                  `json:"totalQuantity"`
		Inspections   []inspectionItemLite `json:"inspections"`
	}

	inspectionBatches := make([]inspectionBatchLite, 0, len(batches))
	for _, b := range batches {
		row := inspectionBatchLite{
			ProductionID:  b.ProductionID,
			Status:        string(b.Status),
			TotalPassed:   b.TotalPassed,
			TotalQuantity: len(b.Inspections),
			Inspections:   make([]inspectionItemLite, 0, len(b.Inspections)),
		}

		for _, it := range b.Inspections {
			row.Inspections = append(row.Inspections, inspectionItemLite{
				ProductID:        it.ProductID,
				ModelID:          it.ModelID,
				InspectionResult: inspectionResultString(it.InspectionResult),
				RGB:              nil,
				Size:             "",
				Color:            "",
				ModelNumber:      "",
				InspectedBy:      stringPtrValue(it.InspectedBy),
				InspectedAt:      timePtrString(it.InspectedAt),
			})
		}

		inspectionBatches = append(inspectionBatches, row)
	}

	var insp inspectionBatchLite
	hasInsp := false
	for _, b := range inspectionBatches {
		if b.ProductionID == pid {
			insp = b
			hasInsp = true
			break
		}
	}

	mintsByPID, err := s.listMintsByProductionIDs(ctx, []string{pid})
	if err != nil {
		return nil, err
	}
	m, hasMint := mintsByPID[pid]

	productName := prod.ProductName

	mintQty := 0
	prodQty := prod.TotalQuantity
	inspStatus := "notYet"

	inspectionItems := make([]querydto.InspectionItemDTO, 0)

	if hasInsp {
		mintQty = insp.TotalPassed
		if insp.Status != "" {
			inspStatus = insp.Status
		}

		for _, it := range insp.Inspections {
			row := querydto.InspectionItemDTO{
				ProductID:        it.ProductID,
				ModelID:          it.ModelID,
				ModelNumber:      it.ModelNumber,
				Size:             it.Size,
				Color:            it.Color,
				RGB:              it.RGB,
				InspectionResult: it.InspectionResult,
				InspectedBy:      it.InspectedBy,
				InspectedAt:      it.InspectedAt,
			}

			inspectionItems = append(inspectionItems, row)
		}
	}

	tokenBlueprintID := ""
	tokenName := ""
	requestedBy := ""
	requestedByName := ""
	var mintedAt *time.Time
	var mintSummary *querydto.MintSummaryDTO

	if hasMint {
		requestedBy = m.CreatedBy
		mintedAt = m.MintedAt
		tokenBlueprintID = m.TokenBlueprintID

		tokenName = s.resolveTokenName(ctx, tokenBlueprintID)
		requestedByName = s.resolveMemberNameByID(ctx, requestedBy)

		products := make([]string, 0, len(m.Products))
		products = append(products, m.Products...)

		mintSummary = &querydto.MintSummaryDTO{
			ID:                 m.ID,
			BrandID:            m.BrandID,
			TokenBlueprintID:   m.TokenBlueprintID,
			CreatedBy:          m.CreatedBy,
			CreatedByName:      requestedByName,
			CreatedAt:          &m.CreatedAt,
			Minted:             m.Minted,
			MintedAt:           m.MintedAt,
			ScheduledBurnDate:  m.ScheduledBurnDate,
			ProductIDs:         products,
			OnChainTxSignature: m.OnChainTxSignature,
		}
	}

	prodSummary := &querydto.ProductionSummaryDTO{
		ID:          prod.ID,
		ProductName: prod.ProductName,
		Quantity:    prodQty,
	}

	var inspSummary *querydto.InspectionSummaryDTO
	if hasInsp {
		inspSummary = &querydto.InspectionSummaryDTO{
			ProductionID: insp.ProductionID,
			Status:       insp.Status,
			TotalPassed:  insp.TotalPassed,
			Quantity:     prodQty,
			ProductName:  "",
			Inspections:  inspectionItems,
		}
	}

	out := &querydto.MintRequestDetailDTO{
		ID:                 pid,
		ProductionID:       pid,
		ProductName:        productName,
		TokenName:          tokenName,
		TokenBlueprintID:   tokenBlueprintID,
		MintQuantity:       mintQty,
		ProductionQuantity: prodQty,
		InspectionStatus:   inspStatus,

		RequestedBy:     requestedBy,
		CreatedByName:   requestedByName,
		RequestedByName: requestedByName,

		MintedAt:       mintedAt,
		Production:     prodSummary,
		Inspection:     inspSummary,
		Mint:           mintSummary,
		ModelMeta:      nil,
		TokenBlueprint: nil,
	}

	return out, nil
}

func inspectionResultString(v *inspectiondom.InspectionResult) string {
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
