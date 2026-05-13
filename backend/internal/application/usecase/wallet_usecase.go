// backend/internal/application/usecase/wallet_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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

// BrandNameResolver (brandId -> brandName)
// - domain/brand の Service.GetNameByID を使う想定
type BrandNameResolver interface {
	GetNameByID(ctx context.Context, brandID string) (string, error)
}

// ProductReader (productId -> product(modelId取得))
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ModelProductBlueprintIDResolver (modelId -> productBlueprintId)
// - port は GetIDByModelID に統一する（productBlueprint.Repository に寄せる）
type ModelProductBlueprintIDResolver interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, error)
}

// ProductBlueprintReader (productBlueprintId -> productBlueprint(productName/contentFiles取得))
type ProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

// WalletUsecase は Wallet 同期ユースケース
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader // 必須（同期APIとして使うなら）
	TokenQuery    TokenQuery          // mint -> token逆引き

	// brandId -> brandName（UI期待値）
	BrandNameResolver BrandNameResolver

	// productName / token content files 逆引き（UI期待値）
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

// TokenQuery を後から差し込むためのセッター
func (uc *WalletUsecase) WithTokenQuery(q TokenQuery) *WalletUsecase {
	if uc != nil {
		uc.TokenQuery = q
	}
	return uc
}

// BrandNameResolver を後から差し込むためのセッター
func (uc *WalletUsecase) WithBrandNameResolver(r BrandNameResolver) *WalletUsecase {
	if uc != nil {
		uc.BrandNameResolver = r
	}
	return uc
}

// ProductReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithProductReader(r ProductReader) *WalletUsecase {
	if uc != nil {
		uc.ProductReader = r
	}
	return uc
}

// ModelProductBlueprintIDResolver を後から差し込むためのセッター
func (uc *WalletUsecase) WithModelProductBlueprintIDResolver(r ModelProductBlueprintIDResolver) *WalletUsecase {
	if uc != nil {
		uc.ModelProductBlueprintID = r
	}
	return uc
}

// ProductBlueprintReader を後から差し込むためのセッター
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

	// TokenQuery
	ErrWalletTokenQueryNotConfigured = errors.New("wallet usecase: token query not configured")
	ErrMintAddressEmpty              = errors.New("wallet usecase: mintAddress is empty")

	// BrandNameResolver
	ErrWalletBrandNameNotConfigured = errors.New("wallet usecase: brand name resolver not configured")

	// ProductName / ProductBlueprint chain
	ErrWalletProductReaderNotConfigured          = errors.New("wallet usecase: product reader not configured")
	ErrWalletModelProductBlueprintNotConfigured  = errors.New("wallet usecase: model->productBlueprint resolver not configured")
	ErrWalletProductBlueprintReaderNotConfigured = errors.New("wallet usecase: productBlueprint reader not configured")
	ErrWalletResolvedModelIDEmpty                = errors.New("wallet usecase: resolved modelId is empty")
	ErrWalletResolvedProductBlueprintIDEmpty     = errors.New("wallet usecase: resolved productBlueprintId is empty")
)

