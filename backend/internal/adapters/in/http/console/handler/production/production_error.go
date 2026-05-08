// backend/internal/adapters/in/http/console/handler/production/production_error.go
package productionHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

func writeProductionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// Production domain errors
	if errors.Is(err, productiondom.ErrInvalidID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productiondom.ErrNotFound) {
		code = http.StatusNotFound

		// ProductBlueprint domain errors（companyId 無しなど、QueryService 側で起きる）
	} else if errors.Is(err, productbpdom.ErrInvalidCompanyID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, productbpdom.ErrInvalidID) {
		code = http.StatusBadRequest
	}

	// ✅ debug: final error response
	log.Printf("[productions] respond error status=%d err=%v", code, err)

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
