package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	query "narratives/internal/application/query"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

type ListHandler struct {
	uc *usecase.ListUsecase
	q  *query.ListQuery
}

func NewListHandler(uc *usecase.ListUsecase) http.Handler {
	return &ListHandler{uc: uc, q: nil}
}

// ✅ NEW: Query を注入できる ctor
func NewListHandlerWithQuery(uc *usecase.ListUsecase, q *query.ListQuery) http.Handler {
	return &ListHandler{uc: uc, q: q}
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")
	log.Printf("[list_handler] request method=%s path=%s rawQuery=%q", r.Method, path, r.URL.RawQuery)

	// ✅ NEW: list新規作成画面のための seed を返す（ここでは永続化しない）
	// GET /lists/create-seed?inventoryId={pbId}__{tbId}&modelIds=a,b,c
	// GET /lists/create-seed?inventoryId=...&modelIds=a&modelIds=b
	if path == "/lists/create-seed" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		log.Printf("[list_handler] GET /lists/create-seed start")
		h.createSeed(w, r)
		return
	}

	if path == "/lists" {
		switch r.Method {
		case http.MethodPost:
			log.Printf("[list_handler] POST /lists start")
			h.create(w, r)
			return
		case http.MethodGet:
			log.Printf("[list_handler] GET /lists start")
			h.listIndex(w, r)
			return
		default:
			methodNotAllowed(w)
			return
		}
	}

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
// GET /lists/create-seed
// ==============================

// createSeed は list新規作成画面に必要な情報だけを揃えて返します。
// - 実際の create（永続化）は POST /lists（usecase.Create）に移譲します。
func (h *ListHandler) createSeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil {
		log.Printf("[list_handler] GET /lists/create-seed aborted: handler is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}

	// Query が無ければ "画面情報を揃える" ができないため Not Implemented 扱い
	if h.q == nil {
		log.Printf("[list_handler] GET /lists/create-seed NOT supported (query is nil)")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	qp := r.URL.Query()

	invID := strings.TrimSpace(qp.Get("inventoryId"))
	if invID == "" {
		invID = strings.TrimSpace(qp.Get("inventory_id"))
	}
	if invID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}

	// modelIds:
	// - modelIds=a&modelIds=b
	// - modelIds=a,b,c
	modelIDs := []string{}
	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			for _, s := range splitCSV(x) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	} else {
		raw := strings.TrimSpace(qp.Get("modelIds"))
		if raw == "" {
			raw = strings.TrimSpace(qp.Get("model_ids"))
		}
		if raw != "" {
			for _, s := range splitCSV(raw) {
				s = strings.TrimSpace(s)
				if s != "" {
					modelIDs = append(modelIDs, s)
				}
			}
		}
	}

	log.Printf("[list_handler] GET /lists/create-seed parsed inventoryId=%q modelIDs=%d", invID, len(modelIDs))

	out, err := h.q.BuildCreateSeed(ctx, invID, modelIDs)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		msg := strings.TrimSpace(err.Error())
		if strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "inventory") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}

		log.Printf("[list_handler] GET /lists/create-seed failed: %v", err)
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ==============================
// GET /lists
// ==============================

