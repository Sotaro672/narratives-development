// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
)

// ------------------------------------------------------------
// Errors
// ------------------------------------------------------------

var (
	ErrPreviewQueryNotConfigured = errors.New("preview_query: not configured")
	ErrInvalidProductID          = errors.New("preview_query: invalid productId")
	ErrInvalidModelID            = errors.New("preview_query: invalid modelId")
	ErrModelIDEmpty              = errors.New("preview_query: resolved modelId is empty")
	ErrModelVariationNotFound    = errors.New("preview_query: model variation not found")
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

// ------------------------------------------------------------
// DTO (optional return shape)
// ------------------------------------------------------------

// PreviewModelInfo is what preview.dart eventually wants to display.
type PreviewModelInfo struct {
	ProductID string `json:"productId"`
	ModelID   string `json:"modelId"`

	ModelNumber  string         `json:"modelNumber"`
	Size         string         `json:"size"`
	Color        string         `json:"color"`
	RGB          int            `json:"rgb"` // Color.RGB は int（0xRRGGBB 想定）
	Measurements map[string]int `json:"measurements,omitempty"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// PreviewQuery resolves preview entry info from productId.
// This struct is intended to be injected as cont.PreviewQ.
type PreviewQuery struct {
	ProductRepo ProductReader
	ModelRepo   ModelVariationReader
}

// NewPreviewQuery constructs PreviewQuery.
//
// NOTE:
// 既存DIが NewPreviewQuery(productRepo) になっている場合は、
// DI側で modelRepo も渡すように更新してください。
func NewPreviewQuery(productRepo ProductReader, modelRepo ModelVariationReader) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo: productRepo,
		ModelRepo:   modelRepo,
	}
}

// ResolveModelIDByProductID resolves modelId from productId.
//
// このメソッドは handler 側 interface（PreviewQuery）互換のために必須です。
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
//
// handler 側（preview_handler.go / preview_me_handler.go）の interface が要求するため必須。
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
	rgb = v.Color.RGB // ✅ int のまま

	return modelNumber, size, colorName, rgb, nil
}

// ResolveModelInfoByProductID resolves modelId AND variation fields
// (modelNumber/size/color/rgb/measurements) from productId.
//
// 将来、handler 側でこのDTOをそのまま返す形にしてもOK。
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
