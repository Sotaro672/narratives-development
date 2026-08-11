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
// Application-specific external ports
// ============================================================

// TokenQuery (assetId -> productId/docId, brandId, metadataUri)
type TokenQuery interface {
	ResolveTokenByAssetID(ctx context.Context, assetID string) (tokendom.ResolveTokenByAssetIDResult, error)
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

// WalletUsecase は Wallet 同期ユースケースです。
//
// IMPORTANT:
// 依存はすべて NewWalletUsecase 経由で注入する。
// struct field は外部から直接差し替えできないように private にする。
type WalletUsecase struct {
	walletRepo    walletdom.Repository
	onchainReader walletdom.OnchainReader
	tokenQuery    TokenQuery

	// brandId -> Brand.Name（UI期待値）
	brandResolver BrandResolver

	// productName 逆引き（UI期待値）
	productReader           ProductReader
	modelProductBlueprintID ModelProductBlueprintIDResolver
	productBlueprintReader  ProductBlueprintReader
}

// NewWalletUsecase is the only wiring entrypoint.
// All dependencies must be routed through this constructor.
func NewWalletUsecase(
	walletRepo walletdom.Repository,
	onchainReader walletdom.OnchainReader,
	tokenQuery TokenQuery,
	brandResolver BrandResolver,
	productReader ProductReader,
	modelProductBlueprintID ModelProductBlueprintIDResolver,
	productBlueprintReader ProductBlueprintReader,
) *WalletUsecase {
	return &WalletUsecase{
		walletRepo:              walletRepo,
		onchainReader:           onchainReader,
		tokenQuery:              tokenQuery,
		brandResolver:           brandResolver,
		productReader:           productReader,
		modelProductBlueprintID: modelProductBlueprintID,
		productBlueprintReader:  productBlueprintReader,
	}
}

var _ OwnedProductResolver = (*WalletUsecase)(nil)

var (
	ErrWalletUsecaseNotConfigured     = errors.New("wallet usecase: not configured")
	ErrWalletSyncAvatarIDEmpty        = errors.New("wallet usecase: avatarID is empty")
	ErrWalletSyncOnchainNotConfigured = errors.New("wallet usecase: onchain reader not configured")
	ErrWalletSyncWalletAddressEmpty   = errors.New("wallet usecase: walletAddress is empty")
	ErrWalletAssetIDNotOwned          = errors.New("wallet usecase: assetId is not owned by avatar")

	// TokenQuery
	ErrWalletTokenQueryNotConfigured = errors.New("wallet usecase: token query not configured")
	ErrAssetIDEmpty                  = errors.New("wallet usecase: assetId is empty")

	// BrandResolver
	ErrWalletBrandResolverNotConfigured = errors.New("wallet usecase: brand resolver not configured")

	// ProductName chain
	ErrWalletProductReaderNotConfigured          = errors.New("wallet usecase: product reader not configured")
	ErrWalletModelProductBlueprintNotConfigured  = errors.New("wallet usecase: model->productBlueprint resolver not configured")
	ErrWalletProductBlueprintReaderNotConfigured = errors.New("wallet usecase: productBlueprint reader not configured")
	ErrWalletResolvedModelIDEmpty                = errors.New("wallet usecase: resolved modelId is empty")
	ErrWalletResolvedProductBlueprintIDEmpty     = errors.New("wallet usecase: resolved productBlueprintId is empty")
)

// HasOwnedProductBlueprint は avatar が指定 productBlueprint の token を
// on-chain 上で現在保有しているかを判定します.
//
// 判定順:
// 1. avatarId から wallet を取得
// 2. walletAddress から on-chain 保有 assetId 一覧を取得
// 3. assetId -> token.productId
// 4. productId -> product.modelId
// 5. modelId -> productBlueprintId
// 6. 指定 productBlueprintId と一致するものがあれば true
//
// assetId ごとの token / product / model 逆引き失敗は、既存の verified purchase
// 判定と同じくその asset をスキップします。
func (uc *WalletUsecase) HasOwnedProductBlueprint(
	ctx context.Context,
	avatarID string,
	productBlueprintID string,
) (bool, error) {
	if uc == nil || uc.walletRepo == nil {
		return false, ErrWalletUsecaseNotConfigured
	}
	if uc.onchainReader == nil {
		return false, ErrWalletSyncOnchainNotConfigured
	}
	if uc.tokenQuery == nil {
		return false, ErrWalletTokenQueryNotConfigured
	}
	if uc.productReader == nil {
		return false, ErrWalletProductReaderNotConfigured
	}
	if uc.modelProductBlueprintID == nil {
		return false, ErrWalletModelProductBlueprintNotConfigured
	}

	if avatarID == "" {
		return false, ErrWalletSyncAvatarIDEmpty
	}
	if productBlueprintID == "" {
		return false, productdom.ErrInvalidID
	}

	w, err := uc.walletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return false, err
	}

	if w.WalletAddress == "" {
		return false, ErrWalletSyncWalletAddressEmpty
	}

	assetIDs, err := uc.onchainReader.ListOwnedAssetIDs(ctx, w.WalletAddress)
	if err != nil {
		return false, err
	}
	if len(assetIDs) == 0 {
		return false, nil
	}

	for _, assetID := range assetIDs {
		if assetID == "" {
			continue
		}

		resolvedToken, err := uc.tokenQuery.ResolveTokenByAssetID(ctx, assetID)
		if err != nil {
			continue
		}

		productID := resolvedToken.ProductID
		if productID == "" {
			continue
		}

		product, err := uc.productReader.GetByID(ctx, productID)
		if err != nil {
			continue
		}

		modelID := product.ModelID
		if modelID == "" {
			continue
		}

		resolvedProductBlueprintID, _, err := uc.modelProductBlueprintID.GetIDByModelID(ctx, modelID)
		if err != nil {
			continue
		}
		if resolvedProductBlueprintID == "" {
			continue
		}

		if resolvedProductBlueprintID == productBlueprintID {
			return true, nil
		}
	}

	return false, nil
}

