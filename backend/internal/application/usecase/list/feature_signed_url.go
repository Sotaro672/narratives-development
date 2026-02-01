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
	listimgdom "narratives/internal/domain/listImage"
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
	out.ID = strings.TrimSpace(out.ID)
	out.Bucket = strings.TrimSpace(out.Bucket)
	out.ObjectPath = strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/")
	out.UploadURL = strings.TrimSpace(out.UploadURL)
	out.PublicURL = strings.TrimSpace(out.PublicURL)
	out.FileName = strings.TrimSpace(out.FileName)
	out.ContentType = strings.TrimSpace(out.ContentType)
	out.ExpiresAt = strings.TrimSpace(out.ExpiresAt)

	// id defaults: objectPath を採用（方針）
	if out.ID == "" && out.ObjectPath != "" {
		out.ID = out.ObjectPath
	}

	// bucket default
	if out.Bucket == "" {
		out.Bucket = listimgdom.DefaultBucket
	}

	// public url default（空なら生成）
	if out.PublicURL == "" && out.Bucket != "" && out.ObjectPath != "" {
		out.PublicURL = listimgdom.PublicURL(out.Bucket, out.ObjectPath)
	}

	// expiresAt default（空なら計算）
	if out.ExpiresAt == "" {
		sec := in.ExpiresInSeconds
		if sec <= 0 {
			sec = 15 * 60
		}
		out.ExpiresAt = time.Now().UTC().Add(time.Duration(sec) * time.Second).Format(time.RFC3339)
	}

	// 必須チェック
	if out.UploadURL == "" || out.Bucket == "" || out.ObjectPath == "" || out.ID == "" {
		return ListImageIssueSignedURLOutput{}, errors.New("signed_url_response_invalid")
	}

	return out, nil
}
