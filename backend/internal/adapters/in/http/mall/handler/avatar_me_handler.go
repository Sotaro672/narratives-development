// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	avatardom "narratives/internal/domain/avatar"

	"narratives/internal/adapters/in/http/middleware"
)

// MeAvatarService resolves + updates avatar by Firebase uid.
//
// Policy (me-only):
// - uid は認証コンテキストから取得し、クライアント入力では受けない
// - avatarId はサーバで uid -> avatarId を解決する
//
// IMPORTANT (avatarIcon policy: Pattern1 / Recommended B):
// - avatarIcon の「文字列(URL)」は原則固定（変えない）
// - 画像実体の更新/削除は GCS object の overwrite / delete で行う
// - PATCH /mall/me/avatar は avatarIcon を受け取っても DB 更新には使わない（フロントが送らない前提だが、受けても無害に捨てる）
//
// Endpoints:
// - GET    /mall/me/avatar
// - PATCH  /mall/me/avatar                 (avatarName/profile/externalLink only; avatarIcon is ignored if present)
// - POST   /mall/me/avatar/icon-upload-url (issue signed PUT url for FIXED objectPath derived from existing avatarIcon URL)
// - DELETE /mall/me/avatar/icon-object     (delete GCS object ONLY; avatarIcon string remains)
type MeAvatarService interface {
	ResolveAvatarPatchByUID(ctx context.Context, uid string) (avatarId string, patch avatardom.AvatarPatch, err error)

	// UpdateAvatarPatchByUID applies a partial update (AvatarPatch) to "me" avatar resolved by uid.
	// The service MUST internally resolve uid -> avatarId and update that avatar (anti-spoof).
	UpdateAvatarPatchByUID(ctx context.Context, uid string, patch avatardom.AvatarPatch) (avatarId string, outPatch avatardom.AvatarPatch, err error)
}

type MeAvatarHandler struct {
	Svc MeAvatarService
}

func NewMeAvatarHandler(svc MeAvatarService) http.Handler {
	return &MeAvatarHandler{Svc: svc}
}

