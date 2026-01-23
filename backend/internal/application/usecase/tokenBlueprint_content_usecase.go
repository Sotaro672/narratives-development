// backend/internal/application/usecase/tokenBlueprint_content_usecase.go
package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Config (token contents)
// ============================================================

// 今後は GCS_SIGNER_EMAIL のみを使用
const envGCSSignerEmail = "GCS_SIGNER_EMAIL"

// 署名付きURLの有効期限（PUT）
const tokenContentsSignedURLTTL = 15 * time.Minute

// tokenContentsBucketName は tokenBlueprint_bucket_usecase.go 側の定義を利用する。
// （このファイルで再定義すると DuplicateDecl になるため禁止）

func gcsSignerEmail() string {
	return strings.TrimSpace(os.Getenv(envGCSSignerEmail))
}

// tokenContentsObjectPath returns stable object path under "{tokenBlueprintId}/contents/".
// fileName は保存上の識別子として使う（表示用ではない）。
func tokenContentsObjectPath(tokenBlueprintID, fileName string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	fn := strings.TrimLeft(strings.TrimSpace(fileName), "/")
	if fn == "" {
		fn = "file"
	}
	return id + "/contents/" + fn
}

// ============================================================
// Usecase: Content (Signed URL + embedded contents ops)
// ============================================================

// TokenBlueprintContentUsecase handles embedded contents operations and signed URL issuing.
type TokenBlueprintContentUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintContentUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintContentUsecase {
	return &TokenBlueprintContentUsecase{tbRepo: tbRepo}
}

// TokenContentsUploadURL is returned to front for direct PUT.
type TokenContentsUploadURL struct {
	UploadURL  string     `json:"uploadUrl"`
	PublicURL  string     `json:"publicUrl"`
	ObjectPath string     `json:"objectPath"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// IssueTokenContentsUploadURL issues V4 signed PUT URL for "{tokenBlueprintId}/contents/{fileName}".
//
// Required:
// - env TOKEN_CONTENTS_BUCKET set
// - env GCS_SIGNER_EMAIL set
// - Cloud Run runtime SA has iam.serviceAccounts.signBlob for signer SA
//
// Note:
// - SignedURL includes ContentType; frontend PUT must match.
func (u *TokenBlueprintContentUsecase) IssueTokenContentsUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	fileName string,
	contentType string,
) (*TokenContentsUploadURL, error) {

	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	// ensure blueprint exists
	if _, err := u.tbRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	// NOTE: bucket 名取得は tokenBlueprint_bucket_usecase.go の実装に一本化
	bucket := tokenContentsBucketName()
	if bucket == "" {
		return nil, fmt.Errorf("token contents bucket is empty (env TOKEN_CONTENTS_BUCKET is required)")
	}

	accessID := gcsSignerEmail()
	if accessID == "" {
		return nil, fmt.Errorf("missing %s env (signer service account email)", envGCSSignerEmail)
	}

	objectPath := tokenContentsObjectPath(id, fileName)

	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create iamcredentials service: %w", err)
	}

	signBytes := func(b []byte) ([]byte, error) {
		name := "projects/-/serviceAccounts/" + accessID
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(b),
		}
		resp, err := iamSvc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}

	expires := time.Now().UTC().Add(tokenContentsSignedURLTTL)

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        expires,
		ContentType:    ct,
	})
	if err != nil {
		return nil, fmt.Errorf("sign gcs url: %w", err)
	}

	// NOTE: public URL 生成も tokenBlueprint_bucket_usecase.go 側の共通関数を利用（重複定義禁止）
	publicURL := gcsObjectPublicURL(bucket, objectPath)

	return &TokenContentsUploadURL{
		UploadURL:  strings.TrimSpace(uploadURL),
		PublicURL:  strings.TrimSpace(publicURL),
		ObjectPath: strings.TrimSpace(objectPath),
		ExpiresAt:  ptr(expires),
	}, nil
}

// ============================================================
// Embedded contents operations (existing)
// ============================================================

// ReplaceContentFiles replaces all embedded contents.
func (u *TokenBlueprintContentUsecase) ReplaceContentFiles(ctx context.Context, blueprintID string, files []tbdom.ContentFile, actorID string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	clean := dedupAndValidateContentFiles(files)
	if clean == nil {
		empty := make([]tbdom.ContentFile, 0)
		clean = empty
	}

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &clean,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

// SetContentVisibility updates visibility for a specific contentId (delegates to domain method).
func (u *TokenBlueprintContentUsecase) SetContentVisibility(ctx context.Context, blueprintID string, contentID string, v tbdom.ContentVisibility, actorID string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint content usecase/repo is nil")
	}

	tb, err := u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	now := time.Now().UTC()
	if err := tb.SetContentVisibility(contentID, v, actorID, now); err != nil {
		return nil, err
	}

	files := tb.ContentFiles
	updated, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &files,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// ============================================================
// internal helpers (contents)
// ============================================================

func normalizeContentFilesPtr(p *[]tbdom.ContentFile) *[]tbdom.ContentFile {
	if p == nil {
		return nil
	}
	clean := dedupAndValidateContentFiles(*p)
	return &clean
}

func dedupAndValidateContentFiles(files []tbdom.ContentFile) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	seen := make(map[string]struct{}, len(files))
	out := make([]tbdom.ContentFile, 0, len(files))

	for _, f := range files {
		// default visibility
		if strings.TrimSpace(string(f.Visibility)) == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}
		if err := f.Validate(); err != nil {
			// validate は domain の責務。ここで落とす。
			panic(err)
		}

		id := strings.TrimSpace(f.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, f)
	}

	return out
}
