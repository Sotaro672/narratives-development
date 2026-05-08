// backend\internal\application\usecase\wallet_usecase.go
package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/iterator"
)

// ============================================================
// Config: token-contents signed URL (wallet resolve)
// - tokenBlueprint パッケージから切り離し（別ディレクトリ/別パッケージになったため）
// ============================================================

// 今後は GCS_SIGNER_EMAIL のみを使用（user memory 方針）
const envGCSSignerEmail = "GCS_SIGNER_EMAIL"

// TOKEN_CONTENTS_BUCKET が未指定の場合のデフォルト（既存の挙動に合わせる）
const defaultTokenContentsBucket = "narratives-development-token-contents"

// 閲覧用（GET）の署名付きURLの有効期限
const tokenContentsViewSignedURLTTL = 15 * time.Minute

func tokenContentsBucketName() string {
	if v := os.Getenv("TOKEN_CONTENTS_BUCKET"); v != "" {
		return v
	}
	return defaultTokenContentsBucket
}

func gcsSignerEmail() string {
	return os.Getenv(envGCSSignerEmail)
}

// tokenContentsObjectPath returns stable object path.
// - "{tokenBlueprintId}/{fileName}"
func tokenContentsObjectPath(tokenBlueprintID, fileName string) string {
	id := strings.Trim(tokenBlueprintID, "/")
	fn := strings.TrimLeft(fileName, "/")
	if fn == "" {
		fn = "file"
	}
	return id + "/" + fn
}

// stable identifier (private bucket では直接GETはできないが、識別子としては有用)
func gcsObjectPublicURL(bucket, objectPath string) string {
	b := strings.Trim(bucket, "/")
	p := strings.TrimLeft(objectPath, "/")
	if b == "" || p == "" {
		return ""
	}
	return "https://storage.googleapis.com/" + b + "/" + p
}

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

// ProductBlueprintReader (productBlueprintId -> productBlueprint(productName取得))
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

	// productName 逆引き（UI期待値）
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

