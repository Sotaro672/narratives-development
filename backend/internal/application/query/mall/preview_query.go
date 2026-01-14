// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	sharedquery "narratives/internal/application/query/shared"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Errors
// ------------------------------------------------------------

var (
	ErrPreviewQueryNotConfigured         = errors.New("preview_query: not configured")
	ErrInvalidProductID                  = errors.New("preview_query: invalid productId")
	ErrInvalidModelID                    = errors.New("preview_query: invalid modelId")
	ErrModelIDEmpty                      = errors.New("preview_query: resolved modelId is empty")
	ErrModelVariationNotFound            = errors.New("preview_query: model variation not found")
	ErrProductBlueprintRepoNotConfigured = errors.New("preview_query: productBlueprint repo not configured")
	ErrProductBlueprintIDEmpty           = errors.New("preview_query: resolved productBlueprintId is empty")
)

// ------------------------------------------------------------
// Ports (dependency interfaces)
// ------------------------------------------------------------

// ProductReader is a minimal read port for preview usecases.
// We only need: productId -> product -> modelId.
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ModelVariationReader is a minimal read port for model variation.
// We need: modelId(=variationId想定) -> variation -> modelNumber/size/color/rgb/measurements.
type ModelVariationReader interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// ProductBlueprintReader is a minimal read port for ProductBlueprint.
// We need: modelId -> productBlueprintId -> productBlueprint(+patch if needed).
type ProductBlueprintReader interface {
	// modelId(=variationId想定) -> productBlueprintId
	GetIDByModelID(ctx context.Context, modelID string) (string, error)

	// productBlueprintId -> patch (display fields)
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)

	// productBlueprintId -> entity (full fields)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// TokenReader is a minimal read port for Token information by productId.
// 想定: tokens/{productId} を読む（存在しない=未mint は nil を返してOK）
type TokenReader interface {
	GetByProductID(ctx context.Context, productID string) (*TokenInfo, error)
}

// ------------------------------------------------------------
// DTO (optional return shape)
// ------------------------------------------------------------

// TokenInfo is a minimal view for token doc (tokens/{productId}) used by preview.
type TokenInfo struct {
	// docID (=productId) をレスポンスに含める用途
	ProductID string `json:"productId"`

	BrandID string `json:"brandId,omitempty"`

	// ✅ NEW: tokenBlueprintId (tokens/{productId}.tokenBlueprintId)
	// order_scan_verify_query.go が scannedTokenBlueprintId を作るために使う
	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`

	// ✅ Off-chain cache (for faster UI)
	ToAddress   string `json:"toAddress,omitempty"`
	MetadataURI string `json:"metadataUri,omitempty"`

	// On-chain results
	MintAddress        string `json:"mintAddress,omitempty"`
	OnChainTxSignature string `json:"onChainTxSignature,omitempty"`

	// mintedAt は UI で表示したいことが多いので返す（不要なら削ってOK）
	MintedAt string `json:"mintedAt,omitempty"`
}

// PreviewModelInfo is what preview.dart eventually wants to display.
type PreviewModelInfo struct {
	ProductID string `json:"productId"`
	ModelID   string `json:"modelId"`

	ModelNumber  string         `json:"modelNumber"`
	Size         string         `json:"size"`
	Color        string         `json:"color"`
	RGB          int            `json:"rgb"` // Color.RGB は int（0xRRGGBB 想定）
	Measurements map[string]int `json:"measurements,omitempty"`

	// modelId -> productBlueprintId -> entity/patch
	ProductBlueprintID string                  `json:"productBlueprintId,omitempty"`
	ProductBlueprint   *pbdom.ProductBlueprint `json:"productBlueprint,omitempty"`

	// UIで Patch を使う場合
	ProductBlueprintPatch *pbdom.Patch `json:"productBlueprintPatch,omitempty"`

	// ✅ tokens/{productId}（あれば）
	Token *TokenInfo `json:"token,omitempty"`

	// ✅ owner_resolve_query.go のみを使う（A案）
	Owner *sharedquery.OwnerResolveResult `json:"owner,omitempty"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// PreviewQuery resolves preview entry info from productId.
// This struct is intended to be injected as cont.PreviewQ.
type PreviewQuery struct {
	ProductRepo          ProductReader
	ModelRepo            ModelVariationReader
	ProductBlueprintRepo ProductBlueprintReader

	// ✅ Optional: tokens/{productId} を読む（nil なら token は返さない）
	TokenRepo TokenReader

	// ✅ Optional: tokens.toAddress -> owner を解決（nil なら owner は返さない）
	OwnerResolveQ *sharedquery.OwnerResolveQuery
}

// NewPreviewQuery constructs PreviewQuery.
//
// ✅ NOTE:
// - DI(container.go) 側が 3 引数で呼ぶ想定に合わせてこちらを「正」とする。
// - TokenRepo / OwnerResolveQ は optional（後から注入可能）
// - PB を返す前提のため ProductBlueprintRepo は必須（ResolveModelInfoByProductID で参照）
func NewPreviewQuery(
	productRepo ProductReader,
	modelRepo ModelVariationReader,
	pbRepo ProductBlueprintReader,
) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo:          productRepo,
		ModelRepo:            modelRepo,
		ProductBlueprintRepo: pbRepo,
		TokenRepo:            nil,
		OwnerResolveQ:        nil,
	}
}

