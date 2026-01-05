package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
)

// TrackingHandler は /trackings 関連のエンドポイントを担当します（単一取得のみ）。
type TrackingHandler struct {
	uc *usecase.TrackingUsecase
}

// NewTrackingHandler はHTTPハンドラを初期化します。
func NewTrackingHandler(uc *usecase.TrackingUsecase) http.Handler {
	return &TrackingHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *TrackingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/trackings/"):
		id := strings.TrimPrefix(r.URL.Path, "/trackings/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /trackings/{id}
func (h *TrackingHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	tr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTrackingErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(tr)
}

// エラーハンドリング
func writeTrackingErr(w http.ResponseWriter, err error) {
	// ドメインのエラー型に依存せず 500 を返す
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
