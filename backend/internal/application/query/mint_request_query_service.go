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
	modeldom "narratives/internal/domain/model"
)

var ErrMintRequestQueryServiceNotConfigured = errors.New("mintRequest query service is not configured")

// ------------------------------------------------------------
// Optional dependency: model variations lister
// - Firestore 実装(ModelRepositoryFS) の ListModelVariationsByProductBlueprintID を
//   そのまま差し込めるように “最小インターフェース” を定義する
// ------------------------------------------------------------

type ModelVariationsLister interface {
	ListModelVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error)
}

// MintRequestQueryService is used by /mint/requests handler.
// It returns management rows: (productionId = docId) + inspection + mint summary.
type MintRequestQueryService struct {
	mintUC       *mintapp.MintUsecase
	productionUC *productionapp.ProductionUsecase
	nameResolver *resolver.NameResolver

	// ★追加: productBlueprintId -> modelVariations を引くため（任意）
	modelRepo ModelVariationsLister
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
		modelRepo:    nil,
	}
}

// ★追加: DI 側で後から差し込めるようにする（既存 constructor を壊さない）
func (s *MintRequestQueryService) SetModelRepo(modelRepo ModelVariationsLister) {
	if s == nil {
		return
	}
	s.modelRepo = modelRepo
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

		// ✅ ProductBlueprintID が正（ここに一本化）
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
	// ------------------------------------------------------------
	start = time.Now()
	batchesAny, err := s.mintUC.ListInspectionBatchesByProductionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// InspectionBatch の実体(struct)に依存せず JSON 経由で吸収する（型ズレ回避）
	type inspectionLite struct {
		ProductionID string `json:"productionId"`
		Status       string `json:"status"`
		TotalPassed  int    `json:"totalPassed"`
		Quantity     int    `json:"quantity"`
		MintID       string `json:"mintId"`
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

	const tokenNameMissLogLimit = 10
	tokenNameMissCount := 0

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

		tokenBlueprintID := ""
		tokenName := ""
		requestedBy := ""
		createdByName := ""
		var mintedAt *time.Time

		if hasMint {
			requestedBy = strings.TrimSpace(m.CreatedBy)
			mintedAt = m.MintedAt
			tokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)

			if s.nameResolver != nil && tokenBlueprintID != "" {
				tokenName = strings.TrimSpace(s.nameResolver.ResolveTokenName(ctx, tokenBlueprintID))
				if tokenName == "" && tokenNameMissCount < tokenNameMissLogLimit {
					tokenNameMissCount++
					log.Printf("[mint_request_qs] WARN: tokenName not resolved tokenBlueprintId=%q (will fallback to id)", tokenBlueprintID)
				}
			}
			if tokenName == "" {
				tokenName = tokenBlueprintID
			}

			if s.nameResolver != nil {
				cb := requestedBy
				createdByName = strings.TrimSpace(s.nameResolver.ResolveCreatedByName(ctx, &cb))
			}
			if createdByName == "" {
				createdByName = requestedBy
			}
		}

		rows = append(rows, querydto.ProductionInspectionMintDTO{
			ID:           pid,
			ProductionID: pid,

			TokenBlueprintID: tokenBlueprintID,

			TokenName:          tokenName,
			ProductName:        productName,
			MintQuantity:       mintQty,
			ProductionQuantity: prodQty,
			InspectionStatus:   inspStatus,
			RequestedBy:        requestedBy,
			CreatedByName:      createdByName,
			MintedAt:           mintedAt,

			Inspection: nil,
			Mint:       mintPtr,
		})
	}

	log.Printf("[mint_request_qs] rows built len=%d sampleRow[0]=%s",
		len(rows), toJSONForLog(sampleFirst(rows), 1500),
	)

	return rows, nil
}

