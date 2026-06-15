package query

import (
	"context"
	"errors"
	"sort"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	inspectiondom "narratives/internal/domain/inspection"
	memberdom "narratives/internal/domain/member"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var ErrMintRequestQueryServiceNotConfigured = errors.New("mintRequest query service is not configured")

type MintRequestQueryService struct {
	productionQuery *CompanyProductionQueryService

	mintRepo   mintdom.MintRepository
	inspRepo   mintdom.MintInspectionRepo
	pbRepo     mintdom.MintProductBlueprintRepo
	tbRepo     tbdom.RepositoryPort
	brandRepo  branddom.Repository
	memberRepo memberdom.Repository
}

func NewMintRequestQueryService(
	productionQuery *CompanyProductionQueryService,
	mintRepo mintdom.MintRepository,
	inspRepo mintdom.MintInspectionRepo,
	pbRepo mintdom.MintProductBlueprintRepo,
	tbRepo tbdom.RepositoryPort,
	brandRepo branddom.Repository,
	memberRepo memberdom.Repository,
) *MintRequestQueryService {
	return &MintRequestQueryService{
		productionQuery: productionQuery,
		mintRepo:        mintRepo,
		inspRepo:        inspRepo,
		pbRepo:          pbRepo,
		tbRepo:          tbRepo,
		brandRepo:       brandRepo,
		memberRepo:      memberRepo,
	}
}

func (s *MintRequestQueryService) ListMintRequestManagementRows(
	ctx context.Context,
	input querydto.ListMintRequestManagementRowsInput,
) ([]querydto.ProductionInspectionMintDTO, error) {
	if s == nil || s.productionQuery == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	filterSet := makeIDSet(input.ProductionIDs)

	prods, err := s.productionQuery.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(prods))
	prodByID := make(map[string]ProductionListItemDTO, len(prods))
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

	batches, err := s.listInspectionBatchesByProductionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	inspByPID := make(map[string]inspectiondom.InspectionBatch, len(batches))
	for _, b := range batches {
		pid := b.ProductionID
		if pid == "" {
			continue
		}
		inspByPID[pid] = b
	}

	mintsByPID, err := s.listMintsByProductionIDs(ctx, ids)
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
		prodQty := p.TotalQuantity
		inspStatus := "notYet"

		if hasInsp {
			mintQty = insp.TotalPassed
			if insp.Status != "" {
				inspStatus = string(insp.Status)
			}
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

			tokenName = s.resolveTokenName(ctx, tokenBlueprintID)
			requestedByName = s.resolveMemberNameByID(ctx, requestedBy)
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

func (s *MintRequestQueryService) ListInspectionBatchesForMint(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {
	return s.listInspectionBatchesByProductionIDs(ctx, productionIDs)
}

func (s *MintRequestQueryService) ListBrandsForMint(
	ctx context.Context,
) (branddom.PageResult[branddom.Brand], error) {
	var empty branddom.PageResult[branddom.Brand]

	if s == nil || s.brandRepo == nil {
		return empty, ErrMintRequestQueryServiceNotConfigured
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return empty, usecase.ErrCompanyIDMissing
	}

	return s.brandRepo.ListByCompanyID(ctx, companyID, branddom.Page{})
}

func (s *MintRequestQueryService) listMintsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]mintdom.Mint, error) {
	if s == nil || s.mintRepo == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return map[string]mintdom.Mint{}, nil
	}

	sort.Strings(ids)

	out := make(map[string]mintdom.Mint, len(ids))
	for _, id := range ids {
		m, err := s.mintRepo.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, mintdom.ErrNotFound) {
				continue
			}
			return nil, err
		}

		out[id] = m
	}

	return out, nil
}

func (s *MintRequestQueryService) listInspectionBatchesByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {
	if s == nil || s.inspRepo == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	sort.Strings(ids)

	out := make([]inspectiondom.InspectionBatch, 0, len(ids))
	for _, id := range ids {
		batch, err := s.inspRepo.GetByProductionID(ctx, id)
		if err != nil {
			if errors.Is(err, inspectiondom.ErrNotFound) {
				continue
			}
			return nil, err
		}

		out = append(out, batch)
	}

	return out, nil
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

func (s *MintRequestQueryService) resolveTokenName(
	ctx context.Context,
	tokenBlueprintID string,
) string {
	if tokenBlueprintID == "" {
		return ""
	}
	if s == nil || s.tbRepo == nil {
		return tokenBlueprintID
	}

	tb, err := s.tbRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return tokenBlueprintID
	}

	if tb.Name == "" {
		return tokenBlueprintID
	}

	return tb.Name
}

func (s *MintRequestQueryService) resolveMemberNameByID(
	ctx context.Context,
	memberID string,
) string {
	if memberID == "" {
		return ""
	}
	if s == nil || s.memberRepo == nil {
		return memberID
	}

	rec, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return memberID
	}

	name := memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)
	if name == "" {
		return memberID
	}

	return name
}

func pageFromMintInput(input querydto.ListTokenBlueprintsForMintInput) domcommon.Page {
	page := input.Page
	perPage := input.PerPage

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 100
	}

	return domcommon.Page{
		Number:  page,
		PerPage: perPage,
	}
}
