// backend/internal/application/usecase/mintRequest_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	mintdom "narratives/internal/domain/mintRequest"
	pbdom "narratives/internal/domain/productBlueprint"
	proddom "narratives/internal/domain/production"
)

// ========================================
// 依存ポート（最小インターフェース）
// ========================================

// ProductBlueprintListRepo は companyId 付きで一覧取得できる既存 FS 実装
// (ProductBlueprintRepositoryFS) に対応する最小インターフェースです。
type ProductBlueprintListRepo interface {
	// repository_fs の List(ctx) と同じシグネチャ
	List(ctx context.Context) ([]pbdom.ProductBlueprint, error)
}

// ProductionListRepo は ProductionRepositoryFS の List と互換のポートです。
type ProductionListRepo interface {
	List(ctx context.Context) ([]proddom.Production, error)
}

// ========================================
// Repository Port (usecase が要求する最小インターフェース)
// ========================================
//
// Firestore 実装 (*firestore.MintRequestRepositoryFS) は
// mintdom.Repository を実装しており、その superset ですが、
// Usecase 側ではここで定義する最小インターフェースだけに依存します。
type MintRequestRepository interface {
	// ID で1件取得
	GetByID(ctx context.Context, id string) (mintdom.MintRequest, error)

	// 既存 MintRequest の更新
	Update(ctx context.Context, mr mintdom.MintRequest) (mintdom.MintRequest, error)

	// 指定された productionId 群に紐づく MintRequest 一覧を取得
	// （backend/internal/domain/mintRequest/repository_port.go に対応）
	ListByProductionIDs(ctx context.Context, productionIDs []string) ([]mintdom.MintRequest, error)
}

// ========================================
// Usecase 本体
// ========================================

type MintRequestUsecase struct {
	repo     MintRequestRepository
	pbRepo   ProductBlueprintListRepo
	prodRepo ProductionListRepo
}

// NewMintRequestUsecase はユースケースを初期化します。
func NewMintRequestUsecase(
	repo MintRequestRepository,
	pbRepo ProductBlueprintListRepo,
	prodRepo ProductionListRepo,
) *MintRequestUsecase {
	return &MintRequestUsecase{
		repo:     repo,
		pbRepo:   pbRepo,
		prodRepo: prodRepo,
	}
}

// ----------------------------------------
// Queries
// ----------------------------------------

// GetByID は ID で MintRequest を取得します。
func (u *MintRequestUsecase) GetByID(
	ctx context.Context,
	id string,
) (mintdom.MintRequest, error) {

	id = strings.TrimSpace(id)
	if id == "" {
		return mintdom.MintRequest{}, mintdom.ErrInvalidID
	}
	return u.repo.GetByID(ctx, id)
}

// ListByCurrentCompany は、context に注入された companyId を起点に
// 1) companyId を持つ productBlueprint を取得
// 2) それらに紐づく productions を取得
// 3) productions の ID 群を使って mintRequests を取得
// という 3 段階で MintRequest 一覧を返します。
func (u *MintRequestUsecase) ListByCurrentCompany(
	ctx context.Context,
) ([]mintdom.MintRequest, error) {

	if u.pbRepo == nil || u.prodRepo == nil || u.repo == nil {
		return nil, errors.New("mintRequest: usecase not initialized")
	}

	// AuthMiddleware → usecase.WithCompanyID(...) により埋め込まれた companyId を取得
	cid := companyIDFromContext(ctx)
	cid = strings.TrimSpace(cid)
	if cid == "" {
		// companyId が無い場合は、テナント未確定としてエラーにしておく
		return nil, errors.New("mintRequest: companyId not found in context")
	}

	// 1) companyId = cid の ProductBlueprint 一覧を取得
	pbs, err := u.pbRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	pbIDSet := make(map[string]struct{})
	for _, pb := range pbs {
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		// 論理削除済みは除外（DeletedAt != nil）
		if pb.DeletedAt != nil {
			continue
		}
		id := strings.TrimSpace(pb.ID)
		if id == "" {
			continue
		}
		pbIDSet[id] = struct{}{}
	}

	if len(pbIDSet) == 0 {
		// 対象となる productBlueprint がない → MintRequest も 0 件
		return []mintdom.MintRequest{}, nil
	}

	// 2) 上記 productBlueprintId を参照している Production 一覧を取得
	prods, err := u.prodRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	prodIDSet := make(map[string]struct{})
	for _, p := range prods {
		pbid := strings.TrimSpace(p.ProductBlueprintID)
		if _, ok := pbIDSet[pbid]; !ok {
			continue
		}
		// ※必要に応じて Production 側の論理削除チェックを追加してもよい
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		prodIDSet[id] = struct{}{}
	}

	if len(prodIDSet) == 0 {
		// 対象となる production がない → MintRequest も 0 件
		return []mintdom.MintRequest{}, nil
	}

	// map → slice へ変換
	prodIDs := make([]string, 0, len(prodIDSet))
	for id := range prodIDSet {
		prodIDs = append(prodIDs, id)
	}

	// 3) productionId IN (...) に紐づく MintRequest 一覧を取得
	mrs, err := u.repo.ListByProductionIDs(ctx, prodIDs)
	if err != nil {
		return nil, err
	}

	return mrs, nil
}

// ----------------------------------------
// Commands（今後の拡張用サンプル）
// ----------------------------------------

// Request は planning → requested への遷移を行います。
func (u *MintRequestUsecase) Request(
	ctx context.Context,
	id string,
	requestedBy string,
	at time.Time,
) (mintdom.MintRequest, error) {

	id = strings.TrimSpace(id)
	if id == "" {
		return mintdom.MintRequest{}, mintdom.ErrInvalidID
	}

	mr, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return mintdom.MintRequest{}, err
	}

	if err := mr.Request(requestedBy, at); err != nil {
		return mintdom.MintRequest{}, err
	}

	return u.repo.Update(ctx, mr)
}

// MarkMinted は requested → minted への遷移を行います。
func (u *MintRequestUsecase) MarkMinted(
	ctx context.Context,
	id string,
	at time.Time,
) (mintdom.MintRequest, error) {

	id = strings.TrimSpace(id)
	if id == "" {
		return mintdom.MintRequest{}, mintdom.ErrInvalidID
	}

	mr, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return mintdom.MintRequest{}, err
	}

	if err := mr.MarkMinted(at); err != nil {
		return mintdom.MintRequest{}, err
	}

	return u.repo.Update(ctx, mr)
}
