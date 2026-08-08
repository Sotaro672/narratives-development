// backend/internal/application/query/console/dto/mint_request_detail_dto.go
package dto

// MintModelMetaEntry is a per-model metadata entry for mint request detail page.
// Keyed by modelId (variationId) on the wire.
type MintModelMetaEntry struct {
	ModelID     string `json:"modelId"`
	ModelNumber string `json:"modelNumber,omitempty"`
	Size        string `json:"size,omitempty"`
	ColorName   string `json:"colorName,omitempty"`
	RGB         *int   `json:"rgb,omitempty"`
}

// MintTaskProgressDTO is progress information calculated from
// mints/{mintID}/products subcollection.
type MintTaskProgressDTO struct {
	Total           int `json:"total"`
	Pending         int `json:"pending"`
	Minting         int `json:"minting"`
	Minted          int `json:"minted"`
	FailedRetryable int `json:"failedRetryable"`
	FailedFatal     int `json:"failedFatal"`
	Percentage      int `json:"percentage"`
}

// MintRequestDetailDTO is a detail DTO for mint request detail page.
// Mint information is fetched separately from /mint/requests.
type MintRequestDetailDTO struct {
	ProductName string `json:"productName"`

	ModelMeta map[string]MintModelMetaEntry `json:"modelMeta,omitempty"`

	Inspection *InspectionSummaryDTO `json:"inspection,omitempty"`
}

type InspectionItemDTO struct {
	ProductID string `json:"productId,omitempty"`
	ModelID   string `json:"modelId"`

	InspectionResult string `json:"inspectionResult,omitempty"`
	InspectedBy      string `json:"inspectedBy,omitempty"`
	InspectedAt      string `json:"inspectedAt,omitempty"`
}

type InspectionSummaryDTO struct {
	ProductionID string `json:"productionId"`
	Status       string `json:"status"`
	TotalPassed  int    `json:"totalPassed"`
	Quantity     int    `json:"quantity"`

	Inspections []InspectionItemDTO `json:"inspections,omitempty"`
}
