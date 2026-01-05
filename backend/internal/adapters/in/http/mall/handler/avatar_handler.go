// backend\internal\adapters\in\http\mall\handler\avatar_handler.go
package handler

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
)

// AvatarHandler は /avatars 関連のエンドポイントを担当します。
// 新しい usecase.AvatarUsecase を利用します。
type AvatarHandler struct {
	uc *uc.AvatarUsecase
}

// NewAvatarHandler はHTTPハンドラを初期化します。
func NewAvatarHandler(avatarUC *uc.AvatarUsecase) http.Handler {
	return &AvatarHandler{uc: avatarUC}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 元のパス（末尾 / を落とす）
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /sns/avatars にも対応するため /sns を剥がして /avatars 系に正規化する
	// - /sns/avatars        -> /avatars
	// - /sns/avatars/{id}   -> /avatars/{id}
	// - /avatars            -> /avatars (そのまま)
	if strings.HasPrefix(path0, "/sns/") {
		path0 = strings.TrimPrefix(path0, "/sns")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	case r.Method == http.MethodGet && path0 == "/avatars":
		// 現行の AvatarUsecase は一覧取得を提供しないため 501 で返す
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})

	case r.Method == http.MethodPost && path0 == "/avatars":
		h.post(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/avatars/"):
		id := strings.TrimPrefix(path0, "/avatars/")
		h.get(w, r, id)

	// ✅ NEW: wallet open endpoint
	// POST /avatars/{id}/wallet
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/avatars/") && strings.HasSuffix(path0, "/wallet"):
		id := strings.TrimSuffix(strings.TrimPrefix(path0, "/avatars/"), "/wallet")
		h.openWallet(w, r, id)

	// ✅ NEW: avatar icon upload-url endpoint (B案)
	// POST /avatars/{id}/icon-upload-url
	// - GCS Signed URL を発行してクライアントが PUT アップロードできるようにする
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/avatars/") && strings.HasSuffix(path0, "/icon-upload-url"):
		id := strings.TrimSuffix(strings.TrimPrefix(path0, "/avatars/"), "/icon-upload-url")
		h.issueIconUploadURL(w, r, id)

	// ✅ NEW: avatar icon register/replace endpoint
	// POST /avatars/{id}/icon
	// - 事前にクライアントが GCS にアップロード済みの object を「登録」する
	// - 既存アイコンがあれば best-effort で削除（usecase側）
	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/avatars/") && strings.HasSuffix(path0, "/icon"):
		id := strings.TrimSuffix(strings.TrimPrefix(path0, "/avatars/"), "/icon")
		h.replaceIcon(w, r, id)

	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(path0, "/avatars/"):
		id := strings.TrimPrefix(path0, "/avatars/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete && strings.HasPrefix(path0, "/avatars/"):
		id := strings.TrimPrefix(path0, "/avatars/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// -----------------------------------------------------------------------------
// GET /avatars/{id}
// aggregate=1|true を付けると Avatar + State + Icons の集約を返します。
// -----------------------------------------------------------------------------
func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_avatar_handler] GET /avatars/%s aggregate=%q\n", id, r.URL.Query().Get("aggregate"))

	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}

		// ✅ avatar icon を保持できているか確認（Aggregate の Icons / AvatarIcon を観測）
		iconsCount := len(data.Icons)
		hasAvatarIconField := data.Avatar.AvatarIcon != nil && strings.TrimSpace(*data.Avatar.AvatarIcon) != ""
		sampleIconURL := ""
		if iconsCount > 0 {
			sampleIconURL = data.Icons[0].URL
		}
		log.Printf(
			"[sns_avatar_handler] GET /avatars/%s aggregate ok iconsCount=%d avatar.avatarIcon_set=%t icons_sample_url=%q\n",
			id,
			iconsCount,
			hasAvatarIconField,
			sampleIconURL,
		)

		_ = json.NewEncoder(w).Encode(data)
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	// ✅ avatar icon を保持できているか確認（avatars.avatarIcon フィールドを観測）
	hasAvatarIconField := avatar.AvatarIcon != nil && strings.TrimSpace(*avatar.AvatarIcon) != ""
	log.Printf(
		"[sns_avatar_handler] GET /avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(avatar.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(avatar)
}

// -----------------------------------------------------------------------------
// POST /avatars
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
		"[sns_avatar_handler] POST /avatars request userId=%q userUid=%q avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		in.UserID,
		maskUID(in.UserUID), // shared
		in.AvatarName,
		ptrStr(in.AvatarIcon),   // shared
		ptrLen(in.Profile),      // shared
		ptrStr(in.ExternalLink), // shared
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[sns_avatar_handler] POST /avatars error=%v\n", err)
		writeAvatarErr(w, err)
		return
	}

	// ✅ avatar icon を保持できているか確認（作成直後の avatar.avatarIcon を観測）
	hasAvatarIconField := created.AvatarIcon != nil && strings.TrimSpace(*created.AvatarIcon) != ""
	log.Printf(
		"[sns_avatar_handler] POST /avatars ok avatarId=%q walletAddress=%q avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		created.ID,
		ptrStr(created.WalletAddress),
		hasAvatarIconField,
		ptrStr(created.AvatarIcon),
	)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// -----------------------------------------------------------------------------
