// backend/internal/application/inspection/dto/list.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ==============================
// Inspector Screen DTO
//  - InspectionBatch + Mint (joined)
// ==============================

type InspectionItemDTO struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId"`
	InspectionResult *string `json:"inspectionResult,omitempty"`
	InspectedBy      *string `json:"inspectedBy,omitempty"`
	InspectedAt      *string `json:"inspectedAt,omitempty"` // RFC3339
}

type InspectionBatchForScreenDTO struct {
	ProductionID string              `json:"productionId"`
	MintID       *string             `json:"mintId,omitempty"` // ★ pointer に合わせる
	Quantity     int                 `json:"quantity"`
	Status       string              `json:"status"`
	TotalPassed  int                 `json:"totalPassed"`
	Inspections  []InspectionItemDTO `json:"inspections"`

	// ★ JOINED
	Mint *MintDTO `json:"mint,omitempty"`
}

func NewInspectionBatchForScreenDTO(b inspectiondom.InspectionBatch, mint *MintDTO) InspectionBatchForScreenDTO {
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
		MintID:       b.MintID, // ★ そのまま渡せる
		Quantity:     b.Quantity,
		Status:       string(b.Status),
		TotalPassed:  b.TotalPassed,
		Inspections:  items,
		Mint:         mint,
	}
}

// ==============================
// Mint DTO (screen)
// ==============================

type MintDTO struct {
	MintID            string  `json:"mintId"`
	InspectionID      string  `json:"inspectionId"` // ★ Mint から取らず、呼び出し元で埋める
	BrandID           string  `json:"brandId"`
	TokenBlueprintID  string  `json:"tokenBlueprintId"`
	CreatedAt         string  `json:"createdAt"` // RFC3339
	CreatedBy         string  `json:"createdBy"`
	Minted            bool    `json:"minted"`
	MintedAt          *string `json:"mintedAt,omitempty"`          // RFC3339
	ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"` // RFC3339
}

// ★ inspectionID は batch.ProductionID (= inspectionId 扱い) を渡す想定
func NewMintDTO(m mintdom.Mint, inspectionID string) MintDTO {
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
		InspectionID:      inspectionID,
		BrandID:           m.BrandID,
		TokenBlueprintID:  m.TokenBlueprintID,
		CreatedAt:         createdAt,
		CreatedBy:         m.CreatedBy,
		Minted:            m.Minted,
		MintedAt:          mintedAt,
		ScheduledBurnDate: scheduled,
	}
}
