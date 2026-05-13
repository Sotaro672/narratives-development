// backend/internal/domain/model/service.go
package model

// ModelNumberFromID は、渡された modelID (variation の ID) に対応する
// ModelVariation を ModelData から探し、その ModelNumber を返します。
//
// 対象の variation が存在しない場合は ErrVariationNotFound を、
// modelID が空の場合は ErrInvalidID を返します。
//
// NOTE:
//   - ModelVariation は interface 化されているため、
//     ModelData.FindVariationByID には依存しない。
//   - 現時点では apparel 用の ApparelModelVariation から ModelNumber を取得する。
func ModelNumberFromID(md *ModelData, modelID string) (string, error) {
	if modelID == "" {
		return "", ErrInvalidID
	}

	if md == nil {
		return "", ErrVariationNotFound
	}

	for _, raw := range md.Variations {
		if raw == nil {
			continue
		}

		if raw.GetID() != modelID {
			continue
		}

		v, ok := toServiceApparelModelVariation(raw)
		if !ok {
			return "", ErrVariationNotFound
		}

		if v.ModelNumber == "" {
			return "", ErrInvalidModelNumber
		}

		return v.ModelNumber, nil
	}

	return "", ErrVariationNotFound
}

func toServiceApparelModelVariation(v ModelVariation) (ApparelModelVariation, bool) {
	if v == nil {
		return ApparelModelVariation{}, false
	}

	switch x := v.(type) {
	case ApparelModelVariation:
		return x, true
	case *ApparelModelVariation:
		if x == nil {
			return ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return ApparelModelVariation{}, false
	}
}