func (h *ListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		log.Printf("[list_handler] GET /lists aborted: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	qp := r.URL.Query()

	// ---- filter ----
	var f listdom.Filter

	if s := strings.TrimSpace(qp.Get("q")); s != "" {
		f.SearchQuery = s
	} else if s := strings.TrimSpace(qp.Get("search")); s != "" {
		f.SearchQuery = s
	}

	if v := strings.TrimSpace(qp.Get("assigneeId")); v != "" {
		f.AssigneeID = &v
	} else if v := strings.TrimSpace(qp.Get("assignee_id")); v != "" {
		f.AssigneeID = &v
	}

	statusesRaw := strings.TrimSpace(qp.Get("statuses"))
	if statusesRaw == "" {
		statusesRaw = strings.TrimSpace(qp.Get("status"))
	}
	if statusesRaw != "" {
		ss := splitCSV(statusesRaw)
		if len(ss) == 1 {
			st := listdom.ListStatus(strings.TrimSpace(ss[0]))
			if st != "" {
				f.Status = &st
			}
		} else if len(ss) > 1 {
			out := make([]listdom.ListStatus, 0, len(ss))
			for _, s := range ss {
				st := listdom.ListStatus(strings.TrimSpace(s))
				if st != "" {
					out = append(out, st)
				}
			}
			f.Statuses = out
		}
	}

	if dv := strings.TrimSpace(qp.Get("deleted")); dv != "" {
		if b, err := strconv.ParseBool(dv); err == nil {
			f.Deleted = &b
		}
	}

	if v := strings.TrimSpace(qp.Get("minPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinPrice = &n
		}
	}
	if v := strings.TrimSpace(qp.Get("maxPrice")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MaxPrice = &n
		}
	}

	// 互換枠: modelNumbers を使っている既存クライアントもいるため残す
	if vv := qp["modelNumbers"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	} else if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	}

	if t := parseRFC3339Ptr(qp.Get("createdFrom")); t != nil {
		f.CreatedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("createdTo")); t != nil {
		f.CreatedTo = t
	}
	if t := parseRFC3339Ptr(qp.Get("updatedFrom")); t != nil {
		f.UpdatedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("updatedTo")); t != nil {
		f.UpdatedTo = t
	}
	if t := parseRFC3339Ptr(qp.Get("deletedFrom")); t != nil {
		f.DeletedFrom = t
	}
	if t := parseRFC3339Ptr(qp.Get("deletedTo")); t != nil {
		f.DeletedTo = t
	}

	// ---- sort ----
	sort := listdom.Sort{} // repo側のデフォルトに任せる

	// ---- page ----
	pageNum := parseIntDefault(qp.Get("page"), 1)
	perPage := parseIntDefault(qp.Get("perPage"), 50)
	page := listdom.Page{Number: pageNum, PerPage: perPage}

	log.Printf("[list_handler] GET /lists parsed page=%d perPage=%d filter={q:%q assignee:%v status:%v statuses:%d deleted:%v}",
		pageNum, perPage, f.SearchQuery, ptrStr(f.AssigneeID), ptrStatus(f.Status), len(f.Statuses), ptrBool(f.Deleted),
	)

	// ✅ Query があれば query 経由で tokenName/assigneeName を解決して返す
	if h.q != nil {
		pr, err := h.q.ListRows(ctx, f, sort, page)
		if err != nil {
			if isNotSupported(err) {
				w.WriteHeader(http.StatusNotImplemented)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
				return
			}
			log.Printf("[list_handler] GET /lists (query) failed: %v", err)
			writeListErr(w, err)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":      pr.Items,
			"totalCount": pr.TotalCount,
			"totalPages": pr.TotalPages,
			"page":       pr.Page,
			"perPage":    pr.PerPage,
		})
		return
	}

	// fallback: usecase のまま返す
	result, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		log.Printf("[list_handler] GET /lists failed: %v", err)
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    result.PerPage,
	})
}

// ==============================
// POST /lists
// ==============================

