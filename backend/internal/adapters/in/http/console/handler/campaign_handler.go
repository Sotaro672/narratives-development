package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	campaigndom "narratives/internal/domain/campaign"
)

// CampaignHandler は /campaigns 関連のエンドポイントを担当します。
type CampaignHandler struct {
	uc *usecase.CampaignUsecase
}

// NewCampaignHandler はHTTPハンドラを初期化します。
func NewCampaignHandler(uc *usecase.CampaignUsecase) http.Handler {
	return &CampaignHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *CampaignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !strings.HasPrefix(r.URL.Path, "/campaigns/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/campaigns/")
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// サブリソース分岐
	if len(parts) > 1 {
		switch r.Method {
		case http.MethodGet:
			switch parts[1] {
			case "aggregate":
				h.getAggregate(w, r, id)
				return
			case "images":
				h.listImages(w, r, id)
				return
			case "performances":
				h.listPerformances(w, r, id)
				return
			default:
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
			return
		}
	}

	// ベース: /campaigns/{id}
	switch r.Method {
	case http.MethodGet:
		// ?aggregate=true|1 なら集約、それ以外は単体
		q := r.URL.Query()
		if agg := q.Get("aggregate"); strings.EqualFold(agg, "true") || agg == "1" {
			h.getAggregate(w, r, id)
			return
		}
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
	}
}

// GET /campaigns/{id}
func (h *CampaignHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	c, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeCampaignErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(c)
}

// GET /campaigns/{id}/aggregate もしくは GET /campaigns/{id}?aggregate=true
func (h *CampaignHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeCampaignErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(agg)
}

// GET /campaigns/{id}/images?page=&perPage=
func (h *CampaignHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	q := r.URL.Query()
	page := parseIntDefault(q.Get("page"), 1)
	per := parseIntDefault(q.Get("perPage"), 50)

	items, err := h.uc.ListImages(ctx, id, page, per)
	if err != nil {
		writeCampaignErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// GET /campaigns/{id}/performances
func (h *CampaignHandler) listPerformances(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	items, err := h.uc.ListPerformances(ctx, id)
	if err != nil {
		writeCampaignErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// エラーハンドリング
func writeCampaignErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case campaigndom.ErrInvalidID:
		code = http.StatusBadRequest
	case campaigndom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
