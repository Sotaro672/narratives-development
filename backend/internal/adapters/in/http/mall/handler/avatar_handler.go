// backend/internal/adapters/in/http/mall/handler/avatar_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	avataruc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
)

type AvatarSummaryRepository interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type AvatarHandler struct {
	uc         *avataruc.AvatarUsecase
	avatarRepo AvatarSummaryRepository
}

func NewAvatarHandler(
	avatarUC *avataruc.AvatarUsecase,
	avatarRepo AvatarSummaryRepository,
) http.Handler {
	return &AvatarHandler{
		uc:         avatarUC,
		avatarRepo: avatarRepo,
	}
}

func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodPost && path0 == "/mall/avatars":
		h.post(w, r)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/mall/avatars/"):
		id, ok := extractIDFromPath(path0, "/mall/avatars/")
		if !ok {
			notFound(w)
			return
		}
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func extractIDFromPath(path0 string, prefix string) (string, bool) {
	if !strings.HasPrefix(path0, prefix) {
		return "", false
	}

	rest := strings.TrimPrefix(path0, prefix)
	if rest == "" {
		return "", false
	}

	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		return "", false
	}

	if len(parts) > 1 {
		return "", false
	}

	return id, true
}

func headString(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}

	if len(b) > max {
		b = b[:max]
	}

	s := string(b)
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func ptrLen(p *string) int {
	if p == nil {
		return 0
	}

	return len([]rune(*p))
}

func (h *AvatarHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	var body struct {
		UserID       string  `json:"userId"`
		UserUID      string  `json:"userUid"`
		AvatarName   string  `json:"avatarName"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if body.AvatarIcon != nil {
		s := *body.AvatarIcon
		if s != "" &&
			!strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_avatar_icon"})
			return
		}
		body.AvatarIcon = &s
	}

	in := avataruc.CreateAvatarInput{
		UserID:       body.UserID,
		UserUID:      body.UserUID,
		AvatarName:   body.AvatarName,
		AvatarIcon:   body.AvatarIcon,
		Profile:      body.Profile,
		ExternalLink: body.ExternalLink,
	}

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAvatarResponse(created))
}

type avatarResponse struct {
	AvatarID string `json:"avatarId"`
	UserID   string `json:"userId"`

	AvatarName string  `json:"avatarName"`
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	WalletAddress *string   `json:"walletAddress,omitempty"`
	Profile       *string   `json:"profile,omitempty"`
	ExternalLink  *string   `json:"externalLink,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func toAvatarResponse(a avatardom.Avatar) avatarResponse {
	return avatarResponse{
		AvatarID:      a.ID,
		UserID:        a.UserID,
		AvatarName:    a.AvatarName,
		AvatarIcon:    a.AvatarIcon,
		WalletAddress: a.WalletAddress,
		Profile:       a.Profile,
		ExternalLink:  a.ExternalLink,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(toAvatarResponse(avatar))
}

func writeAvatarErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, avatardom.ErrInvalidID) ||
		errors.Is(err, avatardom.ErrInvalidUserID) ||
		errors.Is(err, avataruc.ErrInvalidUserUID) ||
		errors.Is(err, avatardom.ErrInvalidAvatarName) ||
		errors.Is(err, avatardom.ErrInvalidProfile) ||
		errors.Is(err, avatardom.ErrInvalidExternalLink) {
		code = http.StatusBadRequest
	}

	if hasErrNotFound(err) {
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func hasErrNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

var _ = avatardom.Avatar{}
