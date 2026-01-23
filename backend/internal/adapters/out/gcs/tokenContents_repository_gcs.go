// backend/internal/adapters/out/gcs/tokenContents_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	gcscommon "narratives/internal/adapters/out/gcs/common"
	tcdom "narratives/internal/domain/tokenContents"
)

// GCS-based implementation of token content repository.
// - Binary contents are stored in GCS.
// - Metadata is derived from GCS Object attributes/metadata.
// - One GCS object represents one TokenContent.
// - Object name is used as ID (id == objectPath).
//
// NOTE (no backward-compat):
// - This repository treats IDs as *object paths* within the configured bucket.
// - It does NOT accept gs://... or https://storage.googleapis.com/... URLs as IDs/inputs.
type TokenContentsRepositoryGCS struct {
	Client *storage.Client
	Bucket string
}

// Default bucket for token contents in this adapter layer.
const defaultTokenContentsBucket = "narratives_development_token_contents"

func NewTokenContentsRepositoryGCS(client *storage.Client, bucket string) *TokenContentsRepositoryGCS {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = defaultTokenContentsBucket
	}
	return &TokenContentsRepositoryGCS{
		Client: client,
		Bucket: b,
	}
}

func (r *TokenContentsRepositoryGCS) bucket() string {
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		return defaultTokenContentsBucket
	}
	return b
}

// rejectURL enforces "no backward-compat" policy.
func rejectURL(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.HasPrefix(s, "gs://") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// ========================
// RepositoryPort impl (reduced contract)
// ========================

// GetByID:
// - id must be an object path within configured bucket.
// - URL forms (gs://..., https://...) are rejected.
func (r *TokenContentsRepositoryGCS) GetByID(ctx context.Context, id string) (*tcdom.TokenContent, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS.GetByID: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tcdom.ErrNotFound
	}
	if rejectURL(id) {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.GetByID: URL-form id is not supported (pass object path only)")
	}

	bucket := r.bucket()
	objectPath := strings.TrimLeft(id, "/")
	if objectPath == "" {
		return nil, tcdom.ErrNotFound
	}

	attrs, err := r.Client.Bucket(bucket).Object(objectPath).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.GetByID: attrs failed: %w", err)
	}

	tc := buildTokenContentFromAttrs(bucket, attrs)
	return &tc, nil
}

// Create:
// - Assumes the object is already uploaded to GCS.
// - in.URL must be an object path (NOT a URL).
// - Updates object metadata (name/type/size) and returns the TokenContent.
func (r *TokenContentsRepositoryGCS) Create(ctx context.Context, in tcdom.CreateTokenContentInput) (*tcdom.TokenContent, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS.Create: nil storage client")
	}

	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: empty URL (expected object path)")
	}
	if rejectURL(raw) {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: URL-form input is not supported (pass object path only)")
	}

	bucket := r.bucket()
	objectPath := strings.TrimLeft(raw, "/")
	if objectPath == "" {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: empty objectPath")
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: attrs failed: %w", err)
	}

	ua := storage.ObjectAttrsToUpdate{}
	meta := map[string]string{}
	for k, v := range attrs.Metadata {
		meta[k] = v
	}

	if n := strings.TrimSpace(in.Name); n != "" {
		meta["name"] = n
	}
	if t := strings.TrimSpace(string(in.Type)); t != "" {
		meta["type"] = t
	}
	if in.Size > 0 {
		meta["size"] = fmt.Sprint(in.Size)
	}

	if len(meta) > 0 {
		ua.Metadata = meta
	}

	newAttrs, err := obj.Update(ctx, ua)
	if err != nil {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Create: update failed: %w", err)
	}

	tc := buildTokenContentFromAttrs(bucket, newAttrs)
	return &tc, nil
}

