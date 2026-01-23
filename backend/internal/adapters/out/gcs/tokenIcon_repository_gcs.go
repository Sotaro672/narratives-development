// backend/internal/adapters/out/gcs/tokenIcon_repository_gcs.go
package gcs

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"

	gcscommon "narratives/internal/adapters/out/gcs/common"
	tidom "narratives/internal/domain/tokenIcon"
)

// TokenIconRepositoryGCS
//   - Token Icon の実体（バイナリ）を GCS に保存するためのアダプタ。
//   - tokenIcon.RepositoryPort（GetByID/Create/Update/Delete）を満たします。
//   - 署名付きURL（PUT）発行は tokenIcon.SignedUploadURLIssuer として実装します。
type TokenIconRepositoryGCS struct {
	Client *storage.Client
	Bucket string

	// ★ Signed URL 発行用（フロントから直接 PUT する）
	// 例: narratives-backend-sa@xxxx.iam.gserviceaccount.com
	SignerEmail string

	// ★ Signed URL の有効期限（未指定なら 15分）
	SignedURLTTL time.Duration
}

// Default bucket for token icons (public).
// env TOKEN_ICON_BUCKET が空のときのフォールバック。
const defaultTokenIconBucket = "narratives-development_token_icon"

// Signed URL TTL のデフォルト
const defaultSignedURLTTL = 15 * time.Minute

func NewTokenIconRepositoryGCS(client *storage.Client, bucket string) *TokenIconRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultTokenIconBucket
	}

	// ★ 署名者（サービスアカウント）を env から拾う（無ければ空のまま。IssueSignedUploadURL でエラーにする）
	// NOTE: GCS_SIGNER_EMAIL のみを使用する（フォールバック削除）
	signer := strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL"))

	ttl := defaultSignedURLTTL
	if v := strings.TrimSpace(os.Getenv("TOKEN_ICON_SIGNED_URL_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			ttl = d
		}
	}

	return &TokenIconRepositoryGCS{
		Client:       client,
		Bucket:       b,
		SignerEmail:  signer,
		SignedURLTTL: ttl,
	}
}

func (r *TokenIconRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultTokenIconBucket
	}
	return b
}

func (r *TokenIconRepositoryGCS) signedURLTTL() time.Duration {
	if r == nil {
		return defaultSignedURLTTL
	}
	if r.SignedURLTTL <= 0 {
		return defaultSignedURLTTL
	}
	return r.SignedURLTTL
}

// ============================================================
// ★ Signed URL (PUT) 発行
//   - フロントが直接 GCS に PUT するための URL を返す
//   - docId 配下に object を作る（例: "{docId}/icon.png"）
//   - ".keep" も必要なら purpose="keep" で発行できる
// ============================================================

func (r *TokenIconRepositoryGCS) IssueSignedUploadURL(
	ctx context.Context,
	in tidom.SignedUploadURLInput,
) (*tidom.SignedUploadURLResult, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	docID := strings.TrimSpace(in.DocID)
	if docID == "" {
		return nil, fmt.Errorf("IssueSignedUploadURL: docId is empty")
	}

	signer := strings.TrimSpace(r.SignerEmail)
	if signer == "" {
		return nil, fmt.Errorf("IssueSignedUploadURL: signerEmail is empty (set GCS_SIGNER_EMAIL)")
	}

	bucket := r.bucket()

	// objectPath 決定
	objectPath := buildObjectPathForTokenIcon(
		docID,
		strings.TrimSpace(in.FileName),
		strings.TrimSpace(in.ContentType),
		strings.TrimSpace(in.Purpose),
	)
	if objectPath == "" {
		return nil, fmt.Errorf("IssueSignedUploadURL: failed to build objectPath")
	}

	// Content-Type（V4署名に含める。PUT時は同じ Content-Type を必ず付ける）
	ct := strings.TrimSpace(in.ContentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

	// IAM Credentials API で SignBlob（鍵ファイル不要）
	signBytes := func(b []byte) ([]byte, error) {
		svc, err := iamcredentials.NewService(ctx)
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("projects/-/serviceAccounts/%s", signer)
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(b),
		}
		resp, err := svc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}

	exp := time.Now().UTC().Add(r.signedURLTTL())
	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		Expires:        exp,
		GoogleAccessID: signer,
		SignBytes:      signBytes,

		// V4 canonical request に含める
		ContentType: ct,
	})
	if err != nil {
		return nil, err
	}

	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenIconBucket)
	return &tidom.SignedUploadURLResult{
		UploadURL:  uploadURL,
		ObjectPath: objectPath,
		PublicURL:  publicURL,
		ExpiresAt:  &exp,
	}, nil
}

// ============================================================
// GetByID
// ============================================================

// GetByID:
// - id is either:
//   - a GCS URL (gs:// or https://storage.googleapis.com/...),
//   - or an object path within the default bucket.
func (r *TokenIconRepositoryGCS) GetByID(ctx context.Context, id string) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return nil, err
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, attrs)
	return &ti, nil
}

// ============================================================
// Create / Update / Delete
// ============================================================

