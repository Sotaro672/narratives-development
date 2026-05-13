// backend\internal\domain\model\common.go
package model

import "time"

type CategoryFields map[string]any

type ModelData struct {
	ProductBlueprintID string
	Variations         []ModelVariation
	UpdatedAt          time.Time
}

type Model = ModelData

type ModelVariation interface {
	GetID() string
	GetProductBlueprintID() string
	GetModelNumber() string
	Validate() error
}
