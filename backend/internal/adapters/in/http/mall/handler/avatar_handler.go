// backend/internal/adapters/in/http/mall/handler/avatar_handler.go
package mallHandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"cloud.google.com/go/storage"

	uc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
)

// AvatarHandler handles ONLY mall buyer-facing endpoints.
//
// Routes (mall only):
// - POST   /mall/avatars
// - GET    /mall/avatars/{id}
// - PATCH  /mall/avatars/{id}
// - PUT    /mall/avatars/{id}
//
// Extensions:
// - POST /mall/avatars/{id}/wallet
// - POST /mall/avatars/{id}/icon-upload-url
// - POST /mall/avatars/{id}/icon
type AvatarHandler struct {
	uc *uc.AvatarUsecase
}

// NewAvatarHandler initializes handler.
func NewAvatarHandler(avatarUC *uc.AvatarUsecase) http.Handler {
	return &AvatarHandler{uc: avatarUC}
}

// ServeHTTP routes requests (mall path only).
func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// ------------------------------------------------------------
	// /mall/avatars (list not supported)
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && path0 == "/mall/avatars":
		// current AvatarUsecase doesn't provide listing
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return

	// POST /mall/avatars
	case r.Method == http.MethodPost && path0 == "/mall/avatars":
		h.post(w, r)
		return

	// ------------------------------------------------------------
	// subroutes MUST be checked before /mall/avatars/{id}
	// ------------------------------------------------------------

	// POST /mall/avatars/{id}/wallet
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/wallet"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/wallet")
		if !ok {
			notFound(w)
			return
		}
		h.openWallet(w, r, id)
		return

	// POST /mall/avatars/{id}/icon-upload-url
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/icon-upload-url"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/icon-upload-url")
		if !ok {
			notFound(w)
			return
		}
		h.issueIconUploadURL(w, r, id)
		return

	// POST /mall/avatars/{id}/icon
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/icon"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/icon")
		if !ok {
			notFound(w)
			return
		}
		h.replaceIcon(w, r, id)
		return

	// ------------------------------------------------------------
	// /mall/avatars/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/mall/avatars/"):
		id, ok := extractIDFromPath(path0, "/mall/avatars/")
		if !ok {
			notFound(w)
			return
		}
		h.get(w, r, id)
		return

	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(path0, "/mall/avatars/"):
		id, ok := extractIDFromPath(path0, "/mall/avatars/")
		if !ok {
			notFound(w)
			return
		}
		h.update(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// -----------------------------------------------------------------------------
// DTOs (Mall response) - ✅ avatarId に寄せる
// -----------------------------------------------------------------------------

// avatarResponse is the mall-facing Avatar DTO.
// It normalizes the ID field name to "avatarId" (absolute schema).
type avatarResponse struct {
	AvatarID string `json:"avatarId"`
	UserID   string `json:"userId"`

	AvatarName string `json:"avatarName"`

	// URL/gs://.../path (optional)
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	AvatarState   avatarstate.AvatarState `json:"avatarState"`
	WalletAddress *string                 `json:"walletAddress,omitempty"`
	Profile       *string                 `json:"profile,omitempty"`
	ExternalLink  *string                 `json:"externalLink,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
	DeletedAt     *time.Time              `json:"deletedAt,omitempty"`
}

func toAvatarResponse(a avatardom.Avatar) avatarResponse {
	return avatarResponse{
		AvatarID:      strings.TrimSpace(a.ID),
		UserID:        strings.TrimSpace(a.UserID),
		AvatarName:    strings.TrimSpace(a.AvatarName),
		AvatarIcon:    a.AvatarIcon,
		AvatarState:   a.AvatarState,
		WalletAddress: a.WalletAddress,
		Profile:       a.Profile,
		ExternalLink:  a.ExternalLink,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		DeletedAt:     a.DeletedAt,
	}
}

// avatarIconResponse is the mall-facing icon DTO.
// It normalizes "avatarId" and keeps other fields stable.
type avatarIconResponse struct {
	ID       string  `json:"id"`
	AvatarID *string `json:"avatarId,omitempty"`
	URL      string  `json:"url"`
	FileName *string `json:"fileName,omitempty"`
	Size     *int64  `json:"size,omitempty"`
}

func toAvatarIconResponse(icon avataricon.AvatarIcon, knownAvatarID string) avatarIconResponse {
	aid := strings.TrimSpace(knownAvatarID)
	var aidPtr *string
	if aid != "" {
		aidPtr = &aid
	}

	// NOTE: icon struct field names are not shown here; we rely on those used in this handler (ID/URL/FileName/Size).
	// If avataricon.AvatarIcon already has AvatarID, it can be wired later; for now, we guarantee avatarId via knownAvatarID.
	return avatarIconResponse{
		ID:       strings.TrimSpace(icon.ID),
		AvatarID: aidPtr,
		URL:      strings.TrimSpace(icon.URL),
		FileName: trimPtr(icon.FileName),
		Size:     icon.Size,
	}
}

// avatarAggregateResponse is the mall-facing aggregate DTO.
type avatarAggregateResponse struct {
	Avatar avatarResponse       `json:"avatar"`
	State  any                  `json:"state,omitempty"` // pass-through (already expected to be "avatarId" in domain)
	Icons  []avatarIconResponse `json:"icons"`
}

// -----------------------------------------------------------------------------
// GET /mall/avatars/{id}
// aggregate=1|true -> Avatar + State + Icons aggregate
// -----------------------------------------------------------------------------
func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[mall_avatar_handler] GET /mall/avatars/%s aggregate=%q\n", id, r.URL.Query().Get("aggregate"))

	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}

		iconsCount := len(data.Icons)
		hasAvatarIconField := data.Avatar.AvatarIcon != nil && strings.TrimSpace(*data.Avatar.AvatarIcon) != ""
		sampleIconURL := ""
		if iconsCount > 0 {
			sampleIconURL = data.Icons[0].URL
		}
		log.Printf(
			"[mall_avatar_handler] GET /mall/avatars/%s aggregate ok iconsCount=%d avatar.avatarIcon_set=%t icons_sample_url=%q\n",
			id,
			iconsCount,
			hasAvatarIconField,
			sampleIconURL,
		)

		icons := make([]avatarIconResponse, 0, len(data.Icons))
		for _, ic := range data.Icons {
			icons = append(icons, toAvatarIconResponse(ic, id))
		}

		out := avatarAggregateResponse{
			Avatar: toAvatarResponse(data.Avatar),
			State:  data.State, // pass-through
			Icons:  icons,
		}

		_ = json.NewEncoder(w).Encode(out)
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := avatar.AvatarIcon != nil && strings.TrimSpace(*avatar.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] GET /mall/avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(avatar.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(toAvatarResponse(avatar))
}

// -----------------------------------------------------------------------------
// POST /mall/avatars
// -----------------------------------------------------------------------------
func (h *AvatarHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	in := uc.CreateAvatarInput{
		UserID:       strings.TrimSpace(body.UserID),
		UserUID:      strings.TrimSpace(body.UserUID),
		AvatarName:   strings.TrimSpace(body.AvatarName),
		AvatarIcon:   trimPtr(body.AvatarIcon),   // shared helper_handler.go
		Profile:      trimPtr(body.Profile),      // shared helper_handler.go
		ExternalLink: trimPtr(body.ExternalLink), // shared helper_handler.go
	}

	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars request userId=%q userUid=%q avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		in.UserID,
		maskUID(in.UserUID), // shared
		in.AvatarName,
		ptrStr(in.AvatarIcon),   // shared
		ptrLen(in.Profile),      // shared
		ptrStr(in.ExternalLink), // shared
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars error=%v\n", err)
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := created.AvatarIcon != nil && strings.TrimSpace(*created.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars ok avatarId=%q walletAddress=%q avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		created.ID,
		ptrStr(created.WalletAddress),
		hasAvatarIconField,
		ptrStr(created.AvatarIcon),
	)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAvatarResponse(created))
}

// -----------------------------------------------------------------------------
// POST /mall/avatars/{id}/wallet
// -----------------------------------------------------------------------------
func (h *AvatarHandler) openWallet(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet request\n", id)

	// conflict if already opened
	a, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet get error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet conflict walletAddress=%q\n", id, ptrStr(a.WalletAddress))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet already opened"})
		return
	}

	updated, err := h.uc.OpenWallet(ctx, id)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/wallet ok walletAddress=%q\n", id, ptrStr(updated.WalletAddress))
	_ = json.NewEncoder(w).Encode(toAvatarResponse(updated))
}

// -----------------------------------------------------------------------------
// POST /mall/avatars/{id}/icon-upload-url
// - issues GCS Signed URL (PUT)
// - client then registers via POST /mall/avatars/{id}/icon
// -----------------------------------------------------------------------------
//
// Cloud Run:
// - if key file exists (GCS_SIGNER_CREDENTIALS/GOOGLE_APPLICATION_CREDENTIALS) -> use keyfile signing
// - else use IAM Credentials API (signBlob) (requires roles/iam.serviceAccountTokenCreator)
func (h *AvatarHandler) issueIconUploadURL(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var body struct {
		FileName *string `json:"fileName,omitempty"`
		MimeType *string `json:"mimeType,omitempty"`
		Size     *int64  `json:"size,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	fileName := strings.TrimSpace(ptrStr(body.FileName))
	mimeType := strings.TrimSpace(ptrStr(body.MimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// bucket
	bucket := strings.TrimSpace(os.Getenv("AVATAR_ICON_BUCKET"))
	if bucket == "" {
		bucket = "narratives-development_avatar_icon"
	}

	// objectPath: "<avatarId>/<random>.<ext>"
	oid := newObjectID()
	ext := guessExt(fileName, mimeType)
	objectPath := fmt.Sprintf("%s/%s%s", id, oid, ext)

	// signer email (Cloud Run)
	// NOTE: GCS_SIGNER_EMAIL のみを使用する（フォールバック削除）
	signerEmail := strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL"))

	// local: key file signing if present
	credPath := strings.TrimSpace(os.Getenv("GCS_SIGNER_CREDENTIALS"))
	if credPath == "" {
		credPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}

	exp := time.Now().Add(15 * time.Minute)

	// ------------------------------------------------------------
	// A) keyfile signing (local)
	// ------------------------------------------------------------
	if credPath != "" {
		email, pk, err := loadServiceAccountKey(credPath)
		if err != nil {
			log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url load key error=%v credPath=%q\n", id, err, credPath)
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
			log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url sign (keyfile) error=%v\n", id, err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
			return
		}

		log.Printf(
			"[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url ok (keyfile) bucket=%q objectPath=%q mimeType=%q size=%v\n",
			id, bucket, objectPath, mimeType, body.Size,
		)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"uploadUrl":  uploadURL,
			"bucket":     bucket,
			"objectPath": objectPath,
			"gsUrl":      fmt.Sprintf("gs://%s/%s", bucket, objectPath),
			"expiresAt":  exp.UTC().Format(time.RFC3339),
		})
		return
	}

	// ------------------------------------------------------------
	// B) IAM Credentials signing (Cloud Run)
	// ------------------------------------------------------------
	if signerEmail == "" {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url missing signer email (set GCS_SIGNER_EMAIL)\n", id)
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
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url sign (iam) error=%v signer=%q bucket=%q objectPath=%q\n", id, err, signerEmail, bucket, objectPath)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
		return
	}

	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url ok (iam) signer=%q bucket=%q objectPath=%q mimeType=%q size=%v\n",
		id, signerEmail, bucket, objectPath, mimeType, body.Size,
	)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadUrl":  uploadURL,
		"bucket":     bucket,
		"objectPath": objectPath,
		"gsUrl":      fmt.Sprintf("gs://%s/%s", bucket, objectPath),
		"expiresAt":  exp.UTC().Format(time.RFC3339),
	})
}

