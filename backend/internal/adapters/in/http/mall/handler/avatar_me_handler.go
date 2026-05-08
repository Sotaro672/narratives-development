// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	avataruc "narratives/internal/application/usecase/avatar"
	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
)

// Policy (me-only):
// - uid は認証コンテキストから取得し、クライアント入力では受けない
// - avatarId はサーバで uid -> avatarId を解決する
//
// IMPORTANT (avatarIcon policy):
// - 画像アップロードは Firebase Storage 側で行う
// - PATCH /mall/me/avatars は avatarName/profile/externalLink/avatarIcon を更新する
// - avatarIcon には Firebase Storage の download URL を保存する
//
// Endpoints (primary only):
// - GET    /mall/me/avatars
// - GET    /mall/me/avatars/state
// - PATCH  /mall/me/avatars        (avatarName/profile/externalLink/avatarIcon)
// - POST   /mall/me/avatars/follow
// - DELETE /mall/me/avatars/follow

type AvatarSummaryRepository interface {
	GetNameAndIconByID(ctx context.Context, id string) (name string, icon string, err error)
}

type MeAvatarHandler struct {
	Repo       *mallfs.MeAvatarRepo
	AvatarUC   *avataruc.AvatarUsecase
	AvatarRepo AvatarSummaryRepository
}

func NewMeAvatarHandler(
	repo *mallfs.MeAvatarRepo,
	avatarUC *avataruc.AvatarUsecase,
	avatarRepo AvatarSummaryRepository,
) http.Handler {
	return &MeAvatarHandler{
		Repo:       repo,
		AvatarUC:   avatarUC,
		AvatarRepo: avatarRepo,
	}
}