// GetWalletByAvatarIDWithReadThroughSync は avatarId から wallet を取得し、
// persisted wallet.assetIds と on-chain 保有 assetId 一覧に差分があれば同期して返します。
//
// IMPORTANT:
// on-chain reader 未設定、walletAddress 空、on-chain 取得失敗、sync 失敗の場合は、
// 既存の GET /mall/me/wallets 挙動に合わせて persisted wallet を返します。
func (uc *WalletUsecase) GetWalletByAvatarIDWithReadThroughSync(
	ctx context.Context,
	avatarID string,
) (walletdom.Wallet, error) {
	if uc == nil || uc.walletRepo == nil {
		return walletdom.Wallet{}, ErrWalletUsecaseNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return walletdom.Wallet{}, ErrWalletSyncAvatarIDEmpty
	}

	w, err := uc.walletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	if uc.onchainReader == nil {
		return w, nil
	}

	addr := w.WalletAddress
	if addr == "" {
		return w, nil
	}

	onchainAssetIDs, err := uc.onchainReader.ListOwnedAssetIDs(ctx, addr)
	if err != nil {
		return w, nil
	}

	walletAssetIDCounts := make(map[string]int, len(w.AssetIDs))
	onchainAssetIDCounts := make(map[string]int, len(onchainAssetIDs))

	for _, assetID := range w.AssetIDs {
		if assetID == "" {
			continue
		}
		walletAssetIDCounts[assetID]++
	}

	for _, assetID := range onchainAssetIDs {
		if assetID == "" {
			continue
		}
		onchainAssetIDCounts[assetID]++
	}

	same := len(walletAssetIDCounts) == len(onchainAssetIDCounts)
	if same {
		for assetID, count := range walletAssetIDCounts {
			if onchainAssetIDCounts[assetID] != count {
				same = false
				break
			}
		}
	}

	if same {
		return w, nil
	}

	synced, err := uc.SyncWalletAssetIDs(ctx, aid)
	if err != nil {
		return w, nil
	}

	return synced, nil
}

