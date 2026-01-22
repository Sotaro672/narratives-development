// backend/internal/adapters/out/gcs/avatarIcon_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	avicon "narratives/internal/domain/avatarIcon"
)

// Env var key (Cloud Run / local .env) for avatar icon bucket.
const EnvAvatarIconBucket = "AVATAR_ICON_BUCKET"

// Fallback bucket name (domain const を優先し、空ならこちら).
const FallbackAvatarIconBucket = "narratives-development_avatar_icon"

// AvatarIconRepositoryGCS is a small GCS adapter for Avatar icons.
//
// ObjectPath policy:
// - 推奨: {avatarId}/{iconId}/{fileName}  (複数アイコン/履歴保持向け)
// - 互換: {avatarId}/{fileName}          (handler が {avatarId}/{random}.{ext} を作るケース)
//
// NOTE:
// - gcs/tokenIcon_repository_gcs.go に sanitizeFileName / isKeepObject / sanitizePathSegment がある前提で再利用する（同一 package gcs）。
type AvatarIconRepositoryGCS struct {
	Client *storage.Client
	Bucket string // optional (if empty, use env or fallback)
}

func NewAvatarIconRepositoryGCS(client *storage.Client, bucket string) *AvatarIconRepositoryGCS {
	return &AvatarIconRepositoryGCS{
		Client: client,
		Bucket: strings.TrimSpace(bucket),
	}
}

// ResolveBucket decides bucket by:
// 1) repository.Bucket
// 2) env AVATAR_ICON_BUCKET
// 3) domain DefaultBucket
// 4) FallbackAvatarIconBucket
func (r *AvatarIconRepositoryGCS) ResolveBucket() string {
	if r != nil {
		if b := strings.TrimSpace(r.Bucket); b != "" {
			return b
		}
	}
	if b := strings.TrimSpace(os.Getenv(EnvAvatarIconBucket)); b != "" {
		return b
	}
	if b := strings.TrimSpace(avicon.DefaultBucket); b != "" {
		return b
	}
	return FallbackAvatarIconBucket
}

// ============================================================
// Object storage port (usecase.AvatarIconObjectStoragePort 互換)
// ============================================================

// EnsurePrefix creates "<prefix>/.keep" object (best-effort).
func (r *AvatarIconRepositoryGCS) EnsurePrefix(ctx context.Context, bucket, prefix string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("AvatarIconRepositoryGCS.EnsurePrefix: storage client is nil")
	}
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.ResolveBucket()
	}

	prefix = strings.TrimSpace(prefix)
	prefix = strings.TrimLeft(prefix, "/")
	if prefix == "" {
		return fmt.Errorf("AvatarIconRepositoryGCS.EnsurePrefix: prefix is empty")
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	keepObject := path.Join(prefix, ".keep")
	obj := r.Client.Bucket(b).Object(keepObject)

	w := obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	w.ContentType = "text/plain"
	_, _ = w.Write([]byte("keep"))
	if err := w.Close(); err != nil {
		// already exists is OK
		if ge, ok := err.(*googleapi.Error); ok && ge.Code == 412 {
			return nil
		}
		return fmt.Errorf("AvatarIconRepositoryGCS.EnsurePrefix: write keep failed: %w", err)
	}
	return nil
}

