// backend/internal/application/usecase/tokenBlueprint_icon_usecase.go
package tokenBlueprint

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
// Config (token icon)
// ============================================================

// 署名に使うサービスアカウントメール（必須）
// 例: narratives-backend-sa@narratives-development-26c2d.iam.gserviceaccount.com
// ※ Cloud Run では env で明示推奨
const envTokenIconSignerEmail = "GCS_SIGNER_EMAIL"

// 署名付きURLの有効期限（PUT）
const tokenIconSignedURLTTL = 15 * time.Minute

// tokenIconSignerEmail returns signer service account email.
func tokenIconSignerEmail() string {
	if v := strings.TrimSpace(os.Getenv(envTokenIconSignerEmail)); v != "" {
		return v
	}
	return ""
}

// tokenIconObjectPath is a stable object path under "{tokenBlueprintId}/".
// 画像を後から差し替えても URL を固定化するため、ファイル名は常に "icon" に寄せる
func tokenIconObjectPath(tokenBlueprintID string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	return id + "/icon"
}

// ============================================================
// Usecase: Icon (Signed URL)
// ============================================================

type TokenBlueprintIconUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintIconUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintIconUsecase {
	return &TokenBlueprintIconUsecase{tbRepo: tbRepo}
}

// TokenIconUploadURL is returned to front for direct PUT.
type TokenIconUploadURL struct {
	UploadURL  string     `json:"uploadUrl"`
	PublicURL  string     `json:"publicUrl"`
	ObjectPath string     `json:"objectPath"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// IssueTokenIconUploadURL issues V4 signed PUT URL for "{tokenBlueprintId}/icon".
//
// Required:
// - env GCS_SIGNER_EMAIL set
// - Cloud Run runtime SA has iam.serviceAccounts.signBlob
//
// Note:
// - SignedURL includes ContentType; frontend PUT must match.
func (u *TokenBlueprintIconUsecase) IssueTokenIconUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	_ string, // fileName: not persisted; kept only to match handler signature
	contentType string,
) (*TokenIconUploadURL, error) {

	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint icon usecase/repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	// ensure blueprint exists
	if _, err := u.tbRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	bucket := tokenIconBucketName()
	if bucket == "" {
		return nil, fmt.Errorf("token icon bucket is empty")
	}

	accessID := tokenIconSignerEmail()
	if accessID == "" {
		return nil, fmt.Errorf("missing %s env (signer service account email)", envTokenIconSignerEmail)
	}

	objectPath := tokenIconObjectPath(id)

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

	expires := time.Now().UTC().Add(tokenIconSignedURLTTL)

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

	publicURL := gcsObjectPublicURL(bucket, objectPath)
	return &TokenIconUploadURL{
		UploadURL:  strings.TrimSpace(uploadURL),
		PublicURL:  strings.TrimSpace(publicURL),
		ObjectPath: strings.TrimSpace(objectPath),
		ExpiresAt:  ptr(expires),
	}, nil
}
