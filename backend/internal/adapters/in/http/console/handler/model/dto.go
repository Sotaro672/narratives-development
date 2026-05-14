// backend/internal/adapters/in/http/console/handler/model/dto.go
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
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`

	// category-specific variation kind.
	// 未指定の場合は既存互換として apparel 扱いにする。
	Kind string `json:"kind,omitempty"`

	// common
	ModelNumber string `json:"modelNumber"`

	// apparel
	Size         string             `json:"size,omitempty"`
	Color        string             `json:"color,omitempty"`
	RGB          int                `json:"rgb"` // ✅ 必須: omitempty を外す（0=黒を送れる）
	Measurements map[string]float64 `json:"measurements,omitempty"`

	// alcohol
	Volume modeldom.Volume `json:"volume,omitempty"`
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type colorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"` // ✅ 必須: omitempty なし（0=黒を正しく返す）
}

type volumeDTO struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type modelVariationDTO struct {
	ID                 string         `json:"id"`
	ProductBlueprintID string         `json:"productBlueprintId"`
	Kind               string         `json:"kind,omitempty"`
	ModelNumber        string         `json:"modelNumber"`
	Size               string         `json:"size,omitempty"`
	Color              *colorDTO      `json:"color,omitempty"`
	Measurements       map[string]int `json:"measurements,omitempty"`
	Volume             *volumeDTO     `json:"volume,omitempty"`
	CreatedAt          *string        `json:"createdAt,omitempty"`
	CreatedBy          *string        `json:"createdBy,omitempty"`
	UpdatedAt          *string        `json:"updatedAt,omitempty"`
	UpdatedBy          *string        `json:"updatedBy,omitempty"`
}

func toModelVariationDTO(mv modeldom.ModelVariation) modelVariationDTO {
	switch v := mv.(type) {
	case modeldom.ApparelModelVariation:
		return toApparelModelVariationDTO(v)

	case *modeldom.ApparelModelVariation:
		if v == nil {
			return modelVariationDTO{}
		}
		return toApparelModelVariationDTO(*v)

	case modeldom.AlcoholModelVariation:
		return toAlcoholModelVariationDTO(v)

	case *modeldom.AlcoholModelVariation:
		if v == nil {
			return modelVariationDTO{}
		}
		return toAlcoholModelVariationDTO(*v)

	default:
		return modelVariationDTO{
			ID:                 mv.GetID(),
			ProductBlueprintID: mv.GetProductBlueprintID(),
			ModelNumber:        mv.GetModelNumber(),
		}
	}
}

func toApparelModelVariationDTO(mv modeldom.ApparelModelVariation) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind:               string(modeldom.ModelVariationKindApparel),
		ModelNumber:        mv.ModelNumber,
		Size:               mv.Size,
		Color: &colorDTO{
			Name: mv.Color.Name,
			RGB:  mv.Color.RGB, // ✅ 0 もそのまま返す（黒）
		},
		Measurements: cloneMeasurementsForDTO(mv.Measurements),
		CreatedAt:    timePtrToRFC3339(&mv.CreatedAt),
		CreatedBy:    mv.CreatedBy,
		UpdatedAt:    timePtrToRFC3339(&mv.UpdatedAt),
		UpdatedBy:    mv.UpdatedBy,
	}
}

func toAlcoholModelVariationDTO(mv modeldom.AlcoholModelVariation) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind:               string(modeldom.ModelVariationKindAlcohol),
		ModelNumber:        mv.ModelNumber,
		Volume: &volumeDTO{
			Value: mv.Volume.Value,
			Unit:  mv.Volume.Unit,
		},
		CreatedAt: timePtrToRFC3339(&mv.CreatedAt),
		CreatedBy: mv.CreatedBy,
		UpdatedAt: timePtrToRFC3339(&mv.UpdatedAt),
		UpdatedBy: mv.UpdatedBy,
	}
}

func toModelVariationDTOs(vars []modeldom.ModelVariation) []modelVariationDTO {
	out := make([]modelVariationDTO, 0, len(vars))
	for _, v := range vars {
		out = append(out, toModelVariationDTO(v))
	}
	return out
}

func cloneMeasurementsForDTO(m modeldom.Measurements) map[string]int {
	if m == nil {
		return nil
	}

	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
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
