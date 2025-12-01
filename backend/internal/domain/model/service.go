// backend/internal/domain/model/service.go
package model

import "strings"

// ModelNumberFromID は、渡された modelID (variation の ID) に対応する
// ModelVariation を ModelData から探し、その ModelNumber を返します。
//
// 対象の variation が存在しない場合は ErrVariationNotFound を、
// modelID が空の場合は ErrInvalidID を返します。
func ModelNumberFromID(md *ModelData, modelID string) (string, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", ErrInvalidID
	}

	v, ok := md.FindVariationByID(modelID)
	if !ok {
		return "", ErrVariationNotFound
	}

	if strings.TrimSpace(v.ModelNumber) == "" {
		return "", ErrInvalidModelNumber
	}

	return v.ModelNumber, nil
}