// Update:
// - Updates metadata on the corresponding GCS object for given id.
// - id must be an object path (NOT a URL).
func (r *TokenContentsRepositoryGCS) Update(ctx context.Context, id string, in tcdom.UpdateTokenContentInput) (*tcdom.TokenContent, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("TokenContentsRepositoryGCS.Update: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tcdom.ErrNotFound
	}
	if rejectURL(id) {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Update: URL-form id is not supported (pass object path only)")
	}

	bucket := r.bucket()
	objectPath := strings.TrimLeft(id, "/")
	if objectPath == "" {
		return nil, tcdom.ErrNotFound
	}

	obj := r.Client.Bucket(bucket).Object(objectPath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, tcdom.ErrNotFound
		}
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Update: attrs failed: %w", err)
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
	setType := func(p *tcdom.ContentType) {
		if p == nil {
			return
		}
		v := strings.TrimSpace(string(*p))
		if v == "" {
			delete(meta, "type")
		} else {
			meta["type"] = v
		}
	}
	setInt64 := func(key string, p *int64) {
		if p == nil {
			return
		}
		meta[key] = fmt.Sprint(*p)
	}

	setStr("name", in.Name)
	setType(in.Type)

	// NOTE: URL is derived; if provided, we store it as metadata only (debug/trace use).
	setStr("url", in.URL)

	setInt64("size", in.Size)

	if len(meta) == 0 {
		tc := buildTokenContentFromAttrs(bucket, attrs)
		return &tc, nil
	}

	newAttrs, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: meta})
	if err != nil {
		return nil, fmt.Errorf("TokenContentsRepositoryGCS.Update: update failed: %w", err)
	}

	tc := buildTokenContentFromAttrs(bucket, newAttrs)
	return &tc, nil
}

// Delete removes the underlying GCS object.
// id must be an object path (NOT a URL).
func (r *TokenContentsRepositoryGCS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("TokenContentsRepositoryGCS.Delete: nil storage client")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tcdom.ErrNotFound
	}
	if rejectURL(id) {
		return fmt.Errorf("TokenContentsRepositoryGCS.Delete: URL-form id is not supported (pass object path only)")
	}

	bucket := r.bucket()
	objectPath := strings.TrimLeft(id, "/")
	if objectPath == "" {
		return tcdom.ErrNotFound
	}

	err := r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return tcdom.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("TokenContentsRepositoryGCS.Delete: delete failed: %w", err)
	}
	return nil
}

// UploadContent:
// - Uploads content to GCS and returns (publicURL, size).
// - ID generation policy: "<unix_nano>_<fileName>"
func (r *TokenContentsRepositoryGCS) UploadContent(ctx context.Context, fileName, contentType string, body io.Reader) (string, int64, error) {
	if r == nil || r.Client == nil {
		return "", 0, errors.New("TokenContentsRepositoryGCS.UploadContent: nil storage client")
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "", 0, fmt.Errorf("TokenContentsRepositoryGCS.UploadContent: empty fileName")
	}

	bucket := r.bucket()
	objectPath := fmt.Sprintf("%d_%s", time.Now().UTC().UnixNano(), fileName)

	w := r.Client.Bucket(bucket).Object(objectPath).NewWriter(ctx)
	if ct := strings.TrimSpace(contentType); ct != "" {
		w.ContentType = ct
	}

	n, err := io.Copy(w, body)
	if cerr := w.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		_ = r.Client.Bucket(bucket).Object(objectPath).Delete(ctx)
		return "", 0, fmt.Errorf("TokenContentsRepositoryGCS.UploadContent: upload failed: %w", err)
	}

	// gcscommon.GCSPublicURL(bucket, objectPath, defaultBucket)
	publicURL := gcscommon.GCSPublicURL(bucket, objectPath, defaultTokenContentsBucket)
	return publicURL, n, nil
}

// ========================
// Helpers
// ========================

// buildTokenContentFromAttrs converts GCS ObjectAttrs → TokenContent.
func buildTokenContentFromAttrs(bucket string, attrs *storage.ObjectAttrs) tcdom.TokenContent {
	name := strings.TrimSpace(attrs.Name)

	meta := attrs.Metadata
	getMeta := func(key string) string {
		if meta == nil {
			return ""
		}
		return strings.TrimSpace(meta[key])
	}

	id := name

	tcName := getMeta("name")
	if tcName == "" {
		tcName = name
		if idx := strings.LastIndex(tcName, "/"); idx >= 0 && idx < len(tcName)-1 {
			tcName = tcName[idx+1:]
		}
	}

	var ctype tcdom.ContentType
	if t := getMeta("type"); t != "" {
		ctype = tcdom.ContentType(t)
	}

	url := getMeta("url")
	if url == "" {
		url = gcscommon.GCSPublicURL(bucket, name, defaultTokenContentsBucket)
	}

	var size int64
	if sz, ok := gcscommon.ParseInt64Meta(meta, "size"); ok {
		size = sz
	} else if attrs.Size > 0 {
		size = attrs.Size
	}

	return tcdom.TokenContent{
		ID:   id,
		Name: tcName,
		Type: ctype,
		URL:  url,
		Size: size,
	}
}
