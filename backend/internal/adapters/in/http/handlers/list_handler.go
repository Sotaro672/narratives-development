// backend/internal/adapters/in/http/handlers/list_handler.go
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"io"
	"log"
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

	// ✅ 末尾スラッシュの揺れを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ 入口ログ（Cloud Run で最初に見える）
	log.Printf("[list_handler] request method=%s path=%s rawQuery=%q", r.Method, path, r.URL.RawQuery)

	// ✅ /lists 直下
	if path == "/lists" {
		switch r.Method {
		case http.MethodPost:
			log.Printf("[list_handler] POST /lists start")
			h.create(w, r)
			return
		case http.MethodGet:
			// 一覧 GET は未対応（現状維持）
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

	// ✅ /lists/{id} 以下
	if !strings.HasPrefix(path, "/lists/") {
		log.Printf("[list_handler] not_found (handler mismatch) method=%s path=%s", r.Method, path)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(path, "/lists/")
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		log.Printf("[list_handler] invalid id method=%s path=%s rest=%q", r.Method, path, rest)
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
			log.Printf("[list_handler] not_found (unknown subresource) method=%s path=%s sub=%q", r.Method, path, parts[1])
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

// ==============================
// ✅ POST /lists
// ==============================

// listCreator は ListUsecase に Create が実装されたときに呼べるようにするための最小インターフェースです。
// ※ ListUsecase に Create を追加していない段階でも、この handler はコンパイルできます。
type listCreator interface {
	Create(ctx context.Context, item listdom.List) (listdom.List, error)
}

// create: POST /lists
//
// ✅ frontend は { "input": { ... } } 形式で送ってくるため、
// まず wrapper を剥がしてから domain へマッピングする。
func (h *ListHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		log.Printf("[list_handler] POST /lists aborted: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	// ✅ body を一度読み取り、ログと decode の両方に使う（サイズ上限あり）
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB
	if err != nil {
		log.Printf("[list_handler] POST /lists read body failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	// ✅ raw log（長すぎる場合は切る）
	raw := string(body)
	if len(raw) > 4000 {
		raw = raw[:4000] + "...(truncated)"
	}
	log.Printf("[list_handler] POST /lists body=%s", raw)

	// ✅ まず wrapper を剥がす: { input: {...} }
	var wrap struct {
		Input json.RawMessage `json:"input"`
	}
	_ = json.Unmarshal(body, &wrap)

	var inputBytes []byte
	if len(wrap.Input) > 0 {
		inputBytes = wrap.Input
	} else {
		// 互換: もし input wrapper が無ければ body を input として扱う
		inputBytes = body
	}

	// ✅ 入力を map で受けて柔軟に読む（frontend 側の key 名揺れに耐える）
	var in map[string]any
	if err := json.Unmarshal(inputBytes, &in); err != nil {
		log.Printf("[list_handler] POST /lists invalid json (input): %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[list_handler] POST /lists input keys=%v", mapKeys(in))

	// --- Extract fields (best-effort) ---
	id := firstString(in, "id", "listId", "listID")
	if strings.TrimSpace(id) == "" {
		id = newListID()
	}

	title := firstString(in, "title", "listingTitle")
	description := firstString(in, "description")
	assigneeID := firstString(in, "assigneeId", "assigneeID")

	// decision -> status
	decision := firstString(in, "decision", "status")
	status := strings.TrimSpace(decision)
	if status == "" {
		// ここは domain 側のデフォルトに任せたいが、空で弾かれることがあるので最低限 "hold" を入れる
		status = "hold"
	}

	createdBy := firstString(in, "createdBy")
	if strings.TrimSpace(createdBy) == "" {
		createdBy = "system"
	}

	createdAt := time.Now().UTC()
	if s := firstString(in, "createdAt"); strings.TrimSpace(s) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
			createdAt = t.UTC()
		}
	}

	// prices: priceRows[] or prices(map)
	prices := extractPrices(in)

	// ✅ domain List を構築（repo が ID 必須のため、ここで必ず ID を入れる）
	item := listdom.List{
		ID:          strings.TrimSpace(id),
		Status:      listdom.ListStatus(strings.TrimSpace(status)),
		AssigneeID:  strings.TrimSpace(assigneeID),
		Title:       strings.TrimSpace(title),
		ImageID:     "", // create 時点では未設定
		Description: strings.TrimSpace(description),
		Prices:      prices, // nil でもOK（repoはlen(prices)==0で何もしない）

		CreatedBy: strings.TrimSpace(createdBy),
		CreatedAt: createdAt,
		// UpdatedBy / UpdatedAt / DeletedAt / DeletedBy は作成時点では空でOK（repo側で補完）
	}

	log.Printf("[list_handler] POST /lists mapped id=%s status=%s assigneeId=%s titleLen=%d descLen=%d prices=%d createdBy=%s createdAt=%s",
		item.ID,
		string(item.Status),
		item.AssigneeID,
		len(item.Title),
		len(item.Description),
		len(item.Prices),
		item.CreatedBy,
		item.CreatedAt.UTC().Format(time.RFC3339),
	)

	// ✅ Create があるか確認して呼び出す（uc.Create を呼ぶ）
	c, ok := any(h.uc).(listCreator)
	if !ok {
		log.Printf("[list_handler] POST /lists not supported: uc.Create is missing")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented_create"})
		return
	}

	log.Printf("[list_handler] POST /lists calling uc.Create id=%s", item.ID)

	created, err := c.Create(ctx, item)
	if err != nil {
		log.Printf("[list_handler] POST /lists uc.Create failed: %v", err)
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	log.Printf("[list_handler] POST /lists uc.Create success id=%s", created.ID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ==============================
// Existing endpoints
// ==============================

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
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not supported") || strings.Contains(msg, "not_supported")
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

// --- JSON helpers ---

func mapKeys(m map[string]any) []string {
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if s, ok := v.(string); ok {
			if strings.TrimSpace(s) != "" {
				return s
			}
			continue
		}
		// number -> string は不要（id など）
	}
	return ""
}

// extractPrices は input の価格情報を map[inventoryId]ListPrice に変換します。
// 対応:
// - priceRows: [{ inventoryId, price }, ...]
// - prices: { "<inventoryId>": { price: 123 } , ... }
func extractPrices(in map[string]any) map[string]listdom.ListPrice {
	// 1) priceRows
	if v, ok := in["priceRows"]; ok && v != nil {
		if rows, ok := v.([]any); ok && len(rows) > 0 {
			out := map[string]listdom.ListPrice{}
			for _, rowAny := range rows {
				row, ok := rowAny.(map[string]any)
				if !ok || row == nil {
					continue
				}
				invID, _ := row["inventoryId"].(string)
				invID = strings.TrimSpace(invID)
				if invID == "" {
					// 他のキー名も一応見る
					if s, ok := row["inventoryID"].(string); ok {
						invID = strings.TrimSpace(s)
					}
				}
				if invID == "" {
					continue
				}

				price := 0
				if pv, ok := row["price"]; ok && pv != nil {
					switch t := pv.(type) {
					case float64:
						price = int(t)
					case int:
						price = t
					}
				}

				out[invID] = listdom.ListPrice{Price: price}
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	// 2) prices map
	if v, ok := in["prices"]; ok && v != nil {
		if pm, ok := v.(map[string]any); ok && len(pm) > 0 {
			out := map[string]listdom.ListPrice{}
			for k, vv := range pm {
				invID := strings.TrimSpace(k)
				if invID == "" || vv == nil {
					continue
				}
				switch t := vv.(type) {
				case map[string]any:
					price := 0
					if pv, ok := t["price"]; ok && pv != nil {
						switch x := pv.(type) {
						case float64:
							price = int(x)
						case int:
							price = x
						}
					}
					out[invID] = listdom.ListPrice{Price: price}
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	return nil
}

// newListID は repo が "missing id" を出さないよう、サーバ側で listID を採番します。
// Firestore の doc id と同等である必要はありません（文字列で一意ならOK）。
func newListID() string {
	// 16 bytes -> base32 (no padding) => 26 chars 程度
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback（最悪）
		return "list_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return strings.ToLower(enc.EncodeToString(b))
}
