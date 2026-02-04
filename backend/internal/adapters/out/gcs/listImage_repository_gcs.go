// backend/internal/adapters/out/gcs/listImage_repository_gcs.go
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

	listuc "narratives/internal/application/usecase/list"
	listimagedom "narratives/internal/domain/listImage"
)

// Env var key (Cloud Run / local .env) for list image bucket.
const EnvListImageBucket = "LIST_IMAGE_BUCKET"

// ListImageRepositoryGCS is a small GCS adapter for List images.
// - Bucket is fixed per environment (single bucket).
// - objectPath is canonical:
//
//	lists/{listId}/images/{imageId}
//
// NOTE:
// - ファイル名の正規化は package gcs の sanitizeFileName を再利用する。
// - ".keep" 除外は package gcs の isKeepObject を再利用する。
// - fileName は objectPath に含めず、metadata / Firestore レコードとして保持する。
type ListImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string // optional override; if empty, use env (required)
}

func NewListImageRepositoryGCS(client *storage.Client, bucket string) *ListImageRepositoryGCS {
	return &ListImageRepositoryGCS{
		Client: client,
		Bucket: strings.TrimSpace(bucket),
	}
}

// ResolveBucket decides bucket by:
// 1) repository.Bucket
// 2) env LIST_IMAGE_BUCKET
//
// IMPORTANT:
// - bucket is required (env-fixed). No fallback.
func (r *ListImageRepositoryGCS) ResolveBucket() (string, error) {
	if r != nil {
		if b := strings.TrimSpace(r.Bucket); b != "" {
			return b, nil
		}
	}
	if b := strings.TrimSpace(os.Getenv(EnvListImageBucket)); b != "" {
		return b, nil
	}
	return "", fmt.Errorf("ListImageRepositoryGCS.ResolveBucket: %s is required", EnvListImageBucket)
}

// ============================================================
// ✅ (Optional) Create-time init: EnsureListBucket
// - Kept only for backward compatibility / historical design.
// - It does NOT create a bucket; it creates a "{prefix}/.keep" object.
// - With single-bucket policy, you typically don't need this.
// ============================================================

func (r *ListImageRepositoryGCS) EnsureListBucket(ctx context.Context, listID string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("ListImageRepositoryGCS.EnsureListBucket: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return fmt.Errorf("ListImageRepositoryGCS.EnsureListBucket: listID is empty")
	}

	bucket, err := r.ResolveBucket()
	if err != nil {
		return err
	}

	prefix := sanitizePathSegment(listID)
	if prefix == "" {
		return fmt.Errorf("ListImageRepositoryGCS.EnsureListBucket: invalid listID")
	}

	// NOTE: This keep path is under canonical root as well.
	keepObject := path.Join("lists", prefix, "images", ".keep")
	obj := r.Client.Bucket(bucket).Object(keepObject)

	// Only create if not exist (best-effort)
	w := obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	w.ContentType = "text/plain"
	_, _ = w.Write([]byte("keep"))
	if err := w.Close(); err != nil {
		// If already exists, it's OK
		if ge, ok := err.(*googleapi.Error); ok && ge.Code == 412 {
			return nil
		}
		return fmt.Errorf("ListImageRepositoryGCS.EnsureListBucket: write keep failed: %w", err)
	}

	return nil
}

// ============================================================
// ✅ Delete (usecase port implementation)
// - usecase.DeleteImage(ctx, listID, imageID) → (this repo)
// - This implementation deletes only the GCS object.
//   (Firestore record deletion is handled by Firestore repo side if needed.)
// ============================================================

