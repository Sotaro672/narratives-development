// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	commondom "narratives/internal/domain/common"
	modeldom "narratives/internal/domain/model"
	orderdom "narratives/internal/domain/order"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
	pbcatdom "narratives/internal/domain/productBlueprintCategory"
	tbdom "narratives/internal/domain/tokenBlueprint"
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

	ErrOrderPurchasedQueryNotConfigured = errors.New("order_purchased_query: not configured")
	ErrInvalidAvatarID                  = errors.New("order_purchased_query: invalid avatarId")

	ErrOrderScanVerifyQueryNotConfigured  = errors.New("order_scan_verify_query: not configured")
	ErrOrderScanVerifyAvatarIDEmpty       = errors.New("order_scan_verify_query: avatarId is empty")
	ErrOrderScanVerifyProductIDEmpty      = errors.New("order_scan_verify_query: productId is empty")
	ErrOrderScanVerifyTokenNotFound       = errors.New("order_scan_verify_query: token not found for productId")
	ErrOrderScanVerifyTokenBlueprintEmpty = errors.New("order_scan_verify_query: tokenBlueprintId is empty")
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
// preview では measurements 補完用途だけに使う。
// modelNumber / size / color / rgb / volume は NameResolver.ResolveModelResolved を正とする。
type ModelVariationReader interface {
	GetByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

// ProductBlueprintReader is a minimal read port for ProductBlueprint.
// We need: modelId -> productBlueprintId -> productBlueprint(+patch if needed).
type ProductBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []pbdom.ModelRef, error)

	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// TokenReader is a minimal read port for Token information by productId.
// 想定: tokens/{productId} を読む（存在しない=未mint は nil を返してOK）
type TokenReader interface {
	GetByProductID(ctx context.Context, productID string) (*dto.TokenInfo, error)
}

// BrandReader resolves brandId -> Brand.
//
// brand.RepositoryPort / brand.Repository の GetByID(ctx, id string) に合わせる。
// preview で必要な brandName / brandIcon は GetByID の結果から組み立てる。
type BrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// AvatarNameIconReader resolves avatarId -> Avatar.
// avatar 側は GetByID port に統一する。
type AvatarNameIconReader interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// TransferReader resolves mintAddress -> transfer records.
type TransferReader interface {
	ListByMintAddress(ctx context.Context, mintAddress string) ([]dto.PreviewTransferInfo, error)
}

// ------------------------------------------------------------
// Purchased / scan verify DTOs
// ------------------------------------------------------------

