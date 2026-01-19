// backend/internal/application/usecase/wallet_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"
)

// ✅ usecase が必要とするIFをここで定義する（domain の Repository に依存しない）
type WalletRepository interface {
	// docId=avatarId
	GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error)
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type OnchainWalletReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}

// ✅ TokenQuery (mintAddress -> productId/docId, brandId, metadataUri)
type TokenQuery interface {
	ResolveTokenByMintAddress(ctx context.Context, mintAddress string) (tokendom.ResolveTokenByMintAddressResult, error)
}

// ✅ BrandNameResolver (brandId -> brandName)
// - domain/brand の Service.GetNameByID を使う想定
type BrandNameResolver interface {
	GetNameByID(ctx context.Context, brandID string) (string, error)
}

// ✅ ProductReader (productId -> product(modelId取得))
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ✅ ModelProductBlueprintIDResolver (modelId -> productBlueprintId)
// - models コレクションの productBlueprintId を直読みする想定
type ModelProductBlueprintIDResolver interface {
	GetProductBlueprintIDByModelID(ctx context.Context, modelID string) (string, error)
}

// ✅ ProductBlueprintReader (productBlueprintId -> productBlueprint(productName取得))
type ProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

// WalletUsecase は Wallet 同期ユースケース
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader // 必須（同期APIとして使うなら）
	TokenQuery    TokenQuery          // mint -> token逆引き

	// ✅ brandId -> brandName（UI期待値）
	BrandNameResolver BrandNameResolver

	// ✅ productName 逆引き（UI期待値）
	ProductReader           ProductReader
	ModelProductBlueprintID ModelProductBlueprintIDResolver
	ProductBlueprintReader  ProductBlueprintReader
}

// コンストラクタ（DI コンテナの呼び出しに合わせて 1 引数）
// OnchainReader / TokenQuery / BrandNameResolver / Product* はセッターで差し込む
func NewWalletUsecase(walletRepo WalletRepository) *WalletUsecase {
	return &WalletUsecase{
		WalletRepo:              walletRepo,
		OnchainReader:           nil,
		TokenQuery:              nil,
		BrandNameResolver:       nil,
		ProductReader:           nil,
		ModelProductBlueprintID: nil,
		ProductBlueprintReader:  nil,
	}
}

// 任意: OnchainReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithOnchainReader(r OnchainWalletReader) *WalletUsecase {
	if uc != nil {
		uc.OnchainReader = r
	}
	return uc
}

// ✅ TokenQuery を後から差し込むためのセッター
func (uc *WalletUsecase) WithTokenQuery(q TokenQuery) *WalletUsecase {
	if uc != nil {
		uc.TokenQuery = q
	}
	return uc
}

// ✅ BrandNameResolver を後から差し込むためのセッター
func (uc *WalletUsecase) WithBrandNameResolver(r BrandNameResolver) *WalletUsecase {
	if uc != nil {
		uc.BrandNameResolver = r
	}
	return uc
}

// ✅ ProductReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithProductReader(r ProductReader) *WalletUsecase {
	if uc != nil {
		uc.ProductReader = r
	}
	return uc
}

// ✅ ModelProductBlueprintIDResolver を後から差し込むためのセッター
func (uc *WalletUsecase) WithModelProductBlueprintIDResolver(r ModelProductBlueprintIDResolver) *WalletUsecase {
	if uc != nil {
		uc.ModelProductBlueprintID = r
	}
	return uc
}

// ✅ ProductBlueprintReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithProductBlueprintReader(r ProductBlueprintReader) *WalletUsecase {
	if uc != nil {
		uc.ProductBlueprintReader = r
	}
	return uc
}

