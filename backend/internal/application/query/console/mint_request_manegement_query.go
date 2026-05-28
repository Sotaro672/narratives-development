// backend\internal\application\query\console\mint_request_management_query.go
package query

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	mintapp "narratives/internal/application/usecase"
	mintdom "narratives/internal/domain/mint"
)

var ErrMintRequestQueryServiceNotConfigured = errors.New("mintRequest query service is not configured")

type MintRequestQueryService struct {
	mintUC       *mintapp.MintUsecase
	productionUC *mintapp.ProductionUsecase
	nameResolver *resolver.NameResolver
}

func NewMintRequestQueryService(
	mintUC *mintapp.MintUsecase,
	productionUC *mintapp.ProductionUsecase,
	nameResolver *resolver.NameResolver,
) *MintRequestQueryService {
	return &MintRequestQueryService{
		mintUC:       mintUC,
		productionUC: productionUC,
		nameResolver: nameResolver,
	}
}

func (s *MintRequestQueryService) ListMintRequestManagementRows(
	ctx context.Context,
	input querydto.ListMintRequestManagementRowsInput,
) ([]querydto.ProductionInspectionMintDTO, error) {
	if s == nil || s.mintUC == nil || s.productionUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	filterSet := makeIDSet(input.ProductionIDs)

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

	ids := make([]string, 0, len(prods))
	prodByID := make(map[string]prodLite, len(prods))
	seen := make(map[string]struct{}, len(prods))

	for _, p := range prods {
		pid := p.ID
		if pid == "" {
			continue
		}

		if len(filterSet) > 0 {
			if _, ok := filterSet[pid]; !ok {
				continue
			}
		}

		if _, ok := seen[pid]; ok {
			continue
		}

		seen[pid] = struct{}{}
		ids = append(ids, pid)
		prodByID[pid] = p
	}

	sort.Strings(ids)

	if len(ids) == 0 {
		return []querydto.ProductionInspectionMintDTO{}, nil
	}

	batchesAny, err := s.mintUC.ListInspectionBatchesByProductionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	type inspectionLite struct {
		ProductionID  string `json:"productionId"`
		Status        string `json:"status"`
		TotalPassed   int    `json:"totalPassed"`
		TotalQuantity int    `json:"totalQuantity"`
		MintID        string `json:"mintId"`
	}

	batches := make([]inspectionLite, 0)
	if b, mErr := json.Marshal(batchesAny); mErr == nil {
		_ = json.Unmarshal(b, &batches)
	}

	inspByPID := make(map[string]inspectionLite, len(batches))
	for _, b := range batches {
		pid := b.ProductionID
		if pid == "" {
			continue
		}
		inspByPID[pid] = b
	}

	mintsByPID, err := s.mintUC.ListMintsByProductionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	rows := make([]querydto.ProductionInspectionMintDTO, 0, len(ids))

	for _, pid := range ids {
		p := prodByID[pid]
		insp, hasInsp := inspByPID[pid]

		m, hasMint := mintsByPID[pid]
		var mintPtr *mintdom.Mint
		if hasMint {
			tmp := m
			mintPtr = &tmp
		}

		mintQty := 0
		prodQty := 0
		inspStatus := "notYet"

		if hasInsp {
			mintQty = insp.TotalPassed
			if insp.Status != "" {
				inspStatus = insp.Status
			}
			if insp.TotalQuantity > 0 {
				prodQty = insp.TotalQuantity
			}
		}

		if prodQty == 0 {
			prodQty = p.TotalQuantity
		}

		tokenBlueprintID := ""
		tokenName := ""
		requestedBy := ""
		requestedByName := ""
		var mintedAt *time.Time

		if hasMint {
			requestedBy = m.CreatedBy
			mintedAt = m.MintedAt
			tokenBlueprintID = m.TokenBlueprintID

			tokenName = resolveTokenName(ctx, s.nameResolver, tokenBlueprintID)
			requestedByName = resolveRequestedByName(ctx, s.nameResolver, requestedBy)
		}

		rows = append(rows, querydto.ProductionInspectionMintDTO{
			ID:           pid,
			ProductionID: pid,

			TokenBlueprintID: tokenBlueprintID,
			TokenName:        tokenName,
			ProductName:      p.ProductName,

			MintQuantity:       mintQty,
			ProductionQuantity: prodQty,
			InspectionStatus:   inspStatus,

			RequestedBy:   requestedBy,
			CreatedByName: requestedByName,
			MintedAt:      mintedAt,

			Inspection: nil,
			Mint:       mintPtr,
		})
	}

	return rows, nil
}

func makeIDSet(ids []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func resolveTokenName(
	ctx context.Context,
	nameResolver *resolver.NameResolver,
	tokenBlueprintID string,
) string {
	if tokenBlueprintID == "" {
		return ""
	}
	if nameResolver == nil {
		return tokenBlueprintID
	}

	name := nameResolver.ResolveTokenName(ctx, tokenBlueprintID)
	if name == "" {
		return tokenBlueprintID
	}

	return name
}

func resolveRequestedByName(
	ctx context.Context,
	nameResolver *resolver.NameResolver,
	memberID string,
) string {
	if memberID == "" {
		return ""
	}
	if nameResolver == nil {
		return memberID
	}

	v := memberID
	name := nameResolver.ResolveRequestedByName(ctx, &v)
	if name == "" {
		return memberID
	}

	return name
}