// NewPreviewQueryWithToken constructs PreviewQuery with TokenRepo.
// DI で 4 引数にしたい場合はこちらを使用。
func NewPreviewQueryWithToken(
	productRepo ProductReader,
	modelRepo ModelVariationReader,
	pbRepo ProductBlueprintReader,
	tokenRepo TokenReader,
) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo:          productRepo,
		ModelRepo:            modelRepo,
		ProductBlueprintRepo: pbRepo,
		TokenRepo:            tokenRepo,
		OwnerResolveQ:        nil,
	}
}

// ResolveModelIDByProductID resolves modelId from productId.
func (q *PreviewQuery) ResolveModelIDByProductID(
	ctx context.Context,
	productID string,
) (string, error) {
	if q == nil || q.ProductRepo == nil {
		return "", ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return "", ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return "", ErrModelIDEmpty
	}

	return modelID, nil
}

// ResolveModelMetaByModelID resolves model metadata from modelId.
// model.Color.RGB は int が正なので、戻り値も int で返します。
func (q *PreviewQuery) ResolveModelMetaByModelID(
	ctx context.Context,
	modelID string,
) (modelNumber string, size string, colorName string, rgb int, err error) {
	if q == nil || q.ModelRepo == nil {
		return "", "", "", 0, ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(modelID)
	if id == "" {
		return "", "", "", 0, ErrInvalidModelID
	}

	v, err := q.ModelRepo.GetModelVariationByID(ctx, id)
	if err != nil {
		return "", "", "", 0, err
	}
	if v == nil {
		return "", "", "", 0, ErrModelVariationNotFound
	}

	modelNumber = strings.TrimSpace(v.ModelNumber)
	size = strings.TrimSpace(v.Size)
	colorName = strings.TrimSpace(v.Color.Name)
	rgb = v.Color.RGB

	return modelNumber, size, colorName, rgb, nil
}

// ResolveModelInfoByProductID resolves modelId AND variation fields
// (modelNumber/size/color/rgb/measurements) from productId,
// and additionally resolves productBlueprintId + (productBlueprint entity + patch) by modelId.
// and optionally resolves tokens/{productId} if TokenRepo is configured.
// and optionally resolves owner (tokens.toAddress -> avatarId/brandId) if OwnerResolveQ is configured.
func (q *PreviewQuery) ResolveModelInfoByProductID(
	ctx context.Context,
	productID string,
) (*PreviewModelInfo, error) {
	if q == nil || q.ProductRepo == nil || q.ModelRepo == nil {
		return nil, ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return nil, ErrModelIDEmpty
	}

	v, err := q.ModelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrModelVariationNotFound
	}

	out := &PreviewModelInfo{
		ProductID:    id,
		ModelID:      modelID,
		ModelNumber:  strings.TrimSpace(v.ModelNumber),
		Size:         strings.TrimSpace(v.Size),
		Color:        strings.TrimSpace(v.Color.Name),
		RGB:          v.Color.RGB,
		Measurements: cloneMeasurements(v.Measurements),
	}

	// ✅ modelId -> productBlueprintId -> productBlueprint(全フィールド) + patch
	if q.ProductBlueprintRepo == nil {
		return nil, ErrProductBlueprintRepoNotConfigured
	}

	pbID, err := q.ProductBlueprintRepo.GetIDByModelID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	pbID = strings.TrimSpace(pbID)
	if pbID == "" {
		return nil, ErrProductBlueprintIDEmpty
	}

	pb, err := q.ProductBlueprintRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	patch, err := q.ProductBlueprintRepo.GetPatchByID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	out.ProductBlueprintID = pbID
	out.ProductBlueprint = &pb
	out.ProductBlueprintPatch = &patch

	// ✅ tokens/{productId}（存在すれば付与）
	if q.TokenRepo != nil {
		tok, err := q.TokenRepo.GetByProductID(ctx, id)
		if err != nil {
			return nil, err
		}
		out.Token = tok

		// ✅ owner 解決（token が無い / toAddress が無い / OwnerResolveQ が無い場合は何もしない）
		if q.OwnerResolveQ != nil && tok != nil {
			addr := strings.TrimSpace(tok.ToAddress)
			if addr != "" {
				res, rerr := q.OwnerResolveQ.Resolve(ctx, addr)
				if rerr == nil && res != nil {
					out.Owner = res
				}
				// rerr は preview を壊さない（owner は付加情報のため）
			}
		}
	}

	return out, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func cloneMeasurements(m map[string]int) map[string]int {
	if m == nil {
		return nil
	}
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