// signBytesWithIAM signs bytes via IAM Credentials API SignBlob.
func signBytesWithIAM(ctx context.Context, signerEmail string, payload []byte) ([]byte, error) {
	c, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	name := fmt.Sprintf("projects/-/serviceAccounts/%s", signerEmail)
	resp, err := c.SignBlob(ctx, &credentialspb.SignBlobRequest{
		Name:    name,
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}
	return resp.SignedBlob, nil
}

// -----------------------------------------------------------------------------
// POST /mall/avatars/{id}/icon
// -----------------------------------------------------------------------------
func (h *AvatarHandler) replaceIcon(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var body struct {
		Bucket     *string `json:"bucket,omitempty"`
		ObjectPath *string `json:"objectPath,omitempty"`
		FileName   *string `json:"fileName,omitempty"`
		Size       *int64  `json:"size,omitempty"`

		// compatibility: client may send gs://... in avatarIcon
		AvatarIcon *string `json:"avatarIcon,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	bucket := strings.TrimSpace(ptrStr(body.Bucket))
	obj := strings.TrimSpace(ptrStr(body.ObjectPath))

	// if avatarIcon is gs://..., parse and override
	if v := strings.TrimSpace(ptrStr(body.AvatarIcon)); v != "" {
		if b, o, ok := avataricon.ParseGCSURL(v); ok {
			bucket = b
			obj = o
		}
	}

	if bucket == "" {
		bucket = "narratives-development_avatar_icon"
	}
	if obj == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath is required"})
		return
	}

	in := uc.ReplaceIconInput{
		Bucket:     bucket,
		ObjectPath: strings.TrimLeft(obj, "/"),
		FileName:   trimPtr(body.FileName),
		Size:       body.Size,
	}

	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars/%s/icon request bucket=%q objectPath=%q fileName=%q size=%v\n",
		id,
		in.Bucket,
		in.ObjectPath,
		ptrStr(in.FileName),
		in.Size,
	)

	ic, err := h.uc.ReplaceAvatarIcon(ctx, id, in)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	// best-effort: also patch avatars.avatarIcon for UI
	updatedAvatarIcon := ""
	if strings.TrimSpace(ic.URL) != "" {
		url := strings.TrimSpace(ic.URL)
		_, _ = h.uc.Update(ctx, id, avatardom.AvatarPatch{AvatarIcon: &url})
		updatedAvatarIcon = url
	}

	hasURL := strings.TrimSpace(ic.URL) != ""
	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars/%s/icon ok iconId=%q url_set=%t url=%q avatar_patch_avatarIcon=%q\n",
		id,
		ic.ID,
		hasURL,
		ic.URL,
		updatedAvatarIcon,
	)

	_ = json.NewEncoder(w).Encode(toAvatarIconResponse(ic, id))
}

// -----------------------------------------------------------------------------
// PATCH/PUT /mall/avatars/{id}
// -----------------------------------------------------------------------------
func (h *AvatarHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if _, ok := raw["walletAddress"]; ok {
		log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s rejected: walletAddress field is not allowed\n", id)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "walletAddress is not allowed in update"})
		return
	}

	bs, merr := json.Marshal(raw)
	if merr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s raw=%q\n", id, headString(bs, 300))

	var body struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}
	if err := json.Unmarshal(bs, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	patch := avatardom.AvatarPatch{
		AvatarName:   trimPtrNilAware(body.AvatarName),
		AvatarIcon:   trimPtrNilAware(body.AvatarIcon),
		Profile:      trimPtrNilAware(body.Profile),
		ExternalLink: trimPtrNilAware(body.ExternalLink),
	}

	log.Printf(
		"[mall_avatar_handler] PATCH/PUT /mall/avatars/%s request avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		id,
		ptrStr(patch.AvatarName),
		ptrStr(patch.AvatarIcon),
		ptrLen(patch.Profile),
		ptrStr(patch.ExternalLink),
	)

	updated, err := h.uc.Update(ctx, id, patch)
	if err != nil {
		log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := updated.AvatarIcon != nil && strings.TrimSpace(*updated.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] PATCH/PUT /mall/avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(updated.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(toAvatarResponse(updated))
}

// -----------------------------------------------------------------------------
// Error handling
// -----------------------------------------------------------------------------
func writeAvatarErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, avatardom.ErrInvalidID) ||
		errors.Is(err, avatardom.ErrInvalidUserID) ||
		errors.Is(err, uc.ErrInvalidUserUID) ||
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

func trimPtrNilAware(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return ptr("")
	}
	return &s
}

func ptr[T any](v T) *T { return &v }

func hasErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

// -----------------------------------------------------------------------------
// helpers (path parsing)
// -----------------------------------------------------------------------------

// extractIDFromPath extracts {id} from "/mall/avatars/{id}" (no subroutes allowed).
func extractIDFromPath(path0 string, prefix string) (string, bool) {
	if !strings.HasPrefix(path0, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path0, prefix)
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if len(parts) > 1 {
		// subroute exists -> not this endpoint
		return "", false
	}
	return id, true
}

// extractIDFromSubroute extracts {id} from "/mall/avatars/{id}<suffix>".
// Example: prefix="/mall/avatars/", suffix="/wallet" => "/mall/avatars/abc/wallet" -> "abc"
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
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if len(parts) > 1 {
		return "", false
	}
	return id, true
}

// -----------------------------------------------------------------------------
// helpers (upload-url)
// -----------------------------------------------------------------------------
func newObjectID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func guessExt(fileName string, mimeType string) string {
	ext := strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
	if ext != "" && len(ext) <= 10 {
		return ext
	}

	mt := strings.ToLower(strings.TrimSpace(mimeType))
	switch mt {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

type serviceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func loadServiceAccountKey(filepath string) (email string, privateKey []byte, err error) {
	bs, err := os.ReadFile(filepath)
	if err != nil {
		return "", nil, err
	}
	var k serviceAccountKey
	if err := json.Unmarshal(bs, &k); err != nil {
		return "", nil, err
	}
	e := strings.TrimSpace(k.ClientEmail)
	pk := strings.TrimSpace(k.PrivateKey)
	if e == "" || pk == "" {
		return "", nil, fmt.Errorf("missing client_email/private_key in credentials")
	}
	return e, []byte(pk), nil
}
