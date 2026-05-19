// backend/internal/application/query/console/dto/production_inspection_mint_dto.go
package dto

import (
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

type ProductionInspectionMintDTO struct {
	ID           string `json:"id"`
	ProductionID string `json:"productionId"`

	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	TokenName   string `json:"tokenName,omitempty"`
	ProductName string `json:"productName,omitempty"`

	MintQuantity       int    `json:"mintQuantity"`
	ProductionQuantity int    `json:"productionQuantity"`
	InspectionStatus   string `json:"inspectionStatus,omitempty"`

	RequestedBy   string     `json:"requestedBy,omitempty"`
	CreatedByName string     `json:"createdByName,omitempty"`
	MintedAt      *time.Time `json:"mintedAt,omitempty"`

	// management view BFF fields.
	ProductBlueprintPatch *MintProductBlueprintPatchDTO `json:"productBlueprintPatch,omitempty"`
	TokenBlueprintPatch   *TokenBlueprintPatchDTO       `json:"tokenBlueprintPatch,omitempty"`
	Rows                  []MintRequestModelRowDTO      `json:"rows,omitempty"`
	TotalStock            int                           `json:"totalStock"`
	UpdatedAt             *time.Time                    `json:"updatedAt,omitempty"`

	Inspection *inspectiondom.InspectionBatch `json:"inspection,omitempty"`
	Mint       *mintdom.Mint                  `json:"mint,omitempty"`
}

type TokenBlueprintPatchDTO struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description"`
	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`
	IconURL     string `json:"iconUrl,omitempty"`
}

type MintRequestModelRowDTO struct {
	ModelID     string `json:"modelId"`
	Kind        string `json:"kind"`
	ModelNumber string `json:"modelNumber,omitempty"`
	Stock       int    `json:"stock"`

	// apparel
	Size      string `json:"size,omitempty"`
	ColorName string `json:"colorName,omitempty"`
	RGB       *int   `json:"rgb,omitempty"`

	// alcohol
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`
}
