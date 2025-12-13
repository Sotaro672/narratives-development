// backend/internal/platform/di/adapters.go
package di

import (
	"context"
	"errors"
	"strings"

	fs "narratives/internal/adapters/out/firestore"
	companydom "narratives/internal/domain/company"
	inspectiondom "narratives/internal/domain/inspection"
	memdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

//
// ========================================
// auth.BootstrapService 用アダプタ
// ========================================
//

// memdom.Repository → auth.MemberRepository
type authMemberRepoAdapter struct {
	repo memdom.Repository
}

// Save: *member を memdom.Repository.Save に委譲
func (a *authMemberRepoAdapter) Save(ctx context.Context, m *memdom.Member) error {
	if m == nil {
		return errors.New("authMemberRepoAdapter.Save: nil member")
	}
	saved, err := a.repo.Save(ctx, *m, nil)
	if err != nil {
		return err
	}
	// Save 側で CreatedAt / UpdatedAt などが上書きされた場合に反映しておく
	*m = saved
	return nil
}

// GetByID: 値戻りをポインタに変換
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

// NewID: Firestore の companies コレクションから DocID を採番
func (a *authCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	if a.repo == nil || a.repo.Client == nil {
		return "", errors.New("authCompanyRepoAdapter.NewID: repo or client is nil")
	}
	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

// Save: companydom.Company を CompanyRepositoryFS.Save に委譲
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
// Firestore 実装 (*fs.InvitationTokenRepositoryFS) を
// usecase.InvitationTokenRepository に合わせてラップする。
//   - ResolveInvitationInfoByToken
//   - CreateInvitationToken
//

type invitationTokenRepoAdapter struct {
	fsRepo *fs.InvitationTokenRepositoryFS
}

// ResolveInvitationInfoByToken は token から InvitationInfo を取得します。
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

// CreateInvitationToken は InvitationInfo を受け取り、
// Firestore 側に招待トークンを作成して token 文字列を返します。
func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a.fsRepo == nil {
		return "", errors.New("invitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}
	// FS 実装は既に (ctx, member.InvitationInfo) を受け取るように
	// 変更済みという前提で、そのまま委譲する。
	return a.fsRepo.CreateInvitationToken(ctx, info)
}

// ========================================
// productBlueprint ドメインサービス用アダプタ
// ========================================
//
// fs.ProductBlueprintRepositoryFS（= domain/productBlueprint.Repository 実装）を
// productBlueprint.Service が期待する ReaderRepository（GetByID + ListIDsByCompany）に
// 合わせるための薄いアダプタです。
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

// ★追加: companyId → productBlueprintIds を repo に委譲
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
//
// usecase.ProductInspectionRepo が期待する
//
//	UpdateInspectionResult(ctx, productID string, result inspection.InspectionResult)
//
// を、ProductRepositoryFS が持つ
//
//	UpdateInspectionResult(ctx, productID string, result product.InspectionResult)
//
// に橋渡しする。
type inspectionProductRepoAdapter struct {
	repo interface {
		UpdateInspectionResult(ctx context.Context, productID string, result productdom.InspectionResult) error
	}
}

// InspectionUsecase.ProductInspectionRepo を満たす
func (a *inspectionProductRepoAdapter) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result inspectiondom.InspectionResult,
) error {
	if a == nil || a.repo == nil {
		return errors.New("inspectionProductRepoAdapter: repo is nil")
	}
	// inspection.InspectionResult → product.InspectionResult に変換して委譲
	return a.repo.UpdateInspectionResult(ctx, productID, productdom.InspectionResult(result))
}

// ========================================
// ProductUsecase 用 ProductQueryRepo アダプタ
// ========================================
//
// 既存の Firestore Repository 群を束ねて usecase.ProductQueryRepo を実装します。
// - productRepo          → products 取得
// - modelRepo            → model variations 取得
// - productionRepo       → productions 取得
// - productBlueprintRepo → product_blueprints 取得
type productQueryRepoAdapter struct {
	productRepo          *fs.ProductRepositoryFS
	modelRepo            *fs.ModelRepositoryFS
	productionRepo       *fs.ProductionRepositoryFS
	productBlueprintRepo *fs.ProductBlueprintRepositoryFS
}

// GetProductByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetProductByID(
	ctx context.Context,
	productID string,
) (productdom.Product, error) {
	if a == nil || a.productRepo == nil {
		return productdom.Product{}, errors.New("productQueryRepoAdapter: productRepo is nil")
	}
	return a.productRepo.GetByID(ctx, productID)
}

// GetModelByID implements usecase.ProductQueryRepo.
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

// GetProductionByID implements usecase.ProductQueryRepo.
func (a *productQueryRepoAdapter) GetProductionByID(
	ctx context.Context,
	productionID string,
) (interface{}, error) {
	if a == nil || a.productionRepo == nil {
		return nil, errors.New("productQueryRepoAdapter: productionRepo is nil")
	}
	// productiondom.Production 型を interface{} として返す
	return a.productionRepo.GetByID(ctx, productionID)
}

// GetProductBlueprintByID implements usecase.ProductQueryRepo.
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
//
// fs.TokenBlueprintRepositoryFS は GetByID が (*TokenBlueprint, error)
// を返すため、NameResolver が期待する
//
//	GetByID(ctx, id) (tokenBlueprint.TokenBlueprint, error)
//
// に合わせる薄いアダプタです。
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