// POST /avatars/{id}/wallet
// -----------------------------------------------------------------------------
func (h *AvatarHandler) openWallet(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_avatar_handler] POST /avatars/%s/wallet request\n", id)

	// すでに walletAddress があるなら衝突扱い
	a, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns_avatar_handler] POST /avatars/%s/wallet get error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		log.Printf("[sns_avatar_handler] POST /avatars/%s/wallet conflict walletAddress=%q\n", id, ptrStr(a.WalletAddress))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet already opened"})
		return
	}

	updated, err := h.uc.OpenWallet(ctx, id)
	if err != nil {
		log.Printf("[sns_avatar_handler] POST /avatars/%s/wallet error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf(
		"[sns_avatar_handler] POST /avatars/%s/wallet ok walletAddress=%q\n",
		id,
		ptrStr(updated.WalletAddress),
	)
	_ = json.NewEncoder(w).Encode(updated)
}

// -----------------------------------------------------------------------------
// POST /avatars/{id}/icon-upload-url
// - GCS Signed URL を発行（PUT 用）
// - 返り値の bucket/objectPath を使って、次に POST /avatars/{id}/icon で登録する
// -----------------------------------------------------------------------------
//
// ✅ Cloud Run 対応:
// - 以前: JSON鍵ファイル (GCS_SIGNER_CREDENTIALS / GOOGLE_APPLICATION_CREDENTIALS) が必須だった
// - 変更後: 鍵ファイルが無い場合は IAM Credentials API (signBlob) で署名する
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

	// bucket は固定（必要なら env で差し替え）
	bucket := strings.TrimSpace(os.Getenv("AVATAR_ICON_BUCKET"))
	if bucket == "" {
		bucket = "narratives-development_avatar_icon"
	}

	// objectPath: "<avatarId>/<random>.<ext>"
	oid := newObjectID()
	ext := guessExt(fileName, mimeType)
	objectPath := fmt.Sprintf("%s/%s%s", id, oid, ext)

	// 署名者 email（Cloud Run の場合これが必須）
	signerEmail := strings.TrimSpace(os.Getenv("AVATAR_ICON_SIGNER_EMAIL"))
	if signerEmail == "" {
		// 互換: 既存の env を流用
		signerEmail = strings.TrimSpace(os.Getenv("TOKEN_ICON_SIGNER_EMAIL"))
	}
	if signerEmail == "" {
		signerEmail = strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL"))
	}

	// ローカル開発: JSON鍵ファイルがあればそれを使う（従来方式）
	credPath := strings.TrimSpace(os.Getenv("GCS_SIGNER_CREDENTIALS"))
	if credPath == "" {
		credPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}

	exp := time.Now().Add(15 * time.Minute)

	// ------------------------------------------------------------
	// A) 鍵ファイル署名（ローカル用）
	// ------------------------------------------------------------
	if credPath != "" {
		email, pk, err := loadServiceAccountKey(credPath)
		if err != nil {
			log.Printf("[sns_avatar_handler] POST /avatars/%s/icon-upload-url load key error=%v credPath=%q\n", id, err, credPath)
			// 鍵ファイルが壊れている等は致命的なので 500
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
			log.Printf("[sns_avatar_handler] POST /avatars/%s/icon-upload-url sign (keyfile) error=%v\n", id, err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
			return
		}

		log.Printf(
			"[sns_avatar_handler] POST /avatars/%s/icon-upload-url ok (keyfile) bucket=%q objectPath=%q mimeType=%q size=%v\n",
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
	// B) IAM Credentials API で署名（Cloud Run 用）
	// ------------------------------------------------------------
	if signerEmail == "" {
		// ここに来るのは「鍵ファイルが無い」かつ「署名者メールも無い」ケース
		log.Printf("[sns_avatar_handler] POST /avatars/%s/icon-upload-url missing signer email (set AVATAR_ICON_SIGNER_EMAIL)\n", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing signer email (set AVATAR_ICON_SIGNER_EMAIL)"})
		return
	}

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		GoogleAccessID: signerEmail,
		Method:         http.MethodPut,
		Expires:        exp,
		ContentType:    mimeType,

		// ✅ Cloud Run: SignBytes を IAM Credentials API で実装
		SignBytes: func(b []byte) ([]byte, error) {
			return signBytesWithIAM(ctx, signerEmail, b)
		},
	})
	if err != nil {
		log.Printf("[sns_avatar_handler] POST /avatars/%s/icon-upload-url sign (iam) error=%v signer=%q bucket=%q objectPath=%q\n", id, err, signerEmail, bucket, objectPath)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
		return
	}

	log.Printf(
		"[sns_avatar_handler] POST /avatars/%s/icon-upload-url ok (iam) signer=%q bucket=%q objectPath=%q mimeType=%q size=%v\n",
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

// signBytesWithIAM signs bytes by calling IAM Credentials API SignBlob.
// - 呼び出し主体: Cloud Run runtime SA（Workload Identity）
// - 署名対象SA: signerEmail（例: narratives-backend-sa@...）
// 必要権限: 呼び出し主体が「署名対象SA」に対して roles/iam.serviceAccountTokenCreator
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
// POST /avatars/{id}/icon  (既存)
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

		// 既存クライアントが avatarIcon 1本で送ってくる場合の救済
		AvatarIcon *string `json:"avatarIcon,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	bucket := strings.TrimSpace(ptrStr(body.Bucket))
	obj := strings.TrimSpace(ptrStr(body.ObjectPath))

	// avatarIcon が gs://... なら bucket/object に分解して優先
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
		"[sns_avatar_handler] POST /avatars/%s/icon request bucket=%q objectPath=%q fileName=%q size=%v\n",
		id,
		in.Bucket,
		in.ObjectPath,
		ptrStr(in.FileName),
		in.Size,
	)

	ic, err := h.uc.ReplaceAvatarIcon(ctx, id, in)
	if err != nil {
		log.Printf("[sns_avatar_handler] POST /avatars/%s/icon error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	// ✅ UI 表示のため、avatars.avatarIcon も最新URLへ更新（best-effort）
	updatedAvatarIcon := ""
	if strings.TrimSpace(ic.URL) != "" {
		url := strings.TrimSpace(ic.URL)
		_, _ = h.uc.Update(ctx, id, avatardom.AvatarPatch{AvatarIcon: &url})
		updatedAvatarIcon = url
	}

	// ✅ avatar icon を保持できているか確認
	hasURL := strings.TrimSpace(ic.URL) != ""
	log.Printf(
		"[sns_avatar_handler] POST /avatars/%s/icon ok iconId=%q url_set=%t url=%q avatar_patch_avatarIcon=%q\n",
		id,
		ic.ID,
		hasURL,
		ic.URL,
		updatedAvatarIcon,
	)

	_ = json.NewEncoder(w).Encode(ic)
}

// -----------------------------------------------------------------------------
// PATCH/PUT /avatars/{id}  (既存)
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
		log.Printf("[sns_avatar_handler] PATCH/PUT /avatars/%s rejected: walletAddress field is not allowed\n", id)
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
	log.Printf("[sns_avatar_handler] PATCH/PUT /avatars/%s raw=%q\n", id, headString(bs, 300))

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
		"[sns_avatar_handler] PATCH/PUT /avatars/%s request avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		id,
		ptrStr(patch.AvatarName),
		ptrStr(patch.AvatarIcon),
		ptrLen(patch.Profile),
		ptrStr(patch.ExternalLink),
	)

	updated, err := h.uc.Update(ctx, id, patch)
	if err != nil {
		log.Printf("[sns_avatar_handler] PATCH/PUT /avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := updated.AvatarIcon != nil && strings.TrimSpace(*updated.AvatarIcon) != ""
	log.Printf(
		"[sns_avatar_handler] PATCH/PUT /avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(updated.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(updated)
}

// -----------------------------------------------------------------------------
// DELETE /avatars/{id}  (既存)
// -----------------------------------------------------------------------------
func (h *AvatarHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_avatar_handler] DELETE /avatars/%s request\n", id)

	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[sns_avatar_handler] DELETE /avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf("[sns_avatar_handler] DELETE /avatars/%s ok\n", id)
	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// エラーハンドリング（既存）
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
// helpers (upload-url)
// -----------------------------------------------------------------------------
func newObjectID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func guessExt(fileName string, mimeType string) string {
	// fileName の拡張子を優先（.png 等）
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
