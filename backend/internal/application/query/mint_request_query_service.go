// backend/internal/application/query/mint_request_query_service.go
package query

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strings"
	"time"

	mintapp "narratives/internal/application/mint"
	productionapp "narratives/internal/application/production"
	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	mintdom "narratives/internal/domain/mint"
)

var ErrMintRequestQueryServiceNotConfigured = errors.New("mintRequest query service is not configured")

// MintRequestQueryService is used by /mint/requests handler.
// It returns management rows: (productionId = docId) + inspection + mint summary.
type MintRequestQueryService struct {
	mintUC       *mintapp.MintUsecase
	productionUC *productionapp.ProductionUsecase
	nameResolver *resolver.NameResolver
}

func NewMintRequestQueryService(
	mintUC *mintapp.MintUsecase,
	productionUC *productionapp.ProductionUsecase,
	nameResolver *resolver.NameResolver,
) *MintRequestQueryService {
	return &MintRequestQueryService{
		mintUC:       mintUC,
		productionUC: productionUC,
		nameResolver: nameResolver,
	}
}

// ListMintRequestManagementRows returns rows for current company.
// Company boundary is expected to be enforced by UC layers (via ctx injected by AuthMiddleware).
func (s *MintRequestQueryService) ListMintRequestManagementRows(ctx context.Context) ([]querydto.ProductionInspectionMintDTO, error) {
	if s == nil || s.mintUC == nil || s.productionUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	// ------------------------------------------------------------
	// 1) productionIds: use ProductionUsecase (already company-scoped)
	// ------------------------------------------------------------
	start := time.Now()
	prodsAny, err := s.productionUC.ListWithAssigneeName(ctx)
	if err != nil {
		return nil, err
	}

	// Convert unknown production type -> lightweight struct via JSON.
	type prodLite struct {
		ID          string `json:"id"`
		Quantity    int    `json:"quantity"`
		ProductName string `json:"productName"`
	}
	prods := make([]prodLite, 0)
	if b, mErr := json.Marshal(prodsAny); mErr == nil {
		_ = json.Unmarshal(b, &prods)
	}

	ids := make([]string, 0, len(prods))
	prodByID := make(map[string]prodLite, len(prods))
	seen := make(map[string]struct{}, len(prods))
	for _, p := range prods {
		pid := strings.TrimSpace(p.ID)
		if pid == "" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
		prodByID[pid] = p
	}
	sort.Strings(ids)

	log.Printf("[mint_request_qs] productions resolved len=%d elapsed=%s sample[0..4]=%v",
		len(ids), time.Since(start), ids[:min(5, len(ids))],
	)

	if len(ids) == 0 {
		return []querydto.ProductionInspectionMintDTO{}, nil
	}

	// ------------------------------------------------------------
	// 2) inspections by productionIds (via mintUC)
	//    - do NOT depend on mintdto.InspectionBatchDTO (it does not exist)
	// ------------------------------------------------------------
	start = time.Now()
	batchesAny, err := s.mintUC.ListInspectionBatchesByProductionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	type inspectionLite struct {
		ProductionID string `json:"productionId"`
		Status       string `json:"status"`
		TotalPassed  int    `json:"totalPassed"`
		Quantity     int    `json:"quantity"`
		ProductName  string `json:"productName"`
	}
	batches := make([]inspectionLite, 0)
	if b, mErr := json.Marshal(batchesAny); mErr == nil {
		_ = json.Unmarshal(b, &batches)
	}

	inspByPID := make(map[string]inspectionLite, len(batches))
	for _, b := range batches {
		pid := strings.TrimSpace(b.ProductionID)
		if pid == "" {
			continue
		}
		inspByPID[pid] = b
	}

	log.Printf("[mint_request_qs] inspections resolved len=%d elapsed=%s sampleKey=%q",
		len(inspByPID), time.Since(start), firstKey(inspByPID),
	)

	// ------------------------------------------------------------
	// 3) mints by inspectionIds (= productionIds)
	// ------------------------------------------------------------
	start = time.Now()
	mintsByPID, err := s.mintUC.ListMintsByInspectionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	log.Printf("[mint_request_qs] mints resolved keys=%d elapsed=%s sampleKey=%q",
		len(mintsByPID), time.Since(start), firstKey(mintsByPID),
	)

	// ------------------------------------------------------------
	// 4) build rows (stable order by ids)
	// ------------------------------------------------------------
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

		productName := strings.TrimSpace(p.ProductName)
		if hasInsp && strings.TrimSpace(insp.ProductName) != "" {
			productName = strings.TrimSpace(insp.ProductName)
		}

		mintQty := 0
		prodQty := 0
		inspStatus := "notYet"
		if hasInsp {
			mintQty = insp.TotalPassed
			if strings.TrimSpace(insp.Status) != "" {
				inspStatus = strings.TrimSpace(insp.Status)
			}
			if insp.Quantity > 0 {
				prodQty = insp.Quantity
			}
		}
		if prodQty == 0 {
			prodQty = p.Quantity
		}

		// tokenName / requestedBy / createdByName / mintedAt
		tokenName := ""
		requestedBy := ""
		createdByName := ""
		var mintedAt *time.Time

		if hasMint {
			requestedBy = strings.TrimSpace(m.CreatedBy)
			mintedAt = m.MintedAt

			tbID := strings.TrimSpace(m.TokenBlueprintID)
			if s.nameResolver != nil && tbID != "" {
				tokenName = strings.TrimSpace(s.nameResolver.ResolveTokenName(ctx, tbID))
			}
			if tokenName == "" {
				tokenName = tbID
			}

			if s.nameResolver != nil && requestedBy != "" {
				createdByName = strings.TrimSpace(s.nameResolver.ResolveMemberName(ctx, requestedBy))
			}
			if createdByName == "" {
				createdByName = requestedBy
			}
		}

		rows = append(rows, querydto.ProductionInspectionMintDTO{
			ID:                 pid,
			ProductionID:       pid,
			TokenName:          tokenName,
			ProductName:        productName,
			MintQuantity:       mintQty,
			ProductionQuantity: prodQty,
			InspectionStatus:   inspStatus,
			RequestedBy:        requestedBy,
			CreatedByName:      createdByName,
			MintedAt:           mintedAt,
			Inspection:         nil,     // 型依存を避ける（必要なら別途ドメイン型で埋める）
			Mint:               mintPtr, // mint は domain 型で返して OK（デバッグ用）
		})
	}

	log.Printf("[mint_request_qs] rows built len=%d sampleRow[0]=%s",
		len(rows), toJSONForLog(sampleFirst(rows), 1500),
	)

	return rows, nil
}

// -----------------------
// helpers
// -----------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sampleFirst[T any](xs []T) any {
	if len(xs) == 0 {
		return nil
	}
	return xs[0]
}

func toJSONForLog(v any, max int) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "<marshal_error>"
	}
	s := string(b)
	if max > 0 && len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}

func firstKey[V any](m map[string]V) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}
