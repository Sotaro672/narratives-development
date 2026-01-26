// backend\internal\application\tokenBlueprint\tokenBlueprint_bucket_usecase.go
package tokenBlueprint

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// ============================================================
// Config
// ============================================================

// Buckets (env)
// - TOKEN_ICON_BUCKET
// - TOKEN_CONTENTS_BUCKET
const (
	defaultTokenIconBucket     = "narratives-development_token_icon"
	defaultTokenContentsBucket = "narratives-development-token-contents"
)

func tokenIconBucketName() string {
	if v := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET")); v != "" {
		return v
	}
	return defaultTokenIconBucket
}

func tokenContentsBucketName() string {
	if v := strings.TrimSpace(os.Getenv("TOKEN_CONTENTS_BUCKET")); v != "" {
		return v
	}
	return defaultTokenContentsBucket
}

// gcsObjectPublicURL returns public HTTPS URL for an object.
// NOTE: tokenBlueprint_content_usecase.go から参照されているため、共通定義としてここに置く。
func gcsObjectPublicURL(bucket, object string) string {
	bucket = strings.TrimSpace(bucket)
	object = strings.TrimLeft(strings.TrimSpace(object), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
}

// keepObjectPath returns "{tokenBlueprintId}/.keep".
func keepObjectPath(tokenBlueprintID string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	return id + "/.keep"
}

// ============================================================
// Usecase
// ============================================================

// TokenBlueprintBucketUsecase ensures required bucket objects exist.
type TokenBlueprintBucketUsecase struct {
	gcs *storage.Client
}

func NewTokenBlueprintBucketUsecase(gcs *storage.Client) *TokenBlueprintBucketUsecase {
	return &TokenBlueprintBucketUsecase{gcs: gcs}
}

// EnsureKeepObjects ensures that BOTH:
// - gs://TOKEN_ICON_BUCKET/{tokenBlueprintId}/.keep
// - gs://TOKEN_CONTENTS_BUCKET/{tokenBlueprintId}/.keep
// exist. If any step fails, it returns error (caller should fail the request).
func (u *TokenBlueprintBucketUsecase) EnsureKeepObjects(ctx context.Context, tokenBlueprintID string) error {
	if u == nil || u.gcs == nil {
		return fmt.Errorf("tokenBlueprint bucket usecase/gcs client is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return fmt.Errorf("tokenBlueprintID is empty")
	}

	iconBucket := tokenIconBucketName()
	contentsBucket := tokenContentsBucketName()
	if iconBucket == "" || contentsBucket == "" {
		return fmt.Errorf("bucket names are empty (icon=%q contents=%q)", iconBucket, contentsBucket)
	}

	log.Printf(
		"[TokenBlueprintBucket] ensure start id=%q iconBucket=%q contentsBucket=%q",
		id, iconBucket, contentsBucket,
	)

	// 1) Ensure buckets are accessible (Attrs)
	if err := u.ensureBucketAccessible(ctx, iconBucket); err != nil {
		log.Printf("[TokenBlueprintBucket] ERROR ensure icon bucket accessible bucket=%s: %v", iconBucket, err)
		return err
	}
	if err := u.ensureBucketAccessible(ctx, contentsBucket); err != nil {
		log.Printf("[TokenBlueprintBucket] ERROR ensure contents bucket accessible bucket=%s: %v", contentsBucket, err)
		return err
	}

	// 2) Ensure ".keep" objects exist (idempotent create)
	iconKeep := keepObjectPath(id)
	contentsKeep := keepObjectPath(id)

	if err := u.ensureKeepObject(ctx, iconBucket, iconKeep); err != nil {
		log.Printf("[TokenBlueprintBucket] ERROR ensure icon keep bucket=%s object=%s: %v", iconBucket, iconKeep, err)
		return err
	}
	if err := u.ensureKeepObject(ctx, contentsBucket, contentsKeep); err != nil {
		log.Printf("[TokenBlueprintBucket] ERROR ensure contents keep bucket=%s object=%s: %v", contentsBucket, contentsKeep, err)
		return err
	}

	log.Printf(
		"[TokenBlueprintBucket] ensure success id=%q iconKeep=%q contentsKeep=%q",
		id, "gs://"+iconBucket+"/"+iconKeep, "gs://"+contentsBucket+"/"+contentsKeep,
	)
	return nil
}

func (u *TokenBlueprintBucketUsecase) ensureBucketAccessible(ctx context.Context, bucket string) error {
	b := strings.TrimSpace(bucket)
	if b == "" {
		return fmt.Errorf("bucket is empty")
	}

	// Attrs => requires storage.buckets.get
	_, err := u.gcs.Bucket(b).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("get bucket attrs %s: %w", b, err)
	}
	return nil
}

// ensureKeepObject creates object only if it does not exist.
// - Uses DoesNotExist condition to be idempotent.
// - If object already exists, treat as success.
func (u *TokenBlueprintBucketUsecase) ensureKeepObject(ctx context.Context, bucket, object string) error {
	bucket = strings.TrimSpace(bucket)
	object = strings.TrimLeft(strings.TrimSpace(object), "/")
	if bucket == "" {
		return fmt.Errorf("bucket is empty")
	}
	if object == "" {
		return fmt.Errorf("object is empty")
	}

	// Try create with precondition: only create if not exists.
	oh := u.gcs.Bucket(bucket).Object(object).If(storage.Conditions{DoesNotExist: true})

	w := oh.NewWriter(ctx)
	w.ContentType = "application/octet-stream"
	w.CacheControl = "no-store"

	// 0-byte payloadでも良いが、運用上の可視性のためタイムスタンプを入れておく。
	_, _ = io.WriteString(w, fmt.Sprintf("created_at=%s\n", time.Now().UTC().Format(time.RFC3339)))

	if err := w.Close(); err != nil {
		// If it already exists, GCS returns Precondition Failed (HTTP 412).
		if isPreconditionFailed(err) {
			log.Printf("[TokenBlueprintBucket] keep already exists bucket=%q object=%q", bucket, object)
			return nil
		}
		return fmt.Errorf("create keep object bucket=%s object=%s: %w", bucket, object, err)
	}

	log.Printf("[TokenBlueprintBucket] keep created bucket=%q object=%q", bucket, object)
	return nil
}

// isPreconditionFailed detects HTTP 412 (object already exists with DoesNotExist condition).
func isPreconditionFailed(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "412") || strings.Contains(strings.ToLower(msg), "precondition")
}