// ListOwnedAssetIDs は walletAddress から on-chain 保有 assetId 一覧を取得します.
//
// Handler など外側の層は onchainReader に直接触らず、この method 経由で取得する。
func (uc *WalletUsecase) ListOwnedAssetIDs(
	ctx context.Context,
	walletAddress string,
) ([]string, error) {
	if uc == nil {
		return nil, ErrWalletUsecaseNotConfigured
	}
	if uc.onchainReader == nil {
		return nil, ErrWalletSyncOnchainNotConfigured
	}

	addr := walletAddress
	if addr == "" {
		return nil, ErrWalletSyncWalletAddressEmpty
	}

	return uc.onchainReader.ListOwnedAssetIDs(ctx, addr)
}

// SyncWalletAssetIDs:
// - on-chain の最新保有一覧で wallet.assetIds を完全同期する
// - 既存 assetIds との merge はしない
//
// IMPORTANT:
// この同期処理は必ず残す。
// WalletPage を開いた時や /mall/me/wallets/sync から呼ばれ、
// Solana network 上の保有 assetId 一覧を Firestore wallet.assetIds に反映する。
func (uc *WalletUsecase) SyncWalletAssetIDs(
	ctx context.Context,
	avatarID string,
) (walletdom.Wallet, error) {
	if uc == nil || uc.walletRepo == nil {
		return walletdom.Wallet{}, ErrWalletUsecaseNotConfigured
	}
	if uc.onchainReader == nil {
		return walletdom.Wallet{}, ErrWalletSyncOnchainNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return walletdom.Wallet{}, ErrWalletSyncAvatarIDEmpty
	}

	// 1) docId=avatarId で wallet を取得（存在が前提）
	w, err := uc.walletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	addr := w.WalletAddress
	if addr == "" {
		return walletdom.Wallet{}, ErrWalletSyncWalletAddressEmpty
	}

	// 2) on-chain から現在の保有 assetId 一覧を取得
	assetIDs, err := uc.onchainReader.ListOwnedAssetIDs(ctx, addr)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	// 3) on-chain の最新一覧で完全置換
	now := time.Now().UTC()
	if err := w.ReplaceAssetIDs(assetIDs, now); err != nil {
		return walletdom.Wallet{}, err
	}

	if err := uc.walletRepo.Save(ctx, aid, w); err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}

// EnsureAvatarOwnsAssetID は avatar が assetId を現在保有していることを確認します.
//
// IMPORTANT:
// - Firestore wallet.assetIds は同期済み read model / cache としてのみ扱います。
// - 所有権判定では persisted wallet.assetIds を信用しません。
// - current ownership は onchainReader -> Bubblegum service -> DAS を正とします。
//
// 判定順:
// 1. avatarId から wallet を取得
// 2. walletAddress から on-chain 保有 assetId 一覧を取得
// 3. 指定 assetId が現在の保有一覧に含まれるか判定
func (uc *WalletUsecase) EnsureAvatarOwnsAssetID(
	ctx context.Context,
	avatarID string,
	assetID string,
) error {
	if uc == nil || uc.walletRepo == nil {
		return ErrWalletUsecaseNotConfigured
	}

	if uc.onchainReader == nil {
		return ErrWalletSyncOnchainNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return ErrWalletSyncAvatarIDEmpty
	}

	if assetID == "" {
		return ErrAssetIDEmpty
	}

	w, err := uc.walletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return err
	}

	addr := w.WalletAddress
	if addr == "" {
		return ErrWalletSyncWalletAddressEmpty
	}

	assetIDs, err := uc.onchainReader.ListOwnedAssetIDs(ctx, addr)
	if err != nil {
		return err
	}

	for _, ownedAssetID := range assetIDs {
		if ownedAssetID == assetID {
			return nil
		}
	}

	return ErrWalletAssetIDNotOwned
}

// ResolveOwnedTokenByAssetIDWithBrandName は avatar の assetId 所有確認後、
// token / product / brand 表示情報を解決します。
func (uc *WalletUsecase) ResolveOwnedTokenByAssetIDWithBrandName(
	ctx context.Context,
	avatarID string,
	assetID string,
) (ResolveTokenByAssetIDWithBrandNameResult, error) {
	if err := uc.EnsureAvatarOwnsAssetID(ctx, avatarID, assetID); err != nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, err
	}

	return uc.ResolveTokenByAssetIDWithBrandName(ctx, assetID)
}

