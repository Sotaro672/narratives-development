// backend/internal/application/query/dto/production_management.go
package dto

import (
	"time"

	productiondom "narratives/internal/domain/production"
)

// ProductionManagementRowDTO is the response DTO for ProductionManagement screen.
//
// Frontend usage (useProductionManagement.tsx):
// - blueprint filter: productBlueprintId + productName
// - brand filter: brandName
// - assignee filter: assigneeId + assigneeName
// - sort keys: totalQuantity / printedAt / createdAt
// - optional labels: printedAtLabel / createdAtLabel
type ProductionManagementRowDTO struct {
	// --- base (Production) ---
	ID                 string                         `json:"id"`
	ProductBlueprintID string                         `json:"productBlueprintId"`
	AssigneeID         string                         `json:"assigneeId"`
	Models             []productiondom.ModelQuantity  `json:"models,omitempty"`
	Status             productiondom.ProductionStatus `json:"status,omitempty"`

	PrintedAt *time.Time `json:"printedAt,omitempty"`
	PrintedBy *string    `json:"printedBy,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy *string    `json:"createdBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`

	// --- computed / resolved for UI ---
	TotalQuantity int    `json:"totalQuantity"`
	ProductName   string `json:"productName,omitempty"`
	BrandName     string `json:"brandName,omitempty"`
	AssigneeName  string `json:"assigneeName,omitempty"`

	// --- labels for UI (optional) ---
	PrintedAtLabel string `json:"printedAtLabel,omitempty"`
	CreatedAtLabel string `json:"createdAtLabel,omitempty"`
}