// GetMintRequestDetail returns detail DTO for a single productionId (= inspectionId = docId).
// detail は productionId をキーに必要データを API で取り直す backend 側実装。
// - production: productionUC.ListWithAssigneeName から 1件抽出
// - inspection: mintUC.ListInspectionBatchesByProductionIDs([pid])
// - mint: mintUC.ListMintsByInspectionIDs([pid])
// - modelMeta: (任意) modelRepo.ListModelVariationsByProductBlueprintID
func (s *MintRequestQueryService) GetMintRequestDetail(
	ctx context.Context,
	productionID string,
) (*querydto.MintRequestDetailDTO, error) {
	if s == nil || s.mintUC == nil || s.productionUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return nil, errors.New("productionId is empty")
	}

	start := time.Now()

	// ------------------------------------------------------------
	// 1) production (company-scoped)
	// ------------------------------------------------------------
	prodsAny, err := s.productionUC.ListWithAssigneeName(ctx)
	if err != nil {
		return nil, err
	}

	type prodLite struct {
		ID                 string `json:"id"`
		Quantity           int    `json:"quantity"`
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
		if strings.TrimSpace(p.ID) == pid {
			prod = p
			foundProd = true
			break
		}
	}
	if !foundProd {
		return nil, errors.New("production not found")
	}

	productBlueprintID := strings.TrimSpace(prod.ProductBlueprintID)
	log.Printf("[mint_request_qs] detail pid=%q production resolved productBlueprintId=%q", pid, productBlueprintID)

	// ------------------------------------------------------------
	// 2) inspection (by pid)
	// ------------------------------------------------------------
	batchesAny, err := s.mintUC.ListInspectionBatchesByProductionIDs(ctx, []string{pid})
	if err != nil {
		return nil, err
	}

	type inspectionItemLite struct {
		ModelID          string `json:"modelId"`
		InspectionResult string `json:"inspectionResult"`
		RGB              *int   `json:"rgb,omitempty"`
		Size             string `json:"size,omitempty"`
		Color            string `json:"color,omitempty"`
		ModelNumber      string `json:"modelNumber,omitempty"`
	}
	type inspectionBatchLite struct {
		ProductionID string               `json:"productionId"`
		Status       string               `json:"status"`
		TotalPassed  int                  `json:"totalPassed"`
		Quantity     int                  `json:"quantity"`
		Inspections  []inspectionItemLite `json:"inspections"`
	}
	batches := make([]inspectionBatchLite, 0)
	if b, mErr := json.Marshal(batchesAny); mErr == nil {
		_ = json.Unmarshal(b, &batches)
	}

	var insp inspectionBatchLite
	hasInsp := false
	for _, b := range batches {
		if strings.TrimSpace(b.ProductionID) == pid {
			insp = b
			hasInsp = true
			break
		}
	}

	// ------------------------------------------------------------
	// 3) mint (by pid)
	// ------------------------------------------------------------
	mintsByPID, err := s.mintUC.ListMintsByInspectionIDs(ctx, []string{pid})
	if err != nil {
		return nil, err
	}
	m, hasMint := mintsByPID[pid]

	// ------------------------------------------------------------
	// 3.5) model variations -> modelMeta（任意）
	// ------------------------------------------------------------
	modelMeta := map[string]querydto.MintModelMetaEntry(nil)

	if productBlueprintID == "" {
		log.Printf("[mint_request_qs] WARN: productBlueprintId is empty, skip model variations (pid=%q)", pid)
	} else if s.modelRepo == nil {
		log.Printf("[mint_request_qs] WARN: modelRepo not configured, skip model variations (pid=%q pbId=%q)", pid, productBlueprintID)
	} else {
		vars, vErr := s.modelRepo.ListModelVariationsByProductBlueprintID(ctx, productBlueprintID)
		if vErr != nil {
			log.Printf("[mint_request_qs] WARN: ListModelVariationsByProductBlueprintID failed pid=%q pbId=%q err=%v", pid, productBlueprintID, vErr)
		} else {
			tmp := make(map[string]querydto.MintModelMetaEntry, len(vars))
			for _, v := range vars {
				id := strings.TrimSpace(v.ID)
				if id == "" {
					continue
				}
				rgb := v.Color.RGB
				tmp[id] = querydto.MintModelMetaEntry{
					ModelNumber: strings.TrimSpace(v.ModelNumber),
					Size:        strings.TrimSpace(v.Size),
					ColorName:   strings.TrimSpace(v.Color.Name),
					RGB:         intPtr(rgb),
				}
			}
			modelMeta = tmp
			log.Printf("[mint_request_qs] modelMeta built pbId=%q len=%d sampleKey=%q sampleVal=%s",
				productBlueprintID,
				len(modelMeta),
				firstKey(modelMeta),
				toJSONForLog(func() any {
					k := firstKey(modelMeta)
					if k == "" {
						return nil
					}
					return modelMeta[k]
				}(), 500),
			)
		}
	}

	// ------------------------------------------------------------
	// 4) compute detail fields
	// ------------------------------------------------------------
	productName := strings.TrimSpace(prod.ProductName)

	mintQty := 0
	prodQty := prod.Quantity
	inspStatus := "notYet"

	inspectionItems := make([]querydto.InspectionItemDTO, 0)

	if hasInsp {
		mintQty = insp.TotalPassed
		if strings.TrimSpace(insp.Status) != "" {
			inspStatus = strings.TrimSpace(insp.Status)
		}
		if insp.Quantity > 0 {
			prodQty = insp.Quantity
		}

		// inspections[] を DTO へ（modelMeta があれば上書き）
		for _, it := range insp.Inspections {
			mid := strings.TrimSpace(it.ModelID)
			ir := strings.TrimSpace(it.InspectionResult)

			row := querydto.InspectionItemDTO{
				ModelID:          mid,
				InspectionResult: ir,
			}

			// backend inspection item が modelNumber/size/color/rgb を持っているなら拾う（保険）
			row.ModelNumber = strings.TrimSpace(it.ModelNumber)
			row.Size = strings.TrimSpace(it.Size)
			row.Color = strings.TrimSpace(it.Color)
			row.RGB = it.RGB

			// modelMeta があればそれを最優先で埋める
			if mid != "" && modelMeta != nil {
				if mm, ok := modelMeta[mid]; ok {
					if strings.TrimSpace(mm.ModelNumber) != "" {
						row.ModelNumber = strings.TrimSpace(mm.ModelNumber)
					}
					if strings.TrimSpace(mm.Size) != "" {
						row.Size = strings.TrimSpace(mm.Size)
					}
					if strings.TrimSpace(mm.ColorName) != "" {
						row.Color = strings.TrimSpace(mm.ColorName)
					}
					if mm.RGB != nil {
						row.RGB = mm.RGB
					}
				}
			}

			inspectionItems = append(inspectionItems, row)
		}
	}

	// mint fields
	tokenBlueprintID := ""
	tokenName := ""
	requestedBy := ""
	createdByName := ""
	var mintedAt *time.Time
	var mintSummary *querydto.MintSummaryDTO

	if hasMint {
		requestedBy = strings.TrimSpace(m.CreatedBy)
		mintedAt = m.MintedAt
		tokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)

		if s.nameResolver != nil && tokenBlueprintID != "" {
			tokenName = strings.TrimSpace(s.nameResolver.ResolveTokenName(ctx, tokenBlueprintID))
		}
		if tokenName == "" {
			tokenName = tokenBlueprintID
		}

		if s.nameResolver != nil {
			cb := requestedBy
			createdByName = strings.TrimSpace(s.nameResolver.ResolveCreatedByName(ctx, &cb))
		}
		if createdByName == "" {
			createdByName = requestedBy
		}

		// mint -> safe summary（products の shape 揺れ回避のため json 経由）
		type mintLite struct {
			ID                string         `json:"id"`
			BrandID           string         `json:"brandId"`
			TokenBlueprintID  string         `json:"tokenBlueprintId"`
			Products          map[string]any `json:"products"`
			CreatedAt         *time.Time     `json:"createdAt"`
			CreatedBy         string         `json:"createdBy"`
			Minted            bool           `json:"minted"`
			MintedAt          *time.Time     `json:"mintedAt"`
			ScheduledBurnDate *time.Time     `json:"scheduledBurnDate"`
		}
		var ml mintLite
		if b, mErr := json.Marshal(m); mErr == nil {
			_ = json.Unmarshal(b, &ml)
		}

		productIDs := make([]string, 0)
		for k := range ml.Products {
			id := strings.TrimSpace(k)
			if id == "" {
				continue
			}
			productIDs = append(productIDs, id)
		}
		sort.Strings(productIDs)

		mintSummary = &querydto.MintSummaryDTO{
			ID:                strings.TrimSpace(ml.ID),
			BrandID:           strings.TrimSpace(ml.BrandID),
			TokenBlueprintID:  strings.TrimSpace(ml.TokenBlueprintID),
			CreatedBy:         strings.TrimSpace(ml.CreatedBy),
			CreatedByName:     strings.TrimSpace(createdByName),
			CreatedAt:         ml.CreatedAt,
			Minted:            ml.Minted,
			MintedAt:          ml.MintedAt,
			ScheduledBurnDate: ml.ScheduledBurnDate,
			ProductIDs:        productIDs,
		}
	}

	// production summary
	prodSummary := &querydto.ProductionSummaryDTO{
		ID:          strings.TrimSpace(prod.ID),
		ProductName: strings.TrimSpace(prod.ProductName),
		Quantity:    prodQty,
	}

	// inspection summary
	var inspSummary *querydto.InspectionSummaryDTO
	if hasInsp {
		inspSummary = &querydto.InspectionSummaryDTO{
			ProductionID: strings.TrimSpace(insp.ProductionID),
			Status:       strings.TrimSpace(insp.Status),
			TotalPassed:  insp.TotalPassed,
			Quantity:     insp.Quantity,
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
		RequestedBy:        requestedBy,
		CreatedByName:      createdByName,
		MintedAt:           mintedAt,
		Production:         prodSummary,
		Inspection:         inspSummary,
		Mint:               mintSummary,
		ModelMeta:          modelMeta,
		TokenBlueprint:     nil,
	}

	log.Printf("[mint_request_qs] detail built pid=%q elapsed=%s dto=%s",
		pid, time.Since(start), toJSONForLog(out, 1500),
	)

	return out, nil
}

// -----------------------
// helpers
// -----------------------

func intPtr(n int) *int {
	v := n
	return &v
}

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