// ============================================================
// ResolveTokenByAssetID
// ============================================================
//
// assetId を受け取り、Firestore tokens を逆引きして
// productId(docId), brandId, metadataUri を返す。
func (uc *WalletUsecase) ResolveTokenByAssetID(
	ctx context.Context,
	assetID string,
) (tokendom.ResolveTokenByAssetIDResult, error) {
	if uc == nil {
		return tokendom.ResolveTokenByAssetIDResult{}, ErrWalletUsecaseNotConfigured
	}
	if uc.tokenQuery == nil {
		return tokendom.ResolveTokenByAssetIDResult{}, ErrWalletTokenQueryNotConfigured
	}

	if assetID == "" {
		return tokendom.ResolveTokenByAssetIDResult{}, ErrAssetIDEmpty
	}

	return uc.tokenQuery.ResolveTokenByAssetID(ctx, assetID)
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
	if uc.brandResolver == nil {
		return "", ErrWalletBrandResolverNotConfigured
	}

	bid := brandID
	if bid == "" {
		return "", branddom.ErrInvalidID
	}

	b, err := uc.brandResolver.GetByID(ctx, bid)
	if err != nil {
		return "", err
	}

	return b.Name, nil
}

// ============================================================
// Result for mall resolve
// ============================================================

type ResolveTokenByAssetIDWithBrandNameResult struct {
	ProductID          string `json:"productId"`
	BrandID            string `json:"brandId"`
	BrandName          string `json:"brandName"`
	MetadataURI        string `json:"metadataUri"`
	AssetID            string `json:"assetId"`
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
}

// ============================================================
// ResolveTokenByAssetIDWithBrandName
//
//	assetId -> (productId, brandId, brandName, metadataUri, productName)
//
// IMPORTANT:
//   - metadata proxy は廃止しない
//   - frontend は metadataUri を /mall/me/wallets/metadata/proxy に渡して
//     blockchain token metadata を取得する
//   - 画像・ファイル表示は metadata.properties.files[] を正とする
//   - Firestore productBlueprint.contentFiles / Firebase Storage URL は表示元として使わない
//
// ============================================================

func (uc *WalletUsecase) ResolveTokenByAssetIDWithBrandName(
	ctx context.Context,
	assetID string,
) (ResolveTokenByAssetIDWithBrandNameResult, error) {
	if uc == nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletUsecaseNotConfigured
	}

	// 1) token reverse lookup
	base, err := uc.ResolveTokenByAssetID(ctx, assetID)
	if err != nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, err
	}

	productID := base.ProductID
	brandID := base.BrandID

	// 2) brandName
	brandName := ""
	if brandID != "" {
		if uc.brandResolver == nil {
			return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletBrandResolverNotConfigured
		}

		n, err := uc.ResolveBrandNameByID(ctx, brandID)
		if err != nil {
			return ResolveTokenByAssetIDWithBrandNameResult{}, err
		}
		brandName = n
	}

	// 3) productId -> modelId
	if uc.productReader == nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletProductReaderNotConfigured
	}

	p, err := uc.productReader.GetByID(ctx, productID)
	if err != nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, err
	}

	modelID := p.ModelID
	if modelID == "" {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletResolvedModelIDEmpty
	}

	// 4) modelId -> productBlueprintId
	if uc.modelProductBlueprintID == nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletModelProductBlueprintNotConfigured
	}

	pbID, _, err := uc.modelProductBlueprintID.GetIDByModelID(ctx, modelID)
	if err != nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, err
	}
	if pbID == "" {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletResolvedProductBlueprintIDEmpty
	}

	// 5) productBlueprintId -> productName
	if uc.productBlueprintReader == nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, ErrWalletProductBlueprintReaderNotConfigured
	}

	pb, err := uc.productBlueprintReader.GetByID(ctx, pbID)
	if err != nil {
		return ResolveTokenByAssetIDWithBrandNameResult{}, err
	}

	productName := pb.ProductName

	return ResolveTokenByAssetIDWithBrandNameResult{
		ProductID:          productID,
		BrandID:            brandID,
		BrandName:          brandName,
		MetadataURI:        base.MetadataURI,
		AssetID:            base.AssetID,
		ProductBlueprintID: pbID,
		ProductName:        productName,
	}, nil
}