func (r *ListImageRepositoryGCS) Delete(ctx context.Context, listID string, imageID string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("ListImageRepositoryGCS.Delete: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listimagedom.ErrInvalidListID
	}
	if imageID == "" {
		return listimagedom.ErrInvalidID
	}
	// Policy A: imageId only
	if strings.Contains(imageID, "/") {
		return fmt.Errorf("ListImageRepositoryGCS.Delete: invalid imageID")
	}

	bucket, err := r.ResolveBucket()
	if err != nil {
		return err
	}

	objPath, err := buildCanonicalListImageObjectPath(listID, imageID)
	if err != nil {
		return err
	}

	obj := r.Client.Bucket(bucket).Object(objPath)
	if err := obj.Delete(ctx); err != nil {
		// NotFound は成功扱い（idempotent）
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}
		// googleapi.Error でも NotFound を吸収
		if ge, ok := err.(*googleapi.Error); ok && ge.Code == 404 {
			return nil
		}
		return fmt.Errorf("ListImageRepositoryGCS.Delete: delete failed: %w", err)
	}

	return nil
}

// ============================================================
// ✅ A) signed-url: IssueSignedURL (usecase port implementation)
// - handler → usecase/list → (this repo)
// ============================================================

// IssueSignedURL issues a signed PUT url for uploading list images.
//
// Canonical policy:
// - out.ID is imageId (Firestore docID).
// - out.ObjectPath is "lists/{listId}/images/{imageId}" (MUST be canonical).
// - out.Bucket MUST be provided (env-fixed). No defaults.
//
// NOTE:
// - fileName is NOT part of objectPath. It is returned and should be persisted in Firestore record / metadata.
// - If you need to change filename later, it won't change objectPath (supports overwrite update policy).
func (r *ListImageRepositoryGCS) IssueSignedURL(
	ctx context.Context,
	in listuc.ListImageIssueSignedURLInput,
) (listuc.ListImageIssueSignedURLOutput, error) {
	_ = ctx // SignedURL generation itself doesn't require ctx; kept for interface consistency.

	if r == nil || r.Client == nil {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf("ListImageRepositoryGCS.IssueSignedURL: storage client is nil")
	}

	listID := strings.TrimSpace(in.ListID)
	if listID == "" {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf("ListImageRepositoryGCS.IssueSignedURL: listID is empty")
	}

	ct := strings.ToLower(strings.TrimSpace(in.ContentType))
	if ct == "" {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf("ListImageRepositoryGCS.IssueSignedURL: contentType is empty")
	}
	if !isSupportedListImageMIME(ct) {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf("ListImageRepositoryGCS.IssueSignedURL: unsupported contentType=%q", ct)
	}

	// optional: size validation (if provided)
	if in.Size > 0 && in.Size > int64(listimagedom.DefaultMaxImageSizeBytes) {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf(
			"ListImageRepositoryGCS.IssueSignedURL: file too large: %d > %d",
			in.Size,
			listimagedom.DefaultMaxImageSizeBytes,
		)
	}

	// Normalize fileName (not used for object path; for record/metadata)
	normName := strings.TrimSpace(in.FileName)
	if normName == "" {
		normName = "image"
	}
	normName = sanitizeFileName(normName)
	normName = ensureExtensionByMIME(normName, ct)

	// Resolve imageID (always generate for signed-url flow)
	imgID := newObjectID()

	// Canonical objectPath: lists/{listId}/images/{imageId}
	objPath, err := buildCanonicalListImageObjectPath(listID, imgID)
	if err != nil {
		return listuc.ListImageIssueSignedURLOutput{}, err
	}

	bucket, err := r.ResolveBucket()
	if err != nil {
		return listuc.ListImageIssueSignedURLOutput{}, err
	}

	// expiry
	sec := in.ExpiresInSeconds
	if sec <= 0 {
		sec = 15 * 60 // default 15 minutes
	}
	if sec > 60*60 {
		sec = 60 * 60 // cap 60 minutes
	}
	expiresAt := time.Now().UTC().Add(time.Duration(sec) * time.Second)

	// Signed PUT URL (V4)
	uploadURL, err := r.Client.Bucket(bucket).SignedURL(objPath, &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     expiresAt,
		ContentType: ct,
	})
	if err != nil {
		return listuc.ListImageIssueSignedURLOutput{}, fmt.Errorf("ListImageRepositoryGCS.IssueSignedURL: signed url failed: %w", err)
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objPath)

	return listuc.ListImageIssueSignedURLOutput{
		ID:           imgID,   // ✅ imageId
		Bucket:       bucket,  // ✅ required
		ObjectPath:   objPath, // ✅ canonical
		UploadURL:    uploadURL,
		PublicURL:    publicURL,
		FileName:     normName,
		ContentType:  ct,
		Size:         in.Size,
		DisplayOrder: in.DisplayOrder,
		ExpiresAt:    expiresAt.UTC().Format(time.RFC3339),
	}, nil
}

