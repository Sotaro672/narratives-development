// backend/internal/adapters/in/http/console/handler/list_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	query "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

type ListImageUploader interface {
	Upload(ctx context.Context, in listimgdom.UploadImageInput) (*listimgdom.ListImage, error)
}

type ListImageDeleter interface {
	Delete(ctx context.Context, imageID string) error
}

type ListHandler struct {
	uc *usecase.ListUsecase

	// ✅ split: listManagement.tsx / listCreate.tsx 向け
	qMgmt *query.ListManagementQuery

	// ✅ split: listDetail.tsx 向け
	qDetail *query.ListDetailQuery

	// ✅ ListImage (optional)
	imgUploader ListImageUploader
	imgDeleter  ListImageDeleter
}

func NewListHandler(uc *usecase.ListUsecase) http.Handler {
	return &ListHandler{uc: uc, qMgmt: nil, qDetail: nil, imgUploader: nil, imgDeleter: nil}
}

// ✅ NEW: 2種類の Query を注入できる ctor
func NewListHandlerWithQueries(
	uc *usecase.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
) http.Handler {
	return &ListHandler{uc: uc, qMgmt: qMgmt, qDetail: qDetail, imgUploader: nil, imgDeleter: nil}
}

// ✅ NEW: ListImage も注入できる ctor
func NewListHandlerWithQueriesAndListImage(
	uc *usecase.ListUsecase,
	qMgmt *query.ListManagementQuery,
	qDetail *query.ListDetailQuery,
	uploader ListImageUploader,
	deleter ListImageDeleter,
) http.Handler {
	return &ListHandler{
		uc:          uc,
		qMgmt:       qMgmt,
		qDetail:     qDetail,
		imgUploader: uploader,
		imgDeleter:  deleter,
	}
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /lists/create-seed?inventoryId={pbId}__{tbId}&modelIds=a,b,c
	if path == "/lists/create-seed" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
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
			methodNotAllowed(w)
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
	id := strings.TrimSpace(parts[0])
	if id == "" {
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
			// /lists/{id}/images
			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.listImages(w, r, id)
					return
				case http.MethodPost:
					// signed URL PUT 後の object をレコード化
					h.saveImageFromGCS(w, r, id)
					return
				default:
					methodNotAllowed(w)
					return
				}
			}

			// /lists/{id}/images/{sub}
			sub := ""
			if len(parts) >= 3 {
				sub = strings.TrimSpace(parts[2])
			}

			// /lists/{id}/images/signed-url で signed URL を発行
			if strings.EqualFold(sub, "signed-url") {
				if r.Method != http.MethodPost {
					methodNotAllowed(w)
					return
				}
				h.issueSignedURL(w, r, id)
				return
			}

			// /lists/{id}/images/{imageId} DELETE
			if r.Method == http.MethodDelete && sub != "" {
				h.deleteImage(w, r, id, sub)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return

		case "primary-image":
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
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
		return
	case http.MethodPut, http.MethodPatch:
		h.update(w, r, id)
		return
	default:
		methodNotAllowed(w)
		return
	}
}

