// backend/internal/application/usecase/wallet_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	branddom "narratives/internal/domain/brand"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"
)

// ============================================================
// Wallet repository / external ports
// ============================================================

// usecase が必要とするIFをここで定義する（domain の Repository に依存しない）
type WalletRepository interface {
	// docId=avatarId
	GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error)
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type OnchainWalletReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}

// TokenQuery (mintAddress -> productId/docId, brandId, metadataUri)
type TokenQuery interface {
	ResolveTokenByMintAddress(ctx context.Context, mintAddress string) (tokendom.ResolveTokenByMintAddressResult, error)
}

// BrandResolver (brandId -> Brand)
//
// brand.RepositoryPort / brand.Repository の GetByID(ctx, id string) に合わせる。
// brand.Service / GetNameByID は使わず、repository の GetByID から Brand.Name を解決する。
type BrandResolver interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// ProductReader (productId -> product(modelId取得))
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ModelProductBlueprintIDResolver (modelId -> productBlueprintId + modelRefs)
//
// repository port の GetIDByModelID に合わせる。
// - productBlueprintID が必要な caller は第1戻り値を使う
// - displayOrder / modelRefs が必要な caller は第2戻り値を使う
type ModelProductBlueprintIDResolver interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []productbpdom.ModelRef, error)
}

// ProductBlueprintReader (productBlueprintId -> productBlueprint(productName取得))
type ProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

// WalletUsecase は Wallet 同期ユースケース
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader
	TokenQuery    TokenQuery

	// brandId -> Brand.Name（UI期待値）
	BrandResolver BrandResolver

	// productName 逆引き（UI期待値）
	ProductReader           ProductReader
	ModelProductBlueprintID ModelProductBlueprintIDResolver
	ProductBlueprintReader  ProductBlueprintReader
}

// NewWalletUsecase is the only wiring entrypoint.
// All dependencies must be routed through this constructor.
func NewWalletUsecase(
	walletRepo WalletRepository,
	onchainReader OnchainWalletReader,
	tokenQuery TokenQuery,
	brandResolver BrandResolver,
	productReader ProductReader,
	modelProductBlueprintID ModelProductBlueprintIDResolver,
	productBlueprintReader ProductBlueprintReader,
) *WalletUsecase {
	return &WalletUsecase{
		WalletRepo:              walletRepo,
		OnchainReader:           onchainReader,
		TokenQuery:              tokenQuery,
		BrandResolver:           brandResolver,
		ProductReader:           productReader,
		ModelProductBlueprintID: modelProductBlueprintID,
		ProductBlueprintReader:  productBlueprintReader,
	}
}

var (
	ErrWalletUsecaseNotConfigured     = errors.New("wallet usecase: not configured")
	ErrWalletSyncAvatarIDEmpty        = errors.New("wallet usecase: avatarID is empty")
	ErrWalletSyncOnchainNotConfigured = errors.New("wallet usecase: onchain reader not configured")
	ErrWalletSyncWalletAddressEmpty   = errors.New("wallet usecase: walletAddress is empty")

	// TokenQuery
	ErrWalletTokenQueryNotConfigured = errors.New("wallet usecase: token query not configured")
	ErrMintAddressEmpty              = errors.New("wallet usecase: mintAddress is empty")

	// BrandResolver
	ErrWalletBrandResolverNotConfigured = errors.New("wallet usecase: brand resolver not configured")

	// ProductName chain
	ErrWalletProductReaderNotConfigured          = errors.New("wallet usecase: product reader not configured")
	ErrWalletModelProductBlueprintNotConfigured  = errors.New("wallet usecase: model->productBlueprint resolver not configured")
	ErrWalletProductBlueprintReaderNotConfigured = errors.New("wallet usecase: productBlueprint reader not configured")
	ErrWalletResolvedModelIDEmpty                = errors.New("wallet usecase: resolved modelId is empty")
	ErrWalletResolvedProductBlueprintIDEmpty     = errors.New("wallet usecase: resolved productBlueprintId is empty")
)

// SyncWalletTokens:
// - on-chain の最新保有一覧で wallet.tokens を完全同期する
// - 既存 tokens との merge はしない
//
// IMPORTANT:
// この同期処理は必ず残す。
// WalletPage を開いた時や /mall/me/wallets/sync から呼ばれ、
// Solana network 上の保有 mint 一覧を Firestore wallet.tokens に反映する。
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	if uc == nil || uc.WalletRepo == nil {
		return walletdom.Wallet{}, ErrWalletUsecaseNotConfigured
	}
	if uc.OnchainReader == nil {
		return walletdom.Wallet{}, ErrWalletSyncOnchainNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return walletdom.Wallet{}, ErrWalletSyncAvatarIDEmpty
	}

	// 1) docId=avatarId で wallet を取得（存在が前提）
	w, err := uc.WalletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	addr := w.WalletAddress
	if addr == "" {
		return walletdom.Wallet{}, ErrWalletSyncWalletAddressEmpty
	}

	// 2) on-chain から現在の保有 mint 一覧を取得
	mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, addr)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	// 3) on-chain の最新一覧で完全置換
	now := time.Now().UTC()
	if err := w.ReplaceTokens(mints, now); err != nil {
		return walletdom.Wallet{}, err
	}

	if err := uc.WalletRepo.Save(ctx, aid, w); err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}