// DeleteObjects deletes multiple GCS objects (best-effort).
func (r *AvatarIconRepositoryGCS) DeleteObjects(ctx context.Context, ops []avicon.GCSDeleteOp) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("AvatarIconRepositoryGCS.DeleteObjects: storage client is nil")
	}
	if len(ops) == 0 {
		return nil
	}

	var firstErr error
	for _, op := range ops {
		b := strings.TrimSpace(op.Bucket)
		if b == "" {
			b = r.ResolveBucket()
		}
		obj := strings.TrimLeft(strings.TrimSpace(op.ObjectPath), "/")
		if obj == "" {
			continue
		}
		if err := r.Client.Bucket(b).Object(obj).Delete(ctx); err != nil {
			if errors.Is(err, storage.ErrObjectNotExist) {
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// ============================================================
// ✅ Signed URL
// ============================================================

// IssueSignedURL issues a signed PUT url for uploading avatar icon images.
// ObjectPath policy: {avatarId}/{iconId}/{fileName}
// Returns ID as objectPath.
func (r *AvatarIconRepositoryGCS) IssueSignedURL(
	ctx context.Context,
	in avicon.IssueSignedURLInput,
) (avicon.IssueSignedURLOutput, error) {
	_ = ctx // SignedURL generation doesn't require ctx; kept for consistency.

	if r == nil || r.Client == nil {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURL: storage client is nil")
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURL: avatarId is empty")
	}

	ct := strings.ToLower(strings.TrimSpace(in.ContentType))
	if ct == "" {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURL: contentType is empty")
	}
	if !isSupportedAvatarIconMIME(ct) {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURL: unsupported contentType=%q", ct)
	}

	// optional: size validation (if provided)
	if in.Size > 0 && avicon.DefaultMaxIconSizeBytes > 0 && in.Size > avicon.DefaultMaxIconSizeBytes {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf(
			"AvatarIconRepositoryGCS.IssueSignedURL: file too large: %d > %d",
			in.Size,
			avicon.DefaultMaxIconSizeBytes,
		)
	}

	// Normalize fileName (reuse sanitizeFileName from tokenIcon_repository_gcs.go)
	normName := strings.TrimSpace(in.FileName)
	if normName == "" {
		normName = "icon"
	}
	normName = sanitizeFileName(normName)
	normName = ensureExtensionByMIME(normName, ct)

	// iconID (new object each time; history-friendly)
	iconID := newObjectID()

	// objectPath
	objPath, err := buildAvatarIconObjectPath(avatarID, iconID, normName)
	if err != nil {
		return avicon.IssueSignedURLOutput{}, err
	}

	bucket := r.ResolveBucket()

	// expiry
	sec := in.ExpiresInSeconds
	if sec <= 0 {
		sec = 15 * 60 // default 15m
	}
	if sec > 60*60 {
		sec = 60 * 60 // cap 60m
	}
	expiresAt := time.Now().UTC().Add(time.Duration(sec) * time.Second)

	uploadURL, err := r.Client.Bucket(bucket).SignedURL(objPath, &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     expiresAt,
		ContentType: ct,
	})
	if err != nil {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURL: signed url failed: %w", err)
	}

	publicURL := avicon.PublicURL(bucket, objPath)

	return avicon.IssueSignedURLOutput{
		ID:          objPath, // ✅ 方針: ID = objectPath
		Bucket:      bucket,
		ObjectPath:  objPath,
		UploadURL:   uploadURL,
		PublicURL:   publicURL,
		FileName:    normName,
		ContentType: ct,
		Size:        in.Size,
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
	}, nil
}

// IssueSignedURLForOverwrite issues a signed PUT url for uploading avatar icon images
// to an EXISTING objectPath (fixed URL overwrite).
//
// This is used when the product expectation is:
// - avatarIcon (public URL) string stays the same
// - only the GCS object bytes are replaced (PUT overwrite) or deleted
//
// objectPath examples:
// - "{avatarId}/{fileName}"                 (legacy/handler style: avatarId/random.png)
// - "{avatarId}/{iconId}/{fileName}"        (recommended style)
//
// NOTE:
//   - This method is intentionally NOT part of usecase.AvatarIconObjectStoragePort compatibility;
//     it can be called by a dedicated "me" icon update flow where the server resolves current avatarIcon URL,
//     parses bucket/objectPath, and issues overwrite signed URL.
func (r *AvatarIconRepositoryGCS) IssueSignedURLForOverwrite(
	ctx context.Context,
	bucket string,
	objectPath string,
	contentType string,
	size int64,
	expiresInSeconds int64,
) (avicon.IssueSignedURLOutput, error) {
	_ = ctx

	if r == nil || r.Client == nil {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: storage client is nil")
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		b = r.ResolveBucket()
	}

	objPath := normalizeObjectPath(objectPath)
	if objPath == "" {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: objectPath is empty")
	}
	// very small safety check (reject traversal)
	if strings.Contains(objPath, "..") {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: invalid objectPath=%q", objPath)
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: contentType is empty")
	}
	if !isSupportedAvatarIconMIME(ct) {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: unsupported contentType=%q", ct)
	}

	// optional: size validation (if provided)
	if size > 0 && avicon.DefaultMaxIconSizeBytes > 0 && size > avicon.DefaultMaxIconSizeBytes {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf(
			"AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: file too large: %d > %d",
			size,
			avicon.DefaultMaxIconSizeBytes,
		)
	}

	sec := expiresInSeconds
	if sec <= 0 {
		sec = 15 * 60
	}
	if sec > 60*60 {
		sec = 60 * 60
	}
	expiresAt := time.Now().UTC().Add(time.Duration(sec) * time.Second)

	uploadURL, err := r.Client.Bucket(b).SignedURL(objPath, &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     expiresAt,
		ContentType: ct,
	})
	if err != nil {
		return avicon.IssueSignedURLOutput{}, fmt.Errorf("AvatarIconRepositoryGCS.IssueSignedURLForOverwrite: signed url failed: %w", err)
	}

	publicURL := avicon.PublicURL(b, objPath)

	// fileName is best-effort (for client debug/logging)
	fn := sanitizeFileName(path.Base(objPath))
	if fn == "" {
		fn = "icon"
	}

	return avicon.IssueSignedURLOutput{
		ID:          objPath, // policy: id = objectPath
		Bucket:      b,
		ObjectPath:  objPath,
		UploadURL:   uploadURL,
		PublicURL:   publicURL,
		FileName:    fn,
		ContentType: ct,
		Size:        size,
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
	}, nil
}

func normalizeObjectPath(p string) string {
	s := strings.TrimSpace(p)
	s = strings.TrimLeft(s, "/")
	// collapse accidental double slashes (best-effort)
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return s
}

func isSupportedAvatarIconMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

// ============================================================
// ✅ usecase.AvatarIconRepo を満たす最小実装（Save 必須）
// ============================================================

// GetByAvatarID lists objects under "{avatarId}/" prefix and returns domain AvatarIcon list.
func (r *AvatarIconRepositoryGCS) GetByAvatarID(ctx context.Context, avatarID string) ([]avicon.AvatarIcon, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("AvatarIconRepositoryGCS.GetByAvatarID: storage client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return []avicon.AvatarIcon{}, nil
	}

	bucket := r.ResolveBucket()
	prefix := sanitizePathSegment(avatarID)
	if prefix == "" {
		return []avicon.AvatarIcon{}, nil
	}
	prefix = prefix + "/"

	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})

	out := make([]avicon.AvatarIcon, 0, 8)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("AvatarIconRepositoryGCS.GetByAvatarID: iterate failed: %w", err)
		}
		if attrs == nil {
			continue
		}

		// ".keep" などは除外（tokenIcon 側 helper を再利用）
		if isKeepObject(attrs.Name) {
			continue
		}

		ic, ok := buildAvatarIconFromAttrs(bucket, attrs)
		if !ok {
			continue
		}

		// 念のため avatarId で絞る
		if ic.AvatarID == nil || strings.TrimSpace(*ic.AvatarID) != avatarID {
			continue
		}

		out = append(out, ic)
	}

	return out, nil
}