func (h *ListHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		log.Printf("[list_handler] POST /lists aborted: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		log.Printf("[list_handler] POST /lists read body failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	raw := string(body)
	if len(raw) > 4000 {
		raw = raw[:4000] + "...(truncated)"
	}
	log.Printf("[list_handler] POST /lists body=%s", raw)

	// ✅ prices はフロント標準の配列（[{modelId, price}, ...]）を正とするため、
	// domain の List 型へ直接 Unmarshal する（domain 側の json tag を正とする）
	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		log.Printf("[list_handler] POST /lists invalid json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ---- normalize / server-side fixups ----
	item.ID = strings.TrimSpace(item.ID)
	item.Title = strings.TrimSpace(item.Title)
	item.AssigneeID = strings.TrimSpace(item.AssigneeID)
	item.InventoryID = strings.TrimSpace(item.InventoryID)
	item.ImageID = strings.TrimSpace(item.ImageID) // ✅ create 時は必須にしない（後で images/primary-image で設定）
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedBy = strings.TrimSpace(item.CreatedBy)

	if item.InventoryID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}
	if item.AssigneeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "assigneeId is required"})
		return
	}
	if item.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "title is required"})
		return
	}
	// ❌ imageId 必須はやめる（後で /images + /primary-image で確定）
	// if item.ImageID == "" { ... }

	if item.CreatedBy == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "createdBy is required"})
		return
	}

	// ✅ ABテスト前提:
	// 1 inventoryId に複数 List を作れるように、ID が inventoryId 固定（または空）の場合はサーバが採番する
	if item.ID == "" || item.ID == item.InventoryID {
		item.ID = buildListID(item.InventoryID)
	}

	// status default
	if strings.TrimSpace(string(item.Status)) == "" {
		item.Status = listdom.ListStatus("listing")
	}

	now := time.Now().UTC()

	// createdAt はサーバ時刻を正とする
	item.CreatedAt = now

	// UpdatedAt もサーバで付与（create 時点で持たせる）
	item.UpdatedAt = &now
	item.UpdatedBy = nil
	item.DeletedAt = nil
	item.DeletedBy = nil

	created, err := h.uc.Create(ctx, item)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		log.Printf("[list_handler] POST /lists uc.Create failed: %v item=%s", err, dumpAsJSON(item))
		writeListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func buildListID(inventoryID string) string {
	inventoryID = strings.TrimSpace(inventoryID)
	suffix := randomHex(8) // 16 chars
	if inventoryID == "" {
		return suffix
	}
	// 読みやすさのため inventoryID を prefix にする（衝突回避は suffix）
	return inventoryID + "__" + suffix
}

func randomHex(nBytes int) string {
	if nBytes <= 0 {
		nBytes = 8
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		// 失敗しても「空」にならないように時刻でフォールバック
		t := time.Now().UTC().UnixNano()
		buf := make([]byte, 8)
		for i := 0; i < 8; i++ {
			buf[i] = byte((t >> (8 * i)) & 0xff)
		}
		return hex.EncodeToString(buf)
	}
	return hex.EncodeToString(b)
}

// ==============================
// GET /lists/{id}
// ==============================

func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	// ✅ A) 採用: GET /lists/{id} は ListDetailDTO を返す
	// - PriceRows に Size/Color/RGB が入る
	// - そのため Query(ListQuery) が必須
	if h == nil || h.q == nil {
		log.Printf("[list_handler] GET /lists/{id} NOT supported (query is nil) id=%q", strings.TrimSpace(id))
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	dto, err := h.q.BuildListDetailDTO(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	// ✅ model metadata が埋まっているか観測できるログ
	if len(dto.PriceRows) > 0 {
		s0 := dto.PriceRows[0]
		log.Printf("[list_handler] GET /lists/{id} detail dto ok id=%q inventoryId=%q priceRows=%d sample={modelId:%q size:%q color:%q rgb:%v stock:%d price:%v}",
			strings.TrimSpace(dto.ID),
			strings.TrimSpace(dto.InventoryID),
			len(dto.PriceRows),
			strings.TrimSpace(s0.ModelID),
			strings.TrimSpace(s0.Size),
			strings.TrimSpace(s0.Color),
			s0.RGB,
			s0.Stock,
			s0.Price,
		)
	} else {
		log.Printf("[list_handler] GET /lists/{id} detail dto ok id=%q inventoryId=%q priceRows=0",
			strings.TrimSpace(dto.ID),
			strings.TrimSpace(dto.InventoryID),
		)
	}

	_ = json.NewEncoder(w).Encode(dto)
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

// ==============================
// error helpers
// ==============================

func writeListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// ✅ domain/list 側の exported error に寄せる（存在しない ErrInvalid* を参照しない）
	switch {
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, listdom.ErrConflict):
		code = http.StatusConflict
	default:
		// 文字列ベースで 400 寄せ（domain 側のエラー定義変更に強くする）
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

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

func ptrStatus(p *listdom.ListStatus) any {
	if p == nil {
		return nil
	}
	return string(*p)
}

func dumpAsJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<json_marshal_failed>"
	}
	return string(b)
}
