// backend/internal/application/query/console/dto/mint_request_detail_dto.go
package dto

import "time"

// MintModelMetaEntry is a per-model metadata entry for mint request detail page.
// Keyed by modelId (variationId) on the wire.
type MintModelMetaEntry struct {
	ModelID     string `json:"modelId"`
	ModelNumber string `json:"modelNumber,omitempty"`
	Size        string `json:"size,omitempty"`
	ColorName   string `json:"colorName,omitempty"`
	RGB         *int   `json:"rgb,omitempty"`
}

// MintRequestDetailDTO is a detail DTO for mint request detail page.
// Key is productionId (= inspectionId = mintId).
type MintRequestDetailDTO struct {
	ID           string `json:"id"`
	ProductionID string `json:"productionId"`

	ProductName string `json:"productName"`
	TokenName   string `json:"tokenName"`

	TokenBlueprintID string `json:"tokenBlueprintId"`

	MintQuantity       int `json:"mintQuantity"`
	ProductionQuantity int `json:"productionQuantity"`

	InspectionStatus string `json:"inspectionStatus"`

	RequestedBy     string `json:"requestedBy"`
	CreatedByName   string `json:"createdByName"`
	RequestedByName string `json:"requestedByName,omitempty"`

	MintedAt *time.Time `json:"mintedAt,omitempty"`

	ModelMeta map[string]MintModelMetaEntry `json:"modelMeta,omitempty"`

	Production     *ProductionSummaryDTO     `json:"production,omitempty"`
	Inspection     *InspectionSummaryDTO     `json:"inspection,omitempty"`
	Mint           *MintSummaryDTO           `json:"mint,omitempty"`
	TokenBlueprint *TokenBlueprintSummaryDTO `json:"tokenBlueprint,omitempty"`
}

type ProductionSummaryDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
}

type InspectionItemDTO struct {
	ProductID string `json:"productId,omitempty"`

	ModelID     string `json:"modelId"`
	ModelNumber string `json:"modelNumber,omitempty"`

	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`

	InspectionResult string `json:"inspectionResult,omitempty"`
	InspectedBy      string `json:"inspectedBy,omitempty"`
	InspectedAt      string `json:"inspectedAt,omitempty"`
}

type InspectionSummaryDTO struct {
	ProductionID    string     `json:"productionId"`
	Status          string     `json:"status"`
	TotalPassed     int        `json:"totalPassed"`
	Quantity        int        `json:"quantity"`
	ProductName     string     `json:"productName,omitempty"`
	InspectedBy     string     `json:"inspectedBy,omitempty"`
	InspectedByName string     `json:"inspectedByName,omitempty"`
	InspectedAt     *time.Time `json:"inspectedAt,omitempty"`

	Inspections []InspectionItemDTO `json:"inspections,omitempty"`
}

type MintSummaryDTO struct {
	ID                 string     `json:"id"`
	BrandID            string     `json:"brandId"`
	TokenBlueprintID   string     `json:"tokenBlueprintId"`
	CreatedBy          string     `json:"createdBy"`
	CreatedByName      string     `json:"createdByName,omitempty"`
	CreatedAt          *time.Time `json:"createdAt,omitempty"`
	Minted             bool       `json:"minted"`
	MintedAt           *time.Time `json:"mintedAt,omitempty"`
	ScheduledBurnDate  *time.Time `json:"scheduledBurnDate,omitempty"`
	ProductIDs         []string   `json:"productIds,omitempty"`
	OnChainTxSignature string     `json:"onChainTxSignature,omitempty"`
}

type TokenBlueprintSummaryDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
	BrandID string `json:"brandId,omitempty"`
	IconURL string `json:"iconUrl,omitempty"`
}
