// backend/internal/application/query/dto/production_management.go
package dto

import (
	"time"

	productiondom "narratives/internal/domain/production"
)

type ProductionManagementRowDTO struct {
	// --- base (Production) ---
	ID                 string                        `json:"id"`
	ProductBlueprintID string                        `json:"productBlueprintId"`
	AssigneeID         string                        `json:"assigneeId"`
	Models             []productiondom.ModelQuantity `json:"models,omitempty"`

	// Status は廃止。Printed(boolean) に統一。
	Printed bool `json:"printed"`

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
}
