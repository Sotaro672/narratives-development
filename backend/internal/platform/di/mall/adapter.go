// backend/internal/platform/di/mall/adapter.go
package mall

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	mallquery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
	usecase "narratives/internal/application/usecase"

	outfs "narratives/internal/adapters/out/firestore"

	avatarstate "narratives/internal/domain/avatarState"
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
// ✅ InventoryUsecase 用: ApplyTransferResult アダプタ（container.go から移譲）
// ============================================================

type inventoryRepoTransferResultAdapter struct {
	*outfs.InventoryRepositoryFS
}

func (a *inventoryRepoTransferResultAdapter) ApplyTransferResult(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) error {
	if a == nil || a.InventoryRepositoryFS == nil {
		return errors.New("inventory repo adapter is nil")
	}

	removed, err := a.InventoryRepositoryFS.ReleaseReservationAfterTransfer(ctx, productID, orderID, now)
	if err != nil {
		return err
	}

	log.Printf(
		"[inventory_repo_adapter.mall] ApplyTransferResult ok productId=%q orderId=%q removed=%d at=%s",
		strings.TrimSpace(productID),
		strings.TrimSpace(orderID),
		removed,
		now.UTC().Format(time.RFC3339),
	)

	return nil
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

// (NOTE)
// adapter.go には「型ズレ/IF差/署名差の吸収」や「配線のための薄い変換」を残し、
// Firestore SDK 直依存の実装は firestore.go に分離する。
var (
	_ = mallquery.ErrPreviewQueryNotConfigured
	_ = sharedquery.ErrOwnerResolveNotConfigured
	_ = usecase.TokenForTransfer{}
)
