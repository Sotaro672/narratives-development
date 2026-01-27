// backend/internal/application/usecase/tokenBlueprint_metadata_usecase.go
package tokenBlueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

// EnsureMetadataURIByTokenBlueprintID loads TokenBlueprint by id and ensures metadataUri if empty.
// Intended for mint-time usage via port adapter/DI.
// Returns the ensured metadataUri string.
func (u *TokenBlueprintMetadataUsecase) EnsureMetadataURIByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (string, error) {
	if u == nil || u.tbRepo == nil {
		return "", fmt.Errorf("tokenBlueprint metadata usecase/repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return "", fmt.Errorf("tokenBlueprintID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if tb == nil {
		return "", fmt.Errorf("tokenBlueprint %s not found", id)
	}

	updated, err := u.EnsureMetadataURI(ctx, tb, actorID)
	if err != nil {
		return "", err
	}
	if updated == nil {
		// best-effort fallback
		uri := strings.TrimSpace(tb.MetadataURI)
		if uri == "" {
			return "", fmt.Errorf("metadataUri is empty after ensure (tokenBlueprintID=%s)", id)
		}
		return uri, nil
	}

	uri := strings.TrimSpace(updated.MetadataURI)
	if uri == "" {
		return "", fmt.Errorf("metadataUri is empty after ensure (tokenBlueprintID=%s)", id)
	}
	return uri, nil
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
	if strings.TrimSpace(tb.ID) == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	// 既に入っていれば何もしない
	if strings.TrimSpace(tb.MetadataURI) != "" {
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
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, fmt.Errorf("metadataUri is empty after upload")
	}

	// 3) DB 更新
	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
		UpdatedAt:   nil,
		UpdatedBy:   ptr(strings.TrimSpace(actorID)),
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
	id := strings.TrimSpace(tb.ID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	// ★期待値: TokenBlueprint の各項目から拾う（固定値禁止）
	name := strings.TrimSpace(tb.Name)
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := strings.TrimSpace(tb.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	// description は空でも許容
	desc := strings.TrimSpace(tb.Description)

	// ------------------------------------------------------------
	// bucket helpers (inline: avoid cross-dir package dependency)
	// ------------------------------------------------------------
	iconBucket := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET"))
	if iconBucket == "" {
		// default should match app/tokenBlueprint bucket policy
		iconBucket = "narratives-development_token_icon"
	}

	contentsBucket := strings.TrimSpace(os.Getenv("TOKEN_CONTENTS_BUCKET"))
	if contentsBucket == "" {
		// default should match app/tokenBlueprint bucket policy
		contentsBucket = "narratives-development-token-contents"
	}

	gcsPublicURL := func(bucket, object string) string {
		bucket = strings.TrimSpace(bucket)
		object = strings.TrimLeft(strings.TrimSpace(object), "/")
		if bucket == "" || object == "" {
			return ""
		}
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
	}

	// ------------------------------------------------------------
	// image/icon URL
	// ------------------------------------------------------------
	if strings.TrimSpace(iconBucket) == "" {
		return nil, fmt.Errorf("token icon bucket is empty")
	}

	// ★永続化 objectPath を優先（空ならドメイン規約で補完）
	iconObjectPath := strings.TrimSpace(tb.TokenIconObjectPath)
	if iconObjectPath == "" {
		iconObjectPath = tbdom.DefaultTokenIconObjectPath(id) // => "{id}/icon"
	}
	imageURL := strings.TrimSpace(gcsPublicURL(iconBucket, iconObjectPath))
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
	if strings.TrimSpace(contentsBucket) == "" {
		return nil, fmt.Errorf("token contents bucket is empty")
	}

	// ★永続化 objectPath を優先（空ならドメイン規約で補完）
	contentsKeepObjectPath := strings.TrimSpace(tb.TokenContentsObjectPath)
	if contentsKeepObjectPath == "" {
		contentsKeepObjectPath = tbdom.DefaultTokenContentsObjectPath(id) // => "{id}/.keep"
	}
	contentsKeepURL := strings.TrimSpace(gcsPublicURL(contentsBucket, contentsKeepObjectPath))
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
		cid := strings.TrimSpace(f.ID)
		if cid == "" {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}

		objectPath := strings.TrimSpace(f.ObjectPath)
		if objectPath == "" {
			// fallback（最低限の互換）
			objectPath = fmt.Sprintf("%s/contents/%s", id, cid)
		}

		uri := strings.TrimSpace(gcsPublicURL(contentsBucket, objectPath))
		if uri == "" {
			continue
		}

		ct := strings.TrimSpace(f.ContentType)
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
