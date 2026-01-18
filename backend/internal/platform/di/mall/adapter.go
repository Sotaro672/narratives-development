// backend/internal/platform/di/mall/adapter.go
package mall

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
	usecase "narratives/internal/application/usecase"

	avatarstate "narratives/internal/domain/avatarState"
	inspectiondom "narratives/internal/domain/inspection"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

//
// ========================================
// mall CatalogQuery 用アダプタ（型ズレ吸収）
// ========================================

type mallCatalogProductBlueprintRepoAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) // value return
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *mallCatalogProductBlueprintRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (*productbpdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("mallCatalogProductBlueprintRepoAdapter: repo is nil")
	}
	v, err := a.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (a *mallCatalogProductBlueprintRepoAdapter) ListIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("mallCatalogProductBlueprintRepoAdapter: repo is nil")
	}
	return a.repo.ListIDsByCompany(ctx, strings.TrimSpace(companyID))
}

// ========================================
// productBlueprint ドメインサービス用アダプタ
// ========================================
type productBlueprintDomainRepoAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *productBlueprintDomainRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInternal
	}
	return a.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (a *productBlueprintDomainRepoAdapter) ListIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if a == nil || a.repo == nil {
		return nil, productbpdom.ErrInternal
	}
	return a.repo.ListIDsByCompany(ctx, strings.TrimSpace(companyID))
}

// ========================================
// inspection 用: products.UpdateInspectionResult アダプタ
// ========================================
type inspectionProductRepoAdapter struct {
	repo interface {
		UpdateInspectionResult(ctx context.Context, productID string, result productdom.InspectionResult) error
	}
}

func (a *inspectionProductRepoAdapter) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result inspectiondom.InspectionResult,
) error {
	if a == nil || a.repo == nil {
		return errors.New("inspectionProductRepoAdapter: repo is nil")
	}
	return a.repo.UpdateInspectionResult(ctx, productID, productdom.InspectionResult(result))
}

// ========================================
// NameResolver 用 TokenBlueprint アダプタ
// ========================================
type tokenBlueprintNameRepoAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
	}
}

func (a *tokenBlueprintNameRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (tbdom.TokenBlueprint, error) {
	if a == nil || a.repo == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprintNameRepoAdapter: repo is nil")
	}
	tb, err := a.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	if tb == nil {
		return tbdom.TokenBlueprint{}, tbdom.ErrNotFound
	}
	return *tb, nil
}

// ========================================
// InventoryQuery 用 TokenBlueprint Patch アダプタ
// ========================================
type tbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
	}
}

func (a *tbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	if a == nil || a.repo == nil {
		return tbdom.Patch{}, errors.New("tbPatchByIDAdapter: repo is nil")
	}
	return a.repo.GetPatchByID(ctx, strings.TrimSpace(id))
}

// ========================================
// Query / List 用アダプタ（container.go から移譲）
// ========================================
type pbQueryRepoAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	}
}

func (a *pbQueryRepoAdapter) ListIDsByCompany(ctx context.Context, companyID string) ([]string, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("pbQueryRepoAdapter: repo is nil")
	}
	return a.repo.ListIDsByCompany(ctx, strings.TrimSpace(companyID))
}

func (a *pbQueryRepoAdapter) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.repo == nil {
		return productbpdom.ProductBlueprint{}, errors.New("pbQueryRepoAdapter: repo is nil")
	}
	return a.repo.GetByID(ctx, strings.TrimSpace(id))
}

type pbIDsByCompanyAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *pbIDsByCompanyAdapter) ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("pbIDsByCompanyAdapter: repo is nil")
	}
	return a.repo.ListIDsByCompany(ctx, strings.TrimSpace(companyID))
}

type pbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func (a *pbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	if a == nil || a.repo == nil {
		return productbpdom.Patch{}, errors.New("pbPatchByIDAdapter: repo is nil")
	}
	return a.repo.GetPatchByID(ctx, strings.TrimSpace(id))
}

type tbGetterAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
	}
}

