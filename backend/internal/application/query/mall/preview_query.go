// backend\internal\application\query\mall\preview_query.go
package mall

import (
	"context"
	"errors"
	"log"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
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
// We need: modelId(=variationId想定) -> variation -> modelNumber/size/color/rgb/measurements.
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

// ResolveModelMetaByModelID resolves model metadata from modelId.
// model.Color.RGB は int が正なので、戻り値も int で返します。
func (q *PreviewQuery) ResolveModelMetaByModelID(
	ctx context.Context,
	modelID string,
) (modelNumber string, size string, colorName string, rgb int, err error) {
	if q == nil || q.ModelRepo == nil {
		return "", "", "", 0, ErrPreviewQueryNotConfigured
	}

	id := modelID
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

	modelNumber = v.ModelNumber
	size = v.Size
	colorName = v.Color.Name
	rgb = v.Color.RGB

	return modelNumber, size, colorName, rgb, nil
}

// ResolveModelInfoByProductID resolves modelId AND variation fields
// (modelNumber/size/color/rgb/measurements) from productId,
// and additionally resolves productBlueprintId + (productBlueprint entity + patch) by modelId.
// and optionally resolves tokens/{productId} if TokenRepo is configured.
// and optionally resolves tokenBlueprint patch if TokenBlueprintRepo is configured and tokenBlueprintId exists.
// and optionally resolves owner (tokens.toAddress -> avatarId/brandId + (avatarName/brandName)) if OwnerResolveQ is configured.
// and optionally resolves brandId->brandName, companyId->companyName (display-only enrichment) if BrandNameRepo/CompanyNameRepo are configured.
// and optionally resolves transfers (mintAddress -> []transfer) if TransferRepo is configured.
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
	if v == nil {
		return nil, ErrModelVariationNotFound
	}

	out := &dto.PreviewModelInfo{
		ProductID:    id,
		ModelID:      modelID,
		ModelNumber:  v.ModelNumber,
		Size:         v.Size,
		Color:        v.Color.Name,
		RGB:          v.Color.RGB,
		Measurements: cloneMeasurements(v.Measurements),
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

	out.ProductBlueprintID = pbID
	out.ProductBlueprint = &pb
	out.ProductBlueprintPatch = &patch

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
