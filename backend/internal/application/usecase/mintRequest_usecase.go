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
// Repository Port (usecase が要求する最小インターフェース)
// ========================================
//
// Firestore 実装 (*firestore.MintRequestRepositoryFS) は
// mintdom.Repository を実装しており、その superset ですが、
// Usecase 側ではここで定義する最小インターフェースだけに依存します。
type MintRequestRepository interface {
	// ID で1件取得
	GetByID(ctx context.Context, id string) (mintdom.MintRequest, error)

	// productionId IN (...) で一覧取得
	ListByProductionIDs(ctx context.Context, productionIDs []string) ([]mintdom.MintRequest, error)

	// 既存 MintRequest の更新
	Update(ctx context.Context, mr mintdom.MintRequest) (mintdom.MintRequest, error)
}

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
// クエリ結果 DTO（アプリケーション層）
// ========================================
//
// フロント側の MintRequestDTO とほぼ 1:1 でマッピングできる形。
type MintRequestQueryResult struct {
	ID                 string
	ProductionID       string
	ProductBlueprintID string
	ProductName        string
	TokenBlueprintID   *string

	MintQuantity       int
	ProductionQuantity int

	Status      string
	RequestedBy *string
	RequestedAt *time.Time
	MintedAt    *time.Time
	BurnDate    *time.Time
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
) ([]MintRequestQueryResult, error) {

	if u.repo == nil || u.pbRepo == nil || u.prodRepo == nil {
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

	pbByID := make(map[string]pbdom.ProductBlueprint)
	for _, pb := range pbs {
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		if pb.DeletedAt != nil {
			// 論理削除済みは除外
			continue
		}
		id := strings.TrimSpace(pb.ID)
		if id == "" {
			continue
		}
		pbByID[id] = pb
	}

	if len(pbByID) == 0 {
		// 対象となる productBlueprint がない → MintRequest も 0 件
		return []MintRequestQueryResult{}, nil
	}

	// 2) 上記 productBlueprintId を参照している Production 一覧を取得
	prods, err := u.prodRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	prodByID := make(map[string]proddom.Production)
	prodIDs := make([]string, 0, len(prods))
	for _, p := range prods {
		pbid := strings.TrimSpace(p.ProductBlueprintID)
		if _, ok := pbByID[pbid]; !ok {
			continue
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}

		prodByID[id] = p
		prodIDs = append(prodIDs, id)
	}

	if len(prodIDs) == 0 {
		// 対象となる production がない → MintRequest も 0 件
		return []MintRequestQueryResult{}, nil
	}

	// 3) productionId IN (...) に紐づく MintRequest 一覧を取得
	mrs, err := u.repo.ListByProductionIDs(ctx, prodIDs)
	if err != nil {
		return nil, err
	}

	// 4) 結果組み立て（productName / productionQuantity を解決）
	results := make([]MintRequestQueryResult, 0, len(mrs))

	for _, mr := range mrs {
		prodID := strings.TrimSpace(mr.ProductionID)
		p, ok := prodByID[prodID]

		var (
			pbID          string
			productName   string
			productionQty int
		)

		if ok {
			pbID = strings.TrimSpace(p.ProductBlueprintID)

			if pb, okPB := pbByID[pbID]; okPB {
				productName = strings.TrimSpace(pb.ProductName)
			}

			// Production の生産量: Models[].Quantity の合計とする
			total := 0
			for _, mq := range p.Models {
				if mq.Quantity > 0 {
					total += mq.Quantity
				}
			}
			productionQty = total
		}

		results = append(results, MintRequestQueryResult{
			ID:                 mr.ID,
			ProductionID:       prodID,
			ProductBlueprintID: pbID,
			ProductName:        productName,
			TokenBlueprintID:   mr.TokenBlueprintID,
			MintQuantity:       mr.MintQuantity,
			ProductionQuantity: productionQty,
			Status:             string(mr.Status),
			RequestedBy:        mr.RequestedBy,
			RequestedAt:        mr.RequestedAt,
			MintedAt:           mr.MintedAt,
			BurnDate:           mr.ScheduledBurnDate,
		})
	}

	return results, nil
}

// ----------------------------------------
// Commands
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
