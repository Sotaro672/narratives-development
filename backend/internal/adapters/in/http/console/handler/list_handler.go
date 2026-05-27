package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

type ListHandler struct {
	uc      *usecase.ListUsecase
	qMgmt   *listmanagementquery.ListManagementQuery
	qDetail *listdetailquery.ListDetailQuery
}

type NewListHandlerParams struct {
	UC      *usecase.ListUsecase
	QMgmt   *listmanagementquery.ListManagementQuery
	QDetail *listdetailquery.ListDetailQuery
}

func NewListHandler(p NewListHandlerParams) http.Handler {
	return &ListHandler{
		uc:      p.UC,
		qMgmt:   p.QMgmt,
		qDetail: p.QDetail,
	}
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == "/lists/create-seed" {
		if r.Method != http.MethodGet {
			listMethodNotAllowed(w)
			return
		}
		h.createSeed(w, r)
		return
	}

	if path == "/lists" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
			return
		case http.MethodGet:
			h.listIndex(w, r)
			return
		default:
			listMethodNotAllowed(w)
			return
		}
	}

	if !strings.HasPrefix(path, "/lists/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(path, "/lists/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "aggregate":
			if r.Method != http.MethodGet {
				listMethodNotAllowed(w)
				return
			}
			h.getAggregate(w, r, id)
			return

		case "images":
			sub := ""
			if len(parts) >= 3 {
				sub = parts[2]
			}

			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, id)
					return
				case http.MethodPost:
					h.saveImageFromGCS(w, r, id)
					return
				default:
					listMethodNotAllowed(w)
					return
				}
			}

			if len(parts) == 3 && sub != "" {
				if r.Method == http.MethodDelete {
					h.deleteImage(w, r, id, sub)
					return
				}

				listMethodNotAllowed(w)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return

		case "primary-image":
			if r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodPatch {
				listMethodNotAllowed(w)
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

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
		return
	case http.MethodPut, http.MethodPatch:
		h.update(w, r, id)
		return
	case http.MethodDelete:
		h.delete(w, r, id)
		return
	default:
		listMethodNotAllowed(w)
		return
	}
}

func listMethodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
}

func listIsNotSupported(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "not_supported") ||
		strings.Contains(msg, "notsupported")
}

func listParseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}

	return n
}

func listSplitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

func writeConsoleListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *ListHandler) createSeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}

	if h.qMgmt == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	qp := r.URL.Query()

	invID := qp.Get("inventoryId")
	if invID == "" {
		invID = qp.Get("inventory_id")
	}
	if invID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inventoryId is required"})
		return
	}

	modelIDs := []string{}

	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			modelIDs = append(modelIDs, listSplitCSV(x)...)
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			modelIDs = append(modelIDs, listSplitCSV(x)...)
		}
	} else {
		raw := qp.Get("modelIds")
		if raw == "" {
			raw = qp.Get("model_ids")
		}
		if raw != "" {
			modelIDs = append(modelIDs, listSplitCSV(raw)...)
		}
	}

	out, err := h.qMgmt.BuildCreateSeed(ctx, invID, modelIDs)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		msg := err.Error()
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "invalid") || strings.Contains(lower, "inventory") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *ListHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

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
	if item.CreatedBy == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "createdBy is required"})
		return
	}

	if item.ID == item.InventoryID {
		item.ID = ""
	}

	if string(item.Status) == "" {
		item.Status = listdom.StatusListing
	}

	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = &now
	item.UpdatedBy = nil

	created, err := h.uc.Create(ctx, item)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeConsoleListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *ListHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	var in listdom.List
	if err := json.Unmarshal(body, &in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeConsoleListErr(w, err)
		return
	}

	if in.Title != "" {
		current.Title = in.Title
	}
	if in.Description != "" {
		current.Description = in.Description
	}
	if string(in.Status) != "" {
		current.Status = listdom.ListStatus(string(in.Status))
	}
	if in.Prices != nil {
		current.Prices = in.Prices
	}
	if in.UpdatedBy != nil {
		current.UpdatedBy = in.UpdatedBy
	}

	now := time.Now().UTC()
	current.UpdatedAt = &now

	updater, ok := any(h.uc).(interface {
		Update(ctx context.Context, item listdom.List) (listdom.List, error)
	})
	if !ok {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	updated, err := updater.Update(ctx, current)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeConsoleListErr(w, err)
		return
	}

	if h.qDetail != nil {
		if dto, e := h.qDetail.BuildListDetailDTO(ctx, id); e == nil {
			_ = json.NewEncoder(w).Encode(dto)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(updated)
}

func (h *ListHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"id": id,
	})
}

