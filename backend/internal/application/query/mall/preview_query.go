// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	commondom "narratives/internal/domain/common"
	modeldom "narratives/internal/domain/model"
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
// We need: modelId(=variationId想定) -> variation -> model display fields.
//
// apparel:
//   - modelNumber
//   - size
//   - color
//   - rgb
//   - measurements
//
// alcohol:
//   - modelNumber
//   - volumeValue
//   - volumeUnit
type ModelVariationReader interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// ProductBlueprintReader is a minimal read port for ProductBlueprint.
// We need: modelId -> productBlueprintId -> productBlueprint(+patch if needed).
type ProductBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, error)
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// TokenReader is a minimal read port for Token information by productId.
// 想定: tokens/{productId} を読む（存在しない=未mint は nil を返してOK）
type TokenReader interface {
	GetByProductID(ctx context.Context, productID string) (*dto.TokenInfo, error)
}

// TokenBlueprintPatchReader is a minimal read port for TokenBlueprint patch.
// preview は tokenBlueprintId -> patch(表示用) だけ欲しい。
type TokenBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

// BrandNameReader resolves brandId -> brandName (display-only enrichment).
// 返り値の ok=false は「存在しない/名前が空などで表示できない」を想定。
type BrandNameReader interface {
	TryGetBrandName(ctx context.Context, brandID string) (name string, ok bool, err error)
}

// BrandNameIconReader resolves brandId -> brandName + brandIcon.
type BrandNameIconReader interface {
	TryGetBrandNameIcon(ctx context.Context, brandID string) (name string, brandIcon string, ok bool, err error)
}

// AvatarNameIconReader resolves avatarId -> avatarName + avatarIcon.
type AvatarNameIconReader interface {
	TryGetAvatarNameIcon(ctx context.Context, avatarID string) (name string, avatarIcon string, ok bool, err error)
}

// CompanyNameReader resolves companyId -> companyName (display-only enrichment).
// 返り値の ok=false は「存在しない/名前が空などで表示できない」を想定。
type CompanyNameReader interface {
	TryGetCompanyName(ctx context.Context, companyID string) (name string, ok bool, err error)
}

// TransferReader resolves mintAddress -> transfer records.
type TransferReader interface {
	ListByMintAddress(ctx context.Context, mintAddress string) ([]dto.PreviewTransferInfo, error)
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

	// Optional: modelId -> apparel/alcohol display fields
	NameResolver *appresolver.NameResolver

	// Optional: tokens/{productId} を読む（nil なら token は返さない）
	TokenRepo TokenReader

	// Optional: tokenBlueprint patch を読む（nil なら tokenBlueprintPatch は返さない）
	TokenBlueprintRepo TokenBlueprintPatchReader

	// Optional: tokens.toAddress -> owner を解決（nil なら owner は返さない）
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// Optional: display-only name resolvers
	BrandNameRepo      BrandNameReader
	BrandNameIconRepo  BrandNameIconReader
	AvatarNameIconRepo AvatarNameIconReader
	CompanyNameRepo    CompanyNameReader

	// Optional: mintAddress -> transfers を解決（nil なら transfers は返さない）
	TransferRepo TransferReader
}

// ------------------------------------------------------------
// Options (wiring helpers)
// ------------------------------------------------------------

type PreviewQueryOption func(q *PreviewQuery)

func WithNameResolver(r *appresolver.NameResolver) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.NameResolver = r
	}
}

func WithTokenRepo(r TokenReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.TokenRepo = r
	}
}

func WithTokenBlueprintRepo(r TokenBlueprintPatchReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.TokenBlueprintRepo = r
	}
}

func WithOwnerResolveQuery(qry *sharedquery.OwnerResolveQuery) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.OwnerResolveQ = qry
	}
}

func WithBrandNameRepo(r BrandNameReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.BrandNameRepo = r
	}
}

func WithBrandNameIconRepo(r BrandNameIconReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.BrandNameIconRepo = r
	}
}

func WithAvatarNameIconRepo(r AvatarNameIconReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.AvatarNameIconRepo = r
	}
}

func WithCompanyNameRepo(r CompanyNameReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.CompanyNameRepo = r
	}
}

