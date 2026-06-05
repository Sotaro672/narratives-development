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
	avatarstate "narratives/internal/domain/avatarState"
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

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path0, "/mall/avatars/") &&
		strings.HasSuffix(path0, "/wallet"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/wallet")
		if !ok {
			notFound(w)
			return
		}
		h.openWallet(w, r, id)
		return

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path0, "/mall/avatars/") &&
		strings.HasSuffix(path0, "/state"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/state")
		if !ok {
			notFound(w)
			return
		}
		h.getState(w, r, id)
		return

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(path0, "/mall/avatars/") &&
		strings.HasSuffix(path0, "/state"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/state")
		if !ok {
			notFound(w)
			return
		}
		h.upsertState(w, r, id)
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

func extractIDFromSubroute(path0 string, prefix string, suffix string) (string, bool) {
	if !strings.HasPrefix(path0, prefix) || !strings.HasSuffix(path0, suffix) {
		return "", false
	}

	rest := strings.TrimPrefix(path0, prefix)
	rest = strings.TrimSuffix(rest, suffix)
	rest = strings.Trim(rest, "/")
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

func (h *AvatarHandler) openWallet(w http.ResponseWriter, r *http.Request, id string) {
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

	a, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	if a.WalletAddress != nil && *a.WalletAddress != "" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet already opened"})
		return
	}

	updated, err := h.uc.OpenWallet(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(toAvatarResponse(updated))
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
		s := strings.TrimSpace(*body.AvatarIcon)
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

type resolvedAvatarFollowRefResponse struct {
	AvatarID   string     `json:"avatarId"`
	AvatarName string     `json:"avatarName,omitempty"`
	AvatarIcon string     `json:"avatarIcon,omitempty"`
	FollowedAt *time.Time `json:"followedAt,omitempty"`
}

type avatarStatePatchResponse struct {
	AvatarID       string                            `json:"avatarId"`
	FollowerCount  *int64                            `json:"followerCount,omitempty"`
	FollowingCount *int64                            `json:"followingCount,omitempty"`
	PostCount      *int64                            `json:"postCount,omitempty"`
	Followers      []resolvedAvatarFollowRefResponse `json:"followers"`
	Following      []resolvedAvatarFollowRefResponse `json:"following"`
	LastActiveAt   time.Time                         `json:"lastActiveAt"`
	UpdatedAt      *time.Time                        `json:"updatedAt,omitempty"`
}

func (h *AvatarHandler) resolveFollowRefs(
	ctx context.Context,
	refs []avatarstate.AvatarFollowRef,
) []resolvedAvatarFollowRefResponse {
	if len(refs) == 0 {
		return []resolvedAvatarFollowRefResponse{}
	}

	out := make([]resolvedAvatarFollowRefResponse, 0, len(refs))

	for _, ref := range refs {
		item := resolvedAvatarFollowRefResponse{
			AvatarID: ref.AvatarID,
		}

		if !ref.FollowedAt.IsZero() {
			t := ref.FollowedAt.UTC()
			item.FollowedAt = &t
		}

		if h != nil && h.avatarRepo != nil && ref.AvatarID != "" {
			a, err := h.avatarRepo.GetByID(ctx, ref.AvatarID)
			if err == nil {
				item.AvatarName = a.AvatarName
				if a.AvatarIcon != nil {
					item.AvatarIcon = *a.AvatarIcon
				}
			}
		}

		out = append(out, item)
	}

	return out
}

func (h *AvatarHandler) toAvatarStatePatchResponse(
	ctx context.Context,
	avatarID string,
	st *avatarstate.AvatarState,
) avatarStatePatchResponse {
	if st == nil {
		return avatarStatePatchResponse{
			AvatarID:  avatarID,
			Followers: []resolvedAvatarFollowRefResponse{},
			Following: []resolvedAvatarFollowRefResponse{},
			UpdatedAt: nil,
		}
	}

	return avatarStatePatchResponse{
		AvatarID:       avatarID,
		FollowerCount:  st.FollowerCount,
		FollowingCount: st.FollowingCount,
		PostCount:      st.PostCount,
		Followers:      h.resolveFollowRefs(ctx, st.Followers),
		Following:      h.resolveFollowRefs(ctx, st.Following),
		LastActiveAt:   st.LastActiveAt,
		UpdatedAt:      st.UpdatedAt,
	}
}

func (h *AvatarHandler) getState(w http.ResponseWriter, r *http.Request, id string) {
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

	data, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(h.toAvatarStatePatchResponse(ctx, id, data.State))
}

func (h *AvatarHandler) upsertState(w http.ResponseWriter, r *http.Request, id string) {
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

	var body struct {
		FollowerCount  *int64                         `json:"followerCount,omitempty"`
		FollowingCount *int64                         `json:"followingCount,omitempty"`
		PostCount      *int64                         `json:"postCount,omitempty"`
		Followers      *[]avatarstate.AvatarFollowRef `json:"followers,omitempty"`
		Following      *[]avatarstate.AvatarFollowRef `json:"following,omitempty"`
		LastActiveAt   *time.Time                     `json:"lastActiveAt,omitempty"`
		UpdatedAt      *time.Time                     `json:"updatedAt,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	patch := avataruc.AvatarStatePatch{
		FollowerCount:  body.FollowerCount,
		FollowingCount: body.FollowingCount,
		PostCount:      body.PostCount,
		Followers:      body.Followers,
		Following:      body.Following,
		LastActiveAt:   body.LastActiveAt,
		UpdatedAt:      body.UpdatedAt,
	}

	updated, err := h.uc.UpdateAvatarState(ctx, id, patch)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

type avatarResponse struct {
	AvatarID string `json:"avatarId"`
	UserID   string `json:"userId"`

	AvatarName string  `json:"avatarName"`
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	AvatarState *avatarstate.AvatarState `json:"avatarState,omitempty"`

	WalletAddress *string   `json:"walletAddress,omitempty"`
	Profile       *string   `json:"profile,omitempty"`
	ExternalLink  *string   `json:"externalLink,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func toAvatarResponse(a avatardom.Avatar) avatarResponse {
	var stPtr *avatarstate.AvatarState
	if a.AvatarState.ID != "" {
		tmp := a.AvatarState
		stPtr = &tmp
	}

	return avatarResponse{
		AvatarID:      a.ID,
		UserID:        a.UserID,
		AvatarName:    a.AvatarName,
		AvatarIcon:    a.AvatarIcon,
		AvatarState:   stPtr,
		WalletAddress: a.WalletAddress,
		Profile:       a.Profile,
		ExternalLink:  a.ExternalLink,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

type avatarAggregateResponse struct {
	Avatar avatarResponse           `json:"avatar"`
	State  *avatarstate.AvatarState `json:"state,omitempty"`
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

	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}

		out := avatarAggregateResponse{
			Avatar: toAvatarResponse(data.Avatar),
			State:  data.State,
		}

		_ = json.NewEncoder(w).Encode(out)
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
		errors.Is(err, avatardom.ErrInvalidExternalLink) ||
		errors.Is(err, avatarstate.ErrInvalidFollowerAvatarID) ||
		errors.Is(err, avatarstate.ErrInvalidFollowingAvatarID) ||
		errors.Is(err, avatarstate.ErrInvalidFollowedAt) ||
		errors.Is(err, avatarstate.ErrDuplicateFollowerAvatar) ||
		errors.Is(err, avatarstate.ErrDuplicateFollowingAvatar) ||
		errors.Is(err, avatarstate.ErrSelfFollowerRelation) ||
		errors.Is(err, avatarstate.ErrSelfFollowingRelation) ||
		errors.Is(err, avatarstate.ErrFollowerCountMismatch) ||
		errors.Is(err, avatarstate.ErrFollowingCountMismatch) {
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
