// backend\internal\adapters\in\http\console\handler\message_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	uc "narratives/internal/application/usecase"
	msgdom "narratives/internal/domain/message"
)

// MessageHandler wires HTTP to MessageUsecase.
type MessageHandler struct {
	uc   *uc.MessageUsecase
	repo msgdom.Repository // for simple GET by id
}

// NewMessageHandler initializes the HTTP handler.
// repo is used only for GET /messages/{id}.
func NewMessageHandler(messageUC *uc.MessageUsecase, repo msgdom.Repository) http.Handler {
	return &MessageHandler{uc: messageUC, repo: repo}
}

// ServeHTTP routes requests.
func (h *MessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	switch {
	// GET /messages/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/messages/"):
		id := strings.TrimPrefix(path, "/messages/")
		h.get(w, r, id)

	// POST /messages/drafts
	case r.Method == http.MethodPost && path == "/messages/drafts":
		h.createDraft(w, r)

	// POST /messages/{id}/send
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/send"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/messages/"), "/send")
		h.send(w, r, strings.Trim(id, "/"))

	// POST /messages/{id}/cancel
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/cancel"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/messages/"), "/cancel")
		h.cancel(w, r, strings.Trim(id, "/"))

	// POST /messages/{id}/delivered
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/delivered"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/messages/"), "/delivered")
		h.delivered(w, r, strings.Trim(id, "/"))

	// POST /messages/{id}/read
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/read"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/messages/"), "/read")
		h.read(w, r, strings.Trim(id, "/"))

	// DELETE /messages/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/messages/"):
		id := strings.TrimPrefix(path, "/messages/")
		h.deleteCascade(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ============ GET /messages/{id} ============
func (h *MessageHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	if h.repo == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	msg, err := h.repo.GetByID(ctx, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// ============ POST /messages/drafts ============
type createDraftRequest struct {
	ID         string               `json:"id"`
	SenderID   string               `json:"senderId"`
	ReceiverID string               `json:"receiverId"`
	Content    string               `json:"content"`
	Images     []createDraftImageIn `json:"images"`
}
type createDraftImageIn struct {
	FileName   string `json:"fileName"`
	ObjectPath string `json:"objectPath,omitempty"`
	FileURL    string `json:"fileUrl,omitempty"`
	FileSize   int64  `json:"fileSize"`
	MimeType   string `json:"mimeType"`
	Width      *int   `json:"width,omitempty"`
	Height     *int   `json:"height,omitempty"`
}

func (h *MessageHandler) createDraft(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	ctx := r.Context()

	var req createDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	now := time.Now().UTC()
	imgs := make([]uc.NewImageInput, 0, len(req.Images))
	for _, im := range req.Images {
		imgs = append(imgs, uc.NewImageInput{
			FileName:   im.FileName,
			ObjectPath: im.ObjectPath,
			FileURL:    im.FileURL,
			FileSize:   im.FileSize,
			MimeType:   im.MimeType,
			Width:      im.Width,
			Height:     im.Height,
			UploadedAt: now,
		})
	}

	out, err := h.uc.CreateDraftMessage(ctx, uc.CreateDraftInput{
		ID:         req.ID,
		SenderID:   req.SenderID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
		Images:     imgs,
	})
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(out)
}

// ============ POST /messages/{id}/send ============
func (h *MessageHandler) send(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	msg, err := h.uc.SendMessage(ctx, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// ============ POST /messages/{id}/cancel ============
func (h *MessageHandler) cancel(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	msg, err := h.uc.CancelMessage(ctx, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// ============ POST /messages/{id}/delivered ============
func (h *MessageHandler) delivered(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	msg, err := h.uc.MarkDelivered(ctx, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// ============ POST /messages/{id}/read ============
func (h *MessageHandler) read(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	msg, err := h.uc.MarkRead(ctx, id)
	if err != nil {
		writeDomainErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(msg)
}

// ============ DELETE /messages/{id} ============
func (h *MessageHandler) deleteCascade(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	if h.uc == nil {
		http.Error(w, `{"error":"not_configured"}`, http.StatusInternalServerError)
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		badRequest(w, "invalid id")
		return
	}
	if err := h.uc.DeleteMessageCascade(ctx, id); err != nil {
		writeDomainErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============ helpers ============
func badRequest(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeDomainErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case msgdom.ErrInvalidID:
		code = http.StatusBadRequest
	case msgdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
