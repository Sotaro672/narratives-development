// backend/internal/adapters/in/http/console/handler/list_handler.go
package consoleHandler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	middleware "narratives/internal/adapters/in/http/middleware"
	query "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
)

type ListHandler struct {
	uc      *usecase.ListUsecase
	qMgmt   *query.ListManagementQuery
	qDetail *query.ListDetailQuery
}

type NewListHandlerParams struct {
	UC      *usecase.ListUsecase
	QMgmt   *query.ListManagementQuery
	QDetail *query.ListDetailQuery
}

func NewListHandler(p NewListHandlerParams) http.Handler {
	return &ListHandler{
		uc:      p.UC,
		qMgmt:   p.QMgmt,
		qDetail: p.QDetail,
	}
}

func requireCurrentFirebaseUID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	uid, _, ok := middleware.CurrentUIDAndEmail(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "unauthorized"},
		)
		return "", false
	}

	return uid, true
}

func requireCurrentCompanyID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "company_id_not_resolved"},
		)
		return "", false
	}

	return companyID, true
}

func (h *ListHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

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
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "not_found"},
		)
		return
	}

	rest := strings.TrimPrefix(path, "/lists/")
	parts := strings.Split(rest, "/")
	id := parts[0]

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid id"},
		)
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "images":
			sub := ""

			if len(parts) >= 3 {
				sub = parts[2]
			}

			if len(parts) == 2 {
				if r.Method == http.MethodPost {
					h.createImageFromFirebaseStorage(
						w,
						r,
						id,
					)
					return
				}

				methodNotAllowed(w)
				return
			}

			if len(parts) == 3 && sub != "" {
				if r.Method == http.MethodDelete {
					h.deleteImage(
						w,
						r,
						id,
						sub,
					)
					return
				}

				methodNotAllowed(w)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(
				map[string]string{"error": "not_found"},
			)
			return

		case "primary-image":
			if r.Method != http.MethodPut {
				methodNotAllowed(w)
				return
			}

			h.setPrimaryImage(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(
				map[string]string{"error": "not_found"},
			)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
		return
	case http.MethodPut:
		h.update(w, r, id)
		return
	case http.MethodDelete:
		h.delete(w, r, id)
		return
	default:
		methodNotAllowed(w)
		return
	}
}

func (h *ListHandler) create(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	uid, ok := requireCurrentFirebaseUID(w, r)
	if !ok {
		return
	}

	companyID, ok := requireCurrentCompanyID(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(
		io.LimitReader(r.Body, 1<<20),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid body"},
		)
		return
	}

	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid json"},
		)
		return
	}

	if item.InventoryID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "inventoryId is required",
			},
		)
		return
	}

	if item.AssigneeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "assigneeId is required",
			},
		)
		return
	}

	if item.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "title is required",
			},
		)
		return
	}

	if item.ID == item.InventoryID {
		item.ID = ""
	}

	if string(item.Status) == "" {
		item.Status = listdom.StatusListing
	}

	now := time.Now().UTC()
	item.CreatedBy = uid
	item.CreatedAt = now
	item.UpdatedAt = &now
	item.UpdatedBy = nil

	created, err := h.uc.Create(
		ctx,
		companyID,
		item,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *ListHandler) update(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	uid, ok := requireCurrentFirebaseUID(w, r)
	if !ok {
		return
	}

	companyID, ok := requireCurrentCompanyID(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(
		io.LimitReader(r.Body, 1<<20),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid body"},
		)
		return
	}

	var item listdom.List
	if err := json.Unmarshal(body, &item); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid json"},
		)
		return
	}

	item.ID = id

	if item.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "id is required"},
		)
		return
	}

	now := time.Now().UTC()
	item.UpdatedBy = &uid
	item.UpdatedAt = &now

	updated, err := h.uc.Update(
		ctx,
		companyID,
		item,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	if h.qDetail != nil {
		dto, err := h.qDetail.BuildListDetailDTO(
			ctx,
			id,
		)
		if err == nil {
			_ = json.NewEncoder(w).Encode(dto)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(updated)
}

func (h *ListHandler) delete(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "id is required"},
		)
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"ok": true,
			"id": id,
		},
	)
}