// Save updates GCS metadata for an already-uploaded object and returns AvatarIcon.
// - opts は現状未使用（互換のため受ける）
func (r *AvatarIconRepositoryGCS) Save(
	ctx context.Context,
	ic avicon.AvatarIcon,
	opts *avicon.SaveOptions, // unused
) (avicon.AvatarIcon, error) {
	_ = opts

	if r == nil || r.Client == nil {
		return avicon.AvatarIcon{}, fmt.Errorf("AvatarIconRepositoryGCS.Save: storage client is nil")
	}

	// bucket/objectPath を URL から解決（基本は NewFromBucketObject が public URL を作る）
	bucket := r.ResolveBucket()
	objectPath := ""

	if b, obj, ok := avicon.ParseGCSURL(strings.TrimSpace(ic.URL)); ok {
		if strings.TrimSpace(b) != "" {
			bucket = strings.TrimSpace(b)
		}
		objectPath = strings.TrimLeft(strings.TrimSpace(obj), "/")
	}
	if objectPath == "" {
		// fallback: ID を objectPath とみなす
		objectPath = strings.TrimLeft(strings.TrimSpace(ic.ID), "/")
	}
	if objectPath == "" {
		return avicon.AvatarIcon{}, avicon.ErrNotFound
	}

	// object exists?
	o := r.Client.Bucket(bucket).Object(objectPath)
	attrs, err := o.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return avicon.AvatarIcon{}, avicon.ErrNotFound
		}
		return avicon.AvatarIcon{}, fmt.Errorf("AvatarIconRepositoryGCS.Save: attrs failed: %w", err)
	}

	// avatarId
	aid := ""
	if ic.AvatarID != nil {
		aid = strings.TrimSpace(*ic.AvatarID)
	}
	if aid == "" {
		// objectPath から推定（"{avatarId}/{...}"）
		if p0, _, ok := splitAvatarIconObjectPath(objectPath); ok {
			aid = p0
		}
	}

	// fileName
	fn := ""
	if ic.FileName != nil {
		fn = strings.TrimSpace(*ic.FileName)
	}
	if fn == "" {
		fn = path.Base(objectPath)
	}
	fn = sanitizeFileName(fn)
	if fn == "" {
		return avicon.AvatarIcon{}, avicon.ErrInvalidFileName
	}

	// size
	var sizePtr *int64
	if ic.Size != nil && *ic.Size >= 0 {
		tmp := *ic.Size
		sizePtr = &tmp
	} else if attrs.Size >= 0 {
		tmp := attrs.Size
		sizePtr = &tmp
	}

	publicURL := avicon.PublicURL(bucket, objectPath)

	// merge metadata
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}
	if aid != "" {
		meta["avatarId"] = aid
	}
	meta["fileName"] = fn
	meta["url"] = publicURL
	if sizePtr != nil {
		meta["size"] = fmt.Sprint(*sizePtr)
	}

	// metadata update
	newAttrs, err := o.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return avicon.AvatarIcon{}, fmt.Errorf("AvatarIconRepositoryGCS.Save: update metadata failed: %w", err)
	}

	// build domain
	id := strings.TrimSpace(ic.ID)
	if id == "" {
		id = strings.TrimSpace(newAttrs.Name) // policy: id = objectPath
	}

	out, derr := avicon.NewFromBucketObject(
		id,
		bucket,
		newAttrs.Name,
		&fn,
		sizePtr,
	)
	if derr != nil {
		// best-effort fallback
		tmp := avicon.AvatarIcon{
			ID:       id,
			URL:      publicURL,
			FileName: &fn,
			Size:     sizePtr,
		}
		if aid != "" {
			aid2 := aid
			tmp.AvatarID = &aid2
		}
		return tmp, nil
	}

	// avatarId を反映（NewFromBucketObject では nil）
	if aid != "" {
		aid2 := aid
		out.SetAvatarID(&aid2)
	}

	// url をメタ優先で差し替え
	_ = out.UpdateURL(publicURL)

	return out, nil
}

