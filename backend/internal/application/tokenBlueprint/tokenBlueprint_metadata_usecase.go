// backend/internal/application/usecase/tokenBlueprint_metadata_usecase.go
package tokenBlueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ArweaveUploader is implemented by infra/arweave.HTTPUploader.
type ArweaveUploader interface {
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

// TokenBlueprintMetadataUsecase handles metadataUri composition policy.
type TokenBlueprintMetadataUsecase struct {
	tbRepo   tbdom.RepositoryPort
	uploader ArweaveUploader
}

func NewTokenBlueprintMetadataUsecase(tbRepo tbdom.RepositoryPort, uploader ArweaveUploader) *TokenBlueprintMetadataUsecase {
	return &TokenBlueprintMetadataUsecase{
		tbRepo:   tbRepo,
		uploader: uploader,
	}
}

// EnsureMetadataURI sets metadataUri if empty.
// Policy:
// - metadataUri が空なら Irys/Arweave に metadata JSON をアップロードして uri を得る
// - アップロード後も uri が空なら必ずエラー（空を許容しない）
func (u *TokenBlueprintMetadataUsecase) EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint metadata usecase/repo is nil")
	}
	if u.uploader == nil {
		return nil, fmt.Errorf("tokenBlueprint metadata uploader is nil")
	}
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}
	if tb.ID == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	// 既に入っていれば何もしない
	if tb.MetadataURI != "" {
		return tb, nil
	}

	// 1) metadata JSON を組み立て（期待値に合わせる）
	data, err := buildTokenBlueprintMetadataJSON(tb)
	if err != nil {
		return nil, err
	}

	// 2) Irys uploader 経由で Arweave にアップロード
	uri, err := u.uploader.UploadMetadata(ctx, data)
	if err != nil {
		return nil, err
	}
	if uri == "" {
		return nil, fmt.Errorf("metadataUri is empty after upload")
	}

	// 3) DB 更新
	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
		UpdatedAt:   nil,
		UpdatedBy:   ptr(actorID),
		DeletedAt:   nil,
		DeletedBy:   nil,
	})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		tb.MetadataURI = uri
		return tb, nil
	}
	return updated, nil
}

// buildTokenBlueprintMetadataJSON builds Metaplex-compatible metadata JSON.
// Expectations:
// - name/symbol/description: TokenBlueprint の項目から連動（固定値禁止）
// - image: TOKEN_ICON_BUCKET の tb.TokenIconObjectPath（空なら規約 "{id}/icon"）の URL
// - properties.files:
//   - icon URL は必ず 1 件
//   - token-contents の “参照パス” も必ず 1 件入れる（tb.TokenContentsObjectPath / 空なら "{id}/.keep"）
//   - tb.ContentFiles があれば、その f.ObjectPath を列挙する（空なら fallback で "{id}/contents/{contentId}"）
func buildTokenBlueprintMetadataJSON(tb *tbdom.TokenBlueprint) ([]byte, error) {
	id := tb.ID
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	// ★期待値: TokenBlueprint の各項目から拾う（固定値禁止）
	name := tb.Name
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := tb.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	// description は空でも許容
	desc := tb.Description

	// ------------------------------------------------------------
	// bucket helpers (inline: avoid cross-dir package dependency)
	// ------------------------------------------------------------
	iconBucket := os.Getenv("TOKEN_ICON_BUCKET")
	if iconBucket == "" {
		// default should match app/tokenBlueprint bucket policy
		iconBucket = "narratives-development_token_icon"
	}

	contentsBucket := os.Getenv("TOKEN_CONTENTS_BUCKET")
	if contentsBucket == "" {
		// default should match app/tokenBlueprint bucket policy
		contentsBucket = "narratives-development-token-contents"
	}

	gcsPublicURL := func(bucket, object string) string {
		if bucket == "" || object == "" {
			return ""
		}
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
	}

	// ------------------------------------------------------------
	// image/icon URL
	// ------------------------------------------------------------
	if iconBucket == "" {
		return nil, fmt.Errorf("token icon bucket is empty")
	}

	// ★永続化 objectPath を優先（空ならドメイン規約で補完）
	iconObjectPath := tb.TokenIconObjectPath
	if iconObjectPath == "" {
		iconObjectPath = tbdom.DefaultTokenIconObjectPath(id) // => "{id}/icon"
	}
	imageURL := gcsPublicURL(iconBucket, iconObjectPath)
	if imageURL == "" {
		return nil, fmt.Errorf("imageURL is empty")
	}

	// ------------------------------------------------------------
	// properties.files
	// ------------------------------------------------------------
	files := make([]map[string]any, 0, 2+len(tb.ContentFiles))

	// 1) icon は必ず入れる
	files = append(files, map[string]any{
		"uri":  imageURL,
		"type": "image/*",
	})

	// 2) token-contents へのパスも必ず入れる（期待値）
	if contentsBucket == "" {
		return nil, fmt.Errorf("token contents bucket is empty")
	}

	// ★永続化 objectPath を優先（空ならドメイン規約で補完）
	contentsKeepObjectPath := tb.TokenContentsObjectPath
	if contentsKeepObjectPath == "" {
		contentsKeepObjectPath = tbdom.DefaultTokenContentsObjectPath(id) // => "{id}/.keep"
	}
	contentsKeepURL := gcsPublicURL(contentsBucket, contentsKeepObjectPath)
	if contentsKeepURL == "" {
		return nil, fmt.Errorf("token contents keep url is empty")
	}

	files = append(files, map[string]any{
		"uri":  contentsKeepURL,
		"type": "application/octet-stream",
	})

	// 3) contentFiles があれば、objectPath を列挙して追加する
	seen := make(map[string]struct{}, len(tb.ContentFiles))
	for _, f := range tb.ContentFiles {
		cid := f.ID
		if cid == "" {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		objectPath := f.ObjectPath
		if objectPath == "" {
			// fallback（最低限の互換）
			objectPath = fmt.Sprintf("%s/contents/%s", id, cid)
		}

		uri := gcsPublicURL(contentsBucket, objectPath)
		if uri == "" {
			continue
		}

		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}

		files = append(files, map[string]any{
			"uri":  uri,
			"type": ct,
		})
	}

	payload := map[string]any{
		"name":        name,
		"symbol":      symbol,
		"description": desc,
		"image":       imageURL,
		"attributes": []map[string]any{
			{"trait_type": "TokenBlueprintID", "value": id},
		},
		"properties": map[string]any{
			"category": "image",
			"files":    files,
		},
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata json: %w", err)
	}
	return b, nil
}
