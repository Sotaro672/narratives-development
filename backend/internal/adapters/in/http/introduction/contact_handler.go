// backend/internal/adapters/in/http/introduction/contact_handler.go
package introduction

import (
	"encoding/json"
	"errors"
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
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Company   string  `json:"company"`
	Message   string  `json:"message"`
	IsRead    bool    `json:"isRead"`
	Source    string  `json:"source"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
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
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

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
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal error"})
		return
	}

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
		IsRead:    c.IsRead,
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
