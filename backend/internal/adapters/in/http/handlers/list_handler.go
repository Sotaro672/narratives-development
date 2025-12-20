package handlers

import (
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
		ss := splitCSV(statusesRaw) // ✅ helpers.go の splitCSV を利用
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

	if vv := qp["modelNumbers"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	} else if vv := qp["inventoryIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				// 現状仕様に合わせて ModelNumbers に寄せる（既存コード踏襲）
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

	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		log.Printf("[list_handler] POST /lists invalid json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	created, err := h.uc.Create(ctx, item)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		log.Printf("[list_handler] POST /lists uc.Create failed: %v", err)
		writeListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ==============================
// GET /lists/{id}
// ==============================

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

// ==============================
// error helpers
// ==============================

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