const (
	meAvatarsPath       = "/mall/me/avatars"
	meAvatarsStatePath  = "/mall/me/avatars/state"
	meAvatarsFollowPath = "/mall/me/avatars/follow"
)

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.Repo == nil {
		log.Printf("[mall_me_avatar_handler] handler_not_initialized (h_nil=%t repo_nil=%t)", h == nil, h == nil || h.Repo == nil)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_handler_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		log.Printf("[mall_me_avatar_handler] unauthorized: missing uid (ok=%t uid=%q)", ok, uid)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == meAvatarsPath:
		h.handleGet(w, r, uid)
		return

	case r.Method == http.MethodGet && path0 == meAvatarsStatePath:
		h.handleGetState(w, r, uid)
		return

	case r.Method == http.MethodPatch && path0 == meAvatarsPath:
		h.handlePatch(w, r, uid)
		return

	case r.Method == http.MethodPost && path0 == meAvatarsFollowPath:
		h.handleFollow(w, r, uid)
		return

	case r.Method == http.MethodDelete && path0 == meAvatarsFollowPath:
		h.handleUnfollow(w, r, uid)
		return

	default:
		log.Printf("[mall_me_avatar_handler] not_found_or_method_not_allowed method=%s path=%s", r.Method, path0)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func strPtrTrim(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type resolvedAvatarFollowRefResponse struct {
	AvatarID   string     `json:"avatarId"`
	AvatarName string     `json:"avatarName,omitempty"`
	AvatarIcon string     `json:"avatarIcon,omitempty"`
	FollowedAt *time.Time `json:"followedAt,omitempty"`
}

type meAvatarStatePatchResponse struct {
	AvatarID       string                            `json:"avatarId"`
	FollowerCount  *int64                            `json:"followerCount,omitempty"`
	FollowingCount *int64                            `json:"followingCount,omitempty"`
	PostCount      *int64                            `json:"postCount,omitempty"`
	Followers      []resolvedAvatarFollowRefResponse `json:"followers"`
	Following      []resolvedAvatarFollowRefResponse `json:"following"`
	LastActiveAt   time.Time                         `json:"lastActiveAt"`
	UpdatedAt      *time.Time                        `json:"updatedAt,omitempty"`
}

func (h *MeAvatarHandler) resolveFollowRefs(
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

		if h != nil && h.AvatarRepo != nil && ref.AvatarID != "" {
			name, icon, err := h.AvatarRepo.GetNameAndIconByID(ctx, ref.AvatarID)
			if err == nil {
				item.AvatarName = name
				item.AvatarIcon = icon
			}
		}

		out = append(out, item)
	}

	return out
}

func (h *MeAvatarHandler) toMeAvatarStatePatchResponse(
	ctx context.Context,
	avatarID string,
	st *avatarstate.AvatarState,
) meAvatarStatePatchResponse {
	if st == nil {
		return meAvatarStatePatchResponse{
			AvatarID:  avatarID,
			Followers: []resolvedAvatarFollowRefResponse{},
			Following: []resolvedAvatarFollowRefResponse{},
		}
	}

	return meAvatarStatePatchResponse{
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

func (h *MeAvatarHandler) resolveAvatarPatchByUID(ctx context.Context, uid string) (string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	avatarId, walletAddress, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	base := avatardom.AvatarPatch{
		UserID:        "",
		AvatarName:    nil,
		AvatarIcon:    nil,
		WalletAddress: strPtrTrim(walletAddress),
		Profile:       nil,
		ExternalLink:  nil,
		DeletedAt:     nil,
	}

	if h.AvatarUC == nil || avatarId == "" {
		base.Sanitize()
		return avatarId, base, nil
	}

	av, gerr := h.AvatarUC.GetByID(ctx, avatarId)
	if gerr != nil {
		base.Sanitize()
		return avatarId, base, nil
	}

	patch := avatardom.AvatarPatch{
		UserID:        av.UserID,
		AvatarName:    strPtrTrim(av.AvatarName),
		AvatarIcon:    av.AvatarIcon,
		WalletAddress: av.WalletAddress,
		Profile:       av.Profile,
		ExternalLink:  av.ExternalLink,
		DeletedAt:     av.DeletedAt,
	}

	if patch.WalletAddress == nil {
		patch.WalletAddress = strPtrTrim(walletAddress)
	}

	patch.Sanitize()
	return avatarId, patch, nil
}

func (h *MeAvatarHandler) resolveAvatarStatePatchByUID(ctx context.Context, uid string) (string, *avatarstate.AvatarState, error) {
	if h == nil || h.Repo == nil {
		return "", nil, errors.New("me avatar handler not configured")
	}

	avatarId, _, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", nil, err
	}

	if avatarId == "" {
		return "", nil, avatardom.ErrInvalidID
	}

	if h.AvatarUC == nil {
		return "", nil, errors.New("avatar usecase not configured")
	}

	agg, err := h.AvatarUC.GetAggregate(ctx, avatarId)
	if err != nil {
		return "", nil, err
	}

	return avatarId, agg.State, nil
}

func (h *MeAvatarHandler) updateAvatarPatchByUID(ctx context.Context, uid string, patch avatardom.AvatarPatch) (string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	if h.AvatarUC == nil {
		return "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarId, walletAddress, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	if avatarId == "" {
		return "", avatardom.AvatarPatch{}, avatardom.ErrInvalidID
	}

	patch.UserID = ""
	patch.WalletAddress = nil
	patch.DeletedAt = nil
	patch.Sanitize()

	updated, uerr := h.AvatarUC.Update(ctx, avatarId, patch)
	if uerr != nil {
		return "", avatardom.AvatarPatch{}, uerr
	}

	out := avatardom.AvatarPatch{
		UserID:        updated.UserID,
		AvatarName:    strPtrTrim(updated.AvatarName),
		AvatarIcon:    updated.AvatarIcon,
		WalletAddress: updated.WalletAddress,
		Profile:       updated.Profile,
		ExternalLink:  updated.ExternalLink,
		DeletedAt:     updated.DeletedAt,
	}

	if out.WalletAddress == nil {
		out.WalletAddress = strPtrTrim(walletAddress)
	}

	out.Sanitize()
	return avatarId, out, nil
}

func (h *MeAvatarHandler) handleGet(w http.ResponseWriter, r *http.Request, uid string) {
	avatarId, patch, err := h.resolveAvatarPatchByUID(r.Context(), uid)
	if err != nil {
		if isNotFoundLike(err) {
			log.Printf("[mall_me_avatar_handler] resolve not_found uid=%q err=%v", uid, err)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] resolve timeout uid=%q err=%v", uid, err)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		log.Printf("[mall_me_avatar_handler] resolve internal_error uid=%q err=%v", uid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	patch.Sanitize()

	if patch.WalletAddress == nil || *patch.WalletAddress == "" {
		log.Printf("[mall_me_avatar_handler] wallet_not_initialized avatarId=%q uid=%q", avatarId, uid)
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet_address_not_initialized"})
		return
	}

	type meAvatarPatchResponse struct {
		AvatarID      string     `json:"avatarId"`
		UserID        string     `json:"userId"`
		AvatarName    *string    `json:"avatarName,omitempty"`
		AvatarIcon    *string    `json:"avatarIcon,omitempty"`
		WalletAddress *string    `json:"walletAddress,omitempty"`
		Profile       *string    `json:"profile,omitempty"`
		ExternalLink  *string    `json:"externalLink,omitempty"`
		DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	}

	out := meAvatarPatchResponse{
		AvatarID:      avatarId,
		UserID:        patch.UserID,
		AvatarName:    patch.AvatarName,
		AvatarIcon:    patch.AvatarIcon,
		WalletAddress: patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
		DeletedAt:     patch.DeletedAt,
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handleGetState(w http.ResponseWriter, r *http.Request, uid string) {
	avatarId, st, err := h.resolveAvatarStatePatchByUID(r.Context(), uid)
	if err != nil {
		if isNotFoundLike(err) {
			log.Printf("[mall_me_avatar_handler] state not_found uid=%q err=%v", uid, err)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] state timeout uid=%q err=%v", uid, err)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		log.Printf("[mall_me_avatar_handler] state internal_error uid=%q err=%v", uid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toMeAvatarStatePatchResponse(r.Context(), avatarId, st))
}

func (h *MeAvatarHandler) handlePatch(w http.ResponseWriter, r *http.Request, uid string) {
	type meAvatarUpdateRequest struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[mall_me_avatar_handler] patch read_body_failed uid=%q err=%v", uid, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	body := string(raw)
	if body == "" {
		log.Printf("[mall_me_avatar_handler] patch empty_body uid=%q", uid)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "empty_body"})
		return
	}

	var req meAvatarUpdateRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		log.Printf("[mall_me_avatar_handler] patch decode_failed uid=%q err=%v body=%q", uid, err, truncate(body, 500))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	trimKeepEmpty := func(p **string) {
		if p == nil || *p == nil {
			return
		}
		s := strings.TrimSpace(**p)
		*p = &s
	}

	trimKeepEmpty(&req.Profile)
	trimKeepEmpty(&req.ExternalLink)
	trimKeepEmpty(&req.AvatarIcon)

	if req.AvatarName != nil {
		s := strings.TrimSpace(*req.AvatarName)
		if s == "" {
			log.Printf("[mall_me_avatar_handler] patch invalid_avatarName uid=%q", uid)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_avatar_name"})
			return
		}
		req.AvatarName = &s
	}

	if req.AvatarIcon != nil {
		s := *req.AvatarIcon
		if s != "" &&
			!strings.HasPrefix(s, "http://") &&
			!strings.HasPrefix(s, "https://") {
			log.Printf("[mall_me_avatar_handler] patch invalid_avatarIcon uid=%q avatarIcon=%q", uid, truncate(s, 200))
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_avatar_icon"})
			return
		}

		log.Printf("[mall_me_avatar_handler] patch avatarIcon_set uid=%q avatarIcon=%q", uid, truncate(s, 200))
	}

	if req.AvatarName == nil &&
		req.Profile == nil &&
		req.ExternalLink == nil &&
		req.AvatarIcon == nil {
		log.Printf("[mall_me_avatar_handler] patch no_fields uid=%q", uid)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no_fields_to_update"})
		return
	}

	patch := avatardom.AvatarPatch{
		UserID:        "",
		AvatarName:    req.AvatarName,
		AvatarIcon:    req.AvatarIcon,
		WalletAddress: nil,
		Profile:       req.Profile,
		ExternalLink:  req.ExternalLink,
		DeletedAt:     nil,
	}

	avatarId, outPatch, uerr := h.updateAvatarPatchByUID(r.Context(), uid, patch)
	if uerr != nil {
		if isNotFoundLike(uerr) {
			log.Printf("[mall_me_avatar_handler] patch not_found uid=%q err=%v", uid, uerr)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}

		if errors.Is(uerr, context.Canceled) || errors.Is(uerr, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] patch timeout uid=%q err=%v", uid, uerr)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		log.Printf("[mall_me_avatar_handler] patch internal_error uid=%q err=%v", uid, uerr)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	outPatch.Sanitize()

	if outPatch.WalletAddress == nil || *outPatch.WalletAddress == "" {
		log.Printf("[mall_me_avatar_handler] patch wallet_not_initialized avatarId=%q uid=%q", avatarId, uid)
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet_address_not_initialized"})
		return
	}

	type meAvatarPatchResponse struct {
		AvatarID      string     `json:"avatarId"`
		UserID        string     `json:"userId"`
		AvatarName    *string    `json:"avatarName,omitempty"`
		AvatarIcon    *string    `json:"avatarIcon,omitempty"`
		WalletAddress *string    `json:"walletAddress,omitempty"`
		Profile       *string    `json:"profile,omitempty"`
		ExternalLink  *string    `json:"externalLink,omitempty"`
		DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	}

	out := meAvatarPatchResponse{
		AvatarID:      avatarId,
		UserID:        outPatch.UserID,
		AvatarName:    outPatch.AvatarName,
		AvatarIcon:    outPatch.AvatarIcon,
		WalletAddress: outPatch.WalletAddress,
		Profile:       outPatch.Profile,
		ExternalLink:  outPatch.ExternalLink,
		DeletedAt:     outPatch.DeletedAt,
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handleFollow(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	if h.AvatarUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	var req struct {
		TargetAvatarID string `json:"targetAvatarId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.TargetAvatarID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "targetAvatarId is required"})
		return
	}

	myAvatarID, _, err := h.resolveAvatarPatchByUID(ctx, uid)
	if err != nil {
		if isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if myAvatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	if myAvatarID == req.TargetAvatarID {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cannot_follow_self"})
		return
	}

	myAgg, err := h.AvatarUC.GetAggregate(ctx, myAvatarID)
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	targetAgg, err := h.AvatarUC.GetAggregate(ctx, req.TargetAvatarID)
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	now := time.Now().UTC()

	myFollowing := upsertFollowRef(nilSafeState(myAgg.State).Following, avatarstate.AvatarFollowRef{
		AvatarID:   req.TargetAvatarID,
		FollowedAt: now,
	})

	targetFollowers := upsertFollowRef(nilSafeState(targetAgg.State).Followers, avatarstate.AvatarFollowRef{
		AvatarID:   myAvatarID,
		FollowedAt: now,
	})

	myState, err := h.AvatarUC.UpdateAvatarState(ctx, myAvatarID, avatarstate.AvatarStatePatch{
		Following:    &myFollowing,
		LastActiveAt: &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	targetState, err := h.AvatarUC.UpdateAvatarState(ctx, req.TargetAvatarID, avatarstate.AvatarStatePatch{
		Followers:    &targetFollowers,
		LastActiveAt: &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	type followResponse struct {
		MeAvatarID     string                  `json:"meAvatarId"`
		TargetAvatarID string                  `json:"targetAvatarId"`
		Following      bool                    `json:"following"`
		MeState        avatarstate.AvatarState `json:"meState"`
		TargetState    avatarstate.AvatarState `json:"targetState"`
	}

	_ = json.NewEncoder(w).Encode(followResponse{
		MeAvatarID:     myAvatarID,
		TargetAvatarID: req.TargetAvatarID,
		Following:      true,
		MeState:        myState,
		TargetState:    targetState,
	})
}

func (h *MeAvatarHandler) handleUnfollow(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	if h.AvatarUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	var req struct {
		TargetAvatarID string `json:"targetAvatarId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.TargetAvatarID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "targetAvatarId is required"})
		return
	}

	myAvatarID, _, err := h.resolveAvatarPatchByUID(ctx, uid)
	if err != nil {
		if isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if myAvatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	if myAvatarID == req.TargetAvatarID {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cannot_unfollow_self"})
		return
	}

	myAgg, err := h.AvatarUC.GetAggregate(ctx, myAvatarID)
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	targetAgg, err := h.AvatarUC.GetAggregate(ctx, req.TargetAvatarID)
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	now := time.Now().UTC()

	myFollowing := removeFollowRef(nilSafeState(myAgg.State).Following, req.TargetAvatarID)
	targetFollowers := removeFollowRef(nilSafeState(targetAgg.State).Followers, myAvatarID)

	myState, err := h.AvatarUC.UpdateAvatarState(ctx, myAvatarID, avatarstate.AvatarStatePatch{
		Following:    &myFollowing,
		LastActiveAt: &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	targetState, err := h.AvatarUC.UpdateAvatarState(ctx, req.TargetAvatarID, avatarstate.AvatarStatePatch{
		Followers:    &targetFollowers,
		LastActiveAt: &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		writeFollowErr(w, err)
		return
	}

	type unfollowResponse struct {
		MeAvatarID     string                  `json:"meAvatarId"`
		TargetAvatarID string                  `json:"targetAvatarId"`
		Following      bool                    `json:"following"`
		MeState        avatarstate.AvatarState `json:"meState"`
		TargetState    avatarstate.AvatarState `json:"targetState"`
	}

	_ = json.NewEncoder(w).Encode(unfollowResponse{
		MeAvatarID:     myAvatarID,
		TargetAvatarID: req.TargetAvatarID,
		Following:      false,
		MeState:        myState,
		TargetState:    targetState,
	})
}

func truncate(s string, max int) string {
	t := s
	if max <= 0 {
		return ""
	}

	if len(t) <= max {
		return t
	}

	return t[:max] + "...(truncated)"
}

func nilSafeState(st *avatarstate.AvatarState) avatarstate.AvatarState {
	if st == nil {
		return avatarstate.AvatarState{}
	}

	return *st
}

func upsertFollowRef(items []avatarstate.AvatarFollowRef, ref avatarstate.AvatarFollowRef) []avatarstate.AvatarFollowRef {
	out := make([]avatarstate.AvatarFollowRef, 0, len(items)+1)
	found := false

	for _, item := range items {
		if item.AvatarID == ref.AvatarID {
			out = append(out, avatarstate.AvatarFollowRef{
				AvatarID:   item.AvatarID,
				FollowedAt: ref.FollowedAt.UTC(),
			})
			found = true
			continue
		}

		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   item.AvatarID,
			FollowedAt: item.FollowedAt.UTC(),
		})
	}

	if !found {
		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   ref.AvatarID,
			FollowedAt: ref.FollowedAt.UTC(),
		})
	}

	return out
}

func removeFollowRef(items []avatarstate.AvatarFollowRef, avatarID string) []avatarstate.AvatarFollowRef {
	if len(items) == 0 {
		return nil
	}

	out := make([]avatarstate.AvatarFollowRef, 0, len(items))

	for _, item := range items {
		if item.AvatarID == avatarID {
			continue
		}

		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   item.AvatarID,
			FollowedAt: item.FollowedAt.UTC(),
		})
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func writeFollowErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	switch {
	case errors.Is(err, avatardom.ErrInvalidID),
		errors.Is(err, avatarstate.ErrInvalidFollowerAvatarID),
		errors.Is(err, avatarstate.ErrInvalidFollowingAvatarID),
		errors.Is(err, avatarstate.ErrInvalidFollowedAt),
		errors.Is(err, avatarstate.ErrDuplicateFollowerAvatar),
		errors.Is(err, avatarstate.ErrDuplicateFollowingAvatar),
		errors.Is(err, avatarstate.ErrSelfFollowerRelation),
		errors.Is(err, avatarstate.ErrSelfFollowingRelation),
		errors.Is(err, avatarstate.ErrFollowerCountMismatch),
		errors.Is(err, avatarstate.ErrFollowingCountMismatch):
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case isNotFoundLike(err):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}