const (
	meAvatarPath           = "/mall/me/avatar"
	meAvatarIconUploadPath = "/mall/me/avatar/icon-upload-url"
	meAvatarIconObjectPath = "/mall/me/avatar/icon-object"
)

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// always JSON
	w.Header().Set("Content-Type", "application/json")

	// ---- logging: entrypoint (must show if routing reaches here) ----
	log.Printf(
		"[mall_me_avatar_handler] HIT method=%s path=%s rawPath=%s query=%q origin=%q",
		r.Method,
		r.URL.Path,
		r.URL.EscapedPath(),
		r.URL.RawQuery,
		strings.TrimSpace(r.Header.Get("Origin")),
	)

	// CORS preflight
	if r.Method == http.MethodOptions {
		log.Printf("[mall_me_avatar_handler] OPTIONS -> 204")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// handler/service readiness
	if h == nil || h.Svc == nil {
		log.Printf("[mall_me_avatar_handler] service_not_initialized (h_nil=%t svc_nil=%t)", h == nil, h == nil || h.Svc == nil)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_service_not_initialized"})
		return
	}

	// auth principal
	uid, ok := middleware.CurrentUserUID(r)
	uid = strings.TrimSpace(uid)
	if !ok || uid == "" {
		log.Printf("[mall_me_avatar_handler] unauthorized: missing uid (ok=%t uid=%q)", ok, uid)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	// ------------------------------------------------------------------
	// routing inside this handler (me avatar + me icon ops)
	// ------------------------------------------------------------------
	switch {
	// GET /mall/me/avatar
	case r.Method == http.MethodGet && path0 == meAvatarPath:
		h.handleGet(w, r, uid)
		return

	// PATCH /mall/me/avatar
	case r.Method == http.MethodPatch && path0 == meAvatarPath:
		h.handlePatch(w, r, uid)
		return

	// POST /mall/me/avatar/icon-upload-url  (issue signed PUT url for FIXED objectPath)
	case r.Method == http.MethodPost && path0 == meAvatarIconUploadPath:
		h.handleIssueIconUploadURL(w, r, uid)
		return

	// DELETE /mall/me/avatar/icon-object    (delete GCS object ONLY; avatarIcon string remains)
	case r.Method == http.MethodDelete && path0 == meAvatarIconObjectPath:
		h.handleDeleteIconObject(w, r, uid)
		return

	default:
		log.Printf("[mall_me_avatar_handler] not_found_or_method_not_allowed method=%s path=%s", r.Method, path0)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *MeAvatarHandler) handleGet(w http.ResponseWriter, r *http.Request, uid string) {
	// ---- resolve ----
	log.Printf("[mall_me_avatar_handler] resolve start uid=%q", maskUID(uid))
	avatarId, patch, err := h.Svc.ResolveAvatarPatchByUID(r.Context(), uid)
	if err != nil {
		if isNotFoundLike(err) {
			log.Printf("[mall_me_avatar_handler] resolve not_found uid=%q err=%v", maskUID(uid), err)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] resolve timeout uid=%q err=%v", maskUID(uid), err)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		log.Printf("[mall_me_avatar_handler] resolve internal_error uid=%q err=%v", maskUID(uid), err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		log.Printf("[mall_me_avatar_handler] resolve empty avatarId uid=%q", maskUID(uid))
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	// sanitize patch (trim + empty -> nil)
	patch.Sanitize()

	// walletAddress required by frontend
	if patch.WalletAddress == nil || strings.TrimSpace(*patch.WalletAddress) == "" {
		log.Printf("[mall_me_avatar_handler] wallet_not_initialized avatarId=%q uid=%q", avatarId, maskUID(uid))
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
		UserID:        strings.TrimSpace(patch.UserID),
		AvatarName:    patch.AvatarName,
		AvatarIcon:    patch.AvatarIcon,
		WalletAddress: patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
		DeletedAt:     patch.DeletedAt,
	}

	log.Printf(
		"[mall_me_avatar_handler] OK avatarId=%q userId=%q avatarName_set=%t avatarIcon_set=%t wallet_set=%t profile_set=%t externalLink_set=%t deletedAt_set=%t",
		out.AvatarID,
		out.UserID,
		out.AvatarName != nil && strings.TrimSpace(*out.AvatarName) != "",
		out.AvatarIcon != nil && strings.TrimSpace(*out.AvatarIcon) != "",
		out.WalletAddress != nil && strings.TrimSpace(*out.WalletAddress) != "",
		out.Profile != nil && strings.TrimSpace(*out.Profile) != "",
		out.ExternalLink != nil && strings.TrimSpace(*out.ExternalLink) != "",
		out.DeletedAt != nil,
	)

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handlePatch(w http.ResponseWriter, r *http.Request, uid string) {
	// PATCH /mall/me/avatar
	//
	// Allowed:
	// - avatarName
	// - profile ("" allowed => clear)
	// - externalLink ("" allowed => clear)
	//
	// avatarIcon:
	// - Accepted for compatibility (Recommended B) BUT IGNORED (DB は更新しない)
	// - Icon update/delete is done via:
	//   - POST   /mall/me/avatar/icon-upload-url  (signed PUT for fixed objectPath)
	//   - DELETE /mall/me/avatar/icon-object      (delete object only)
	type meAvatarUpdateRequest struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"` // accepted but ignored
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[mall_me_avatar_handler] patch read_body_failed uid=%q err=%v", maskUID(uid), err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}
	body := strings.TrimSpace(string(raw))
	if body == "" {
		log.Printf("[mall_me_avatar_handler] patch empty_body uid=%q", maskUID(uid))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "empty_body"})
		return
	}

	var req meAvatarUpdateRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		log.Printf("[mall_me_avatar_handler] patch decode_failed uid=%q err=%v body=%q", maskUID(uid), err, truncate(body, 500))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	// normalize:
	// - avatarName: trim; empty is invalid if provided
	// - profile/externalLink: trim; empty string is kept (=> clear)
	trimKeepEmpty := func(p **string) {
		if p == nil || *p == nil {
			return
		}
		s := strings.TrimSpace(**p)
		*p = &s // keep "" to represent "clear"
	}
	trimKeepEmpty(&req.Profile)
	trimKeepEmpty(&req.ExternalLink)

	if req.AvatarName != nil {
		s := strings.TrimSpace(*req.AvatarName)
		if s == "" {
			log.Printf("[mall_me_avatar_handler] patch invalid_avatarName uid=%q", maskUID(uid))
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_avatar_name"})
			return
		}
		req.AvatarName = &s
	}

	// avatarIcon is ignored (but we log if present to make behavior explicit)
	if req.AvatarIcon != nil {
		log.Printf(
			"[mall_me_avatar_handler] patch avatarIcon_ignored uid=%q avatarIcon=%q",
			maskUID(uid),
			truncate(strings.TrimSpace(*req.AvatarIcon), 200),
		)
	}

	// must have at least one *effective* field to update (avatarIcon does not count)
	if req.AvatarName == nil && req.Profile == nil && req.ExternalLink == nil {
		log.Printf("[mall_me_avatar_handler] patch no_fields uid=%q", maskUID(uid))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no_fields_to_update"})
		return
	}

	patch := avatardom.AvatarPatch{
		UserID:        "", // never updated here
		AvatarName:    req.AvatarName,
		AvatarIcon:    nil,              // never updated here (固定URL)
		WalletAddress: nil,              // never updated here
		Profile:       req.Profile,      // "" allowed => clear
		ExternalLink:  req.ExternalLink, // "" allowed => clear
		DeletedAt:     nil,              // never updated here
	}

	log.Printf(
		"[mall_me_avatar_handler] patch start uid=%q avatarName_set=%t profile_set=%t externalLink_set=%t",
		maskUID(uid),
		patch.AvatarName != nil,
		patch.Profile != nil,
		patch.ExternalLink != nil,
	)

	avatarId, outPatch, uerr := h.Svc.UpdateAvatarPatchByUID(r.Context(), uid, patch)
	if uerr != nil {
		if isNotFoundLike(uerr) {
			log.Printf("[mall_me_avatar_handler] patch not_found uid=%q err=%v", maskUID(uid), uerr)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}
		if errors.Is(uerr, context.Canceled) || errors.Is(uerr, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] patch timeout uid=%q err=%v", maskUID(uid), uerr)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}
		log.Printf("[mall_me_avatar_handler] patch internal_error uid=%q err=%v", maskUID(uid), uerr)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		log.Printf("[mall_me_avatar_handler] patch empty avatarId uid=%q", maskUID(uid))
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	// sanitize patch (trim + empty -> nil)
	// NOTE: outPatch.Sanitize() が空文字を nil に落とす実装だと「clear結果」が消える可能性があります。
	// ここでは呼び出しを維持しますが、clear をレスポンスで保持したいなら Sanitize() の仕様も点検してください。
	outPatch.Sanitize()

	if outPatch.WalletAddress == nil || strings.TrimSpace(*outPatch.WalletAddress) == "" {
		log.Printf("[mall_me_avatar_handler] patch wallet_not_initialized avatarId=%q uid=%q", avatarId, maskUID(uid))
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
		UserID:        strings.TrimSpace(outPatch.UserID),
		AvatarName:    outPatch.AvatarName,
		AvatarIcon:    outPatch.AvatarIcon,
		WalletAddress: outPatch.WalletAddress,
		Profile:       outPatch.Profile,
		ExternalLink:  outPatch.ExternalLink,
		DeletedAt:     outPatch.DeletedAt,
	}

	log.Printf(
		"[mall_me_avatar_handler] patch OK avatarId=%q userId=%q avatarName_set=%t avatarIcon_set=%t wallet_set=%t profile_set=%t externalLink_set=%t deletedAt_set=%t",
		out.AvatarID,
		out.UserID,
		out.AvatarName != nil && strings.TrimSpace(*out.AvatarName) != "",
		out.AvatarIcon != nil && strings.TrimSpace(*out.AvatarIcon) != "",
		out.WalletAddress != nil && strings.TrimSpace(*out.WalletAddress) != "",
		out.Profile != nil,      // may be "" for clear
		out.ExternalLink != nil, // may be "" for clear
		out.DeletedAt != nil,
	)

	_ = json.NewEncoder(w).Encode(out)
}

// POST /mall/me/avatar/icon-upload-url
// - issues Signed URL (PUT) for FIXED objectPath derived from existing avatarIcon URL (DB).
// - overwrites the same object so avatarIcon URL string can remain constant.
func (h *MeAvatarHandler) handleIssueIconUploadURL(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	// resolve avatarId + current patch (anti-spoof + to get fixed avatarIcon url)
	avatarId, patch, err := h.Svc.ResolveAvatarPatchByUID(ctx, uid)
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
	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	// fixed avatarIcon URL must exist in DB (Pattern1)
	iconURL := ""
	if patch.AvatarIcon != nil {
		iconURL = strings.TrimSpace(*patch.AvatarIcon)
	}
	if iconURL == "" {
		log.Printf("[mall_me_avatar_handler] POST %s avatarIcon_not_set avatarId=%q uid=%q", meAvatarIconUploadPath, avatarId, maskUID(uid))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_icon_not_set"})
		return
	}

	// parse bucket/objectPath from existing avatarIcon url
	bucket, objectPath, ok := parseBucketObjectFromAvatarIconURL(iconURL)
	if !ok {
		log.Printf("[mall_me_avatar_handler] POST %s invalid_avatarIcon_url avatarId=%q avatarIcon=%q", meAvatarIconUploadPath, avatarId, truncate(iconURL, 300))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed_to_parse_avatar_icon_url"})
		return
	}

	// request body (optional)
	var body struct {
		MimeType *string `json:"mimeType,omitempty"`
		Size     *int64  `json:"size,omitempty"`
	}
	// Allow empty body
	if r.Body != nil {
		bs, _ := io.ReadAll(r.Body)
		s := strings.TrimSpace(string(bs))
		if s != "" {
			if err := json.Unmarshal([]byte(s), &body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
				return
			}
		}
	}

	mimeType := strings.TrimSpace(ptrStr(body.MimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// signer email (Cloud Run)
	// NOTE: GCS_SIGNER_EMAIL のみを使用する（フォールバック削除）
	signerEmail := strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL"))

	// local: key file signing if present
	credPath := strings.TrimSpace(os.Getenv("GCS_SIGNER_CREDENTIALS"))
	if credPath == "" {
		credPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}

	exp := time.Now().Add(15 * time.Minute)

	// canonical public url for response (keep stable)
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectPath)

	// A) keyfile signing (local)
	if credPath != "" {
		email, pk, err := loadServiceAccountKey(credPath)
		if err != nil {
			log.Printf("[mall_me_avatar_handler] POST %s load key error=%v credPath=%q", meAvatarIconUploadPath, err, credPath)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load signer key"})
			return
		}

		uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
			GoogleAccessID: email,
			PrivateKey:     pk,
			Method:         http.MethodPut,
			Expires:        exp,
			ContentType:    mimeType,
		})
		if err != nil {
			log.Printf("[mall_me_avatar_handler] POST %s sign (keyfile) error=%v", meAvatarIconUploadPath, err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
			return
		}

		log.Printf(
			"[mall_me_avatar_handler] POST %s ok (keyfile) avatarId=%q bucket=%q objectPath=%q mimeType=%q size=%v",
			meAvatarIconUploadPath, avatarId, bucket, objectPath, mimeType, body.Size,
		)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"uploadUrl":   uploadURL,
			"bucket":      bucket,
			"objectPath":  objectPath,
			"gsUrl":       fmt.Sprintf("gs://%s/%s", bucket, objectPath),
			"publicUrl":   publicURL,
			"expiresAt":   exp.UTC().Format(time.RFC3339),
			"contentType": mimeType,
		})
		return
	}

	// B) IAM Credentials signing (Cloud Run)
	if signerEmail == "" {
		log.Printf("[mall_me_avatar_handler] POST %s missing signer email (set GCS_SIGNER_EMAIL)", meAvatarIconUploadPath)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing signer email (set GCS_SIGNER_EMAIL)"})
		return
	}

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		GoogleAccessID: signerEmail,
		Method:         http.MethodPut,
		Expires:        exp,
		ContentType:    mimeType,
		SignBytes: func(b []byte) ([]byte, error) {
			return signBytesWithIAM(ctx, signerEmail, b)
		},
	})
	if err != nil {
		log.Printf("[mall_me_avatar_handler] POST %s sign (iam) error=%v signer=%q bucket=%q objectPath=%q", meAvatarIconUploadPath, err, signerEmail, bucket, objectPath)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
		return
	}

	log.Printf(
		"[mall_me_avatar_handler] POST %s ok (iam) avatarId=%q signer=%q bucket=%q objectPath=%q mimeType=%q size=%v",
		meAvatarIconUploadPath, avatarId, signerEmail, bucket, objectPath, mimeType, body.Size,
	)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadUrl":   uploadURL,
		"bucket":      bucket,
		"objectPath":  objectPath,
		"gsUrl":       fmt.Sprintf("gs://%s/%s", bucket, objectPath),
		"publicUrl":   publicURL,
		"expiresAt":   exp.UTC().Format(time.RFC3339),
		"contentType": mimeType,
	})
}

