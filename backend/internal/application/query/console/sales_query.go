// backend/internal/application/query/console/sales_query.go
package query

import (
	"context"
	"errors"

	sharedquery "narratives/internal/application/query/shared"
	avatastatedom "narratives/internal/domain/avatarState"
	branddom "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	walletdom "narratives/internal/domain/wallet"
)

type SalesOwner struct {
	AvatarID       string `json:"avatarId"`
	AvatarName     string `json:"avatarName"`
	AvatarIcon     string `json:"avatarIcon"`
	FollowerCount  int64  `json:"followerCount"`
	FollowingCount int64  `json:"followingCount"`
	PostCount      int64  `json:"postCount"`
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

type tokenResolveByMintReader interface {
	ResolveTokenByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (tokendom.ResolveTokenByMintAddressResult, error)
}

type productReader interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
}

type productBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []pbdom.ModelRef, error)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
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

type avatarNameAndIconReader interface {
	GetNameAndIconByID(ctx context.Context, id string) (name string, icon string, err error)
}

type avatarStateReader interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatastatedom.AvatarState, error)
}

type SalesQuery struct {
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            brandReader
	mintRepo             mintListByTokenBlueprintReader
	tokenQueryRepo       tokenResolveByMintReader
	productRepo          productReader
	productBlueprintRepo productBlueprintReader
	walletRepo           walletAddressByMintReader
	ownerResolver        ownerResolveReader
	avatarRepo           avatarNameAndIconReader
	avatarStateRepo      avatarStateReader
}

func NewSalesQuery(
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo brandReader,
	mintRepo mintListByTokenBlueprintReader,
	tokenQueryRepo tokenResolveByMintReader,
	productRepo productReader,
	productBlueprintRepo productBlueprintReader,
	walletRepo walletAddressByMintReader,
	ownerResolver ownerResolveReader,
	avatarRepo avatarNameAndIconReader,
	avatarStateRepo avatarStateReader,
) *SalesQuery {
	return &SalesQuery{
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
		mintRepo:             mintRepo,
		tokenQueryRepo:       tokenQueryRepo,
		productRepo:          productRepo,
		productBlueprintRepo: productBlueprintRepo,
		walletRepo:           walletRepo,
		ownerResolver:        ownerResolver,
		avatarRepo:           avatarRepo,
		avatarStateRepo:      avatarStateRepo,
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
	if q.tokenQueryRepo == nil {
		return SalesQueryResult{}, errors.New("tokenQueryRepo is nil")
	}
	if q.productRepo == nil {
		return SalesQueryResult{}, errors.New("productRepo is nil")
	}
	if q.productBlueprintRepo == nil {
		return SalesQueryResult{}, errors.New("productBlueprintRepo is nil")
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
	if q.avatarStateRepo == nil {
		return SalesQueryResult{}, errors.New("avatarStateRepo is nil")
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

	modelIDs := make([]string, 0, len(mintAddresses))
	seenModelIDs := make(map[string]struct{}, len(mintAddresses))

	productBlueprints := make([]SalesProductBlueprint, 0, len(mintAddresses))
	seenProductBlueprintIDs := make(map[string]struct{}, len(mintAddresses))

	for _, mintAddress := range mintAddresses {
		if mintAddress == "" {
			continue
		}

		tokenResult, err := q.tokenQueryRepo.ResolveTokenByMintAddress(ctx, mintAddress)
		if err != nil {
			if errors.Is(err, tokendom.ErrNotFound) {
				continue
			}
			return nil, nil, err
		}

		productID := tokenResult.ProductID
		if productID == "" {
			continue
		}

		product, err := q.productRepo.GetByID(ctx, productID)
		if err != nil {
			if errors.Is(err, productdom.ErrNotFound) {
				continue
			}
			return nil, nil, err
		}

		modelID := product.ModelID
		if modelID == "" {
			continue
		}

		if _, ok := seenModelIDs[modelID]; !ok {
			seenModelIDs[modelID] = struct{}{}
			modelIDs = append(modelIDs, modelID)
		}

		productBlueprintID, _, err := q.productBlueprintRepo.GetIDByModelID(ctx, modelID)
		if err != nil {
			continue
		}
		if productBlueprintID == "" {
			continue
		}
		if _, ok := seenProductBlueprintIDs[productBlueprintID]; ok {
			continue
		}

		productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
		if err != nil {
			continue
		}

		seenProductBlueprintIDs[productBlueprintID] = struct{}{}
		productBlueprints = append(productBlueprints, SalesProductBlueprint{
			ProductBlueprintID: productBlueprintID,
			ProductName:        productBlueprint.ProductName,
		})
	}

	return modelIDs, productBlueprints, nil
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

		avatarName, avatarIcon, err := q.avatarRepo.GetNameAndIconByID(ctx, owner.AvatarID)
		if err != nil {
			return nil, err
		}

		avatarState, err := q.avatarStateRepo.GetByAvatarID(ctx, owner.AvatarID)
		if err != nil {
			if !errors.Is(err, avatastatedom.ErrNotFound) {
				return nil, err
			}
		}

		seen[owner.AvatarID] = struct{}{}
		result = append(result, SalesOwner{
			AvatarID:       owner.AvatarID,
			AvatarName:     avatarName,
			AvatarIcon:     avatarIcon,
			FollowerCount:  int64Value(avatarState.FollowerCount),
			FollowingCount: int64Value(avatarState.FollowingCount),
			PostCount:      int64Value(avatarState.PostCount),
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

func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