// SyncWalletTokens:
// - on-chain の最新保有一覧で wallet.tokens を完全同期する
// - 既存 tokens との merge はしない
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	log.Printf("[SyncWalletTokens] start avatarID_raw=%q", avatarID)

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
	log.Printf("[SyncWalletTokens] avatarID=%q", aid)

	// 1) docId=avatarId で wallet を取得（存在が前提）
	w, err := uc.WalletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	addr := w.WalletAddress
	if addr == "" {
		return walletdom.Wallet{}, ErrWalletSyncWalletAddressEmpty
	}
	log.Printf("[SyncWalletTokens] wallet loaded avatarID=%q walletAddress=%q tokens_before=%s", aid, addr, walletTokensCountSummary(w))

	// 2) on-chain から現在の保有 mint 一覧を取得
	mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, addr)
	if err != nil {
		return walletdom.Wallet{}, err
	}
	log.Printf("[SyncWalletTokens] onchain mints fetched walletAddress=%q mints_count=%d mints_sample=%s", addr, len(mints), summarizeStringsAbbrev(mints, 10))

	// 3) on-chain の最新一覧で完全置換
	now := time.Now().UTC()
	log.Printf(
		"[SyncWalletTokens] ReplaceTokens input avatarID=%q walletAddress=%q now=%s existing_count=%d existing_sample=%s onchain_count=%d onchain_sample=%s tokens_before=%s",
		aid,
		addr,
		now.Format(time.RFC3339Nano),
		len(w.Tokens),
		summarizeStringsAbbrev(w.Tokens, 10),
		len(mints),
		summarizeStringsAbbrev(mints, 10),
		walletTokensCountSummary(w),
	)

	if err := w.ReplaceTokens(mints, now); err != nil {
		log.Printf("[SyncWalletTokens] ReplaceTokens error avatarID=%q walletAddress=%q err=%v", aid, addr, err)
		return walletdom.Wallet{}, err
	}

	log.Printf("[SyncWalletTokens] ReplaceTokens ok avatarID=%q walletAddress=%q tokens_after=%s", aid, addr, walletTokensCountSummary(w))

	if err := uc.WalletRepo.Save(ctx, aid, w); err != nil {
		return walletdom.Wallet{}, err
	}
	log.Printf("[SyncWalletTokens] saved avatarID=%q walletAddress=%q", aid, addr)

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

	bid := brandID
	if bid == "" {
		return "", branddom.ErrInvalidID
	}

	name, err := uc.BrandNameResolver.GetNameByID(ctx, bid)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ============================================================
// Result for mall resolve
// ============================================================

type TokenContentFile struct {
	// Firebase Storage / metadata properties files 由来
	FileName string `json:"fileName"`
	Type     string `json:"type"`
	URI      string `json:"uri"`

	// 互換用。
	// 旧実装では signed URL を viewUri として返していた。
	// GCS 廃止後は Firebase Storage downloadURL を viewUri として返す。
	ViewURI string `json:"viewUri"`

	// 旧 GCS 実装との互換用。
	// Firebase Storage 前提では Bucket/PublicURI/ViewExpiresAt は基本的に空。
	ObjectPath    string     `json:"objectPath,omitempty"`
	Bucket        string     `json:"bucket,omitempty"`
	PublicURI     string     `json:"publicUri,omitempty"`
	ViewExpiresAt *time.Time `json:"viewExpiresAt,omitempty"`
}

