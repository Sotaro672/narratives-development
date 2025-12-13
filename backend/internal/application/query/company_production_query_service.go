package query

import (
	"context"
	"strings"
	"time"

	dto "narratives/internal/application/production/dto"
	resolver "narratives/internal/application/resolver"

	memberdom "narratives/internal/domain/member"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Ports (query service needs minimal read ports)
// ============================================================

type ProductBlueprintQueryRepo interface {
	// companyId → productBlueprintIds
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// productBlueprintId → BrandID 解決（brandName を引くため）
	// ※ 実装側が値返却 / ポインタ返却で揺れる場合があるため、
	//    ここは一旦「値返却」に寄せています（必要なら合わせて実装側を修正してください）。
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

type ProductionQueryRepo interface {
	// productBlueprintIds → productions
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]productiondom.Production, error)
}

// ============================================================
// Service
// ============================================================

// CompanyProductionQueryService enforces the ONLY list route:
// companyId -> productBlueprintIds -> productions.
//
// This service is meant for "query/read" usecases (list pages).
// It prevents any "list without companyId" leakage at the application boundary.
type CompanyProductionQueryService struct {
	pbRepo       ProductBlueprintQueryRepo
	prodRepo     ProductionQueryRepo
	nameResolver *resolver.NameResolver
	now          func() time.Time
}

func NewCompanyProductionQueryService(
	pbRepo ProductBlueprintQueryRepo,
	prodRepo ProductionQueryRepo,
	nameResolver *resolver.NameResolver,
) *CompanyProductionQueryService {
	return &CompanyProductionQueryService{
		pbRepo:       pbRepo,
		prodRepo:     prodRepo,
		nameResolver: nameResolver,
		now:          time.Now,
	}
}

// ============================================================
// Public APIs
// ============================================================

// ListProductionIDsByCurrentCompany returns production IDs only.
// Useful for select options etc.
func (s *CompanyProductionQueryService) ListProductionIDsByCurrentCompany(
	ctx context.Context,
) ([]string, error) {
	rows, err := s.listProductionsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, p := range rows {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

// ListProductionsWithAssigneeName is for GET /productions list page.
// It returns dto.ProductionListItemDTO (same DTO you already use).
func (s *CompanyProductionQueryService) ListProductionsWithAssigneeName(
	ctx context.Context,
) ([]dto.ProductionListItemDTO, error) {
	list, err := s.listProductionsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []dto.ProductionListItemDTO{}, nil
	}

	// cache: pbID -> brandID, brandID -> brandName
	pbBrandCache := map[string]string{}
	brandNameCache := map[string]string{}

	out := make([]dto.ProductionListItemDTO, 0, len(list))

	for _, p := range list {
		assigneeName := ""
		productName := ""
		brandID := ""
		brandName := ""

		// assignee name / product name (NameResolver)
		if s.nameResolver != nil {
			assigneeName = strings.TrimSpace(
				s.nameResolver.ResolveAssigneeName(ctx, strings.TrimSpace(p.AssigneeID)),
			)
			productName = strings.TrimSpace(
				s.nameResolver.ResolveProductName(ctx, strings.TrimSpace(p.ProductBlueprintID)),
			)
		}

		// brandID (pbRepo.GetByID)
		pbID := strings.TrimSpace(p.ProductBlueprintID)
		if pbID != "" && s.pbRepo != nil {
			if cached, ok := pbBrandCache[pbID]; ok {
				brandID = cached
			} else {
				pb, err := s.pbRepo.GetByID(ctx, pbID)
				if err == nil {
					brandID = strings.TrimSpace(extractBrandIDFromProductBlueprint(pb))
					pbBrandCache[pbID] = brandID
				}
			}
		}

		// brandName (NameResolver)
		if s.nameResolver != nil && strings.TrimSpace(brandID) != "" {
			if cached, ok := brandNameCache[brandID]; ok {
				brandName = cached
			} else {
				brandName = strings.TrimSpace(s.nameResolver.ResolveBrandName(ctx, brandID))
				brandNameCache[brandID] = brandName
			}
		}

		// total quantity
		totalQty := 0
		for _, mq := range p.Models {
			if mq.Quantity > 0 {
				totalQty += mq.Quantity
			}
		}

		// labels
		printedAtLabel := ""
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			printedAtLabel = p.PrintedAt.In(time.Local).Format("2006/01/02 15:04")
		}

		createdAtLabel := ""
		if !p.CreatedAt.IsZero() {
			createdAtLabel = p.CreatedAt.In(time.Local).Format("2006/01/02 15:04")
		}

		out = append(out, dto.ProductionListItemDTO{
			Production:     p,
			ProductName:    productName,
			BrandName:      brandName,
			AssigneeName:   assigneeName,
			TotalQuantity:  totalQty,
			PrintedAtLabel: printedAtLabel,
			CreatedAtLabel: createdAtLabel,
		})
	}

	return out, nil
}

// ============================================================
// Core (single allowed route)
// ============================================================

func (s *CompanyProductionQueryService) listProductionsByCurrentCompany(
	ctx context.Context,
) ([]productiondom.Production, error) {
	cid := strings.TrimSpace(companyIDFromContext(ctx))
	if cid == "" {
		// ★ companyId 無しの list を絶対禁止（全社漏洩の根本対策）
		return nil, productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return nil, productbpdom.ErrInternal
	}

	// 1) companyId → productBlueprintIds
	pbIDs, err := s.pbRepo.ListIDsByCompany(ctx, cid)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []productiondom.Production{}, nil
	}

	// 2) productBlueprintIds → productions
	rows, err := s.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []productiondom.Production{}, nil
	}

	// 3) safety: pbIDs set check
	set := make(map[string]struct{}, len(pbIDs))
	for _, id := range pbIDs {
		if tid := strings.TrimSpace(id); tid != "" {
			set[tid] = struct{}{}
		}
	}

	out := make([]productiondom.Production, 0, len(rows))
	for _, p := range rows {
		if _, ok := set[strings.TrimSpace(p.ProductBlueprintID)]; !ok {
			continue
		}
		out = append(out, p)
	}

	return out, nil
}

// ============================================================
// Helpers
// ============================================================

// extractBrandIDFromProductBlueprint absorbs possible "value vs pointer" drifts
// by keeping extraction in one place.
//
// If your productbpdom.ProductBlueprint is always a value type, this is trivial.
// If later you switch the port to return *ProductBlueprint, just overload here.
func extractBrandIDFromProductBlueprint(pb productbpdom.ProductBlueprint) string {
	// value case
	return strings.TrimSpace(pb.BrandID)
}

// ============================================================
// Context helper
// ============================================================

// companyIDFromContext extracts companyId from context.
// It tries common string keys and then falls back to currentMember.
func companyIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 1) common string keys
	for _, k := range []string{"companyId", "companyID", "company_id"} {
		if v := ctx.Value(k); v != nil {
			if s, ok := v.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					return t
				}
			}
		}
	}

	// 2) currentMember ( *memberdom.Member )
	for _, k := range []string{"currentMember", "member"} {
		if v := ctx.Value(k); v != nil {
			if m, ok := v.(*memberdom.Member); ok && m != nil {
				if t := strings.TrimSpace(m.CompanyID); t != "" {
					return t
				}
			}
		}
	}

	return ""
}
