package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

// ListHandler は /lists 関連のエンドポイントを担当します。
type ListHandler struct {
	uc *usecase.ListUsecase
}

// NewListHandler はHTTPハンドラを初期化します。
func NewListHandler(uc *usecase.ListUsecase) http.Handler {
	return &ListHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ✅ /lists 直下
	if r.URL.Path == "/lists" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
			return
		default:
			// 一覧 GET はこのユースケースでは未対応（現状維持）
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusNotImplemented)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
				return
			}
			methodNotAllowed(w)
			return
		}
	}

	if !strings.HasPrefix(r.URL.Path, "/lists/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/lists/")
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
		case "primary-image":
			// 代表画像の設定
			if r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodPatch {
				methodNotAllowed(w)
				return
			}
			h.setPrimaryImage(w, r, id)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// /lists/{id}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	h.get(w, r, id)
}

// ✅ POST /lists
// - frontend/console/inventory/src/presentation/pages/listCreate.tsx の入力を List レコードとして作成する想定
// - ListUsecase に Create が無い場合は 501 を返す
func (h *ListHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// まずは domain の List をそのまま受ける（UI → API の形はこの JSON に合わせる）
	// ※ List の JSON タグ/構造は listdom 側で定義されている前提
	var item listdom.List
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// createdAt が空（zero）でも良いが、最低限 now を与えたい場合はここで補完
	// listdom.List のフィールドが分からないため「必須」にはしない（usecase 側で補完してOK）
	now := time.Now().UTC()
	_ = now

	// 最も一般的な形：Create(ctx context.Context, item listdom.List)
	if c, ok := any(h.uc).(interface {
		Create(ctx interface {
			Deadline() (time.Time, bool)
			Done() <-chan struct{}
			Err() error
			Value(key any) any
		}, item listdom.List) (listdom.List, error)
	}); ok {
		created, err := c.Create(ctx, item)
		if err != nil {
			writeListErr(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
		return
	}

	// Create が実装されていない
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented_create"})
}

// GET /lists/{id}
func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	item, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}

// GET /lists/{id}/images
func (h *ListHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

// POST /lists/{id}/images
// Body:
//
//	{
//	  "id":"...",           // ListImage.ID（必須）
//	  "fileName":"...",     // 任意（実装による）
//	  "bucket":"",          // optional, empty = default bucket
//	  "objectPath":"...",
//	  "size":123,           // bytes
//	  "displayOrder":0,     // int
//	  "createdBy":"user",   // optional（実装で system 等にフォールバック可）
//	  "createdAt":"..."     // optional RFC3339, default now
//	}
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	var req struct {
		ID           string  `json:"id"`
		FileName     string  `json:"fileName"`
		Bucket       string  `json:"bucket"`
		ObjectPath   string  `json:"objectPath"`
		Size         int64   `json:"size"`
		DisplayOrder int     `json:"displayOrder"`
		CreatedBy    string  `json:"createdBy"`
		CreatedAt    *string `json:"createdAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.ObjectPath) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id and objectPath are required"})
		return
	}

	ca := time.Now().UTC()
	if req.CreatedAt != nil && strings.TrimSpace(*req.CreatedAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedAt)); err == nil {
			ca = t.UTC()
		}
	}

	img, err := h.uc.SaveImageFromGCS(
		ctx,
		strings.TrimSpace(req.ID),
		strings.TrimSpace(listID),
		strings.TrimSpace(req.Bucket),
		strings.TrimSpace(req.ObjectPath),
		req.Size,
		req.DisplayOrder,
		strings.TrimSpace(req.CreatedBy),
		ca,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(img)
}

// PUT|POST|PATCH /lists/{id}/primary-image
// Body:
//
//	{
//	  "imageId":"...",
//	  "updatedBy":"...",     // optional
//	  "now":"..."            // optional RFC3339, default now
//	}
func (h *ListHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	var req struct {
		ImageID   string  `json:"imageId"`
		UpdatedBy *string `json:"updatedBy"`
		Now       *string `json:"now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	imageID := strings.TrimSpace(req.ImageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	now := time.Now().UTC()
	if req.Now != nil && strings.TrimSpace(*req.Now) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.Now)); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, listID, imageID, now, normalizeStrPtr(req.UpdatedBy))
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}

// GET /lists/{id}/aggregate
func (h *ListHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(agg)
}

// エラーハンドリング
func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, listdom.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ==============================
// このファイル内の共通ヘルパー
// ==============================

func methodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
}

// 共通の not supported エラー型は非公開のため、メッセージベースで判定
func isNotSupported(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not supported")
}

// 空白トリムして空なら nil、値があればポインタを返す
func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