// PurchasedPair is a resolved (modelId, tokenBlueprintId) pair derived from an eligible order item.
type PurchasedPair struct {
	OrderID          string `json:"orderId"`
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// OrderPurchasedResult is the purchased-side query output.
// - Pairs は orderId 単位で返す（同一 modelId/tokenBlueprintId が複数回出る可能性あり）
type OrderPurchasedResult struct {
	AvatarID string          `json:"avatarId"`
	Pairs    []PurchasedPair `json:"pairs"`
}

// ModelTokenPair is a minimal pair used for matching.
type ModelTokenPair struct {
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

type VerifyInput struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`
}

type VerifyResult struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`

	// scan side
	ScannedModelID          string `json:"scannedModelId"`
	ScannedTokenBlueprintID string `json:"scannedTokenBlueprintId"`

	// purchased side (dedup list)
	PurchasedPairs []ModelTokenPair `json:"purchasedPairs"`

	// verdict
	Matched bool            `json:"matched"`
	Match   *ModelTokenPair `json:"match,omitempty"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// PreviewQuery resolves preview entry info from productId.
// This struct is intended to be injected as cont.PreviewQ.
//
// It also owns scan verification dependencies so NewPreviewQuery is the
// single construction entry point for preview + order scan verification.
type PreviewQuery struct {
	ProductRepo          ProductReader
	ModelRepo            ModelVariationReader
	ProductBlueprintRepo ProductBlueprintReader

	// order scan verify / purchased-side resolver
	OrderRepo orderdom.Repository

	// modelId -> apparel/alcohol display fields
	NameResolver *appresolver.NameResolver

	// tokens/{productId} を読む
	TokenRepo TokenReader

	// tokenBlueprint を読む
	TokenBlueprintRepo tbdom.RepositoryPort

	// tokens.toAddress -> owner を解決
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// display-only name resolvers
	BrandRepo          BrandReader
	AvatarNameIconRepo AvatarNameIconReader

	// mintAddress -> transfers を解決
	TransferRepo TransferReader
}

// ------------------------------------------------------------
// Constructor
// ------------------------------------------------------------

// NewPreviewQuery constructs PreviewQuery.
// This is the only entry point for wiring preview and scan verification dependencies.
func NewPreviewQuery(
	productRepo ProductReader,
	modelRepo ModelVariationReader,
	pbRepo ProductBlueprintReader,
	orderRepo orderdom.Repository,
	nameResolver *appresolver.NameResolver,
	tokenRepo TokenReader,
	tokenBlueprintRepo tbdom.RepositoryPort,
	ownerResolveQ *sharedquery.OwnerResolveQuery,
	brandRepo BrandReader,
	avatarNameIconRepo AvatarNameIconReader,
	transferRepo TransferReader,
) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo:          productRepo,
		ModelRepo:            modelRepo,
		ProductBlueprintRepo: pbRepo,
		OrderRepo:            orderRepo,
		NameResolver:         nameResolver,
		TokenRepo:            tokenRepo,
		TokenBlueprintRepo:   tokenBlueprintRepo,
		OwnerResolveQ:        ownerResolveQ,
		BrandRepo:            brandRepo,
		AvatarNameIconRepo:   avatarNameIconRepo,
		TransferRepo:         transferRepo,
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

	id := productID
	if id == "" {
		return "", ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	modelID := p.ModelID
	if modelID == "" {
		return "", ErrModelIDEmpty
	}

	return modelID, nil
}

// ResolveModelInfoByProductID resolves modelId AND variation fields
// from productId.
// It supports both apparel and alcohol:
// - apparel: modelNumber / size / color / rgb / measurements
// - alcohol: modelNumber / volumeValue / volumeUnit
func (q *PreviewQuery) ResolveModelInfoByProductID(
	ctx context.Context,
	productID string,
) (*dto.PreviewModelInfo, error) {
	if q == nil || q.ProductRepo == nil || q.ModelRepo == nil || q.NameResolver == nil {
		return nil, ErrPreviewQueryNotConfigured
	}

	id := productID
	if id == "" {
		return nil, ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	modelID := p.ModelID
	if modelID == "" {
		return nil, ErrModelIDEmpty
	}

	// measurements 補完用。
	// modelNumber / size / color / rgb / volume は NameResolver.ResolveModelResolved を正とする。
	mv, err := q.ModelRepo.GetByID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if mv == nil {
		return nil, ErrModelVariationNotFound
	}

	// modelId -> productBlueprintId -> productBlueprint(全フィールド) + patch
	if q.ProductBlueprintRepo == nil {
		return nil, ErrProductBlueprintRepoNotConfigured
	}

	pbID, _, err := q.ProductBlueprintRepo.GetIDByModelID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if pbID == "" {
		return nil, ErrProductBlueprintIDEmpty
	}

	pb, err := q.ProductBlueprintRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	category := pb.ProductBlueprintCategory

	out := &dto.PreviewModelInfo{
		ProductID: productID,
		ModelID:   modelID,

		ProductBlueprintCategoryCode: category.Code,
		ProductBlueprintCategoryKind: category.Kind,
		ProductBlueprintCategoryName: category.NameJa,
		ProductBlueprintCategory:     &category,

		ProductBlueprintID: pbID,
		ProductBlueprint:   &pb,
	}

	if schema, ok := pbcatdom.GetCategoryInputSchema(category.Code); ok {
		out.CategoryInputSchema = &schema
	}

	if err := q.fillResolvedModelInfo(ctx, out, modelID, mv, category.Kind); err != nil {
		return nil, err
	}

	// tokens/{productId}（存在すれば付与）
	if q.TokenRepo != nil {
		tok, err := q.TokenRepo.GetByProductID(ctx, id)
		if err != nil {
			return nil, err
		}
		out.Token = tok

		// brandId -> brandName（tokens側）
		if tok != nil && tok.BrandID != "" {
			if brandName := q.resolveBrandNameForPreview(ctx, tok.BrandID, out); brandName != "" {
				tok.BrandName = brandName
			}
		}

		// tokenBlueprint.Patch は domain/tokenBlueprint 側の Patch を再利用する。
		if tok != nil && q.TokenBlueprintRepo != nil && tok.TokenBlueprintID != "" {
			tb, perr := q.TokenBlueprintRepo.GetByID(ctx, tok.TokenBlueprintID)
			if perr == nil && tb != nil {
				tbPatch := tbdom.NewPatchFromTokenBlueprint(tb)

				if tbPatch.BrandID != "" && tbPatch.BrandName == "" {
					tbPatch.BrandName = q.resolveBrandNameForPreview(ctx, tbPatch.BrandID, out)
				}

				out.TokenBlueprintPatch = &tbPatch
			}
		}

		// owner 解決（token が無い / toAddress が無い / OwnerResolveQ が無い場合は何もしない）
		if q.OwnerResolveQ != nil && tok != nil {
			addr := tok.ToAddress
			if addr != "" {
				res, rerr := q.OwnerResolveQ.Resolve(ctx, addr)
				if rerr == nil && res != nil {
					switch res.OwnerType {
					case sharedquery.OwnerTypeAvatar:
						res.BrandID = ""
						res.BrandName = ""
					case sharedquery.OwnerTypeBrand:
						res.AvatarID = ""
						res.AvatarName = ""
					default:
					}
					out.Owner = res
				}
			}
		}

		// transfer 解決（token が無い / mintAddress が無い / TransferRepo が無い場合は何もしない）
		if q.TransferRepo != nil && tok != nil && tok.MintAddress != "" {
			transfers, terr := q.TransferRepo.ListByMintAddress(ctx, tok.MintAddress)
			if terr == nil {
				out.Transfers = q.resolveTransferOwners(ctx, transfers)
			}
		}
	}

	return out, nil
}

// ListEligiblePairsByAvatarID resolves eligible transfer pairs through order.Repository.
//
// Repository-side condition:
// - order.avatarId == avatarID
// - order.paid == true
// - item.transferred == false
// - item.modelId is not empty
// - item.inventoryId is not empty
//
// This query then derives:
// - modelId from item.modelId
// - tokenBlueprintId from item.inventoryId 2nd segment
func (q *PreviewQuery) ListEligiblePairsByAvatarID(ctx context.Context, avatarID string) (OrderPurchasedResult, error) {
	if q == nil || q.OrderRepo == nil {
		return OrderPurchasedResult{}, ErrOrderPurchasedQueryNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return OrderPurchasedResult{}, ErrInvalidAvatarID
	}

	items, err := q.OrderRepo.ListEligibleTransferItemsByAvatarID(ctx, aid)
	if err != nil {
		return OrderPurchasedResult{}, err
	}

	pairs := make([]PurchasedPair, 0, len(items))

	for _, item := range items {
		if item.ModelID == "" {
			continue
		}
		if item.InventoryID == "" {
			continue
		}

		parts := strings.Split(item.InventoryID, "__")
		if len(parts) < 2 || parts[1] == "" {
			continue
		}

		tokenBlueprintID := parts[1]

		pairs = append(pairs, PurchasedPair{
			OrderID:          item.OrderID,
			ModelID:          item.ModelID,
			TokenBlueprintID: tokenBlueprintID,
		})
	}

	return OrderPurchasedResult{
		AvatarID: aid,
		Pairs:    pairs,
	}, nil
}

// VerifyMatch verifies whether the scanned pair exists in purchased(untransferred) pairs.
func (q *PreviewQuery) VerifyMatch(ctx context.Context, in VerifyInput) (VerifyResult, error) {
	if q == nil || q.OrderRepo == nil || q.ProductRepo == nil || q.NameResolver == nil {
		return VerifyResult{}, ErrOrderScanVerifyQueryNotConfigured
	}

	avatarID := in.AvatarID
	productID := in.ProductID

	if avatarID == "" {
		return VerifyResult{}, ErrOrderScanVerifyAvatarIDEmpty
	}
	if productID == "" {
		return VerifyResult{}, ErrOrderScanVerifyProductIDEmpty
	}

	// 1) scan side: productId -> modelId + tokenBlueprintId(tokens/{productId}.tokenBlueprintId)
	info, err := q.ResolveModelInfoByProductID(ctx, productID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: preview resolve failed: %w", err)
	}
	if info == nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: preview resolve returned nil")
	}

	scannedModelID := info.ModelID
	if scannedModelID == "" {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: scanned modelId is empty")
	}

	// token must exist (tokens/{productId} が存在する or TokenRepo 注入されている)
	if info.Token == nil {
		return VerifyResult{}, ErrOrderScanVerifyTokenNotFound
	}

	// scanned tokenBlueprintId is tokens/{productId}.tokenBlueprintId (docId=productId)
	scannedTokenBlueprintID := info.Token.TokenBlueprintID
	if scannedTokenBlueprintID == "" {
		return VerifyResult{}, ErrOrderScanVerifyTokenBlueprintEmpty
	}

	// 2) purchased side: avatarId -> paid orders -> items.transfer=false -> (modelId,tbId)
	purchased, err := q.ListEligiblePairsByAvatarID(ctx, avatarID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: purchased pairs resolve failed: %w", err)
	}

	// 3) dedup to []ModelTokenPair
	seen := map[string]struct{}{}
	outPairs := make([]ModelTokenPair, 0, len(purchased.Pairs))

	for _, p := range purchased.Pairs {
		modelID := p.ModelID
		tokenBlueprintID := p.TokenBlueprintID
		if modelID == "" || tokenBlueprintID == "" {
			continue
		}

		key := modelID + "::" + tokenBlueprintID
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		outPairs = append(outPairs, ModelTokenPair{
			ModelID:          modelID,
			TokenBlueprintID: tokenBlueprintID,
		})
	}

	// 4) match
	var match *ModelTokenPair
	for i := range outPairs {
		p := outPairs[i]
		if p.ModelID == scannedModelID && p.TokenBlueprintID == scannedTokenBlueprintID {
			cp := p
			match = &cp
			break
		}
	}

	return VerifyResult{
		AvatarID:                avatarID,
		ProductID:               productID,
		ScannedModelID:          scannedModelID,
		ScannedTokenBlueprintID: scannedTokenBlueprintID,
		PurchasedPairs:          outPairs,
		Matched:                 match != nil,
		Match:                   match,
	}, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (q *PreviewQuery) getBrandNameIcon(
	ctx context.Context,
	brandID string,
) (branddom.NameIcon, error) {
	if q == nil || q.BrandRepo == nil {
		return branddom.NameIcon{}, ErrPreviewQueryNotConfigured
	}
	if brandID == "" {
		return branddom.NameIcon{}, branddom.ErrInvalidID
	}

	b, err := q.BrandRepo.GetByID(ctx, brandID)
	if err != nil {
		return branddom.NameIcon{}, err
	}

	return branddom.NameIcon{
		Name:      b.Name,
		BrandIcon: b.BrandIcon,
	}, nil
}

func (q *PreviewQuery) resolveBrandNameForPreview(
	ctx context.Context,
	brandID string,
	out *dto.PreviewModelInfo,
) string {
	if q == nil || q.BrandRepo == nil || brandID == "" {
		return ""
	}

	ni, err := q.getBrandNameIcon(ctx, brandID)
	if err != nil || ni.Name == "" {
		return ""
	}

	if out != nil && out.BrandName == "" {
		out.BrandName = ni.Name
	}

	return ni.Name
}

func (q *PreviewQuery) fillResolvedModelInfo(
	ctx context.Context,
	out *dto.PreviewModelInfo,
	modelID string,
	mv modeldom.ModelVariation,
	categoryKind commondom.ProductCategoryKind,
) error {
	if out == nil {
		return ErrPreviewQueryNotConfigured
	}
	if q == nil || q.NameResolver == nil {
		return ErrPreviewQueryNotConfigured
	}

	resolved := q.NameResolver.ResolveModelResolved(ctx, modelID)
	if resolved.Kind == "" && resolved.ModelNumber == "" {
		return ErrModelVariationNotFound
	}

	modelKind := resolved.Kind
	if modelKind == "" {
		modelKind = string(categoryKind)
	}

	out.ModelKind = commondom.ProductCategoryKind(modelKind)
	out.ModelNumber = resolved.ModelNumber
	out.ModelLabel = buildPreviewModelLabel(
		modelKind,
		resolved.ModelNumber,
		resolved.Size,
		resolved.Color,
		resolved.VolumeValue,
		resolved.VolumeUnit,
	)

	out.Size = resolved.Size
	out.Color = resolved.Color
	if resolved.RGB != nil {
		out.RGB = *resolved.RGB
	}

	out.VolumeValue = resolved.VolumeValue
	out.VolumeUnit = resolved.VolumeUnit

	// measurements は apparel のみに存在するため、preview 側で補完する。
	// model 表示値そのものは resolver の結果を正とする。
	if modelKind == string(modeldom.ModelVariationKindApparel) {
		if apparelVariation, ok := toPreviewApparelModelVariation(mv); ok {
			out.Measurements = cloneMeasurements(apparelVariation.Measurements)
		}
	}

	return nil
}

func buildPreviewModelLabel(
	kind string,
	modelNumber string,
	size string,
	color string,
	volumeValue *int,
	volumeUnit string,
) string {
	switch kind {
	case "alcohol":
		if volumeValue != nil && volumeUnit != "" {
			if modelNumber != "" {
				return fmt.Sprintf("%s / %d%s", modelNumber, *volumeValue, volumeUnit)
			}
			return fmt.Sprintf("%d%s", *volumeValue, volumeUnit)
		}
		return modelNumber

	default:
		if modelNumber != "" && size != "" && color != "" {
			return fmt.Sprintf("%s / %s / %s", modelNumber, size, color)
		}
		if modelNumber != "" && size != "" {
			return fmt.Sprintf("%s / %s", modelNumber, size)
		}
		if modelNumber != "" && color != "" {
			return fmt.Sprintf("%s / %s", modelNumber, color)
		}
		if size != "" && color != "" {
			return fmt.Sprintf("%s / %s", size, color)
		}
		if modelNumber != "" {
			return modelNumber
		}
		if size != "" {
			return size
		}
		return color
	}
}

func toPreviewApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
	if v == nil {
		return modeldom.ApparelModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x, true
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return modeldom.ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.ApparelModelVariation{}, false
	}
}

func (q *PreviewQuery) resolveTransferOwners(
	ctx context.Context,
	transfers []dto.PreviewTransferInfo,
) []dto.PreviewTransferInfo {
	if len(transfers) == 0 {
		return transfers
	}

	out := make([]dto.PreviewTransferInfo, 0, len(transfers))
	for _, tr := range transfers {
		item := tr

		if q != nil && q.OwnerResolveQ != nil {
			if tr.FromWalletAddress != "" {
				if res, err := q.OwnerResolveQ.Resolve(ctx, tr.FromWalletAddress); err == nil && res != nil {
					switch res.OwnerType {
					case sharedquery.OwnerTypeAvatar:
						item.FromAvatarID = res.AvatarID
						item.FromBrandID = ""
						item.FromBrandName = ""
						item.FromBrandIcon = ""
						q.fillAvatarTransferDisplay(ctx, res.AvatarID, &item.FromAvatarName, &item.FromAvatarIcon, res.AvatarName)
					case sharedquery.OwnerTypeBrand:
						item.FromBrandID = res.BrandID
						item.FromAvatarID = ""
						item.FromAvatarName = ""
						item.FromAvatarIcon = ""
						q.fillBrandTransferDisplay(ctx, res.BrandID, &item.FromBrandName, &item.FromBrandIcon, res.BrandName)
					default:
					}
				}
			}

			if tr.ToWalletAddress != "" {
				if res, err := q.OwnerResolveQ.Resolve(ctx, tr.ToWalletAddress); err == nil && res != nil {
					switch res.OwnerType {
					case sharedquery.OwnerTypeAvatar:
						item.ToAvatarID = res.AvatarID
						item.ToBrandID = ""
						item.ToBrandName = ""
						item.ToBrandIcon = ""
						q.fillAvatarTransferDisplay(ctx, res.AvatarID, &item.ToAvatarName, &item.ToAvatarIcon, res.AvatarName)
					case sharedquery.OwnerTypeBrand:
						item.ToBrandID = res.BrandID
						item.ToAvatarID = ""
						item.ToAvatarName = ""
						item.ToAvatarIcon = ""
						q.fillBrandTransferDisplay(ctx, res.BrandID, &item.ToBrandName, &item.ToBrandIcon, res.BrandName)
					default:
					}
				}
			}
		}

		out = append(out, item)
	}

	return out
}

func (q *PreviewQuery) fillBrandTransferDisplay(
	ctx context.Context,
	brandID string,
	nameOut *string,
	iconOut *string,
	fallbackName string,
) {
	if nameOut == nil || iconOut == nil {
		return
	}

	*nameOut = ""
	*iconOut = ""

	if q == nil || brandID == "" {
		if fallbackName != "" {
			*nameOut = fallbackName
		}
		return
	}

	if q.BrandRepo != nil {
		ni, err := q.getBrandNameIcon(ctx, brandID)
		if err == nil && ni.Name != "" {
			*nameOut = ni.Name
			*iconOut = ni.BrandIcon
			return
		}
	}

	if fallbackName != "" {
		*nameOut = fallbackName
	}
}

func (q *PreviewQuery) fillAvatarTransferDisplay(
	ctx context.Context,
	avatarID string,
	nameOut *string,
	iconOut *string,
	fallbackName string,
) {
	if nameOut == nil || iconOut == nil {
		return
	}

	*nameOut = ""
	*iconOut = ""

	if q == nil || avatarID == "" {
		if fallbackName != "" {
			*nameOut = fallbackName
		}
		return
	}

	if q.AvatarNameIconRepo != nil {
		avatar, err := q.AvatarNameIconRepo.GetByID(ctx, avatarID)
		if err == nil && avatar.AvatarName != "" {
			*nameOut = avatar.AvatarName
			if avatar.AvatarIcon != nil {
				*iconOut = *avatar.AvatarIcon
			}
			return
		}
	}

	if fallbackName != "" {
		*nameOut = fallbackName
	}
}
