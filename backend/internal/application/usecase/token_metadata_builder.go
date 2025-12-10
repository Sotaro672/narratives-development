// backend/internal/application/usecase/token_metadata_builder.go
package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenMetadataBuilder は TokenBlueprint から NFT メタデータ JSON を生成する責務を持ちます。
type TokenMetadataBuilder struct{}

// NewTokenMetadataBuilder はビルダーのコンストラクタです。
func NewTokenMetadataBuilder() *TokenMetadataBuilder {
	return &TokenMetadataBuilder{}
}

// BuildFromBlueprint は TokenBlueprint から Arweave 用メタデータ JSON を生成します。
func (b *TokenMetadataBuilder) BuildFromBlueprint(pb tbdom.TokenBlueprint) ([]byte, error) {
	name := strings.TrimSpace(pb.Name) // ドメイン側のフィールド名に合わせて調整
	symbol := strings.TrimSpace(pb.Symbol)

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("token blueprint name or symbol is empty")
	}

	metadata := map[string]interface{}{
		"name":   name,
		"symbol": symbol,
	}

	// description フィールドが存在する場合だけ追加する
	if desc := strings.TrimSpace(pb.Description); desc != "" {
		metadata["description"] = desc
	}

	// 画像 URL 周りは、TokenBlueprint にフィールドが増えた段階で追加する
	// 例:
	// if pb.ImageURL != "" { ... }

	return json.Marshal(metadata)
}