// ============================================================
// Helpers
// ============================================================

func buildAvatarIconObjectPath(avatarID, iconID, fileName string) (string, error) {
	aid := sanitizePathSegment(avatarID)
	iid := sanitizePathSegment(iconID)
	fn := strings.TrimSpace(fileName) // sanitizeFileName 済み想定

	if aid == "" {
		return "", fmt.Errorf("avatarIcon: invalid avatarID for object path")
	}
	if iid == "" {
		return "", fmt.Errorf("avatarIcon: invalid iconID for object path")
	}
	if fn == "" {
		return "", fmt.Errorf("avatarIcon: invalid fileName for object path")
	}

	// {avatarId}/{iconId}/{fileName}
	return path.Join(aid, iid, fn), nil
}

// buildAvatarIconFromAttrs converts GCS attrs to domain AvatarIcon (best-effort).
func buildAvatarIconFromAttrs(bucket string, attrs *storage.ObjectAttrs) (avicon.AvatarIcon, bool) {
	if attrs == nil {
		return avicon.AvatarIcon{}, false
	}

	obj := strings.TrimSpace(attrs.Name)
	if obj == "" {
		return avicon.AvatarIcon{}, false
	}

	meta := attrs.Metadata
	getMeta := func(k string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[k])
	}

	avatarID := getMeta("avatarId")
	if avatarID == "" {
		// supports both "{avatarId}/{fileName}" and "{avatarId}/{iconId}/{fileName}"
		if aid, _, ok := splitAvatarIconObjectPath(obj); ok {
			avatarID = aid
		}
	}
	var aidPtr *string
	if strings.TrimSpace(avatarID) != "" {
		tmp := strings.TrimSpace(avatarID)
		aidPtr = &tmp
	}

	fileName := getMeta("fileName")
	if fileName == "" {
		fileName = path.Base(obj)
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		return avicon.AvatarIcon{}, false
	}
	fnPtr := &fileName

	// url
	urlStr := getMeta("url")
	if urlStr == "" {
		urlStr = avicon.PublicURL(strings.TrimSpace(bucket), obj)
	}

	// size
	var sizePtr *int64
	size := attrs.Size
	if v := getMeta("size"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			size = n
		}
	}
	if size >= 0 {
		tmp := size
		sizePtr = &tmp
	}

	id := obj // policy: id = objectPath
	ic, err := avicon.New(
		id,
		urlStr,
		aidPtr,
		fnPtr,
		sizePtr,
	)
	if err != nil {
		// best-effort fallback
		tmp := avicon.AvatarIcon{
			ID:       id,
			AvatarID: aidPtr,
			URL:      urlStr,
			FileName: fnPtr,
			Size:     sizePtr,
		}
		return tmp, true
	}
	return ic, true
}

// splitAvatarIconObjectPath supports BOTH patterns:
//
// A) "{avatarId}/{iconId}/{fileName}" (len(parts) >= 3)
// B) "{avatarId}/{fileName}"         (len(parts) == 2)  ※ handler style: avatarId/random.png
//
// It returns:
// - avatarID: parts[0]
// - iconID:   parts[1] (best-effort; for pattern B this is fileName, but callers only need "avatarID")
// - ok:       true if at least 2 segments and avatarId is non-empty
func splitAvatarIconObjectPath(objectPath string) (avatarID string, iconID string, ok bool) {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return "", "", false
	}

	avatarID = strings.TrimSpace(parts[0])
	iconID = strings.TrimSpace(parts[1]) // pattern B: this is fileName; pattern A: iconId

	if avatarID == "" || iconID == "" {
		return "", "", false
	}
	return avatarID, iconID, true
}
