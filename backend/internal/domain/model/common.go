// backend/internal/domain/model/common.go
package model

// ModelVariationはModel variationが共通して提供する
// 読み取り・検証用の契約です。
type ModelVariation interface {
	GetID() string

	GetProductBlueprintID() string

	GetKind() ModelVariationKind

	GetModelNumber() string

	Validate() error
}
