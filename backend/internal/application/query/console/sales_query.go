// backend/internal/application/query/console/sales_query.go
package query

import (
	"context"
	"errors"

	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	tokendom "narratives/internal/domain/token"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	walletdom "narratives/internal/domain/wallet"
)

type SalesOwner struct {
	AvatarID   string `json:"avatarId"`
	AvatarName string `json:"avatarName"`
	AvatarIcon string `json:"avatarIcon"`
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
	MintAddresses     []string                `json:"mintAddresses"`
	ModelIDs          []string                `json:"modelIds"`
	ProductBlueprints []SalesProductBlueprint `json:"productBlueprints"`
	Owners            []SalesOwner            `json:"owners"`
}

type SalesQueryResult struct {
	CompanyID string     `json:"companyId"`
	Rows      []SalesRow `json:"rows"`
}

type mintListByTokenBlueprintReader interface {
	ListMintAddressesByTokenBlueprintID(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokendom.ListMintAddressesByTokenBlueprintIDResult, error)
}

type walletAddressByMintReader interface {
	GetWalletAddressByMintAddress(ctx context.Context, mintAddress string) (string, error)
}

type ownerResolveReader interface {
	Resolve(ctx context.Context, walletAddress string) (*sharedquery.OwnerResolveResult, error)
}

type brandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type avatarReader interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type mintProductBlueprintResolver interface {
	ResolveByMintAddresses(
		ctx context.Context,
		mintAddresses []string,
	) (appresolver.MintProductBlueprintResolveResult, error)
}

type SalesQuery struct {
	tokenBlueprintRepo           tokenblueprintdom.RepositoryPort
	brandRepo                    brandReader
	mintRepo                     mintListByTokenBlueprintReader
	walletRepo                   walletAddressByMintReader
	ownerResolver                ownerResolveReader
	avatarRepo                   avatarReader
	mintProductBlueprintResolver mintProductBlueprintResolver
}

func NewSalesQuery(
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo brandReader,
	mintRepo mintListByTokenBlueprintReader,
	walletRepo walletAddressByMintReader,
	ownerResolver ownerResolveReader,
	avatarRepo avatarReader,
	mintProductBlueprintResolver mintProductBlueprintResolver,
) *SalesQuery {
	return &SalesQuery{
		tokenBlueprintRepo:           tokenBlueprintRepo,
		brandRepo:                    brandRepo,
		mintRepo:                     mintRepo,
		walletRepo:                   walletRepo,
		ownerResolver:                ownerResolver,
		avatarRepo:                   avatarRepo,
		mintProductBlueprintResolver: mintProductBlueprintResolver,
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
	if q.mintRepo == nil {
		return SalesQueryResult{}, errors.New("mintRepo is nil")
	}
	if q.walletRepo == nil {
		return SalesQueryResult{}, errors.New("walletRepo is nil")
	}
	if q.ownerResolver == nil {
		return SalesQueryResult{}, errors.New("ownerResolver is nil")
	}
	if q.avatarRepo == nil {
		return SalesQueryResult{}, errors.New("avatarRepo is nil")
	}
	if q.mintProductBlueprintResolver == nil {
		return SalesQueryResult{}, errors.New("mintProductBlueprintResolver is nil")
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

		result, err := q.mintRepo.ListMintAddressesByTokenBlueprintID(ctx, tb.ID)
		if err != nil {
			return SalesQueryResult{}, err
		}

		mintAddresses := uniqueStrings(result.MintAddresses)

		modelIDs, productBlueprints, err := q.resolveProductBlueprints(ctx, mintAddresses)
		if err != nil {
			return SalesQueryResult{}, err
		}

		owners, err := q.resolveSalesOwners(ctx, mintAddresses)
		if err != nil {
			return SalesQueryResult{}, err
		}

		rows = append(rows, SalesRow{
			TokenBlueprintID:  tb.ID,
			TokenName:         tb.Name,
			BrandID:           tb.BrandID,
			BrandName:         brandName,
			MintAddresses:     mintAddresses,
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
	mintAddresses []string,
) ([]string, []SalesProductBlueprint, error) {
	if len(mintAddresses) == 0 {
		return []string{}, []SalesProductBlueprint{}, nil
	}
	if q == nil {
		return nil, nil, errors.New("sales query is nil")
	}
	if q.mintProductBlueprintResolver == nil {
		return nil, nil, errors.New("mintProductBlueprintResolver is nil")
	}

	resolved, err := q.mintProductBlueprintResolver.ResolveByMintAddresses(
		ctx,
		mintAddresses,
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
	mintAddresses []string,
) ([]SalesOwner, error) {
	if len(mintAddresses) == 0 {
		return []SalesOwner{}, nil
	}

	result := make([]SalesOwner, 0, len(mintAddresses))
	seen := make(map[string]struct{}, len(mintAddresses))

	for _, mintAddress := range mintAddresses {
		if mintAddress == "" {
			continue
		}

		walletAddress, err := q.walletRepo.GetWalletAddressByMintAddress(ctx, mintAddress)
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

		avatar, err := q.avatarRepo.GetByID(ctx, owner.AvatarID)
		if err != nil {
			return nil, err
		}

		avatarIcon := ""
		if avatar.AvatarIcon != nil {
			avatarIcon = *avatar.AvatarIcon
		}

		seen[owner.AvatarID] = struct{}{}
		result = append(result, SalesOwner{
			AvatarID:   owner.AvatarID,
			AvatarName: avatar.AvatarName,
			AvatarIcon: avatarIcon,
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
