package model

import "time"

type ModelData struct {
	ProductBlueprintID string
	Variations         []ModelVariation
	UpdatedAt          time.Time
}

func (m ModelData) Validate() error {
	if m.ProductBlueprintID == "" {
		return ErrInvalidBlueprintID
	}

	variationIDs := make(
		map[string]struct{},
		len(m.Variations),
	)

	for _, variation := range m.Variations {
		if variation == nil {
			return ErrInvalid
		}

		if variation.GetProductBlueprintID() !=
			m.ProductBlueprintID {
			return ErrProductMismatch
		}

		if err := variation.Validate(); err != nil {
			return err
		}

		variationID := variation.GetID()

		if _, exists := variationIDs[variationID]; exists {
			return ErrDuplicateVariationID
		}

		variationIDs[variationID] = struct{}{}
	}

	return nil
}

type ModelVariation interface {
	GetID() string
	GetProductBlueprintID() string
	GetKind() ModelVariationKind
	GetModelNumber() string
	Validate() error
}