func (h *ListHandler) listIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	qp := r.URL.Query()

	var f listdom.Filter

	if s := qp.Get("q"); s != "" {
		f.SearchQuery = s
	} else if s := qp.Get("search"); s != "" {
		f.SearchQuery = s
	}

	if v := qp.Get("assigneeId"); v != "" {
		f.AssigneeID = &v
	} else if v := qp.Get("assignee_id"); v != "" {
		f.AssigneeID = &v
	}

	statusesRaw := qp.Get("statuses")
	if statusesRaw == "" {
		statusesRaw = qp.Get("status")
	}

	if statusesRaw != "" {
		ss := listSplitCSV(statusesRaw)
		if len(ss) == 1 {
			st := listdom.ListStatus(ss[0])
			if st != "" {
				f.Status = &st
			}
		} else if len(ss) > 1 {
			out := make([]listdom.ListStatus, 0, len(ss))
			for _, s := range ss {
				st := listdom.ListStatus(s)
				if st != "" {
					out = append(out, st)
				}
			}
			f.Statuses = out
		}
	}

	if v := qp.Get("minPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinPrice = &n
		}
	}

	if v := qp.Get("maxPrice"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MaxPrice = &n
		}
	}

	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			f.ModelNumbers = append(f.ModelNumbers, listSplitCSV(x)...)
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
		for _, x := range vv {
			f.ModelNumbers = append(f.ModelNumbers, listSplitCSV(x)...)
		}
	}

	sort := listdom.Sort{}

	pageNum := listParseIntDefault(qp.Get("page"), 1)
	perPage := listParseIntDefault(qp.Get("perPage"), 50)
	page := listdom.Page{Number: pageNum, PerPage: perPage}

	if h.qMgmt != nil {
		pr, err := h.qMgmt.ListRows(ctx, f, sort, page)
		if err != nil {
			if listIsNotSupported(err) {
				w.WriteHeader(http.StatusNotImplemented)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
				return
			}
			writeConsoleListErr(w, err)
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

	result, err := h.uc.List(ctx, f, sort, page)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    perPage,
	})
}

func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.qDetail == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	dto, err := h.qDetail.BuildListDetailDTO(ctx, id)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func (h *ListHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(agg)
}

func (h *ListHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(items)
}

func (h *ListHandler) deleteImage(w http.ResponseWriter, r *http.Request, listID string, imageID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	if err := h.uc.DeleteImage(ctx, listID, imageID); err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"listId":  listID,
		"imageId": imageID,
	})
}

// saveImageFromGCS keeps the historical function name for routing compatibility.
// Current policy:
// - frontend uploads images directly to Firebase Storage.
// - backend receives and stores only the Firebase Storage download URL.
// - backend does not validate or persist objectPath, fileName, contentType, or size.
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	var req struct {
		ID           string `json:"id"`
		URL          string `json:"url"`
		DisplayOrder int    `json:"displayOrder"`
		CreatedBy    string `json:"createdBy,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.URL = strings.TrimSpace(req.URL)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	if strings.Contains(req.ID, "/") || strings.Contains(req.ID, "://") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid image id"})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "displayOrder must be >= 0"})
		return
	}

	now := time.Now().UTC()

	img, err := h.uc.SaveImage(
		ctx,
		listdom.ListImage{
			ID:           req.ID,
			ListID:       listID,
			URL:          req.URL,
			DisplayOrder: req.DisplayOrder,
			CreatedAt:    now,
			CreatedBy:    req.CreatedBy,
		},
	)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(img)
}

func (h *ListHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

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
	if req.Now != nil && *req.Now != "" {
		if t, err := time.Parse(time.RFC3339, *req.Now); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, listID, imageID, now, req.UpdatedBy)
	if err != nil {
		if listIsNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(item)
}
