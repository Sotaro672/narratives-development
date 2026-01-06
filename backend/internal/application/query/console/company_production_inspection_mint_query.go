// backend/internal/application/query/company_production_inspection_mint_query_service.go
package query

import (
	"context"
	"strings"

	qdto "narratives/internal/application/query/console/dto"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ============================================================
// Ports
// ============================================================

// ProductionIDProvider is satisfied by CompanyProductionQueryService.
type ProductionIDProvider interface {
	ListProductionIDsByCurrentCompany(ctx context.Context) ([]string, error)
}

// InspectionGetter reads inspections/{id}.
type InspectionGetter interface {
	GetByID(ctx context.Context, id string) (*inspectiondom.InspectionBatch, error)
}

// MintGetter reads mints/{id}.
type MintGetter interface {
	GetByID(ctx context.Context, id string) (*mintdom.Mint, error)
}

// ------------------------------------------------------------
// Optional adapters (when repositories return VALUE not pointer)
// ------------------------------------------------------------

// InspectionGetterFromValueRepo adapts GetByID(ctx,id) (InspectionBatch, error)
// into GetByID(ctx,id) (*InspectionBatch, error).
type InspectionGetterFromValueRepo struct {
	Repo interface {
		GetByID(ctx context.Context, id string) (inspectiondom.InspectionBatch, error)
	}
}

func (a *InspectionGetterFromValueRepo) GetByID(ctx context.Context, id string) (*inspectiondom.InspectionBatch, error) {
	v, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// MintGetterFromValueRepo adapts GetByID(ctx,id) (Mint, error)
// into GetByID(ctx,id) (*Mint, error).
type MintGetterFromValueRepo struct {
	Repo interface {
		GetByID(ctx context.Context, id string) (mintdom.Mint, error)
	}
}

func (a *MintGetterFromValueRepo) GetByID(ctx context.Context, id string) (*mintdom.Mint, error) {
	v, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ============================================================
// Service
// ============================================================

// CompanyProductionInspectionMintQueryService
// - gets productionIds from CompanyProductionQueryService (company boundary enforced there)
// - fetches inspections/{productionId} and mints/{productionId}
type CompanyProductionInspectionMintQueryService struct {
	prodIDs    ProductionIDProvider
	insGetter  InspectionGetter
	mintGetter MintGetter
}

func NewCompanyProductionInspectionMintQueryService(
	prodIDs ProductionIDProvider,
	insGetter InspectionGetter,
	mintGetter MintGetter,
) *CompanyProductionInspectionMintQueryService {
	return &CompanyProductionInspectionMintQueryService{
		prodIDs:    prodIDs,
		insGetter:  insGetter,
		mintGetter: mintGetter,
	}
}

// ListInspectionAndMintsByCurrentCompany returns rows keyed by productionId.
// Missing inspection/mint docs are returned as nil (not an error).
func (s *CompanyProductionInspectionMintQueryService) ListInspectionAndMintsByCurrentCompany(
	ctx context.Context,
) ([]qdto.ProductionInspectionMintDTO, error) {
	if s.prodIDs == nil {
		return nil, status.Error(codes.Internal, "prodIDs provider is nil")
	}

	ids, err := s.prodIDs.ListProductionIDsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []qdto.ProductionInspectionMintDTO{}, nil
	}

	out := make([]qdto.ProductionInspectionMintDTO, 0, len(ids))

	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}

		var ins *inspectiondom.InspectionBatch
		var mint *mintdom.Mint

		// inspections/{productionId}
		if s.insGetter != nil {
			v, err := s.insGetter.GetByID(ctx, id)
			if err != nil {
				if !isNotFound(err) {
					return nil, err
				}
			} else {
				ins = v
			}
		}

		// mints/{productionId}
		if s.mintGetter != nil {
			v, err := s.mintGetter.GetByID(ctx, id)
			if err != nil {
				if !isNotFound(err) {
					return nil, err
				}
			} else {
				mint = v
			}
		}

		out = append(out, qdto.ProductionInspectionMintDTO{
			ProductionID: id,
			Inspection:   ins,
			Mint:         mint,
		})
	}

	return out, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Firestore / gRPC
	if status.Code(err) == codes.NotFound {
		return true
	}
	// Some implementations return "not found" as plain text.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no such document")
}