type ResolveTokenByMintAddressWithBrandNameResult struct {
	ProductID          string `json:"productId"`
	BrandID            string `json:"brandId"`
	BrandName          string `json:"brandName"`
	MetadataURI        string `json:"metadataUri"`
	MintAddress        string `json:"mintAddress"`
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`

	// productBlueprintId と同一値を基本とする
	TokenBlueprintID string `json:"tokenBlueprintId"`

	// Firestore productBlueprint.contentFiles[] から抽出した Firebase Storage downloadURL 群
	TokenContentsFiles []TokenContentFile `json:"tokenContentsFiles"`
}

// ============================================================
// ResolveTokenByMintAddressWithBrandName
//
//	mintAddress -> (productId, brandId, brandName, metadataUri, productName)
//	+ productBlueprint.contentFiles[] から tokenContentsFiles を返す
//
// GCS 廃止後:
//   - token-contents bucket は列挙しない
//   - GCS Signed URL は発行しない
//   - GCS_SIGNER_EMAIL は使わない
//   - metadata.properties.files[] ではなく Firestore productBlueprint.contentFiles[] を表示用ファイルの正とする
//   - contentFiles[].url は Firebase Storage downloadURL として ViewURI / URI に入れて返す
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
		if uc.BrandNameResolver == nil {
			return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletBrandNameNotConfigured
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
	pbID, err := uc.ModelProductBlueprintID.GetIDByModelID(ctx, modelID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	if pbID == "" {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletResolvedProductBlueprintIDEmpty
	}

	// 5) productBlueprintId -> productName + contentFiles
	if uc.ProductBlueprintReader == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletProductBlueprintReaderNotConfigured
	}
	pb, err := uc.ProductBlueprintReader.GetByID(ctx, pbID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	productName := pb.ProductName

	// 6) Firestore productBlueprint.contentFiles[] から Firebase Storage downloadURL を抽出する
	files, err := resolveTokenContentsFromProductBlueprint(pb)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	return ResolveTokenByMintAddressWithBrandNameResult{
		ProductID:          productID,
		BrandID:            brandID,
		BrandName:          brandName,
		MetadataURI:        base.MetadataURI,
		MintAddress:        base.MintAddress,
		ProductBlueprintID: pbID,
		ProductName:        productName,

		TokenBlueprintID:   pbID,
		TokenContentsFiles: files,
	}, nil
}

// ---------------------------
// log helpers
// ---------------------------

func abbrev(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:6] + "..." + s[len(s)-6:]
}

func summarizeStringsAbbrev(ss []string, max int) string {
	if len(ss) == 0 {
		return "[]"
	}
	if max <= 0 {
		max = 10
	}
	n := len(ss)
	limit := n
	if n > max {
		limit = max
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, abbrev(ss[i]))
	}
	if n <= max {
		return "[" + strings.Join(out, ",") + "]"
	}
	return "[" + strings.Join(out, ",") + fmt.Sprintf(",...(+%d)", n-max) + "]"
}

func walletTokensCountSummary(w walletdom.Wallet) string {
	return fmt.Sprintf("Tokens=%d", len(w.Tokens))
}

// ---------------------------
// productBlueprint.contentFiles -> token content urls
// ---------------------------

// productBlueprintContentFilesJSON は productBlueprint.ProductBlueprint を JSON 化して
// contentFiles を抽出するための中間 DTO。
// domain 側の ContentFile 型名に直接依存しないようにする。
type productBlueprintContentFilesJSON struct {
	ContentFiles []productBlueprintContentFileJSON `json:"contentFiles"`
}

type productBlueprintContentFileJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	URI         string `json:"uri"`
	ViewURI     string `json:"viewUri"`
	ObjectPath  string `json:"objectPath"`
	Visibility  string `json:"visibility"`
}

// resolveTokenContentsFromProductBlueprint converts Firestore productBlueprint.contentFiles[]
// to TokenContentFile[].
//
// Expected Firestore shape:
//
//	contentFiles[].name
//	contentFiles[].contentType
//	contentFiles[].url
//	contentFiles[].objectPath
//	contentFiles[].type
//
// url is expected to be Firebase Storage downloadURL.
func resolveTokenContentsFromProductBlueprint(pb productbpdom.ProductBlueprint) ([]TokenContentFile, error) {
	raw, err := json.Marshal(pb)
	if err != nil {
		return nil, fmt.Errorf("marshal productBlueprint for contentFiles: %w", err)
	}

	var dto productBlueprintContentFilesJSON
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("unmarshal productBlueprint contentFiles: %w", err)
	}

	out := make([]TokenContentFile, 0, len(dto.ContentFiles))
	for _, f := range dto.ContentFiles {
		uri := firstNonEmptyString(f.URL, f.URI, f.ViewURI)
		if uri == "" {
			continue
		}

		fileName := firstNonEmptyString(f.Name, f.FileName)
		if fileName == "" {
			fileName = "file"
		}

		if isKeepFileNameOrPath(fileName) || isKeepFileNameOrPath(f.ObjectPath) || isKeepURI(uri) {
			continue
		}

		ct := f.ContentType
		if ct == "" {
			ct = f.Type
		}
		if ct == "" {
			ct = "application/octet-stream"
		}

		out = append(out, TokenContentFile{
			FileName:   fileName,
			Type:       ct,
			URI:        uri,
			ViewURI:    uri,
			ObjectPath: f.ObjectPath,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].FileName == out[j].FileName {
			return out[i].URI < out[j].URI
		}
		return out[i].FileName < out[j].FileName
	})

	return out, nil
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		s := strings.TrimSpace(v)
		if s != "" {
			return s
		}
	}
	return ""
}

func isKeepFileNameOrPath(s string) bool {
	if s == "" {
		return false
	}

	p := strings.TrimSpace(s)
	p = strings.TrimSuffix(p, "/")

	return p == ".keep" ||
		strings.HasSuffix(p, "/.keep") ||
		strings.HasSuffix(p, ".keep")
}

func isKeepURI(raw string) bool {
	if raw == "" {
		return false
	}

	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}

	// query / fragment は .keep 判定には不要なので除去
	if idx := strings.Index(s, "#"); idx >= 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "?"); idx >= 0 {
		s = s[:idx]
	}

	return isKeepFileNameOrPath(s)
}
