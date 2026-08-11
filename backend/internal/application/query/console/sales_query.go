// backend\internal\application\query\console\sales_query.go
package query

import (
	"context"
	"errors"

	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	branddom "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	tokendom "narratives/internal/domain/token"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	walletdom "narratives/internal/domain/wallet"
)

type SalesOwner struct {
	AvatarID string `json:"avatarId"`
}

type SalesProductBlueprint struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
}

type SalesRow struct {
	TokenBlueprintID  string                  `json:"tokenBlueprintId"`
	TokenName         string                  `json:"tokenName"`
	BrandID           string                  `json:"brandId"`
	BrandName         string                  `json:"brandName"`
	AssetIDs          []string                `json:"assetIds"`
	ModelIDs          []string                `json:"modelIds"`
	ProductBlueprints []SalesProductBlueprint `json:"productBlueprints"`
	Owners            []SalesOwner            `json:"owners"`
}

type SalesQueryResult struct {
	CompanyID string     `json:"companyId"`
	Rows      []SalesRow `json:"rows"`
}

type assetListByTokenBlueprintReader interface {
	ListAssetIDsByTokenBlueprintID(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokendom.ListAssetIDsByTokenBlueprintIDResult, error)
}

type walletAddressByAssetReader interface {
	GetWalletAddressByAssetID(ctx context.Context, assetID string) (string, error)
}

type ownerResolveReader interface {
	Resolve(ctx context.Context, walletAddress string) (*sharedquery.OwnerResolveResult, error)
}

type brandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type assetProductBlueprintResolver interface {
	ResolveByAssetIDs(
		ctx context.Context,
		assetIDs []string,
	) (appresolver.MintProductBlueprintResolveResult, error)
}

type SalesQuery struct {
	tokenBlueprintRepo            tokenblueprintdom.RepositoryPort
	brandRepo                     brandReader
	assetRepo                     assetListByTokenBlueprintReader
	walletRepo                    walletAddressByAssetReader
	ownerResolver                 ownerResolveReader
	assetProductBlueprintResolver assetProductBlueprintResolver
}

func NewSalesQuery(
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo brandReader,
	assetRepo assetListByTokenBlueprintReader,
	walletRepo walletAddressByAssetReader,
	ownerResolver ownerResolveReader,
	assetProductBlueprintResolver assetProductBlueprintResolver,
) *SalesQuery {
	return &SalesQuery{
		tokenBlueprintRepo:            tokenBlueprintRepo,
		brandRepo:                     brandRepo,
		assetRepo:                     assetRepo,
		walletRepo:                    walletRepo,
		ownerResolver:                 ownerResolver,
		assetProductBlueprintResolver: assetProductBlueprintResolver,
	}
}

