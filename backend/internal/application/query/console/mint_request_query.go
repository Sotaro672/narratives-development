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
	modeldom "narratives/internal/domain/model"
)

var ErrMintRequestQueryServiceNotConfigured = errors.New("mintRequest query service is not configured")

type ModelVariationsGetter interface {
	GetModelVariations(ctx context.Context, productID string) ([]modeldom.ModelVariation, error)
}

type MintRequestQueryService struct {
	mintUC       *mintapp.MintUsecase
	productionUC *mintapp.ProductionUsecase
	nameResolver *resolver.NameResolver
	modelRepo    ModelVariationsGetter
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
		modelRepo:    nil,
	}
}

func (s *MintRequestQueryService) SetModelRepo(modelRepo ModelVariationsGetter) {
	if s == nil {
		return
	}
	s.modelRepo = modelRepo
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

func (s *MintRequestQueryService) ListMintListRowsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]querydto.MintListRowDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	mintsByProductionID, err := s.mintUC.ListMintsByProductionIDs(ctx, productionIDs)
	if err != nil {
		return nil, err
	}

	if len(mintsByProductionID) == 0 {
		return map[string]querydto.MintListRowDTO{}, nil
	}

	keys := make([]string, 0, len(mintsByProductionID))
	for k := range mintsByProductionID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]querydto.MintListRowDTO, len(mintsByProductionID))

	for _, productionID := range keys {
		m := mintsByProductionID[productionID]

		tokenName := resolveTokenName(ctx, s.nameResolver, m.TokenBlueprintID)
		createdByName := resolveMemberName(ctx, s.nameResolver, m.CreatedBy)

		var mintedAt *string
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			v := m.MintedAt.UTC().Format(time.RFC3339)
			mintedAt = &v
		}

		out[productionID] = querydto.MintListRowDTO{
			InspectionID:   productionID,
			MintID:         m.ID,
			TokenBlueprint: m.TokenBlueprintID,
			TokenName:      tokenName,
			CreatedByName:  createdByName,
			MintedAt:       mintedAt,
		}
	}

	return out, nil
}

func (s *MintRequestQueryService) ListMintDTOsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]querydto.MintDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	mintsByProductionID, err := s.mintUC.ListMintsByProductionIDs(ctx, productionIDs)
	if err != nil {
		return nil, err
	}

	if len(mintsByProductionID) == 0 {
		return map[string]querydto.MintDTO{}, nil
	}

	listRows, _ := s.ListMintListRowsByProductionIDs(ctx, productionIDs)

	keys := make([]string, 0, len(mintsByProductionID))
	for k := range mintsByProductionID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]querydto.MintDTO, len(mintsByProductionID))

	for _, productionID := range keys {
		m := mintsByProductionID[productionID]

		createdByName := m.CreatedBy
		tokenName := m.TokenBlueprintID
		if row, ok := listRows[productionID]; ok {
			if row.CreatedByName != "" {
				createdByName = row.CreatedByName
			}
			if row.TokenName != "" {
				tokenName = row.TokenName
			}
		}

		out[productionID] = buildMintDTO(productionID, m, tokenName, createdByName)
	}

	return out, nil
}

func (s *MintRequestQueryService) GetMintByProductionID(
	ctx context.Context,
	productionID string,
) (*querydto.MintDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}
	if productionID == "" {
		return nil, errors.New("productionId is empty")
	}

	mintsByProductionID, err := s.mintUC.ListMintsByProductionIDs(ctx, []string{productionID})
	if err != nil {
		return nil, err
	}

	m, ok := mintsByProductionID[productionID]
	if !ok {
		return nil, mintdom.ErrNotFound
	}

	tokenName := resolveTokenName(ctx, s.nameResolver, m.TokenBlueprintID)
	createdByName := resolveMemberName(ctx, s.nameResolver, m.CreatedBy)

	out := buildMintDTO(productionID, m, tokenName, createdByName)
	return &out, nil
}

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

	productBlueprintID := prod.ProductBlueprintID

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

	modelMeta := map[string]querydto.MintModelMetaEntry(nil)

	if productBlueprintID != "" && s.modelRepo != nil {
		vars, vErr := s.modelRepo.GetModelVariations(ctx, productBlueprintID)
		if vErr == nil {
			tmp := make(map[string]querydto.MintModelMetaEntry, len(vars))
			for _, raw := range vars {
				v, ok := toApparelModelVariation(raw)
				if !ok {
					continue
				}

				id := v.ID
				if id == "" {
					continue
				}

				rgb := v.Color.RGB
				tmp[id] = querydto.MintModelMetaEntry{
					ModelID:     id,
					ModelNumber: v.ModelNumber,
					Size:        v.Size,
					ColorName:   v.Color.Name,
					RGB:         intPtr(rgb),
				}
			}
			modelMeta = tmp
		}
	}

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
			mid := it.ModelID

			row := querydto.InspectionItemDTO{
				ProductID:        it.ProductID,
				ModelID:          mid,
				ModelNumber:      it.ModelNumber,
				Size:             it.Size,
				Color:            it.Color,
				RGB:              it.RGB,
				InspectionResult: it.InspectionResult,
				InspectedBy:      it.InspectedBy,
				InspectedAt:      it.InspectedAt,
			}

			if mid != "" && modelMeta != nil {
				if mm, ok := modelMeta[mid]; ok {
					if mm.ModelNumber != "" {
						row.ModelNumber = mm.ModelNumber
					}
					if mm.Size != "" {
						row.Size = mm.Size
					}
					if mm.ColorName != "" {
						row.Color = mm.ColorName
					}
					if mm.RGB != nil {
						row.RGB = mm.RGB
					}
				}
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
		ModelMeta:      modelMeta,
		TokenBlueprint: nil,
	}

	return out, nil
}

func buildMintDTO(
	productionID string,
	m mintdom.Mint,
	tokenName string,
	createdByName string,
) querydto.MintDTO {
	products := make([]string, 0, len(m.Products))
	products = append(products, m.Products...)

	var createdAt *string
	if !m.CreatedAt.IsZero() {
		v := m.CreatedAt.UTC().Format(time.RFC3339)
		createdAt = &v
	}

	var mintedAt *string
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		v := m.MintedAt.UTC().Format(time.RFC3339)
		mintedAt = &v
	}

	var scheduledBurnDate *string
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		v := m.ScheduledBurnDate.UTC().Format(time.RFC3339)
		scheduledBurnDate = &v
	}

	return querydto.MintDTO{
		ID:                 m.ID,
		InspectionID:       productionID,
		BrandID:            m.BrandID,
		TokenBlueprintID:   m.TokenBlueprintID,
		TokenName:          tokenName,
		Products:           products,
		CreatedBy:          m.CreatedBy,
		CreatedByName:      createdByName,
		CreatedAt:          createdAt,
		Minted:             m.Minted,
		MintedAt:           mintedAt,
		ScheduledBurnDate:  scheduledBurnDate,
		OnChainTxSignature: m.OnChainTxSignature,
	}
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

func resolveMemberName(
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

	name := nameResolver.ResolveMemberName(ctx, memberID)
	if name == "" {
		return memberID
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

func intPtr(n int) *int {
	v := n
	return &v
}

func toApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
	if v == nil {
		return modeldom.ApparelModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x, true
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return modeldom.ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.ApparelModelVariation{}, false
	}
}
