// backend/internal/application/usecase/tokenBlueprint_metadata_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
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

// EnsureMetadataURI sets metadataUri if empty.
// Current policy (before you switch timing to mint):
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
// - image: TOKEN_ICON_BUCKET の "{tokenBlueprintId}/icon" の URL
// - properties.files:
//   - icon URL は必ず 1 件
//   - token-contents の “参照パス” も必ず 1 件入れる（{id}/.keep を利用）
//   - tb.ContentFiles があれば、その contents URL も追加列挙する
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
	// image/icon URL
	// ------------------------------------------------------------
	iconBucket := tokenIconBucketName()
	if strings.TrimSpace(iconBucket) == "" {
		return nil, fmt.Errorf("token icon bucket is empty")
	}

	// tokenBlueprint_icon_usecase.go と完全一致させる（重要）
	iconObjectPath := tokenIconObjectPath(id) // => "{id}/icon"
	imageURL := strings.TrimSpace(gcsObjectPublicURL(iconBucket, iconObjectPath))
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
	//    contents bucket は非公開でも OK（文字列として載せるだけ）。
	//    “存在が保証される” オブジェクトとして {id}/.keep を使う。
	contentsBucket := tokenContentsBucketName()
	if strings.TrimSpace(contentsBucket) == "" {
		return nil, fmt.Errorf("token contents bucket is empty")
	}

	contentsKeepObjectPath := keepObjectPath(id) // => "{id}/.keep"（EnsureKeepObjects により作成される想定）
	contentsKeepURL := strings.TrimSpace(gcsObjectPublicURL(contentsBucket, contentsKeepObjectPath))
	if contentsKeepURL == "" {
		return nil, fmt.Errorf("token contents keep url is empty")
	}

	files = append(files, map[string]any{
		"uri":  contentsKeepURL,
		"type": "application/octet-stream",
	})

	// 3) contentFiles があれば、個別の contents オブジェクトも列挙して追加する
	//    ContentFile の構造が確定できないため、最低限 f.ID を object 名に採用する。
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

		objectPath := tokenContentsObjectPath(id, cid) // => "{id}/contents/{cid}"
		uri := strings.TrimSpace(gcsObjectPublicURL(contentsBucket, objectPath))
		if uri == "" {
			continue
		}

		files = append(files, map[string]any{
			"uri":  uri,
			"type": "application/octet-stream",
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
