// backend/internal/application/query/dto/mint_request_detail_dto.go
package dto

import "time"

// MintRequestDetailDTO is a detail DTO for mint request detail page.
// Key is productionId (= inspectionId = docId).
//
// Design goals:
// - Frontend can render detail page by only calling GET /mint/requests/{productionId}
// - Keep DTO independent from domain structs (no mintdom.Mint import)
// - Allow backward/forward compatibility via optional fields
type MintRequestDetailDTO struct {
	// stable ids
	ID           string `json:"id"`
	ProductionID string `json:"productionId"`

	// resolved display fields
	ProductName string `json:"productName"`
	TokenName   string `json:"tokenName"`

	// ids for navigation / updates
	TokenBlueprintID string `json:"tokenBlueprintId"`

	// quantities
	MintQuantity       int `json:"mintQuantity"`       // = inspection.totalPassed
	ProductionQuantity int `json:"productionQuantity"` // = inspection.quantity (fallback: production.quantity)

	// statuses
	InspectionStatus string `json:"inspectionStatus"` // inspecting/completed/notYet ...

	// requester (mint.createdBy)
	RequestedBy   string `json:"requestedBy"`
	CreatedByName string `json:"createdByName"` // resolved member name (compat: list uses this name)

	// minted timestamp (optional)
	MintedAt *time.Time `json:"mintedAt,omitempty"`

	// optional nested summaries for detail page
	Production     *ProductionSummaryDTO     `json:"production,omitempty"`
	Inspection     *InspectionSummaryDTO     `json:"inspection,omitempty"`
	Mint           *MintSummaryDTO           `json:"mint,omitempty"`
	TokenBlueprint *TokenBlueprintSummaryDTO `json:"tokenBlueprint,omitempty"`
}

// ProductionSummaryDTO is a minimal production summary for detail page.
type ProductionSummaryDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
}

// InspectionSummaryDTO is a minimal inspection summary for detail page.
type InspectionSummaryDTO struct {
	ProductionID    string     `json:"productionId"`
	Status          string     `json:"status"`
	TotalPassed     int        `json:"totalPassed"`
	Quantity        int        `json:"quantity"`
	ProductName     string     `json:"productName,omitempty"`
	InspectedBy     string     `json:"inspectedBy,omitempty"`
	InspectedByName string     `json:"inspectedByName,omitempty"`
	InspectedAt     *time.Time `json:"inspectedAt,omitempty"`
}

// MintSummaryDTO is a mint summary (safe for frontend).
// Note: Products is represented as productIds to avoid Firestore map-shape leaking to UI.
type MintSummaryDTO struct {
	ID                string     `json:"id"`
	BrandID           string     `json:"brandId"`
	TokenBlueprintID  string     `json:"tokenBlueprintId"`
	CreatedBy         string     `json:"createdBy"`
	CreatedByName     string     `json:"createdByName,omitempty"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	Minted            bool       `json:"minted"`
	MintedAt          *time.Time `json:"mintedAt,omitempty"`
	ScheduledBurnDate *time.Time `json:"scheduledBurnDate,omitempty"`
	ProductIDs        []string   `json:"productIds,omitempty"`
}

// TokenBlueprintSummaryDTO is an optional token blueprint summary for detail page.
type TokenBlueprintSummaryDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
	BrandID string `json:"brandId,omitempty"`
}