// ==============================
// POST /lists/{id}/images/signed-url
// ==============================
//
// 必須キーを handler 側で "固定の形" に正規化して返す。
// - uploadUrl / bucket / objectPath / id / publicUrl を必ず返す（可能な限り）
func (h *ListHandler) issueSignedURL(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req struct {
		FileName         string `json:"fileName"`
		ContentType      string `json:"contentType"`
		Size             int64  `json:"size"`
		DisplayOrder     int    `json:"displayOrder"`
		ExpiresInSeconds int    `json:"expiresInSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	ct := strings.ToLower(strings.TrimSpace(req.ContentType))
	if ct == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "contentType is required"})
		return
	}

	if _, ok := listimgdom.SupportedImageMIMEs[ct]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported contentType"})
		return
	}

	if req.Size > 0 && req.Size > int64(listimgdom.DefaultMaxImageSizeBytes) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
		return
	}

	out, err := h.uc.IssueImageSignedURL(ctx, usecase.ListImageIssueSignedURLInput{
		ListID:           listID,
		FileName:         strings.TrimSpace(req.FileName),
		ContentType:      ct,
		Size:             req.Size,
		DisplayOrder:     req.DisplayOrder,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	// 必須チェック（usecase 側でも行うが二重化）
	if strings.TrimSpace(out.UploadURL) == "" ||
		strings.TrimSpace(out.Bucket) == "" ||
		strings.TrimSpace(out.ObjectPath) == "" ||
		strings.TrimSpace(out.ID) == "" {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "signed_url_response_invalid"})
		return
	}

	type resp struct {
		ID           string `json:"id"`
		Bucket       string `json:"bucket"`
		ObjectPath   string `json:"objectPath"`
		PublicURL    string `json:"publicUrl"`
		UploadURL    string `json:"uploadUrl"`
		ExpiresAt    string `json:"expiresAt"`
		ContentType  string `json:"contentType"`
		Size         int64  `json:"size"`
		DisplayOrder int    `json:"displayOrder"`
		FileName     string `json:"fileName"`
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp{
		ID:           strings.TrimSpace(out.ID),
		Bucket:       strings.TrimSpace(out.Bucket),
		ObjectPath:   strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/"),
		PublicURL:    strings.TrimSpace(out.PublicURL),
		UploadURL:    strings.TrimSpace(out.UploadURL),
		ExpiresAt:    strings.TrimSpace(out.ExpiresAt),
		ContentType:  strings.TrimSpace(out.ContentType),
		Size:         out.Size,
		DisplayOrder: out.DisplayOrder,
		FileName:     strings.TrimSpace(out.FileName),
	})
}

// ==============================
// GET /lists/create-seed
// ==============================

func (h *ListHandler) createSeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}

	// management query が無いと seed を作れない
	if h.qMgmt == nil {
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

	out, err := h.qMgmt.BuildCreateSeed(ctx, invID, modelIDs)
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

	// modelIds のみ採用。
	if vv := qp["modelIds"]; len(vv) > 0 {
		for _, x := range vv {
			x = strings.TrimSpace(x)
			if x != "" {
				f.ModelNumbers = append(f.ModelNumbers, x)
			}
		}
	} else if vv := qp["model_ids"]; len(vv) > 0 {
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

	// management query があれば query 経由で返す
	if h.qMgmt != nil {
		pr, err := h.qMgmt.ListRows(ctx, f, sort, page)
		if err != nil {
			if isNotSupported(err) {
				w.WriteHeader(http.StatusNotImplemented)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
				return
			}
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
		writeListErr(w, err)
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

// ==============================
// POST /lists
// ==============================

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

	item.ID = strings.TrimSpace(item.ID)
	item.Title = strings.TrimSpace(item.Title)
	item.AssigneeID = strings.TrimSpace(item.AssigneeID)
	item.InventoryID = strings.TrimSpace(item.InventoryID)
	item.ImageID = strings.TrimSpace(item.ImageID)
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
	if item.CreatedBy == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "createdBy is required"})
		return
	}

	// listId は inventoryId と独立したIDを使う（事故防止）
	if item.ID == item.InventoryID {
		item.ID = ""
	}

	if strings.TrimSpace(string(item.Status)) == "" {
		item.Status = listdom.ListStatus("listing")
	}

	now := time.Now().UTC()
	item.CreatedAt = now
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
		writeListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ==============================
// PUT|PATCH /lists/{id}
// ==============================

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

	log.Printf("[list_handler] UPDATE received method=%s id=%q bytes=%d", r.Method, strings.TrimSpace(id), len(body))

	var in listdom.List
	if err := json.Unmarshal(body, &in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	if t := strings.TrimSpace(in.Title); t != "" {
		current.Title = t
	}
	if d := strings.TrimSpace(in.Description); d != "" {
		current.Description = d
	}

	if st := strings.TrimSpace(string(in.Status)); st != "" {
		current.Status = listdom.ListStatus(st)
	}

	if in.Prices != nil {
		current.Prices = in.Prices
	}

	if in.UpdatedBy != nil {
		current.UpdatedBy = normalizeStrPtr(in.UpdatedBy)
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
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
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

// ==============================
// GET /lists/{id}
// ==============================

func (h *ListHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.qDetail == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	dto, err := h.qDetail.BuildListDetailDTO(ctx, id)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
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

// DELETE /lists/{id}/images/{imageId}
func (h *ListHandler) deleteImage(w http.ResponseWriter, r *http.Request, listID string, imageID string) {
	ctx := r.Context()

	if h == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "handler is nil"})
		return
	}
	if h.imgDeleter == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return
	}

	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	if err := h.imgDeleter.Delete(ctx, imageID); err != nil {
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"listId":  strings.TrimSpace(listID),
		"imageId": imageID,
	})
}

// POST /lists/{id}/images
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req struct {
		ID           string `json:"id"`
		FileName     string `json:"fileName"` // kept for request compatibility; not used by usecase
		Bucket       string `json:"bucket"`
		ObjectPath   string `json:"objectPath"`
		Size         int64  `json:"size"`
		DisplayOrder int    `json:"displayOrder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.ObjectPath = strings.TrimSpace(req.ObjectPath)

	if req.ID == "" || req.ObjectPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id and objectPath are required"})
		return
	}

	img, err := h.uc.SaveImageFromGCS(
		ctx,
		req.ID,
		strings.TrimSpace(listID),
		req.Bucket,
		req.ObjectPath,
		req.Size,
		req.DisplayOrder,
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
	case errors.Is(err, listdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, listdom.ErrConflict):
		code = http.StatusConflict
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
