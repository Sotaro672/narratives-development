// backend/internal/adapters/in/http/introduction/contact_handler.go
package introduction

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	contactuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	contact "narratives/internal/domain/contact"
)

type ContactHandler struct {
	uc *contactuc.ContactUsecase
}

func NewContactHandler(uc *contactuc.ContactUsecase) *ContactHandler {
	return &ContactHandler{uc: uc}
}

type createContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

type contactResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Company   string         `json:"company"`
	Message   string         `json:"message"`
	Status    contact.Status `json:"status"`
	Source    string         `json:"source"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt *string        `json:"updatedAt"`
}

func (h *ContactHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/introduction/contacts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"error": "method not allowed",
			})
		}
	})

	mux.HandleFunc("/introduction/contacts/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/introduction/contacts/"):]
		if id == "" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleGetByID(w, r, id)
		case http.MethodPut, http.MethodPatch:
			h.handleUpdate(w, r, id)
		case http.MethodDelete:
			h.handleDelete(w, r, id)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
	})
}

func (h *ContactHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createContactRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Printf("[contact] invalid create request json: err=%v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	log.Printf("[contact] create request received: email=%s source=%s", req.Email, req.Source)

	out, err := h.uc.Create(r.Context(), contactuc.CreateInput{
		Name:    req.Name,
		Email:   req.Email,
		Company: req.Company,
		Message: req.Message,
		Source:  req.Source,
	})
	if err != nil {
		if errors.Is(err, contact.ErrInvalidName) ||
			errors.Is(err, contact.ErrInvalidEmail) ||
			errors.Is(err, contact.ErrInvalidCompany) ||
			errors.Is(err, contact.ErrInvalidMessage) {
			log.Printf("[contact] validation failed: email=%s err=%v", req.Email, err)
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		log.Printf("[contact] create failed: email=%s err=%v", req.Email, err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal error"})
		return
	}

	log.Printf("[contact] create succeeded: id=%s email=%s", out.ID, out.Email)
	writeJSON(w, http.StatusCreated, toResponse(out))
}

func (h *ContactHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	c, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("[contact] get by id failed: id=%s err=%v", id, err)
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, toResponse(c))
}

func (h *ContactHandler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var filter contact.Filter
	if s := q.Get("status"); s != "" {
		st := contact.Status(s)
		filter.Status = &st
	}

	page := common.Page{
		Number:  parseIntDefault(q.Get("page"), 1),
		PerPage: parseIntDefault(q.Get("perPage"), 20),
	}

	sort := common.Sort{
		Column: q.Get("sort"),
		Order:  common.SortOrder(q.Get("order")),
	}

	res, err := h.uc.List(r.Context(), filter, sort, page)
	if err != nil {
		log.Printf("[contact] list failed: err=%v", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	items := make([]contactResponse, 0, len(res.Items))
	for _, c := range res.Items {
		items = append(items, toResponse(c))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"totalCount": res.TotalCount,
		"totalPages": res.TotalPages,
		"page":       res.Page,
		"perPage":    res.PerPage,
	})
}

func (h *ContactHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var patch contact.Patch
	if err := decodeJSON(r, &patch); err != nil {
		log.Printf("[contact] update invalid json: id=%s err=%v", id, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	updated, err := h.uc.Update(r.Context(), id, patch)
	if err != nil {
		log.Printf("[contact] update failed: id=%s err=%v", id, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *ContactHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.uc.Delete(r.Context(), id); err != nil {
		log.Printf("[contact] delete failed: id=%s err=%v", id, err)
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func toResponse(c contact.Contact) contactResponse {
	createdAt := ""
	if !c.CreatedAt.IsZero() {
		createdAt = c.CreatedAt.UTC().Format(timeRFC3339())
	}

	var updatedAt *string

	return contactResponse{
		ID:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		Company:   c.Company,
		Message:   c.Message,
		Status:    c.Status,
		Source:    c.Source,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func timeRFC3339() string {
	return time.RFC3339Nano
}