func WithTransferRepo(r TransferReader) PreviewQueryOption {
	return func(q *PreviewQuery) {
		q.TransferRepo = r
	}
}

// NewPreviewQuery constructs PreviewQuery.
// This is the only entry point for wiring dependencies.
func NewPreviewQuery(
	productRepo ProductReader,
	modelRepo ModelVariationReader,
	pbRepo ProductBlueprintReader,
	opts ...PreviewQueryOption,
) *PreviewQuery {
	q := &PreviewQuery{
		ProductRepo:          productRepo,
		ModelRepo:            modelRepo,
		ProductBlueprintRepo: pbRepo,
		NameResolver:         nil,
		TokenRepo:            nil,
		TokenBlueprintRepo:   nil,
		OwnerResolveQ:        nil,
		BrandNameRepo:        nil,
		BrandNameIconRepo:    nil,
		AvatarNameIconRepo:   nil,
		CompanyNameRepo:      nil,
		TransferRepo:         nil,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(q)
		}
	}

	return q
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
	if q == nil || q.ProductRepo == nil || q.ModelRepo == nil {
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

	v, err := q.ModelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if v == nil || *v == nil {
		return nil, ErrModelVariationNotFound
	}

	// modelId -> productBlueprintId -> productBlueprint(全フィールド) + patch
	if q.ProductBlueprintRepo == nil {
		return nil, ErrProductBlueprintRepoNotConfigured
	}

	pbID, err := q.ProductBlueprintRepo.GetIDByModelID(ctx, modelID)
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

	patch, err := q.ProductBlueprintRepo.GetPatchByID(ctx, pbID)
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

		ProductBlueprintID:    pbID,
		ProductBlueprint:      &pb,
		ProductBlueprintPatch: &patch,
	}

	if schema, ok := pbcatdom.GetCategoryInputSchema(category.Code); ok {
		out.CategoryInputSchema = &schema
	}

	if err := q.fillResolvedModelInfo(ctx, out, modelID, *v, category.Kind); err != nil {
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
		if tok != nil && tok.BrandID != "" && q.BrandNameRepo != nil {
			name, ok, nerr := q.BrandNameRepo.TryGetBrandName(ctx, tok.BrandID)
			if nerr != nil {
				log.Printf(`[mall.preview] brandName resolve failed: brandId=%q err=%v`, tok.BrandID, nerr)
			} else if ok {
				tok.BrandName = name
				if out.BrandName == "" {
					out.BrandName = name
				}
			}
		}

		repoNil := q.TokenBlueprintRepo == nil
		tbID := ""
		if tok != nil {
			tbID = tok.TokenBlueprintID
		}
		log.Printf(
			`[mall.preview] tokenBlueprintPatch check repoNil=%t hasToken=%t tokenBlueprintId=%q`,
			repoNil,
			tok != nil,
			tbID,
		)

		if tok != nil && !repoNil {
			if tbID == "" {
				log.Printf(`[mall.preview] tokenBlueprintPatch skipped: tokenBlueprintId is empty`)
			} else {
				log.Printf(`[mall.preview] tokenBlueprintPatch fetching: tokenBlueprintId=%q`, tbID)

				tbPatch, perr := q.TokenBlueprintRepo.GetPatchByID(ctx, tbID)
				if perr != nil {
					log.Printf(
						`[mall.preview] tokenBlueprintPatch fetch failed: tokenBlueprintId=%q err=%v`,
						tbID,
						perr,
					)
				} else {
					if tbPatch.BrandID != "" && tbPatch.BrandName == "" && q.BrandNameRepo != nil {
						name, ok, nerr := q.BrandNameRepo.TryGetBrandName(ctx, tbPatch.BrandID)
						if nerr != nil {
							log.Printf(`[mall.preview] brandName resolve failed: brandId=%q err=%v`, tbPatch.BrandID, nerr)
						} else if ok {
							tbPatch.BrandName = name
							if out.BrandName == "" {
								out.BrandName = name
							}
						}
					}
					if tbPatch.CompanyID != "" && q.CompanyNameRepo != nil {
						cn, ok, cerr := q.CompanyNameRepo.TryGetCompanyName(ctx, tbPatch.CompanyID)
						if cerr != nil {
							log.Printf(`[mall.preview] companyName resolve failed: companyId=%q err=%v`, tbPatch.CompanyID, cerr)
						} else if ok {
							out.CompanyName = cn
						}
					}

					out.TokenBlueprintPatch = &tbPatch
					log.Printf(
						`[mall.preview] tokenBlueprintPatch attached: tokenBlueprintId=%q`,
						tbID,
					)
				}
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
			if terr != nil {
				log.Printf(`[mall.preview] transfer resolve failed: mintAddress=%q err=%v`, tok.MintAddress, terr)
			} else {
				out.Transfers = q.resolveTransferOwners(ctx, transfers)
			}
		}
	}

	return out, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

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

	if q != nil && q.NameResolver != nil {
		resolved := q.NameResolver.ResolveModelResolved(ctx, modelID)
		if resolved.Kind != "" || resolved.ModelNumber != "" {
			out.ModelKind = commondom.ProductCategoryKind(resolved.Kind)
			out.ModelNumber = resolved.ModelNumber
			out.ModelLabel = buildPreviewModelLabel(
				resolved.Kind,
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

			if resolved.Kind == "apparel" {
				if apparelVariation, ok := toPreviewApparelModelVariation(mv); ok {
					out.Measurements = cloneMeasurements(apparelVariation.Measurements)
				}
			}

			return nil
		}
	}

	switch string(categoryKind) {
	case "alcohol":
		alcoholVariation, ok := toPreviewAlcoholModelVariation(mv)
		if !ok {
			return ErrModelVariationNotFound
		}

		value := alcoholVariation.Volume.Value

		out.ModelKind = categoryKind
		out.ModelNumber = alcoholVariation.ModelNumber
		out.VolumeValue = &value
		out.VolumeUnit = alcoholVariation.Volume.Unit
		out.ModelLabel = buildPreviewModelLabel(
			"alcohol",
			alcoholVariation.ModelNumber,
			"",
			"",
			&value,
			alcoholVariation.Volume.Unit,
		)

		return nil

	default:
		apparelVariation, ok := toPreviewApparelModelVariation(mv)
		if !ok {
			return ErrModelVariationNotFound
		}

		rgb := apparelVariation.Color.RGB

		out.ModelKind = categoryKind
		out.ModelNumber = apparelVariation.ModelNumber
		out.Size = apparelVariation.Size
		out.Color = apparelVariation.Color.Name
		out.RGB = rgb
		out.Measurements = cloneMeasurements(apparelVariation.Measurements)
		out.ModelLabel = buildPreviewModelLabel(
			"apparel",
			apparelVariation.ModelNumber,
			apparelVariation.Size,
			apparelVariation.Color.Name,
			nil,
			"",
		)

		return nil
	}
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

func toPreviewAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
	if v == nil {
		return modeldom.AlcoholModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.AlcoholModelVariation:
		return x, true
	case *modeldom.AlcoholModelVariation:
		if x == nil {
			return modeldom.AlcoholModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.AlcoholModelVariation{}, false
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

	if q.BrandNameIconRepo != nil {
		name, icon, ok, err := q.BrandNameIconRepo.TryGetBrandNameIcon(ctx, brandID)
		if err != nil {
			log.Printf(`[mall.preview] brandNameIcon resolve failed: brandId=%q err=%v`, brandID, err)
		} else if ok {
			*nameOut = name
			*iconOut = icon
			return
		}
	}

	if fallbackName != "" {
		*nameOut = fallbackName
		return
	}

	if q.BrandNameRepo != nil {
		name, ok, err := q.BrandNameRepo.TryGetBrandName(ctx, brandID)
		if err != nil {
			log.Printf(`[mall.preview] brandName resolve failed: brandId=%q err=%v`, brandID, err)
		} else if ok {
			*nameOut = name
		}
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
		name, icon, ok, err := q.AvatarNameIconRepo.TryGetAvatarNameIcon(ctx, avatarID)
		if err != nil {
			log.Printf(`[mall.preview] avatarNameIcon resolve failed: avatarId=%q err=%v`, avatarID, err)
		} else if ok {
			*nameOut = name
			*iconOut = icon
			return
		}
	}

	if fallbackName != "" {
		*nameOut = fallbackName
	}
}
