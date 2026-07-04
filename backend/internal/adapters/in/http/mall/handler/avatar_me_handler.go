// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	avataruc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
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
// Endpoints:
// - GET    /mall/me/avatars
// - PATCH  /mall/me/avatars        (avatarName/profile/externalLink/avatarIcon)
// - DELETE /mall/me/avatars

type MeAvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error)
}

type MeAvatarHandler struct {
	Repo     MeAvatarResolver
	AvatarUC *avataruc.AvatarUsecase
}

func NewMeAvatarHandler(
	repo MeAvatarResolver,
	avatarUC *avataruc.AvatarUsecase,
) http.Handler {
	return &MeAvatarHandler{
		Repo:     repo,
		AvatarUC: avatarUC,
	}
}

const meAvatarsPath = "/mall/me/avatars"

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

	case r.Method == http.MethodPatch && path0 == meAvatarsPath:
		h.handlePatch(w, r, uid)
		return

	case r.Method == http.MethodDelete && path0 == meAvatarsPath:
		h.handleDelete(w, r, uid)
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

func (h *MeAvatarHandler) ResolveAvatarByUID(
	ctx context.Context,
	uid string,
) (string, string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	avatarID, walletAddress, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", "", avatardom.AvatarPatch{}, err
	}

	if avatarID == "" {
		return "", walletAddress, avatardom.AvatarPatch{}, avatardom.ErrInvalidID
	}

	base := avatardom.AvatarPatch{
		UserID:        "",
		AvatarName:    nil,
		AvatarIcon:    nil,
		WalletAddress: strPtrTrim(walletAddress),
		Profile:       nil,
		ExternalLink:  nil,
	}

	if h.AvatarUC == nil {
		base.Sanitize()
		return avatarID, walletAddress, base, nil
	}

	av, gerr := h.AvatarUC.GetByID(ctx, avatarID)
	if gerr != nil {
		base.Sanitize()
		return avatarID, walletAddress, base, nil
	}

	patch := avatardom.AvatarPatch{
		UserID:        av.UserID,
		AvatarName:    strPtrTrim(av.AvatarName),
		AvatarIcon:    av.AvatarIcon,
		WalletAddress: av.WalletAddress,
		Profile:       av.Profile,
		ExternalLink:  av.ExternalLink,
	}

	if patch.WalletAddress == nil {
		patch.WalletAddress = strPtrTrim(walletAddress)
	}

	patch.Sanitize()
	return avatarID, walletAddress, patch, nil
}

func (h *MeAvatarHandler) updateAvatarPatchByUID(
	ctx context.Context,
	uid string,
	patch avatardom.AvatarPatch,
) (string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	if h.AvatarUC == nil {
		return "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarID, walletAddress, _, err := h.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	patch.UserID = ""
	patch.WalletAddress = nil
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
	}

	if out.WalletAddress == nil {
		out.WalletAddress = strPtrTrim(walletAddress)
	}

	out.Sanitize()
	return avatarID, out, nil
}

func (h *MeAvatarHandler) handleGet(w http.ResponseWriter, r *http.Request, uid string) {
	avatarID, _, patch, err := h.ResolveAvatarByUID(r.Context(), uid)
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
		AvatarID      string  `json:"avatarId"`
		UserID        string  `json:"userId"`
		AvatarName    *string `json:"avatarName,omitempty"`
		AvatarIcon    *string `json:"avatarIcon,omitempty"`
		WalletAddress *string `json:"walletAddress,omitempty"`
		Profile       *string `json:"profile,omitempty"`
		ExternalLink  *string `json:"externalLink,omitempty"`
	}

	out := meAvatarPatchResponse{
		AvatarID:      avatarID,
		UserID:        patch.UserID,
		AvatarName:    patch.AvatarName,
		AvatarIcon:    patch.AvatarIcon,
		WalletAddress: patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
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
		AvatarID      string  `json:"avatarId"`
		UserID        string  `json:"userId"`
		AvatarName    *string `json:"avatarName,omitempty"`
		AvatarIcon    *string `json:"avatarIcon,omitempty"`
		WalletAddress *string `json:"walletAddress,omitempty"`
		Profile       *string `json:"profile,omitempty"`
		ExternalLink  *string `json:"externalLink,omitempty"`
	}

	out := meAvatarPatchResponse{
		AvatarID:      avatarID,
		UserID:        outPatch.UserID,
		AvatarName:    outPatch.AvatarName,
		AvatarIcon:    outPatch.AvatarIcon,
		WalletAddress: outPatch.WalletAddress,
		Profile:       outPatch.Profile,
		ExternalLink:  outPatch.ExternalLink,
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handleDelete(w http.ResponseWriter, r *http.Request, uid string) {
	if h == nil || h.AvatarUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	avatarID, _, _, err := h.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	if err := h.AvatarUC.Delete(r.Context(), avatarID); err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