func isSupportedListImageMIME(mime string) bool {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime == "" {
		return false
	}
	for k := range listimagedom.SupportedImageMIMEs {
		if strings.ToLower(strings.TrimSpace(k)) == mime {
			return true
		}
	}
	return false
}

// ============================================================
// usecase required methods
// - ListByListID(ctx, listID) ([]ListImage, error)
// - GetByID(ctx, id) (ListImage, error)
// - SaveFromBucketObject(ctx, id, listID, bucket, objectPath, size, displayOrder) (ListImage, error)
// ============================================================

// ListByListID lists images under "lists/{listId}/images/" prefix.
// It returns domain ListImage items (best-effort).
func (r *ListImageRepositoryGCS) ListByListID(ctx context.Context, listID string) ([]listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("ListImageRepositoryGCS.ListByListID: storage client is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listimagedom.ListImage{}, nil
	}

	bucket, err := r.ResolveBucket()
	if err != nil {
		return nil, err
	}

	lid := sanitizePathSegment(listID)
	if lid == "" {
		return []listimagedom.ListImage{}, nil
	}

	prefix := path.Join("lists", lid, "images") + "/"
	it := r.Client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})

	out := make([]listimagedom.ListImage, 0, 16)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ListImageRepositoryGCS.ListByListID: iterate failed: %w", err)
		}
		if attrs == nil {
			continue
		}

		// ".keep" は除外
		if isKeepObject(attrs.Name) {
			continue
		}

		li, ok := buildListImageFromAttrs(bucket, attrs)
		if !ok {
			continue
		}

		// listID で絞り込み（メタ不正/パス不正の保険）
		if strings.TrimSpace(li.ListID) != listID {
			continue
		}

		out = append(out, li)
	}

	return out, nil
}

// GetByID gets a ListImage by id.
// id can be:
// - imageId (Firestore docID)  => resolve to canonical objectPath "lists/{listId}/images/{imageId}" only when listId can be determined via metadata lookup is not possible.
// - objectPath within the bucket (e.g. "lists/{listId}/images/{imageId}")
// - https://storage.googleapis.com/{bucket}/{objectPath}
//
// NOTE:
// - For canonical design, prefer calling GetByID with objectPath or URL when using GCS as source-of-truth.
// - In your current system, Firestore (/lists/{listId}/images/{imageId}) should be source-of-truth for imageId lookup.
func (r *ListImageRepositoryGCS) GetByID(ctx context.Context, id string) (listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: storage client is nil")
	}

	bucket, objectPath, err := resolveBucketObjectForListImage(strings.TrimSpace(id), "")
	if err != nil {
		return listimagedom.ListImage{}, err
	}

	// bucket must be resolved (env-fixed) when not embedded in URL
	if bucket == "" {
		b, berr := r.ResolveBucket()
		if berr != nil {
			return listimagedom.ListImage{}, berr
		}
		bucket = b
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: attrs failed: %w", err)
	}

	li, ok := buildListImageFromAttrs(bucket, attrs)
	if !ok {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.GetByID: failed to build list image from attrs")
	}
	return li, nil
}

