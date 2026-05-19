// backend/internal/application/usecase/tokenBlueprint_metadata_usecase.go
package tokenBlueprint

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

func NewTokenBlueprintMetadataUsecase(
	tbRepo tbdom.RepositoryPort,
	uploader ArweaveUploader,
) *TokenBlueprintMetadataUsecase {
	return &TokenBlueprintMetadataUsecase{
		tbRepo:   tbRepo,
		uploader: uploader,
	}
}

// EnsureMetadataURI sets metadataUri if empty.
//
// Firebase Storage migration policy:
// - metadataUri が空なら Irys/Arweave に metadata JSON をアップロードして uri を得る
// - metadata JSON の image/files は Firebase Storage downloadURL を使う
func (u *TokenBlueprintMetadataUsecase) EnsureMetadataURI(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
	actorID string,
) (*tbdom.TokenBlueprint, error) {
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

	// 1) metadata JSON を組み立てる
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
//
// Firebase Storage migration policy:
// - image は tb.IconURL を使う
// - properties.files は tb.IconURL と tb.ContentFiles[].URL を使う
// - tb.TokenIconObjectPath / tb.TokenContentsObjectPath から URL を生成しない
// - TOKEN_ICON_BUCKET / TOKEN_CONTENTS_BUCKET は使わない
// - https://storage.googleapis.com/... は組み立てない
// - .keep は metadata に入れない
// - 旧式 fallback は作らない
func buildTokenBlueprintMetadataJSON(tb *tbdom.TokenBlueprint) ([]byte, error) {
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint is nil")
	}

	id := strings.TrimSpace(tb.ID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprint.ID is empty")
	}

	name := strings.TrimSpace(tb.Name)
	if name == "" {
		return nil, fmt.Errorf("tokenBlueprint.name is empty")
	}

	symbol := strings.TrimSpace(tb.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint.symbol is empty")
	}

	desc := strings.TrimSpace(tb.Description)

	// ------------------------------------------------------------
	// image/icon URL
	// ------------------------------------------------------------
	// Firebase Storage 移行後:
	// - IconURL は frontend が Firebase Storage に upload した後、
	//   getDownloadURL() で取得した downloadURL。
	// - objectPath から URL を組み立てない。
	imageURL := strings.TrimSpace(tb.IconURL)
	if imageURL == "" {
		return nil, fmt.Errorf("tokenBlueprint.iconUrl is empty")
	}

	// ------------------------------------------------------------
	// properties.files
	// ------------------------------------------------------------
	files := make([]map[string]any, 0, 1+len(tb.ContentFiles))

	// 1) icon は必ず入れる
	files = append(files, map[string]any{
		"uri":  imageURL,
		"type": "image/*",
	})

	// 2) contentFiles があれば、保存済み Firebase Storage downloadURL を列挙する
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

		uri := strings.TrimSpace(f.URL)
		if uri == "" {
			return nil, fmt.Errorf("tokenBlueprint.contentFiles[%s].url is empty", cid)
		}

		ct := strings.TrimSpace(f.ContentType)
		if ct == "" {
			ct = "application/octet-stream"
		}

		file := map[string]any{
			"uri":  uri,
			"type": ct,
		}

		if name := strings.TrimSpace(f.Name); name != "" {
			file["name"] = name
		}

		files = append(files, file)
	}

	payload := map[string]any{
		"name":        name,
		"symbol":      symbol,
		"description": desc,
		"image":       imageURL,
		"attributes": []map[string]any{
			{
				"trait_type": "TokenBlueprintID",
				"value":      id,
			},
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
