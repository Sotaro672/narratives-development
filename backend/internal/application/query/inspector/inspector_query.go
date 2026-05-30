// backend/internal/application/query/inspector/inspector_query.go
package inspector

import (
	"context"
	"fmt"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------

// InspectionRepository は inspector 画面表示 query が必要とする
// inspection 永続化ポートです。
type InspectionRepository interface {
	GetByProductionID(
		ctx context.Context,
		productionID string,
	) (inspectiondom.InspectionBatch, error)
}

// MintGetter は productionId / mintId 共通IDから mint を 1 件取得するための
// query 用ポートです。
//
// AMOL/Narratives では production / inspection / mint の docId は同一値として扱うため、
// mint repository 側は GetByID のままにします。
type MintGetter interface {
	GetByID(ctx context.Context, id string) (mintdom.Mint, error)
}

// ------------------------------------------------------------
// Query Service
// ------------------------------------------------------------

type QueryService struct {
	inspectionRepo InspectionRepository
	mintRepo       MintGetter // nil 許容
}

func NewQueryService(
	inspectionRepo InspectionRepository,
	mintRepo MintGetter,
) *QueryService {
	return &QueryService{
		inspectionRepo: inspectionRepo,
		mintRepo:       mintRepo,
	}
}

// GetByProductionID は productionId から InspectionBatch と関連 Mint を取得し、
// inspector 画面用 DTO として返します。
//
// production / inspection / mint の docId は同一値として扱うため、
// inspection 取得も mint join も productionID を起点にします。
func (q *QueryService) GetByProductionID(
	ctx context.Context,
	productionID string,
) (InspectionBatchForScreenDTO, error) {
	if q == nil || q.inspectionRepo == nil {
		return InspectionBatchForScreenDTO{}, fmt.Errorf("inspection query: inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return InspectionBatchForScreenDTO{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := q.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return InspectionBatchForScreenDTO{}, err
	}

	var mintDTO *MintDTO
	if q.mintRepo != nil {
		m, err := q.mintRepo.GetByID(ctx, pid)
		if err == nil {
			dto := NewMintDTO(m, pid)
			mintDTO = &dto
		} else if err != mintdom.ErrNotFound {
			return InspectionBatchForScreenDTO{}, err
		}
	}

	return NewInspectionBatchForScreenDTO(batch, mintDTO), nil
}

// ------------------------------------------------------------
// Inspector Screen DTO
// - InspectionBatch + Mint joined
// ------------------------------------------------------------

type InspectionItemDTO struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId"`
	InspectionResult *string `json:"inspectionResult,omitempty"`
	InspectedBy      *string `json:"inspectedBy,omitempty"`
	InspectedAt      *string `json:"inspectedAt,omitempty"` // RFC3339
}

type InspectionBatchForScreenDTO struct {
	ProductionID string              `json:"productionId"`
	MintID       *string             `json:"mintId,omitempty"`
	Quantity     int                 `json:"quantity"`
	Status       string              `json:"status"`
	TotalPassed  int                 `json:"totalPassed"`
	Inspections  []InspectionItemDTO `json:"inspections"`

	// joined
	Mint *MintDTO `json:"mint,omitempty"`
}

func NewInspectionBatchForScreenDTO(
	b inspectiondom.InspectionBatch,
	mint *MintDTO,
) InspectionBatchForScreenDTO {
	items := make([]InspectionItemDTO, 0, len(b.Inspections))

	for _, it := range b.Inspections {
		var res *string
		if it.InspectionResult != nil {
			s := string(*it.InspectionResult)
			res = &s
		}

		var inspectedAt *string
		if it.InspectedAt != nil && !it.InspectedAt.IsZero() {
			s := it.InspectedAt.UTC().Format(time.RFC3339)
			inspectedAt = &s
		}

		items = append(items, InspectionItemDTO{
			ProductID:        it.ProductID,
			ModelID:          it.ModelID,
			InspectionResult: res,
			InspectedBy:      it.InspectedBy,
			InspectedAt:      inspectedAt,
		})
	}

	return InspectionBatchForScreenDTO{
		ProductionID: b.ProductionID,
		MintID:       b.MintID,
		Quantity:     b.Quantity,
		Status:       string(b.Status),
		TotalPassed:  b.TotalPassed,
		Inspections:  items,
		Mint:         mint,
	}
}

// ------------------------------------------------------------
// Mint DTO
// ------------------------------------------------------------

type MintDTO struct {
	MintID            string  `json:"mintId"`
	ProductionID      string  `json:"productionId"`
	BrandID           string  `json:"brandId"`
	TokenBlueprintID  string  `json:"tokenBlueprintId"`
	CreatedAt         string  `json:"createdAt"` // RFC3339
	CreatedBy         string  `json:"createdBy"`
	Minted            bool    `json:"minted"`
	MintedAt          *string `json:"mintedAt,omitempty"`          // RFC3339
	ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"` // RFC3339
}

// NewMintDTO は productionID に batch.ProductionID を渡す想定です。
func NewMintDTO(m mintdom.Mint, productionID string) MintDTO {
	createdAt := ""
	if !m.CreatedAt.IsZero() {
		createdAt = m.CreatedAt.UTC().Format(time.RFC3339)
	}

	var mintedAt *string
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		s := m.MintedAt.UTC().Format(time.RFC3339)
		mintedAt = &s
	}

	var scheduled *string
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		s := m.ScheduledBurnDate.UTC().Format(time.RFC3339)
		scheduled = &s
	}

	return MintDTO{
		MintID:            m.ID,
		ProductionID:      productionID,
		BrandID:           m.BrandID,
		TokenBlueprintID:  m.TokenBlueprintID,
		CreatedAt:         createdAt,
		CreatedBy:         m.CreatedBy,
		Minted:            m.Minted,
		MintedAt:          mintedAt,
		ScheduledBurnDate: scheduled,
	}
}