type SignedTokenContentFile struct {
	// stable keys (Firestoreに保存してOK)
	FileName   string `json:"fileName"`
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"objectPath"` // "{tokenBlueprintId}/{fileName}"

	Type      string `json:"type"`
	PublicURI string `json:"publicUri"`

	// 期限付き（Firestoreには保存しないのが推奨）
	ViewURI       string     `json:"viewUri"`
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

	// 追加（metadata から抽出して返却）
	TokenBlueprintID   string                   `json:"tokenBlueprintId"`
	TokenContentsFiles []SignedTokenContentFile `json:"tokenContentsFiles"`
}

// ============================================================
// ResolveTokenByMintAddressWithBrandName
//
//	mintAddress -> (productId, brandId, brandName, metadataUri, productName)
//	+ metadataUri を取得して tokenBlueprintId と token-contents の署名付きURLを返す
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

	// 5) productBlueprintId -> productName
	if uc.ProductBlueprintReader == nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, ErrWalletProductBlueprintReaderNotConfigured
	}
	pb, err := uc.ProductBlueprintReader.GetByID(ctx, pbID)
	if err != nil {
		return ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	productName := pb.ProductName

	// 6) metadataUri を取得して tokenBlueprintId と token-contents files を抽出し、署名付きURLへ変換
	tbID, signedFiles, err := resolveSignedTokenContentsFromMetadata(ctx, base.MetadataURI)
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

		TokenBlueprintID:   tbID,
		TokenContentsFiles: signedFiles,
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
// metadata -> signed token-contents urls
// ---------------------------

type tokenMetadataJSON struct {
	Attributes []struct {
		TraitType string `json:"trait_type"`
		Value     string `json:"value"`
	} `json:"attributes"`
	Properties struct {
		Files []struct {
			Type string `json:"type"`
			URI  string `json:"uri"`
		} `json:"files"`
	} `json:"properties"`
}

func resolveSignedTokenContentsFromMetadata(ctx context.Context, metadataURI string) (tokenBlueprintID string, files []SignedTokenContentFile, err error) {
	u := metadataURI
	if u == "" {
		return "", nil, fmt.Errorf("metadataUri is empty")
	}

	meta, err := fetchTokenMetadata(ctx, u)
	if err != nil {
		return "", nil, err
	}

	// 1) metadata.attributes から TokenBlueprintID を抽出
	tbID := ""
	for _, a := range meta.Attributes {
		if a.TraitType == "TokenBlueprintID" {
			tbID = a.Value
			break
		}
	}
	if tbID == "" {
		return "", nil, fmt.Errorf("TokenBlueprintID not found in metadata attributes")
	}

	// 2) token-contents bucket を直接列挙して tokenContentsFiles を構築する
	//    （metadata.properties.files の形式やホストに依存しない。GCS実体が正となる）
	bucket := tokenContentsBucketName()
	if bucket == "" {
		return "", nil, fmt.Errorf("token contents bucket is empty (env TOKEN_CONTENTS_BUCKET is required)")
	}

	objs, err := listTokenContentsObjects(ctx, bucket, tbID)
	if err != nil {
		return "", nil, err
	}

	out := make([]SignedTokenContentFile, 0, len(objs))
	for _, o := range objs {
		fileName := o.FileName
		if fileName == "" {
			continue
		}

		viewURL, viewExpiresAt, err := issueTokenContentsViewSignedURL(ctx, tbID, fileName)
		if err != nil {
			return "", nil, fmt.Errorf("issue view signed url failed file=%q: %w", fileName, err)
		}

		objectPath := tokenContentsObjectPath(tbID, fileName)

		ct := o.ContentType
		if ct == "" {
			ct = detectMimeTypeByExt(fileName)
		}

		out = append(out, SignedTokenContentFile{
			FileName:      fileName,
			Bucket:        bucket,
			ObjectPath:    objectPath,
			Type:          ct,
			PublicURI:     gcsObjectPublicURL(bucket, objectPath),
			ViewURI:       viewURL,
			ViewExpiresAt: viewExpiresAt,
		})
	}

	return tbID, out, nil
}

func fetchTokenMetadata(ctx context.Context, metadataURI string) (*tokenMetadataJSON, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", metadataURI, nil)
	if err != nil {
		return nil, fmt.Errorf("create metadata request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("fetch metadata status=%d body=%q", resp.StatusCode, string(b))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var meta tokenMetadataJSON
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return &meta, nil
}

type listedTokenContentObject struct {
	FileName    string
	ContentType string
}

// listTokenContentsObjects lists direct children under "{tokenBlueprintId}/" in token-contents bucket.
// - excludes ".keep"
// - excludes subdirectories via Delimiter "/"
func listTokenContentsObjects(ctx context.Context, bucketName string, tokenBlueprintID string) ([]listedTokenContentObject, error) {
	b := bucketName
	if b == "" {
		return nil, fmt.Errorf("bucket name is empty")
	}

	id := strings.Trim(tokenBlueprintID, "/")
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	defer client.Close()

	prefix := id + "/"
	it := client.Bucket(b).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: "/",
	})

	out := make([]listedTokenContentObject, 0)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list objects failed: %w", err)
		}

		// subdirectory
		if attrs.Prefix != "" {
			continue
		}
		if path.Base(attrs.Name) == ".keep" {
			continue
		}

		// attrs.Name: "{tokenBlueprintId}/{fileName}"
		fn := strings.TrimPrefix(attrs.Name, prefix)
		fn = strings.TrimLeft(fn, "/")
		if fn == "" {
			continue
		}

		out = append(out, listedTokenContentObject{
			FileName:    fn,
			ContentType: attrs.ContentType,
		})
	}

	// stable order
	sort.Slice(out, func(i, j int) bool {
		return out[i].FileName < out[j].FileName
	})

	return out, nil
}

func detectMimeTypeByExt(fileName string) string {
	mt := mime.TypeByExtension(strings.ToLower(path.Ext(fileName)))
	if mt == "" {
		return "application/octet-stream"
	}
	// mime.TypeByExtension may include charset; keep as-is (frontで表示用途のため問題なし)
	return mt
}

// TokenBlueprintContentUsecase と同方式で token-contents の GET 署名付きURLを発行
func issueTokenContentsViewSignedURL(ctx context.Context, tokenBlueprintID string, fileName string) (string, *time.Time, error) {
	bucket := tokenContentsBucketName()
	if bucket == "" {
		return "", nil, fmt.Errorf("token contents bucket is empty (env TOKEN_CONTENTS_BUCKET is required)")
	}

	accessID := gcsSignerEmail()
	if accessID == "" {
		return "", nil, fmt.Errorf("missing %s env (signer service account email)", envGCSSignerEmail)
	}

	// tokenBlueprintId/fileName
	objectPath := tokenContentsObjectPath(tokenBlueprintID, fileName)
	objectPath = strings.TrimLeft(path.Clean("/"+objectPath), "/") // sanitize

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("create iamcredentials service: %w", err)
	}

	signBytes := func(b []byte) ([]byte, error) {
		name := "projects/-/serviceAccounts/" + accessID
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(b),
		}
		resp, err := iamSvc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}

	viewExpires := time.Now().UTC().Add(tokenContentsViewSignedURLTTL)

	// GET は ContentType を設定しない（署名ヘッダに含まれてしまい、ブラウザの素fetch/imgが失敗する）
	viewURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        viewExpires,
	})
	if err != nil {
		return "", nil, fmt.Errorf("sign gcs view url: %w", err)
	}

	return viewURL, &viewExpires, nil
}
