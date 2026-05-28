// backend\internal\application\query\console\mint_request_detail_query.go
package query

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	querydto "narratives/internal/application/query/console/dto"
)

func (s *MintRequestQueryService) GetMintRequestDetail(
	ctx context.Context,
	productionID string,
) (*querydto.MintRequestDetailDTO, error) {
	if s == nil || s.mintUC == nil || s.productionUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	pid := productionID
	if pid == "" {
		return nil, errors.New("productionId is empty")
	}

	prodsAny, err := s.productionUC.ListWithAssigneeName(ctx)
	if err != nil {
		return nil, err
	}

	type prodLite struct {
		ID                 string `json:"id"`
		TotalQuantity      int    `json:"totalQuantity"`
		ProductName        string `json:"productName"`
		ProductBlueprintID string `json:"ProductBlueprintID"`
	}

	prods := make([]prodLite, 0)
	if b, mErr := json.Marshal(prodsAny); mErr == nil {
		_ = json.Unmarshal(b, &prods)
	}

	var prod prodLite
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

	batchesAny, err := s.mintUC.ListInspectionBatchesByProductionIDs(ctx, []string{pid})
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

	batches := make([]inspectionBatchLite, 0)
	if b, mErr := json.Marshal(batchesAny); mErr == nil {
		_ = json.Unmarshal(b, &batches)
	}

	var insp inspectionBatchLite
	hasInsp := false
	for _, b := range batches {
		if b.ProductionID == pid {
			insp = b
			hasInsp = true
			break
		}
	}

	mintsByPID, err := s.mintUC.ListMintsByProductionIDs(ctx, []string{pid})
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
		if insp.TotalQuantity > 0 {
			prodQty = insp.TotalQuantity
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

		tokenName = resolveTokenName(ctx, s.nameResolver, tokenBlueprintID)
		requestedByName = resolveRequestedByName(ctx, s.nameResolver, requestedBy)

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
			Quantity:     insp.TotalQuantity,
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