// DELETE /mall/me/avatar/icon-object
// - deletes ONLY the GCS object derived from existing avatarIcon URL (DB).
// - does NOT update avatar.avatarIcon field (URL string remains).
func (h *MeAvatarHandler) handleDeleteIconObject(w http.ResponseWriter, r *http.Request, uid string) {
	ctx := r.Context()

	avatarId, patch, err := h.Svc.ResolveAvatarPatchByUID(ctx, uid)
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
	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	iconURL := ""
	if patch.AvatarIcon != nil {
		iconURL = strings.TrimSpace(*patch.AvatarIcon)
	}
	if iconURL == "" {
		log.Printf("[mall_me_avatar_handler] DELETE %s avatarIcon_not_set ok avatarId=%q uid=%q", meAvatarIconObjectPath, avatarId, maskUID(uid))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	bucket, objectPath, ok := parseBucketObjectFromAvatarIconURL(iconURL)
	if !ok {
		log.Printf("[mall_me_avatar_handler] DELETE %s invalid_avatarIcon_url avatarId=%q avatarIcon=%q", meAvatarIconObjectPath, avatarId, truncate(iconURL, 300))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed_to_parse_avatar_icon_url"})
		return
	}

	c, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("[mall_me_avatar_handler] DELETE %s storage.NewClient failed avatarId=%q err=%v", meAvatarIconObjectPath, avatarId, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed_to_init_storage_client"})
		return
	}
	defer func() { _ = c.Close() }()

	obj := c.Bucket(bucket).Object(objectPath)
	if err := obj.Delete(ctx); err != nil {
		// treat "not exist" as OK (idempotent delete)
		if errors.Is(err, storage.ErrObjectNotExist) {
			log.Printf("[mall_me_avatar_handler] DELETE %s object_not_exist ok avatarId=%q bucket=%q objectPath=%q", meAvatarIconObjectPath, avatarId, bucket, objectPath)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		log.Printf("[mall_me_avatar_handler] DELETE %s delete_failed avatarId=%q bucket=%q objectPath=%q err=%v", meAvatarIconObjectPath, avatarId, bucket, objectPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed_to_delete_object"})
		return
	}

	log.Printf("[mall_me_avatar_handler] DELETE %s ok avatarId=%q bucket=%q objectPath=%q", meAvatarIconObjectPath, avatarId, bucket, objectPath)
	w.WriteHeader(http.StatusNoContent)
}

func truncate(s string, max int) string {
	t := strings.TrimSpace(s)
	if max <= 0 {
		return ""
	}
	if len(t) <= max {
		return t
	}
	return t[:max] + "...(truncated)"
}

// parseBucketObjectFromAvatarIconURL parses bucket/objectPath from avatarIcon URL string.
// Supported:
// - gs://<bucket>/<objectPath>
// - https://storage.googleapis.com/<bucket>/<objectPath>
// - http://storage.googleapis.com/<bucket>/<objectPath>
func parseBucketObjectFromAvatarIconURL(u string) (bucket, objectPath string, ok bool) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", "", false
	}

	// gs://bucket/object
	if strings.HasPrefix(u, "gs://") {
		rest := strings.TrimPrefix(u, "gs://")
		rest = strings.TrimLeft(rest, "/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		b := strings.TrimSpace(parts[0])
		o := strings.TrimLeft(strings.TrimSpace(parts[1]), "/")
		if b == "" || o == "" {
			return "", "", false
		}
		return b, o, true
	}

	// public URL
	const p = "https://storage.googleapis.com/"
	var rest string
	switch {
	case strings.HasPrefix(u, p):
		rest = strings.TrimPrefix(u, p)
	default:
		return "", "", false
	}

	rest = strings.TrimLeft(rest, "/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	b := strings.TrimSpace(parts[0])
	o := strings.TrimLeft(strings.TrimSpace(parts[1]), "/")
	if b == "" || o == "" {
		return "", "", false
	}
	return b, o, true
}