// Create:
// - 「Upload 済みの GCS オブジェクト」に対してメタデータ（file_name/size/url）を付与します。
// - in.URL は publicURL / gs:// / https://storage.googleapis.com/... / objectPath のいずれでもOK。
// - 戻り値 TokenIcon.ID は objectPath（= newAttrs.Name）です。
func (r *TokenIconRepositoryGCS) Create(
	ctx context.Context,
	in tidom.CreateTokenIconInput,
) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return nil, fmt.Errorf("TokenIconRepositoryGCS.Create: empty URL")
	}

	bucket, objectPath, err := resolveBucketObject(raw, r.bucket())
	if err != nil {
		return nil, err
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	// merge metadata
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if fn := strings.TrimSpace(in.FileName); fn != "" {
		meta["file_name"] = fn
	}
	if in.Size > 0 {
		meta["size"] = fmt.Sprint(in.Size)
	}

	// URL は “公開 URL” を優先して入れる（Create入力がgs://等でも閲覧側はこれを使える）
	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenIconBucket)
	meta["url"] = publicURL

	newAttrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return nil, err
	}

	ti := tidom.TokenIcon{
		ID:       newAttrs.Name,
		URL:      strings.TrimSpace(meta["url"]),
		FileName: strings.TrimSpace(meta["file_name"]),
		Size:     newAttrs.Size,
	}
	if ti.URL == "" {
		ti.URL = publicURL
	}
	if ti.FileName == "" {
		ti.FileName = lastSegment(newAttrs.Name)
	}
	if v := strings.TrimSpace(meta["size"]); v != "" {
		if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
			ti.Size = sz
		}
	}

	return &ti, nil
}

// Update updates GCS object metadata for the icon identified by id.
func (r *TokenIconRepositoryGCS) Update(
	ctx context.Context,
	id string,
	in tidom.UpdateTokenIconInput,
) (*tidom.TokenIcon, error) {
	if r.Client == nil {
		return nil, errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return nil, err
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tidom.ErrNotFound
		}
		return nil, err
	}

	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	setStr := func(key string, p *string) {
		if p == nil {
			return
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			delete(meta, key)
		} else {
			meta[key] = v
		}
	}
	setInt64 := func(key string, p *int64) {
		if p == nil {
			return
		}
		meta[key] = fmt.Sprint(*p)
	}

	setStr("file_name", in.FileName)
	setStr("url", in.URL)
	setInt64("size", in.Size)

	newAttrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return nil, err
	}

	ti := buildTokenIconFromAttrs(bucket, newAttrs)
	return &ti, nil
}

// Delete deletes the underlying GCS object.
func (r *TokenIconRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("TokenIconRepositoryGCS: nil storage client")
	}

	bucket, objectPath, err := resolveBucketObject(id, r.bucket())
	if err != nil {
		return err
	}

	err = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return tidom.ErrNotFound
	}
	return err
}

// ============================================================
// Helpers
// ============================================================

func buildTokenIconFromAttrs(bucket string, attrs *storage.ObjectAttrs) tidom.TokenIcon {
	name := strings.TrimSpace(attrs.Name)
	meta := attrs.Metadata

	getMeta := func(key string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[key])
	}

	id := name

	url := getMeta("url")
	if url == "" {
		url = gcscommon.GCSPublicURL(bucket, name, defaultTokenIconBucket)
	}

	fileName := getMeta("file_name")
	if fileName == "" {
		fileName = lastSegment(name)
	}

	size := attrs.Size
	if meta != nil {
		if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
			size = sz
		}
	}

	return tidom.TokenIcon{
		ID:       id,
		URL:      url,
		FileName: fileName,
		Size:     size,
	}
}

func resolveBucketObject(id string, fallbackBucket string) (bucket string, objectPath string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", tidom.ErrNotFound
	}

	if b, obj, ok := gcscommon.ParseGCSURL(id); ok {
		bucket, objectPath = b, obj
	} else {
		bucket = strings.TrimSpace(fallbackBucket)
		objectPath = id
	}

	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if bucket == "" || objectPath == "" {
		return "", "", tidom.ErrNotFound
	}
	return bucket, objectPath, nil
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func lastSegment(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if i := strings.LastIndex(p, "/"); i >= 0 && i < len(p)-1 {
		return p[i+1:]
	}
	return p
}

func buildObjectPathForTokenIcon(docID, fileName, contentType, purpose string) string {
	docID = strings.TrimLeft(strings.TrimSpace(docID), "/")
	if docID == "" {
		return ""
	}

	// keep を作りたい場合（「フォルダ作成」用）
	if strings.EqualFold(strings.TrimSpace(purpose), "keep") {
		return docID + "/.keep"
	}

	fileName = sanitizeFileName(fileName)

	// ファイル名が無い場合は icon + 拡張子推定
	if fileName == "" {
		ext := guessExtFromContentType(contentType)
		if ext == "" {
			ext = ".bin"
		}
		fileName = "icon" + ext
	}

	// docID 配下に配置
	// 例: "{docId}/icon.png" / "{docId}/myfile.png"
	return docID + "/" + filepath.Base(fileName)
}

func guessExtFromContentType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(ct))
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}
