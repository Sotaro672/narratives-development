// backend\internal\adapters\in\http\console\handler\inquiry_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
	"net/http"
	"strings"
	"time"
)

// InquiryHandler は /inquiries 関連のエンドポイントを担当します。
type InquiryHandler struct {
	uc *usecase.InquiryUsecase
}

// NewInquiryHandler はHTTPハンドラを初期化します。
func NewInquiryHandler(uc *usecase.InquiryUsecase) http.Handler {
	return &InquiryHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *InquiryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !strings.HasPrefix(r.URL.Path, "/inquiries/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/inquiries/")
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// サブリソース
	if len(parts) > 1 {
		switch parts[1] {
		case "aggregate":
			if r.Method != http.MethodGet {
				methodNotAllowed(w)
				return
			}
			h.getAggregate(w, r, id)
			return
		case "images":
			switch r.Method {
			case http.MethodGet:
				h.listImages(w, r, id)
				return
			case http.MethodPost:
				h.saveImageFromGCS(w, r, id)
				return
			default:
				methodNotAllowed(w)
				return
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// /inquiries/{id}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	h.get(w, r, id)
}

// GET /inquiries/{id}
func (h *InquiryHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	in, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(in)
}

// GET /inquiries/{id}/images
func (h *InquiryHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// POST /inquiries/{id}/images
// Body:
//
//	{
//	  "fileName":"...",
//	  "bucket":"",          // optional, empty = default bucket
//	  "objectPath":"...",
//	  "fileSize":123,
//	  "mimeType":"image/png",
//	  "width":123,          // optional
//	  "height":456,         // optional
//	  "createdAt":"...",    // optional RFC3339, default now
//	  "createdBy":"user"    // optional, impl may default "system"
//	}
func (h *InquiryHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	var req struct {
		FileName   string  `json:"fileName"`
		Bucket     string  `json:"bucket"`
		ObjectPath string  `json:"objectPath"`
		FileSize   int64   `json:"fileSize"`
		MimeType   string  `json:"mimeType"`
		Width      *int    `json:"width"`
		Height     *int    `json:"height"`
		CreatedAt  *string `json:"createdAt"`
		CreatedBy  string  `json:"createdBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	ca := time.Now().UTC()
	if req.CreatedAt != nil && strings.TrimSpace(*req.CreatedAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedAt)); err == nil {
			ca = t.UTC()
		}
	}

	im, err := h.uc.SaveImageFromGCS(
		ctx,
		id,
		strings.TrimSpace(req.FileName),
		strings.TrimSpace(req.Bucket),
		strings.TrimSpace(req.ObjectPath),
		req.FileSize,
		strings.TrimSpace(req.MimeType),
		req.Width,
		req.Height,
		ca,
		strings.TrimSpace(req.CreatedBy),
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeInquiryErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(im)
}

// GET /inquiries/{id}/aggregate
func (h *InquiryHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(agg)
}

// エラーハンドリング
func writeInquiryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, inquirydom.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, inquirydom.ErrNotFound):
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
