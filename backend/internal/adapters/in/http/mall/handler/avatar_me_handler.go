// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	mallfs "narratives/internal/adapters/out/firestore/mall"
	mallquery "narratives/internal/application/query/mall"
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

type AvatarStateResolvedQuery interface {
	GetResolvedByAvatarID(ctx context.Context, avatarID string) (mallquery.AvatarStateResolvedView, error)
}

type MeAvatarHandler struct {
	Repo             *mallfs.MeAvatarRepo
	AvatarUC         *avataruc.AvatarUsecase
	AvatarStateQuery AvatarStateResolvedQuery
}

func NewMeAvatarHandler(
	repo *mallfs.MeAvatarRepo,
	avatarUC *avataruc.AvatarUsecase,
	avatarStateQuery AvatarStateResolvedQuery,
) http.Handler {
	return &MeAvatarHandler{
		Repo:             repo,
		AvatarUC:         avatarUC,
		AvatarStateQuery: avatarStateQuery,
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
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_handler_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
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

func (h *MeAvatarHandler) resolveAvatarIDByUID(ctx context.Context, uid string) (string, string, error) {
	if h == nil || h.Repo == nil {
		return "", "", errors.New("me avatar handler not configured")
	}

	avatarID, walletAddress, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", "", err
	}

	if avatarID == "" {
		return "", walletAddress, avatardom.ErrInvalidID
	}

	return avatarID, walletAddress, nil
}

func (h *MeAvatarHandler) resolveAvatarPatchByUID(ctx context.Context, uid string) (string, avatardom.AvatarPatch, error) {
	avatarID, walletAddress, err := h.resolveAvatarIDByUID(ctx, uid)
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

	if h.AvatarUC == nil {
		base.Sanitize()
		return avatarID, base, nil
	}

	av, gerr := h.AvatarUC.GetByID(ctx, avatarID)
	if gerr != nil {
		base.Sanitize()
		return avatarID, base, nil
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
	return avatarID, patch, nil
}

func (h *MeAvatarHandler) updateAvatarPatchByUID(ctx context.Context, uid string, patch avatardom.AvatarPatch) (string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	if h.AvatarUC == nil {
		return "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarID, walletAddress, err := h.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	patch.UserID = ""
	patch.WalletAddress = nil
	patch.DeletedAt = nil
	patch.Sanitize()

	updated, uerr := h.AvatarUC.Update(ctx, avatarID, patch)
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
	return avatarID, out, nil
}

func (h *MeAvatarHandler) handleGet(w http.ResponseWriter, r *http.Request, uid string) {
	avatarID, patch, err := h.resolveAvatarPatchByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	if avatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	patch.Sanitize()

	if patch.WalletAddress == nil || *patch.WalletAddress == "" {
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
		AvatarID:      avatarID,
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
	if h == nil || h.AvatarStateQuery == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_state_query_not_configured"})
		return
	}

	avatarID, _, err := h.resolveAvatarIDByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	out, err := h.AvatarStateQuery.GetResolvedByAvatarID(r.Context(), avatarID)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	body := string(raw)
	if body == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "empty_body"})
		return
	}

	var req meAvatarUpdateRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	if req.AvatarName == nil &&
		req.Profile == nil &&
		req.ExternalLink == nil &&
		req.AvatarIcon == nil {
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

	avatarID, outPatch, uerr := h.updateAvatarPatchByUID(r.Context(), uid, patch)
	if uerr != nil {
		writeMeAvatarErr(w, uerr)
		return
	}

	if avatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	outPatch.Sanitize()

	if outPatch.WalletAddress == nil || *outPatch.WalletAddress == "" {
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
		AvatarID:      avatarID,
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

	if h == nil || h.AvatarUC == nil {
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

	myAvatarID, _, err := h.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	result, err := h.AvatarUC.FollowAvatar(ctx, myAvatarID, req.TargetAvatarID)
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
		MeAvatarID:     result.MeAvatarID,
		TargetAvatarID: result.TargetAvatarID,
		Following:      result.Following,
		MeState:        result.MeState,
		TargetState:    result.TargetState,
	})
}

func (h *MeAvatarHandler) handleUnfollow(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	if h == nil || h.AvatarUC == nil {
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

	myAvatarID, _, err := h.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	result, err := h.AvatarUC.UnfollowAvatar(ctx, myAvatarID, req.TargetAvatarID)
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
		MeAvatarID:     result.MeAvatarID,
		TargetAvatarID: result.TargetAvatarID,
		Following:      result.Following,
		MeState:        result.MeState,
		TargetState:    result.TargetState,
	})
}

func writeMeAvatarErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	switch {
	case isNotFoundLike(err):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return

	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		w.WriteHeader(http.StatusRequestTimeout)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
		return

	case errors.Is(err, avatardom.ErrInvalidID):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return

	case errors.Is(err, avatardom.ErrInvalidAvatarName),
		errors.Is(err, avatardom.ErrInvalidAvatarIcon),
		errors.Is(err, avatardom.ErrInvalidProfile),
		errors.Is(err, avatardom.ErrInvalidExternalLink):
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}
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
