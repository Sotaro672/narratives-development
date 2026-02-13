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

	// ✅ TrimSpace をしない（渡された値をそのまま使う）
	// in.ListID / in.FileName / in.ContentType はそのまま issuer に渡す

	out, err := uc.imageSignedURLIssuer.IssueSignedURL(ctx, in)
	if err != nil {
		return ListImageIssueSignedURLOutput{}, err
	}

	// ✅ TrimSpace をしない（issuer が返した値をそのまま扱う）
	// ただし canonical 判定で先頭 "/" が付くケースだけは仕様上許容しないため、
	// objectPath の先頭 "/" 除去のみは維持する（TrimSpace は使わない）。
	out.ObjectPath = strings.TrimLeft(out.ObjectPath, "/")

	// ✅ public url default（空なら生成）
	// NOTE: PublicURL depends on bucket. If issuer forgets bucket, this must fail.
	if out.PublicURL == "" && out.Bucket != "" && out.ObjectPath != "" {
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
