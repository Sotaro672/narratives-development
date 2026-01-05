// backend/internal/adapters/in/http/console/handler/account_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
	accountdom "narratives/internal/domain/account"
)

// AccountHandler は /accounts 関連のエンドポイントを担当します。
type AccountHandler struct {
	uc *uc.AccountUsecase
}

// NewAccountHandler はHTTPハンドラを初期化します。
func NewAccountHandler(accountUC *uc.AccountUsecase /* other deps */) http.Handler {
	return &AccountHandler{
		uc: accountUC,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AccountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/accounts":
		h.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/accounts/"):
		id := strings.TrimPrefix(r.URL.Path, "/accounts/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /accounts
func (h *AccountHandler) list(w http.ResponseWriter, _ *http.Request) {
	// まだ未実装（Usecase に一覧APIがないため）
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
}

// GET /accounts/{id}
func (h *AccountHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	account, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAccountErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(account)
}

// エラーハンドリング
func writeAccountErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case accountdom.ErrInvalidID:
		code = http.StatusBadRequest
	case accountdom.ErrNotFound:
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