func (a *tbGetterAdapter) GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error) {
	if a == nil || a.repo == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprint getter adapter is nil")
	}
	tb, err := a.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	if tb == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprint not found")
	}
	return *tb, nil
}

// ============================================================
// PreviewQuery: Firestore “直読み” adapters (moved from container.go)
// ============================================================

// previewProductReaderFS: Firestore -> domain.Product (for PreviewQuery)
type previewProductReaderFS struct {
	fs *firestore.Client
}

func (r previewProductReaderFS) GetByID(ctx context.Context, productID string) (productdom.Product, error) {
	if r.fs == nil {
		return productdom.Product{}, mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return productdom.Product{}, mallquery.ErrInvalidProductID
	}

	doc, err := r.fs.Collection("products").Doc(id).Get(ctx)
	if err != nil {
		return productdom.Product{}, err
	}

	var p productdom.Product
	if err := doc.DataTo(&p); err != nil {
		return productdom.Product{}, err
	}

	p.ID = doc.Ref.ID
	return p, nil
}

// previewProductBlueprintReaderFS: ProductBlueprintReader adapter (for PreviewQuery)
type previewProductBlueprintReaderFS struct {
	fs *firestore.Client
	pb interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func (r previewProductBlueprintReaderFS) GetIDByModelID(ctx context.Context, modelID string) (string, error) {
	if r.fs == nil {
		return "", mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(modelID)
	if id == "" {
		return "", mallquery.ErrInvalidModelID
	}

	snap, err := r.fs.Collection("models").Doc(id).Get(ctx)
	if err != nil {
		return "", err
	}

	data := snap.Data()
	if data == nil {
		return "", nil
	}

	for _, k := range []string{"productBlueprintId", "productBlueprintID", "product_blueprint_id"} {
		if v, ok := data[k]; ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s, nil
				}
			}
		}
	}

	return "", nil
}

func (r previewProductBlueprintReaderFS) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	if r.pb == nil {
		return productbpdom.Patch{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetPatchByID(ctx, id)
}

func (r previewProductBlueprintReaderFS) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	if r.pb == nil {
		return productbpdom.ProductBlueprint{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetByID(ctx, id)
}

// ============================================================
// SharedQuery OwnerResolve: Firestore “直読み” adapters (moved from container.go)
// ============================================================

var (
	errOwnerResolveCollectionEmpty = errors.New("di.mall: owner resolve collection is empty")
)

type brandWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func (r brandWalletAddressReaderFS) FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := strings.TrimSpace(r.col)
	if col == "" {
		return "", errOwnerResolveCollectionEmpty
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return strings.TrimSpace(doc.Ref.ID), nil
}

type avatarWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func (r avatarWalletAddressReaderFS) FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := strings.TrimSpace(r.col)
	if col == "" {
		return "", errOwnerResolveCollectionEmpty
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return strings.TrimSpace(doc.Ref.ID), nil
}

// ============================================================
// Transfer: Firestore “直読み” adapters (moved from container.go)
// ============================================================

var (
	errTokenResolverNotConfigured = errors.New("di.mall: tokenResolverFS not configured")
	errTokenDocNotFound           = errors.New("di.mall: token doc not found")
)

type tokenResolverFS struct {
	fs  *firestore.Client
	col string
}

func (r *tokenResolverFS) ResolveTokenByProductID(ctx context.Context, productID string) (usecase.TokenForTransfer, error) {
	if r == nil || r.fs == nil {
		return usecase.TokenForTransfer{}, errTokenResolverNotConfigured
	}
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return usecase.TokenForTransfer{}, errors.New("tokenResolverFS: productId is empty")
	}
	col := strings.TrimSpace(r.col)
	if col == "" {
		col = "tokens"
	}

	snap, err := r.fs.Collection(col).Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return usecase.TokenForTransfer{}, errTokenDocNotFound
		}
		return usecase.TokenForTransfer{}, err
	}
	raw := snap.Data()
	if raw == nil {
		return usecase.TokenForTransfer{}, errTokenDocNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						return s
					}
				}
			}
		}
		return ""
	}

	return usecase.TokenForTransfer{
		ProductID: pid,
		BrandID:   getStr("brandId", "brandID"),
		MintAddress: getStr(
			"mintAddress",
			"mint_address",
		),
		TokenBlueprintID: getStr(
			"tokenBlueprintId",
			"tokenBlueprintID",
			"token_blueprint_id",
		),
		ToAddress: getStr("toAddress", "to_address"),
	}, nil
}

