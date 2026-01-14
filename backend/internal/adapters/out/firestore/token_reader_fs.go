// backend/internal/adapters/out/firestore/token_reader_fs.go
package firestore

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mallquery "narratives/internal/application/query/mall"
)

type TokenReaderFS struct {
	Client *firestore.Client
}

func NewTokenReaderFS(client *firestore.Client) *TokenReaderFS {
	return &TokenReaderFS{Client: client}
}

// preview_query.go の mall.TokenReader を満たす
func (r *TokenReaderFS) GetByProductID(ctx context.Context, productID string) (*mallquery.TokenInfo, error) {
	if r == nil || r.Client == nil {
		return nil, mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil // ✅ token が無いのは正常（未mintなど）
		}
		return nil, err
	}

	raw := snap.Data()
	if raw == nil {
		// doc はあるが中身が無いケースは token無し相当として扱う
		log.Printf(`[token_reader_fs] tokens/%s raw=nil`, mask(id))
		return nil, nil
	}

	// ✅ DEBUG: raw を出す（型も確認できるように）
	// Firestore の map は value が interface{} なので %+v で十分に形が見える。
	log.Printf(`[token_reader_fs] tokens/%s raw=%+v`, mask(id), raw)
	// keyごとの型も見たい時用（必要なら活かしてください）
	for k, v := range raw {
		log.Printf(`[token_reader_fs] tokens/%s raw[%s]=%T %v`, mask(id), k, v, v)
	}

	// ✅ docID=productId 前提なので、ProductID はまず引数由来で固定
	out := &mallquery.TokenInfo{
		ProductID: id,
	}

	// map から柔軟に読む（Firestoreがスキーマレスなので）
	if v, ok := raw["brandId"].(string); ok {
		out.BrandID = strings.TrimSpace(v)
	} else if v, ok := raw["brandID"].(string); ok {
		out.BrandID = strings.TrimSpace(v)
	}

	// ✅ tokenBlueprintId を拾う（命名揺れ吸収）
	if v, ok := raw["tokenBlueprintId"].(string); ok {
		out.TokenBlueprintID = strings.TrimSpace(v)
	} else if v, ok := raw["tokenBlueprintID"].(string); ok {
		out.TokenBlueprintID = strings.TrimSpace(v)
	} else if v, ok := raw["token_blueprint_id"].(string); ok {
		out.TokenBlueprintID = strings.TrimSpace(v)
	}

	if v, ok := raw["mintAddress"].(string); ok {
		out.MintAddress = strings.TrimSpace(v)
	} else if v, ok := raw["mint_address"].(string); ok {
		out.MintAddress = strings.TrimSpace(v)
	}

	// ✅ A案: tokens にキャッシュする
	if v, ok := raw["toAddress"].(string); ok {
		out.ToAddress = strings.TrimSpace(v)
	} else if v, ok := raw["to_address"].(string); ok {
		out.ToAddress = strings.TrimSpace(v)
	}

	if v, ok := raw["metadataUri"].(string); ok {
		out.MetadataURI = strings.TrimSpace(v)
	} else if v, ok := raw["metadataURI"].(string); ok {
		out.MetadataURI = strings.TrimSpace(v)
	} else if v, ok := raw["metadata_uri"].(string); ok {
		out.MetadataURI = strings.TrimSpace(v)
	}

	// ✅ 命名揺れ吸収: onChainTxSignature 系
	if v, ok := raw["onChainTxSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["onchainTxSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["txSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["signature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	}

	// mintedAt（任意）
	// Firestore の timestamp のことが多いので、string以外も吸収してログで確認できるようにする
	if v, ok := raw["mintedAt"].(string); ok {
		out.MintedAt = strings.TrimSpace(v)
	} else if v, ok := raw["minted_at"].(string); ok {
		out.MintedAt = strings.TrimSpace(v)
	} else if v, ok := raw["mintedAt"]; ok && v != nil {
		// 文字列で取れない場合は fmt.Sprint で落とす（デバッグ用。必要ならDTO側をtimestamp対応に変更）
		out.MintedAt = strings.TrimSpace(fmt.Sprint(v))
	}

	log.Printf(
		`[token_reader_fs] tokens/%s mapped brandId=%q tokenBlueprintId=%q toAddress=%q mintAddress=%q`,
		mask(id),
		mask(out.BrandID),
		mask(out.TokenBlueprintID),
		mask(out.ToAddress),
		mask(out.MintAddress),
	)

	return out, nil
}

func mask(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
