package productBlueprint

import (
	"encoding/json"
	"net/http"

	pbdom "narratives/internal/domain/productBlueprint"
)

func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case pbdom.IsInvalid(err):
		code = http.StatusBadRequest
	case pbdom.IsNotFound(err):
		code = http.StatusNotFound
	case pbdom.IsConflict(err):
		code = http.StatusConflict
	case pbdom.IsUnauthorized(err):
		code = http.StatusUnauthorized
	case pbdom.IsForbidden(err):
		code = http.StatusForbidden
	default:
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
