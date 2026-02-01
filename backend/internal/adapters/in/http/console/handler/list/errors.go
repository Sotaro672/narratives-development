// backend/internal/adapters/in/http/console/handler/list/errors.go
//
// Responsibility:
// - usecase/domain の error を HTTP status へマッピングし、統一した JSON エラーレスポンスを返す。
package list

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	listdom "narratives/internal/domain/list"
)

func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	// NOTE: listdom.ErrConflict が存在しない場合は、この case を削除してください。
	// case errors.Is(err, listdom.ErrConflict):
	// 	code = http.StatusConflict
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