// SaveFromBucketObject:
// - already-uploaded object を前提に、GCS metadata を整えて domain ListImage を返す。
// - objectPath は canonical: "lists/{listId}/images/{imageId}" を必須。
// - id は imageId を採用（Firestore docID と同義）
//
// IMPORTANT:
// - objectPath と id(imageId) の整合性を検証する。
// - fileName は objectPath に含めず、metadata に保持する。
func (r *ListImageRepositoryGCS) SaveFromBucketObject(
	ctx context.Context,
	id string, // ✅ imageId
	listID string,
	bucket string,
	objectPath string,
	size int64,
	displayOrder int,
) (listimagedom.ListImage, error) {
	if r == nil || r.Client == nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: storage client is nil")
	}

	imageID := strings.TrimSpace(id)
	if imageID == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidID
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidListID
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		bk, err := r.ResolveBucket()
		if err != nil {
			return listimagedom.ListImage{}, err
		}
		b = bk
	}

	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: objectPath is empty")
	}

	// ✅ canonical validation: lists/{listId}/images/{imageId}
	if err := validateCanonicalObjectPath(listID, imageID, obj); err != nil {
		return listimagedom.ListImage{}, err
	}

	// object exists?
	o := r.Client.Bucket(b).Object(obj)
	attrs, err := o.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return listimagedom.ListImage{}, listimagedom.ErrNotFound
		}
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: attrs failed: %w", err)
	}

	// fileName: metadata 優先（signed-url responseの fileName を Firestore に保存する想定）
	// fallback: use object base (imageId) - not ideal, but keeps system alive
	fn := ""
	if attrs.Metadata != nil {
		fn = strings.TrimSpace(attrs.Metadata["fileName"])
	}
	if fn == "" {
		fn = "image"
	}
	fn = sanitizeFileName(fn)
	if fn == "" {
		return listimagedom.ListImage{}, listimagedom.ErrInvalidFileName
	}

	// size reconcile
	finalSize := size
	if finalSize <= 0 {
		finalSize = attrs.Size
	}
	if finalSize < 0 {
		finalSize = 0
	}

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)

	// merge metadata
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	// ✅ listId/imageId は引数を正とする
	meta["listId"] = listID
	meta["imageId"] = imageID
	meta["fileName"] = fn
	meta["url"] = publicURL
	meta["size"] = fmt.Sprint(finalSize)
	meta["displayOrder"] = fmt.Sprint(displayOrder)

	// metadata update (best-effort)
	newAttrs, err := o.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return listimagedom.ListImage{}, fmt.Errorf("ListImageRepositoryGCS.SaveFromBucketObject: update metadata failed: %w", err)
	}

	// domain object: id is imageId, URL derived from canonical objectPath
	li, derr := listimagedom.NewFromGCSObject(
		imageID, // ✅ id is imageId
		listID,
		fn,
		finalSize,
		displayOrder,
		b,
		strings.TrimSpace(newAttrs.Name), // objectPath
	)
	if derr != nil {
		// best-effort fallback（domain validate が落ちても UI を止めない）
		tmp := listimagedom.ListImage{
			ID:           imageID,
			ListID:       listID,
			URL:          publicURL,
			FileName:     fn,
			Size:         finalSize,
			DisplayOrder: displayOrder,
		}
		return tmp, nil
	}

	// URL は publicURL を採用
	if strings.TrimSpace(publicURL) != "" {
		_ = li.UpdateURL(publicURL)
	}

	return li, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

// buildCanonicalListImageObjectPath returns canonical objectPath:
//
//	lists/{listId}/images/{imageId}
func buildCanonicalListImageObjectPath(listID, imageID string) (string, error) {
	lid := sanitizePathSegment(listID)
	iid := sanitizePathSegment(imageID)

	if lid == "" {
		return "", fmt.Errorf("listImage: invalid listID for object path")
	}
	if iid == "" {
		return "", fmt.Errorf("listImage: invalid imageID for object path")
	}

	return path.Join("lists", lid, "images", iid), nil
}

func validateCanonicalObjectPath(listID, imageID, objectPath string) error {
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if obj == "" {
		return fmt.Errorf("listImage: objectPath is empty")
	}

	want, err := buildCanonicalListImageObjectPath(listID, imageID)
	if err != nil {
		return err
	}

	// must be exact match
	if obj != want {
		return fmt.Errorf("listImage: objectPath not canonical: got=%q want=%q", obj, want)
	}
	return nil
}