var (
	ErrWalletUsecaseNotConfigured     = errors.New("wallet usecase: not configured")
	ErrWalletSyncAvatarIDEmpty        = errors.New("wallet usecase: avatarID is empty")
	ErrWalletSyncOnchainNotConfigured = errors.New("wallet usecase: onchain reader not configured")
	ErrWalletSyncWalletAddressEmpty   = errors.New("wallet usecase: walletAddress is empty")

	// ✅ TokenQuery
	ErrWalletTokenQueryNotConfigured = errors.New("wallet usecase: token query not configured")
	ErrMintAddressEmpty              = errors.New("wallet usecase: mintAddress is empty")

	// ✅ BrandNameResolver
	ErrWalletBrandNameNotConfigured = errors.New("wallet usecase: brand name resolver not configured")

	// ✅ ProductName chain
	ErrWalletProductReaderNotConfigured          = errors.New("wallet usecase: product reader not configured")
	ErrWalletModelProductBlueprintNotConfigured  = errors.New("wallet usecase: model->productBlueprint resolver not configured")
	ErrWalletProductBlueprintReaderNotConfigured = errors.New("wallet usecase: productBlueprint reader not configured")
	ErrWalletResolvedModelIDEmpty                = errors.New("wallet usecase: resolved modelId is empty")
	ErrWalletResolvedProductBlueprintIDEmpty     = errors.New("wallet usecase: resolved productBlueprintId is empty")
)

// SyncWalletTokens: 既存のまま
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	if uc == nil || uc.WalletRepo == nil {
		return walletdom.Wallet{}, ErrWalletUsecaseNotConfigured
	}
	if uc.OnchainReader == nil {
		return walletdom.Wallet{}, ErrWalletSyncOnchainNotConfigured
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return walletdom.Wallet{}, ErrWalletSyncAvatarIDEmpty
	}

	// 1) docId=avatarId で wallet を取得（存在が前提）
	w, err := uc.WalletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return walletdom.Wallet{}, ErrWalletSyncWalletAddressEmpty
	}

	// 2) on-chain から mint 一覧を取得
	mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, addr)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	// 3) 置換して保存
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
// ✅ ResolveTokenByMintAddress
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

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, ErrMintAddressEmpty
	}

	return uc.TokenQuery.ResolveTokenByMintAddress(ctx, m)
}

// ============================================================
// ✅ ResolveBrandNameByID
// ============================================================
func (uc *WalletUsecase) ResolveBrandNameByID(
	ctx context.Context,
	brandID string,
) (string, error) {
	if uc == nil {
		return "", ErrWalletUsecaseNotConfigured
	}
	if uc.BrandNameResolver == nil {
		return "", ErrWalletBrandNameNotConfigured
	}

	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return "", branddom.ErrInvalidID
	}

	name, err := uc.BrandNameResolver.GetNameByID(ctx, bid)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

// ============================================================
// ✅ Result for mall resolve
// ============================================================

type ResolveTokenByMintAddressWithBrandNameResult struct {
	ProductID          string
	BrandID            string
	BrandName          string
	MetadataURI        string
	MintAddress        string
	ProductBlueprintID string
	ProductName        string
}

// ============================================================
// ✅ ResolveTokenByMintAddressWithBrandName
//
//	mintAddress -> (productId, brandId, brandName, metadataUri, productName)
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

	productID := strings.TrimSpace(base.ProductID)
	brandID := strings.TrimSpace(base.BrandID)

	// 2) brandName
	brandName := ""
	if brandID != "" {
		if uc.BrandNameResolver == nil {
			return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletBrandNameNotConfigured
		}
		n, err := uc.ResolveBrandNameByID(ctx, brandID)
		if err != nil {
			return ResolveTokenByMintAddressWithBrandNameResult{}, err
		}
		brandName = strings.TrimSpace(n)
	}

	// 3) productId -> modelId
	if uc.ProductReader == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletProductReaderNotConfigured
	}
	p, err := uc.ProductReader.GetByID(ctx, productID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletResolvedModelIDEmpty
	}

	// 4) modelId -> productBlueprintId
	if uc.ModelProductBlueprintID == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletModelProductBlueprintNotConfigured
	}
	pbID, err := uc.ModelProductBlueprintID.GetProductBlueprintIDByModelID(ctx, modelID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	pbID = strings.TrimSpace(pbID)
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
	productName := strings.TrimSpace(pb.ProductName)

	return ResolveTokenByMintAddressWithBrandNameResult{
		ProductID:          productID,
		BrandID:            brandID,
		BrandName:          brandName,
		MetadataURI:        strings.TrimSpace(base.MetadataURI),
		MintAddress:        strings.TrimSpace(base.MintAddress),
		ProductBlueprintID: pbID,
		ProductName:        productName,
	}, nil
}
