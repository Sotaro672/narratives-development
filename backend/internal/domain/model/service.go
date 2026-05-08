// backend/internal/domain/model/service.go
package model

// ModelNumberFromID は、渡された modelID (variation の ID) に対応する
// ModelVariation を ModelData から探し、その ModelNumber を返します。
//
// 対象の variation が存在しない場合は ErrVariationNotFound を、
// modelID が空の場合は ErrInvalidID を返します。
func ModelNumberFromID(md *ModelData, modelID string) (string, error) {
	if modelID == "" {
		return "", ErrInvalidID
	}

	v, ok := md.FindVariationByID(modelID)
	if !ok {
		return "", ErrVariationNotFound
	}

	if v.ModelNumber == "" {
		return "", ErrInvalidModelNumber
	}

	return v.ModelNumber, nil
}
