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
		errors.Is(err, modeldom.ErrInvalidBlueprintID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, modeldom.ErrNotFound) {
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
