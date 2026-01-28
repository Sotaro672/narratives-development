package model

import (
	"time"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// Request DTOs
// ------------------------------------------------------------

// Request struct for CREATE / UPDATE
type createModelVariationRequest struct {
	ProductBlueprintID string             `json:"productBlueprintId,omitempty"`
	ModelNumber        string             `json:"modelNumber"`
	Size               string             `json:"size"`
	Color              string             `json:"color"`
	RGB                int                `json:"rgb"` // ✅ 必須: omitempty を外す（0=黒を送れる）
	Measurements       map[string]float64 `json:"measurements,omitempty"`
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type colorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"` // ✅ 必須: omitempty なし（0=黒を正しく返す）
}

type modelVariationDTO struct {
	ID                 string         `json:"id"`
	ProductBlueprintID string         `json:"productBlueprintId"`
	ModelNumber        string         `json:"modelNumber"`
	Size               string         `json:"size"`
	Color              colorDTO       `json:"color"`
	Measurements       map[string]int `json:"measurements,omitempty"`
	CreatedAt          *string        `json:"createdAt,omitempty"`
	CreatedBy          *string        `json:"createdBy,omitempty"`
	UpdatedAt          *string        `json:"updatedAt,omitempty"`
	UpdatedBy          *string        `json:"updatedBy,omitempty"`
}

func toModelVariationDTO(mv modeldom.ModelVariation) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		ModelNumber:        mv.ModelNumber,
		Size:               mv.Size,
		Color: colorDTO{
			Name: mv.Color.Name,
			RGB:  mv.Color.RGB, // ✅ 0 もそのまま返す（黒）
		},
		Measurements: mv.Measurements,
		CreatedAt:    timePtrToRFC3339(&mv.CreatedAt),
		CreatedBy:    mv.CreatedBy,
		UpdatedAt:    timePtrToRFC3339(&mv.UpdatedAt),
		UpdatedBy:    mv.UpdatedBy,
	}
}

func toModelVariationDTOs(vars []modeldom.ModelVariation) []modelVariationDTO {
	out := make([]modelVariationDTO, 0, len(vars))
	for _, v := range vars {
		out = append(out, toModelVariationDTO(v))
	}
	return out
}

func timePtrToRFC3339(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