func (q *SalesQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
) (SalesQueryResult, error) {
	if q == nil {
		return SalesQueryResult{}, errors.New("sales query is nil")
	}
	if q.tokenBlueprintRepo == nil {
		return SalesQueryResult{}, errors.New("tokenBlueprintRepo is nil")
	}
	if q.brandRepo == nil {
		return SalesQueryResult{}, errors.New("brandRepo is nil")
	}
	if q.assetRepo == nil {
		return SalesQueryResult{}, errors.New("assetRepo is nil")
	}
	if q.walletRepo == nil {
		return SalesQueryResult{}, errors.New("walletRepo is nil")
	}
	if q.ownerResolver == nil {
		return SalesQueryResult{}, errors.New("ownerResolver is nil")
	}
	if q.assetProductBlueprintResolver == nil {
		return SalesQueryResult{}, errors.New("assetProductBlueprintResolver is nil")
	}
	if companyID == "" {
		return SalesQueryResult{}, errors.New("companyID is empty")
	}

	page := common.Page{
		Number:  1,
		PerPage: 1000,
	}

	tokenBlueprints, err := q.tokenBlueprintRepo.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		return SalesQueryResult{}, err
	}

	rows := make([]SalesRow, 0, len(tokenBlueprints.Items))

	for _, tb := range tokenBlueprints.Items {
		if tb.ID == "" {
			continue
		}

		brandName := ""
		if tb.BrandID != "" {
			brand, err := q.brandRepo.GetByID(ctx, tb.BrandID)
			if err != nil {
				return SalesQueryResult{}, err
			}
			brandName = brand.Name
		}

		result, err := q.assetRepo.ListAssetIDsByTokenBlueprintID(ctx, tb.ID)
		if err != nil {
			return SalesQueryResult{}, err
		}

		assetIDs := uniqueStrings(result.AssetIDs)

		modelIDs, productBlueprints, err := q.resolveProductBlueprints(ctx, assetIDs)
		if err != nil {
			return SalesQueryResult{}, err
		}

		owners, err := q.resolveSalesOwners(ctx, assetIDs)
		if err != nil {
			return SalesQueryResult{}, err
		}

		rows = append(rows, SalesRow{
			TokenBlueprintID:  tb.ID,
			TokenName:         tb.Name,
			BrandID:           tb.BrandID,
			BrandName:         brandName,
			AssetIDs:          assetIDs,
			ModelIDs:          modelIDs,
			ProductBlueprints: productBlueprints,
			Owners:            owners,
		})
	}

	return SalesQueryResult{
		CompanyID: companyID,
		Rows:      rows,
	}, nil
}

func (q *SalesQuery) resolveProductBlueprints(
	ctx context.Context,
	assetIDs []string,
) ([]string, []SalesProductBlueprint, error) {
	if len(assetIDs) == 0 {
		return []string{}, []SalesProductBlueprint{}, nil
	}
	if q == nil {
		return nil, nil, errors.New("sales query is nil")
	}
	if q.assetProductBlueprintResolver == nil {
		return nil, nil, errors.New("assetProductBlueprintResolver is nil")
	}

	resolved, err := q.assetProductBlueprintResolver.ResolveByAssetIDs(
		ctx,
		assetIDs,
	)
	if err != nil {
		return nil, nil, err
	}

	productBlueprints := make(
		[]SalesProductBlueprint,
		0,
		len(resolved.ProductBlueprints),
	)

	for _, pb := range resolved.ProductBlueprints {
		if pb.ProductBlueprintID == "" {
			continue
		}

		productBlueprints = append(productBlueprints, SalesProductBlueprint{
			ProductBlueprintID: pb.ProductBlueprintID,
			ProductName:        pb.ProductName,
		})
	}

	return uniqueStrings(resolved.ModelIDs), productBlueprints, nil
}

func (q *SalesQuery) resolveSalesOwners(
	ctx context.Context,
	assetIDs []string,
) ([]SalesOwner, error) {
	if len(assetIDs) == 0 {
		return []SalesOwner{}, nil
	}

	result := make([]SalesOwner, 0, len(assetIDs))
	seen := make(map[string]struct{}, len(assetIDs))

	for _, assetID := range assetIDs {
		if assetID == "" {
			continue
		}

		walletAddress, err := q.walletRepo.GetWalletAddressByAssetID(ctx, assetID)
		if err != nil {
			if errors.Is(err, walletdom.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if walletAddress == "" {
			continue
		}

		owner, err := q.ownerResolver.Resolve(ctx, walletAddress)
		if err != nil {
			if errors.Is(err, sharedquery.ErrOwnerNotFound) {
				continue
			}
			return nil, err
		}
		if owner == nil {
			continue
		}
		if owner.OwnerType != sharedquery.OwnerTypeAvatar {
			continue
		}
		if owner.AvatarID == "" {
			continue
		}
		if _, ok := seen[owner.AvatarID]; ok {
			continue
		}

		seen[owner.AvatarID] = struct{}{}
		result = append(result, SalesOwner{
			AvatarID: owner.AvatarID,
		})
	}

	return result, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		result = append(result, v)
	}

	return result
}