// ============================================================
// ResolveTokenByMintAddress
// ============================================================
//
// mintAddress を受け取り、Firestore tokens を逆引きして
// productId(docId), brandId, metadataUri を返す。
func (uc *WalletUsecase) ResolveTokenByMintAddress(
	ctx context.Context,
	mintAddress string,
) (tokendom.ResolveTokenByMintAddressResult, error) {
	if uc == nil {
		return tokendom.ResolveTokenByMintAddressResult{}, ErrWalletUsecaseNotConfigured
	}
	if uc.TokenQuery == nil {
		return tokendom.ResolveTokenByMintAddressResult{}, ErrWalletTokenQueryNotConfigured
	}

	m := mintAddress
	if m == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, ErrMintAddressEmpty
	}

	return uc.TokenQuery.ResolveTokenByMintAddress(ctx, m)
}

// ============================================================
// ResolveBrandNameByID
// ============================================================
//
// brand.RepositoryPort / brand.Repository の GetByID(ctx, id string) に合わせ、
// Brand.Name を返す。
func (uc *WalletUsecase) ResolveBrandNameByID(
	ctx context.Context,
	brandID string,
) (string, error) {
	if uc == nil {
		return "", ErrWalletUsecaseNotConfigured
	}
	if uc.BrandResolver == nil {
		return "", ErrWalletBrandResolverNotConfigured
	}

	bid := brandID
	if bid == "" {
		return "", branddom.ErrInvalidID
	}

	b, err := uc.BrandResolver.GetByID(ctx, bid)
	if err != nil {
		return "", err
	}

	return b.Name, nil
}

// ============================================================
// Result for mall resolve
// ============================================================

type ResolveTokenByMintAddressWithBrandNameResult struct {
	ProductID          string `json:"productId"`
	BrandID            string `json:"brandId"`
	BrandName          string `json:"brandName"`
	MetadataURI        string `json:"metadataUri"`
	MintAddress        string `json:"mintAddress"`
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
}

// ============================================================
// ResolveTokenByMintAddressWithBrandName
//
//	mintAddress -> (productId, brandId, brandName, metadataUri, productName)
//
// IMPORTANT:
//   - metadata proxy は廃止しない
//   - frontend は metadataUri を /mall/me/wallets/metadata/proxy に渡して
//     blockchain token metadata を取得する
//   - 画像・ファイル表示は metadata.properties.files[] を正とする
//   - Firestore productBlueprint.contentFiles / Firebase Storage URL は表示元として使わない
//
// ============================================================

func (uc *WalletUsecase) ResolveTokenByMintAddressWithBrandName(
	ctx context.Context,
	mintAddress string,
) (ResolveTokenByMintAddressWithBrandNameResult, error) {
	if uc == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletUsecaseNotConfigured
	}

	// 1) token reverse lookup
	base, err := uc.ResolveTokenByMintAddress(ctx, mintAddress)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	productID := base.ProductID
	brandID := base.BrandID

	// 2) brandName
	brandName := ""
	if brandID != "" {
		if uc.BrandResolver == nil {
			return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletBrandResolverNotConfigured
		}
		n, err := uc.ResolveBrandNameByID(ctx, brandID)
		if err != nil {
			return ResolveTokenByMintAddressWithBrandNameResult{}, err
		}
		brandName = n
	}

	// 3) productId -> modelId
	if uc.ProductReader == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletProductReaderNotConfigured
	}
	p, err := uc.ProductReader.GetByID(ctx, productID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	modelID := p.ModelID
	if modelID == "" {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletResolvedModelIDEmpty
	}

	// 4) modelId -> productBlueprintId
	if uc.ModelProductBlueprintID == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletModelProductBlueprintNotConfigured
	}

	pbID, _, err := uc.ModelProductBlueprintID.GetIDByModelID(ctx, modelID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	if pbID == "" {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletResolvedProductBlueprintIDEmpty
	}

	// 5) productBlueprintId -> productName
	if uc.ProductBlueprintReader == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletProductBlueprintReaderNotConfigured
	}

	pb, err := uc.ProductBlueprintReader.GetByID(ctx, pbID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	productName := pb.ProductName

	return ResolveTokenByMintAddressWithBrandNameResult{
		ProductID:          productID,
		BrandID:            brandID,
		BrandName:          brandName,
		MetadataURI:        base.MetadataURI,
		MintAddress:        base.MintAddress,
		ProductBlueprintID: pbID,
		ProductName:        productName,
	}, nil
}
