// backend/internal/adapters/in/http/introduction/contact_handler.go
package introduction

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	contactuc "narratives/internal/application/usecase"
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
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}

		h.handleCreate(w, r)
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

func toResponse(c contact.Contact) contactResponse {
	createdAt := ""
	if !c.CreatedAt.IsZero() {
		createdAt = c.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	return contactResponse{
		ID:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		Company:   c.Company,
		Message:   c.Message,
		Status:    c.Status,
		Source:    c.Source,
		CreatedAt: createdAt,
		UpdatedAt: nil,
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
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