func (h *ListHandler) listIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	if h == nil || h.qMgmt == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_implemented",
			},
		)
		return
	}

	qp := r.URL.Query()
	var filter listdom.Filter

	if value := qp.Get("q"); value != "" {
		filter.SearchQuery = value
	} else if value := qp.Get("search"); value != "" {
		filter.SearchQuery = value
	}

	if value := qp.Get("assigneeId"); value != "" {
		filter.AssigneeID = &value
	} else if value := qp.Get("assignee_id"); value != "" {
		filter.AssigneeID = &value
	}

	statusesRaw := qp.Get("statuses")
	if statusesRaw == "" {
		statusesRaw = qp.Get("status")
	}

	if statusesRaw != "" {
		statuses := splitCSV(statusesRaw)

		if len(statuses) == 1 {
			status := listdom.ListStatus(statuses[0])
			if status != "" {
				filter.Status = &status
			}
		} else if len(statuses) > 1 {
			filter.Statuses = make(
				[]listdom.ListStatus,
				0,
				len(statuses),
			)

			for _, value := range statuses {
				status := listdom.ListStatus(value)
				if status != "" {
					filter.Statuses = append(
						filter.Statuses,
						status,
					)
				}
			}
		}
	}

	if value := qp.Get("minPrice"); value != "" {
		if price, err := strconv.Atoi(value); err == nil {
			filter.MinPrice = &price
		}
	}

	if value := qp.Get("maxPrice"); value != "" {
		if price, err := strconv.Atoi(value); err == nil {
			filter.MaxPrice = &price
		}
	}

	if values := qp["modelIds"]; len(values) > 0 {
		for _, value := range values {
			filter.ModelIDs = append(
				filter.ModelIDs,
				splitCSV(value)...,
			)
		}
	} else if values := qp["model_ids"]; len(values) > 0 {
		for _, value := range values {
			filter.ModelIDs = append(
				filter.ModelIDs,
				splitCSV(value)...,
			)
		}
	}

	sortOptions := listdom.Sort{}
	pageNumber := parseIntDefault(
		qp.Get("page"),
		1,
	)
	perPage := parseIntDefault(
		qp.Get("perPage"),
		50,
	)

	page := listdom.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}

	result, err := h.qMgmt.ListRows(
		ctx,
		filter,
		sortOptions,
		page,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"items":      result.Items,
			"totalCount": result.TotalCount,
			"totalPages": result.TotalPages,
			"page":       result.Page,
			"perPage":    result.PerPage,
		},
	)
}

func (h *ListHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if h == nil || h.qDetail == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_implemented",
			},
		)
		return
	}

	dto, err := h.qDetail.BuildListDetailDTO(
		ctx,
		id,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func (h *ListHandler) deleteImage(
	w http.ResponseWriter,
	r *http.Request,
	listID string,
	imageID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid listId"},
		)
		return
	}

	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "imageId is required",
			},
		)
		return
	}

	if err := h.uc.DeleteImage(
		ctx,
		listID,
		imageID,
	); err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"ok":      true,
			"listId":  listID,
			"imageId": imageID,
		},
	)
}

// createImageFromFirebaseStorage stores a list image record.
//
// Current policy:
// - frontend uploads images directly to Firebase Storage.
// - backend receives and stores only the Firebase Storage download URL.
// - backend does not validate or persist objectPath, fileName, contentType, or size.
func (h *ListHandler) createImageFromFirebaseStorage(
	w http.ResponseWriter,
	r *http.Request,
	listID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	uid, ok := requireCurrentFirebaseUID(w, r)
	if !ok {
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid listId"},
		)
		return
	}

	var request struct {
		ID           string `json:"id"`
		URL          string `json:"url"`
		DisplayOrder int    `json:"displayOrder"`
	}

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid json"},
		)
		return
	}

	request.ID = strings.TrimSpace(request.ID)
	request.URL = strings.TrimSpace(request.URL)

	if request.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "id is required"},
		)
		return
	}

	if strings.Contains(request.ID, "/") ||
		strings.Contains(request.ID, "://") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid image id"},
		)
		return
	}

	if request.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "url is required"},
		)
		return
	}

	if request.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "displayOrder must be >= 0",
			},
		)
		return
	}

	now := time.Now().UTC()
	image, err := h.uc.CreateImage(
		ctx,
		listdom.ListImage{
			ID:           request.ID,
			ListID:       listID,
			URL:          request.URL,
			DisplayOrder: request.DisplayOrder,
			CreatedAt:    now,
			CreatedBy:    uid,
		},
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(image)
}

func (h *ListHandler) setPrimaryImage(
	w http.ResponseWriter,
	r *http.Request,
	listID string,
) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "usecase is nil"},
		)
		return
	}

	uid, ok := requireCurrentFirebaseUID(w, r)
	if !ok {
		return
	}

	var request struct {
		ImageID string `json:"imageId"`
	}

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{"error": "invalid json"},
		)
		return
	}

	imageID := strings.TrimSpace(request.ImageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "imageId is required",
			},
		)
		return
	}

	now := time.Now().UTC()
	item, err := h.uc.SetPrimaryImage(
		ctx,
		listID,
		imageID,
		now,
		&uid,
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_implemented",
				},
			)
			return
		}

		writeConsoleListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(item)
}
