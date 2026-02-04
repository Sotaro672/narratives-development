// backend/internal/application/usecase/list/feature_signed_url.go
//
// Responsibility:
// - ListImage の signed-url 発行（アップロード前段）を提供する。
// - 返却 DTO の正規化（trim / default 補完）と必須項目検証を行う。
//
// Features:
// - IssueImageSignedURL
package list

import (
	"context"
	"errors"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
)

func (uc *ListUsecase) IssueImageSignedURL(ctx context.Context, in ListImageIssueSignedURLInput) (ListImageIssueSignedURLOutput, error) {
	if uc.imageSignedURLIssuer == nil {
		return ListImageIssueSignedURLOutput{}, usecase.ErrNotSupported("List.IssueImageSignedURL")
	}

	in.ListID = strings.TrimSpace(in.ListID)
	in.FileName = strings.TrimSpace(in.FileName)
	in.ContentType = strings.TrimSpace(in.ContentType)

	out, err := uc.imageSignedURLIssuer.IssueSignedURL(ctx, in)
	if err != nil {
		return ListImageIssueSignedURLOutput{}, err
	}

	// ---- normalize (trim + defaults) ----
	out.ID = strings.TrimSpace(out.ID) // ✅ expected: imageId
	out.Bucket = strings.TrimSpace(out.Bucket)
	out.ObjectPath = strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/")
	out.UploadURL = strings.TrimSpace(out.UploadURL)
	out.PublicURL = strings.TrimSpace(out.PublicURL)
	out.FileName = strings.TrimSpace(out.FileName)
	out.ContentType = strings.TrimSpace(out.ContentType)
	out.ExpiresAt = strings.TrimSpace(out.ExpiresAt)

	// ✅ bucket default is NOT allowed here.
	// Bucket must be provided by issuer (env-fixed).
	// If empty, we treat it as invalid response.

	// ✅ public url default（空なら生成）
	// NOTE: PublicURL depends on bucket. If issuer forgets bucket, this must fail.
	if out.PublicURL == "" && out.Bucket != "" && out.ObjectPath != "" {
		// public URL convention is issuer-specific; if you always use storage.googleapis.com, keep this.
		// Otherwise, issuer should return PublicURL explicitly.
		out.PublicURL = "https://storage.googleapis.com/" + out.Bucket + "/" + out.ObjectPath
	}

	// ✅ expiresAt default（空なら計算）
	if out.ExpiresAt == "" {
		sec := in.ExpiresInSeconds
		if sec <= 0 {
			sec = 15 * 60
		}
		out.ExpiresAt = time.Now().UTC().Add(time.Duration(sec) * time.Second).Format(time.RFC3339)
	}

	// ✅ canonical objectPath check: lists/{listId}/images/{imageId}
	// Must satisfy:
	// - starts with "lists/{listId}/images/"
	// - ends with "{imageId}" (imageId == out.ID)
	if in.ListID != "" && out.ObjectPath != "" && out.ID != "" {
		prefix := "lists/" + in.ListID + "/images/"
		if !strings.HasPrefix(out.ObjectPath, prefix) {
			return ListImageIssueSignedURLOutput{}, errors.New("signed_url_object_path_not_canonical")
		}
		if !strings.HasSuffix(out.ObjectPath, "/"+out.ID) && out.ObjectPath != prefix+out.ID {
			// Handles both:
			// - "lists/{listId}/images/{imageId}"
			// (Suffix check above is defensive; second condition is the exact match)
			return ListImageIssueSignedURLOutput{}, errors.New("signed_url_object_path_id_mismatch")
		}
	}

	// ✅ Required checks
	// - UploadURL: must exist
	// - Bucket: must be provided by issuer (env-fixed)
	// - ObjectPath: must exist and be canonical
	// - ID: must be imageId (docId)
	if out.UploadURL == "" || out.Bucket == "" || out.ObjectPath == "" || out.ID == "" {
		return ListImageIssueSignedURLOutput{}, errors.New("signed_url_response_invalid")
	}

	return out, nil
}
