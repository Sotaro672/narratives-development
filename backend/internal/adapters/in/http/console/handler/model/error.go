// backend/internal/adapters/in/http/console/handler/model/error.go
package model

import (
	"encoding/json"
	"errors"
	"net/http"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// 共通エラー処理
// ------------------------------------------------------------
func writeModelErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, modeldom.ErrInvalidID) ||
		errors.Is(err, modeldom.ErrInvalidProductID) ||
		errors.Is(err, modeldom.ErrInvalidBlueprintID) ||
		errors.Is(err, modeldom.ErrInvalidModelNumber) ||
		errors.Is(err, modeldom.ErrInvalidSize) ||
		errors.Is(err, modeldom.ErrInvalidColor) ||
		errors.Is(err, modeldom.ErrInvalidMeasurements) ||
		errors.Is(err, modeldom.ErrInvalidVolume) ||
		errors.Is(err, modeldom.ErrInvalidVolumeUnit) ||
		errors.Is(err, modeldom.ErrInvalid) {
		code = http.StatusBadRequest
	} else if errors.Is(err, modeldom.ErrNotFound) {
		code = http.StatusNotFound
	} else if errors.Is(err, modeldom.ErrConflict) {
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
