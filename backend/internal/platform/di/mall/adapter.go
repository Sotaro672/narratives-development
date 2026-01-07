// backend/internal/platform/di/mall/adapter.go
package mall

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	fs "narratives/internal/adapters/out/firestore"
	companydom "narratives/internal/domain/company"
	inspectiondom "narratives/internal/domain/inspection"
	listdom "narratives/internal/domain/list"
	memdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"

	avatarstate "narratives/internal/domain/avatarState"

	mallquerydto "narratives/internal/application/query/mall/dto"
)

//
// ========================================
// mall CatalogQuery 用アダプタ（型ズレ吸収）
// ========================================
//
// mallquery.InventoryRepository が DTO を返すため、
// outfs.InventoryRepositoryFS を直接渡せない（wrong type for GetByID）。
// → Firestore から DTO を直接 DataTo する実装で吸収する。
//

type mallCatalogInventoryRepoAdapter struct {
	Client *firestore.Client
}

func (a *mallCatalogInventoryRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (*mallquerydto.CatalogInventoryDTO, error) {
	if a == nil || a.Client == nil {
		return nil, errors.New("mallCatalogInventoryRepoAdapter: client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("mallCatalogInventoryRepoAdapter: id is empty")
	}

	// NOTE: コレクション名は一般的な "inventories" を採用。
	// 実データが異なる場合はここだけ差し替えてください。
	snap, err := a.Client.Collection("inventories").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var dto mallquerydto.CatalogInventoryDTO
	if err := snap.DataTo(&dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

//
// mallquery.ProductBlueprintRepository は GetByID が *ProductBlueprint を返す。
// outfs.ProductBlueprintRepositoryFS は値を返すため、ポインタ化する薄いアダプタを挟む。
//

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

//
// ========================================
// auth.BootstrapService 用アダプタ
// ========================================
//

// memdom.Repository → auth.MemberRepository
type authMemberRepoAdapter struct {
	repo memdom.Repository
}

func (a *authMemberRepoAdapter) Save(ctx context.Context, m *memdom.Member) error {
	if m == nil {
		return errors.New("authMemberRepoAdapter.Save: nil member")
	}
	saved, err := a.repo.Save(ctx, *m, nil)
	if err != nil {
		return err
	}
	*m = saved
	return nil
}

func (a *authMemberRepoAdapter) GetByID(ctx context.Context, id string) (*memdom.Member, error) {
	v, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CompanyRepositoryFS → auth.CompanyRepository
type authCompanyRepoAdapter struct {
	repo *fs.CompanyRepositoryFS
}

func (a *authCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	if a.repo == nil || a.repo.Client == nil {
		return "", errors.New("authCompanyRepoAdapter.NewID: repo or client is nil")
	}
	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

func (a *authCompanyRepoAdapter) Save(ctx context.Context, c *companydom.Company) error {
	if c == nil {
		return errors.New("authCompanyRepoAdapter.Save: nil company")
	}
	saved, err := a.repo.Save(ctx, *c, nil)
	if err != nil {
		return err
	}
	*c = saved
	return nil
}

//
// ========================================
// InvitationTokenRepository 用アダプタ
// ========================================
//

type invitationTokenRepoAdapter struct {
	fsRepo *fs.InvitationTokenRepositoryFS
}

func (a *invitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a.fsRepo == nil {
		return memdom.InvitationInfo{}, errors.New("invitationTokenRepoAdapter.ResolveInvitationInfoByToken: fsRepo is nil")
	}

	it, err := a.fsRepo.FindByToken(ctx, token)
	if err != nil {
		return memdom.InvitationInfo{}, err
	}

	return memdom.InvitationInfo{
		MemberID:         it.MemberID,
		CompanyID:        it.CompanyID,
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}, nil
}

func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a.fsRepo == nil {
		return "", errors.New("invitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}
	return a.fsRepo.CreateInvitationToken(ctx, info)
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
// ProductUsecase 用 ProductQueryRepo アダプタ
// ========================================
type productQueryRepoAdapter struct {
	productRepo          *fs.ProductRepositoryFS
	modelRepo            *fs.ModelRepositoryFS
	productionRepo       *fs.ProductionRepositoryFS
	productBlueprintRepo *fs.ProductBlueprintRepositoryFS
}

func (a *productQueryRepoAdapter) GetProductByID(
	ctx context.Context,
	productID string,
) (productdom.Product, error) {
	if a == nil || a.productRepo == nil {
		return productdom.Product{}, errors.New("productQueryRepoAdapter: productRepo is nil")
	}
	return a.productRepo.GetByID(ctx, productID)
}

func (a *productQueryRepoAdapter) GetModelByID(
	ctx context.Context,
	modelID string,
) (modeldom.ModelVariation, error) {
	if a == nil || a.modelRepo == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo is nil")
	}
	mv, err := a.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return modeldom.ModelVariation{}, err
	}
	if mv == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo returned nil model variation")
	}
	return *mv, nil
}

func (a *productQueryRepoAdapter) GetProductionByID(
	ctx context.Context,
	productionID string,
) (interface{}, error) {
	if a == nil || a.productionRepo == nil {
		return nil, errors.New("productQueryRepoAdapter: productionRepo is nil")
	}
	return a.productionRepo.GetByID(ctx, productionID)
}

func (a *productQueryRepoAdapter) GetProductBlueprintByID(
	ctx context.Context,
	bpID string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.productBlueprintRepo == nil {
		return productbpdom.ProductBlueprint{}, errors.New("productQueryRepoAdapter: productBlueprintRepo is nil")
	}
	return a.productBlueprintRepo.GetByID(ctx, bpID)
}

// ========================================
// NameResolver 用 TokenBlueprint アダプタ
// ========================================
type tokenBlueprintNameRepoAdapter struct {
	repo *fs.TokenBlueprintRepositoryFS
}

func (a *tokenBlueprintNameRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (tbdom.TokenBlueprint, error) {
	if a == nil || a.repo == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprintNameRepoAdapter: repo is nil")
	}
	tb, err := a.repo.GetByID(ctx, id)
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
	return a.repo.ListIDsByCompany(ctx, companyID)
}

func (a *pbQueryRepoAdapter) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return a.repo.GetByID(ctx, id)
}

type pbIDsByCompanyAdapter struct {
	repo interface {
		ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
	}
}

func (a *pbIDsByCompanyAdapter) ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error) {
	return a.repo.ListIDsByCompany(ctx, companyID)
}

type pbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func (a *pbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	return a.repo.GetPatchByID(ctx, id)
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
	tb, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	if tb == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprint not found")
	}
	return *tb, nil
}

// ============================================================
// ✅ Adapter: ListRepositoryFS -> usecase.ListPatcher
// ============================================================
type listPatcherAdapter struct {
	repo interface {
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	}
}

func (a *listPatcherAdapter) UpdateImageID(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)
	if listID == "" {
		return listdom.List{}, listdom.ErrNotFound
	}

	patch := listdom.ListPatch{
		ImageID:   &imageID,
		UpdatedAt: &now,
		UpdatedBy: updatedBy,
	}
	return a.repo.Update(ctx, listID, patch)
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