func resolveBucketObjectForListImage(id string, fallbackBucket string) (bucket string, objectPath string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", listimagedom.ErrNotFound
	}

	// domain helper: ParseGCSURL supports storage.googleapis.com / storage.cloud.google.com
	if b, obj, ok := listimagedom.ParseGCSURL(id); ok {
		bucket, objectPath = strings.TrimSpace(b), strings.TrimLeft(strings.TrimSpace(obj), "/")
	} else {
		// if the caller passed objectPath directly, bucket may be empty here and should be resolved by caller
		bucket = strings.TrimSpace(fallbackBucket)
		objectPath = strings.TrimLeft(strings.TrimSpace(id), "/")
	}

	if objectPath == "" {
		return "", "", listimagedom.ErrNotFound
	}
	return bucket, objectPath, nil
}

// buildListImageFromAttrs converts GCS attrs to domain ListImage (best-effort).
func buildListImageFromAttrs(bucket string, attrs *storage.ObjectAttrs) (listimagedom.ListImage, bool) {
	if attrs == nil {
		return listimagedom.ListImage{}, false
	}

	obj := strings.TrimSpace(attrs.Name)
	if obj == "" {
		return listimagedom.ListImage{}, false
	}

	meta := attrs.Metadata
	getMeta := func(k string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[k])
	}

	// listID: metadata 優先 → fallback to canonical path
	listID := getMeta("listId")
	imageID := getMeta("imageId")

	if listID == "" || imageID == "" {
		lid, iid, ok := splitCanonicalListImageObjectPath(obj)
		if !ok {
			// 期待ポリシー外の object は除外（安全側）
			return listimagedom.ListImage{}, false
		}
		if listID == "" {
			listID = lid
		}
		if imageID == "" {
			imageID = iid
		}
	}

	if strings.TrimSpace(listID) == "" || strings.TrimSpace(imageID) == "" {
		return listimagedom.ListImage{}, false
	}

	// fileName
	fileName := getMeta("fileName")
	if fileName == "" {
		fileName = "image"
	}
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		return listimagedom.ListImage{}, false
	}

	// url
	url := getMeta("url")
	if url == "" {
		url = fmt.Sprintf("https://storage.googleapis.com/%s/%s", strings.TrimSpace(bucket), obj)
	}

	// size
	size := attrs.Size
	if v := getMeta("size"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			size = n
		}
	}

	// displayOrder
	displayOrder := 0
	if v := getMeta("displayOrder"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			displayOrder = n
		}
	}

	// id is imageId (Firestore docID)
	id := strings.TrimSpace(imageID)

	li, err := listimagedom.NewFromGCSObject(
		id,
		listID,
		fileName,
		size,
		displayOrder,
		strings.TrimSpace(bucket),
		obj,
	)
	if err != nil {
		// best-effort fallback
		tmp := listimagedom.ListImage{
			ID:           id,
			ListID:       listID,
			URL:          url,
			FileName:     fileName,
			Size:         size,
			DisplayOrder: displayOrder,
		}
		return tmp, true
	}

	// constructor の URL は PublicURL(bucket,obj) になるので、メタの url を優先したい場合は差し替え
	if strings.TrimSpace(url) != "" {
		_ = li.UpdateURL(url)
	}
	return li, true
}

// splitCanonicalListImageObjectPath expects "lists/{listId}/images/{imageId}".
func splitCanonicalListImageObjectPath(objectPath string) (listID string, imageID string, ok bool) {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	// lists/{listId}/images/{imageId}
	if len(parts) < 4 {
		return "", "", false
	}
	if strings.TrimSpace(parts[0]) != "lists" {
		return "", "", false
	}
	if strings.TrimSpace(parts[2]) != "images" {
		return "", "", false
	}
	listID = strings.TrimSpace(parts[1])
	imageID = strings.TrimSpace(parts[3])
	if listID == "" || imageID == "" {
		return "", "", false
	}
	return listID, imageID, true
}
