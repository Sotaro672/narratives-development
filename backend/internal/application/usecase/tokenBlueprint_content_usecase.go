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

// 閲覧用（GET）の署名付きURLの有効期限
// - バケットが private のため、storage.googleapis.com の素URLは 403 になる
// - 画面表示には GET の署名付きURLが必要
const tokenContentsViewSignedURLTTL = 15 * time.Minute

// tokenContentsBucketName は tokenBlueprint_bucket_usecase.go 側の定義を利用する。
// （このファイルで再定義すると DuplicateDecl になるため禁止）

func gcsSignerEmail() string {
	return strings.TrimSpace(os.Getenv(envGCSSignerEmail))
}

// tokenContentsObjectPath returns stable object path.
//
// あなたの実パスに合わせる:
// - narratives-development-token-contents/{tokenBlueprintId}/{contentId}
//
// fileName はここでは contentId として扱う想定。
func tokenContentsObjectPath(tokenBlueprintID, fileName string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	fn := strings.TrimLeft(strings.TrimSpace(fileName), "/")
	if fn == "" {
		fn = "file"
	}
	return id + "/" + fn
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
// 追加:
// - ViewURL: private bucket の閲覧用（GET）署名付きURL
type TokenContentsUploadURL struct {
	UploadURL     string     `json:"uploadUrl"`
	PublicURL     string     `json:"publicUrl"`
	ViewURL       string     `json:"viewUrl"`
	ObjectPath    string     `json:"objectPath"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	ViewExpiresAt *time.Time `json:"viewExpiresAt,omitempty"`
}

// IssueTokenContentsUploadURL issues V4 signed PUT URL for "{tokenBlueprintId}/{contentId}".
// 併せて、閲覧用の V4 signed GET URL（ViewURL）も発行する。
//
// Required:
// - env TOKEN_CONTENTS_BUCKET set
// - env GCS_SIGNER_EMAIL set
// - Cloud Run runtime SA has iam.serviceAccounts.signBlob for signer SA
//
// Notes:
// - PUT の SignedURL は ContentType を署名に含むため、frontend の PUT は Content-Type を一致させる必要がある。
// - GET の SignedURL は ContentType を署名に含めない（ブラウザが Content-Type ヘッダを送らないため）。
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

	// PUT 用 Content-Type（未指定は octet-stream）
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

	// --------------------------
	// PUT (upload) signed URL
	// --------------------------
	expires := time.Now().UTC().Add(tokenContentsSignedURLTTL)

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        expires,

		// PUT は Content-Type を署名に含める（frontend PUT で一致必須）
		ContentType: ct,
	})
	if err != nil {
		return nil, fmt.Errorf("sign gcs upload url: %w", err)
	}

	// NOTE: public URL 生成も tokenBlueprint_bucket_usecase.go 側の共通関数を利用（重複定義禁止）
	// private bucket では直接 GET できないが、安定識別子としては有用（DB保存やログ等）
	publicURL := gcsObjectPublicURL(bucket, objectPath)

	// --------------------------
	// GET (view) signed URL
	// --------------------------
	viewExpires := time.Now().UTC().Add(tokenContentsViewSignedURLTTL)

	// GET は ContentType を設定しない（署名ヘッダに含まれてしまい、ブラウザの素fetch/imgが失敗する）
	viewURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        viewExpires,
	})
	if err != nil {
		return nil, fmt.Errorf("sign gcs view url: %w", err)
	}

	return &TokenContentsUploadURL{
		UploadURL:     strings.TrimSpace(uploadURL),
		PublicURL:     strings.TrimSpace(publicURL),
		ViewURL:       strings.TrimSpace(viewURL),
		ObjectPath:    strings.TrimSpace(objectPath),
		ExpiresAt:     ptr(expires),
		ViewExpiresAt: ptr(viewExpires),
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