type tokenOwnerUpdaterFS struct {
	fs  *firestore.Client
	col string
}

func (r *tokenOwnerUpdaterFS) UpdateToAddressByProductID(ctx context.Context, productID string, newToAddress string, now time.Time, txSignature string) error {
	if r == nil || r.fs == nil {
		return errTokenResolverNotConfigured
	}
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return errors.New("tokenOwnerUpdaterFS: productId is empty")
	}
	addr := strings.TrimSpace(newToAddress)
	if addr == "" {
		return errors.New("tokenOwnerUpdaterFS: newToAddress is empty")
	}
	col := strings.TrimSpace(r.col)
	if col == "" {
		col = "tokens"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	ref := r.fs.Collection(col).Doc(pid)

	_, err := ref.Set(ctx, map[string]any{
		"toAddress":       addr,
		"updatedAt":       now,
		"lastTxSignature": strings.TrimSpace(txSignature),
		"ownerUpdatedAt":  now,
	}, firestore.MergeAll)
	return err
}

// ============================================================
// Adapters (DI layer) to absorb signature drift
// ============================================================

// ---- AvatarState adapter ----

type avatarStateGetter interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
}

type avatarStateUpserterV2 interface {
	Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error)
}

type avatarStateUpserterV1 interface {
	Upsert(ctx context.Context, avatarID string) error
}

type avatarStateRepoAdapter struct {
	repo any
}

func (a *avatarStateRepoAdapter) GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	if a == nil || a.repo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}
	g, ok := a.repo.(avatarStateGetter)
	if !ok {
		return avatarstate.AvatarState{}, errors.New("avatarState repo missing GetByAvatarID")
	}
	return g.GetByAvatarID(ctx, avatarID)
}

func (a *avatarStateRepoAdapter) Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if a == nil || a.repo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}

	if v2, ok := a.repo.(avatarStateUpserterV2); ok {
		return v2.Upsert(ctx, s)
	}

	if v1, ok := a.repo.(avatarStateUpserterV1); ok {
		aid := strings.TrimSpace(s.ID)
		if aid == "" {
			return avatarstate.AvatarState{}, errors.New("avatarState upsert: id is empty")
		}
		if err := v1.Upsert(ctx, aid); err != nil {
			return avatarstate.AvatarState{}, err
		}
		return a.GetByAvatarID(ctx, aid)
	}

	return avatarstate.AvatarState{}, errors.New("avatarState repo missing Upsert")
}

// ---- PostImage adapter ----

type postImageIssuerV2 interface {
	IssueSignedUploadURL(ctx context.Context, avatarID, fileName, contentType string, expiresIn time.Duration) (string, string, string, error)
}

type postImageIssuerV1 interface {
	IssueSignedUploadURL(ctx context.Context, avatarID, fileName, contentType string, expiresIn time.Duration) (string, error)
}

type postImageRepoAdapter struct {
	repo any
}

func (a *postImageRepoAdapter) IssueSignedUploadURL(ctx context.Context, avatarID, fileName, contentType string, expiresIn time.Duration) (string, string, string, error) {
	if a == nil || a.repo == nil {
		return "", "", "", errors.New("postImage repo not configured")
	}
	if v2, ok := a.repo.(postImageIssuerV2); ok {
		return v2.IssueSignedUploadURL(ctx, avatarID, fileName, contentType, expiresIn)
	}
	if _, ok := a.repo.(postImageIssuerV1); ok {
		return "", "", "", errors.New("postImage repo has legacy IssueSignedUploadURL signature; expected (uploadURL, publicURL, objectPath, error)")
	}
	return "", "", "", errors.New("postImage repo missing IssueSignedUploadURL")
}
