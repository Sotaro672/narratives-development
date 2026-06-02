// backend/internal/application/usecase/productBlueprintReview_usecase.go
package usecase

import (
	"context"
	"math"
	"time"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
	productdom "narratives/internal/domain/product"
	pbdomain "narratives/internal/domain/productBlueprint"
	pbr "narratives/internal/domain/productBlueprintReview"
)

// wallet_usecase.go で既に定義されている IF/Err を再利用する（ここでは再定義しない）
// - WalletRepository
// - OnchainWalletReader
// - TokenQuery
// - ProductReader
// - ModelProductBlueprintIDResolver
// - ErrWalletUsecaseNotConfigured / ErrWalletSyncOnchainNotConfigured / ... etc

// AvatarName/Icon の軽量取得（N+1 を軽量化する想定のポート）
type AvatarNameIconGetter interface {
	GetNameAndIconByID(ctx context.Context, id string) (name string, icon string, err error)
}

// Brand 取得
type BrandGetter interface {
	GetByID(ctx context.Context, brandID string) (branddom.Brand, error)
}

// Assignee(Member) 名取得（member.Service をそのまま注入できる）
type AssigneeNameGetter interface {
	GetNameLastFirstByUID(ctx context.Context, uid string) (string, error)
}

// handler/画面へ渡す DTO（Review + AvatarName/Icon を同梱）
type ProductBlueprintReviewListItem struct {
	pbr.Review

	AvatarName string `json:"AvatarName"`
	AvatarIcon string `json:"AvatarIcon"`
}

// management 用: aggregate + BrandName/AssigneeName（PascalCase JSON）
type ProductBlueprintReviewAggregateItem struct {
	ID                 string `json:"ID"`
	ProductBlueprintID string `json:"ProductBlueprintID"`

	ProductName string `json:"ProductName"`

	BrandID   string `json:"BrandID"`
	BrandName string `json:"BrandName"`

	AssigneeID   string `json:"AssigneeID"`
	AssigneeName string `json:"AssigneeName"`

	Rating1Count int `json:"Rating1Count"`
	Rating2Count int `json:"Rating2Count"`
	Rating3Count int `json:"Rating3Count"`
	Rating4Count int `json:"Rating4Count"`
	Rating5Count int `json:"Rating5Count"`

	TotalCount    int     `json:"TotalCount"`
	AverageRating float64 `json:"AverageRating"`
}

type ProductBlueprintReviewUsecase struct {
	ReviewRepo pbr.Repository

	// aggregates 用
	ProductBlueprintRepo pbdomain.Repository

	// name resolvers (best-effort)
	BrandGetter        BrandGetter
	AssigneeNameGetter AssigneeNameGetter

	WalletRepo              WalletRepository
	OnchainReader           OnchainWalletReader
	TokenQuery              TokenQuery
	ProductReader           ProductReader
	ModelProductBlueprintID ModelProductBlueprintIDResolver

	// avatarId -> (avatarName, avatarIcon)
	// 実体は avatar.Repository を注入して使う想定
	AvatarRepo AvatarNameIconGetter

	now func() time.Time
}

func NewProductBlueprintReviewUsecase(
	reviewRepo pbr.Repository,
	walletRepo WalletRepository,
) *ProductBlueprintReviewUsecase {
	return &ProductBlueprintReviewUsecase{
		ReviewRepo:              reviewRepo,
		ProductBlueprintRepo:    nil,
		BrandGetter:             nil,
		AssigneeNameGetter:      nil,
		WalletRepo:              walletRepo,
		OnchainReader:           nil,
		TokenQuery:              nil,
		ProductReader:           nil,
		ModelProductBlueprintID: nil,
		AvatarRepo:              nil,
		now:                     time.Now,
	}
}

func (uc *ProductBlueprintReviewUsecase) WithProductBlueprintRepo(r pbdomain.Repository) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.ProductBlueprintRepo = r
	}
	return uc
}

// BrandRepository 注入
func (uc *ProductBlueprintReviewUsecase) WithBrandRepository(r branddom.Repository) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.BrandGetter = r
	}
	return uc
}

// MemberService 注入（member.Service をそのまま渡せる）
func (uc *ProductBlueprintReviewUsecase) WithMemberService(s *memdom.Service) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.AssigneeNameGetter = s
	}
	return uc
}

func (uc *ProductBlueprintReviewUsecase) WithOnchainReader(r OnchainWalletReader) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.OnchainReader = r
	}
	return uc
}

func (uc *ProductBlueprintReviewUsecase) WithTokenQuery(q TokenQuery) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.TokenQuery = q
	}
	return uc
}

func (uc *ProductBlueprintReviewUsecase) WithProductReader(r ProductReader) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.ProductReader = r
	}
	return uc
}

func (uc *ProductBlueprintReviewUsecase) WithModelProductBlueprintIDResolver(r ModelProductBlueprintIDResolver) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.ModelProductBlueprintID = r
	}
	return uc
}

// AvatarRepo 注入（avatar.Repository をそのまま渡せる）
func (uc *ProductBlueprintReviewUsecase) WithAvatarRepo(r avatardom.Repository) *ProductBlueprintReviewUsecase {
	if uc != nil {
		uc.AvatarRepo = r
	}
	return uc
}

func (uc *ProductBlueprintReviewUsecase) WithNow(f func() time.Time) *ProductBlueprintReviewUsecase {
	if uc != nil && f != nil {
		uc.now = f
	}
	return uc
}

// ============================================================
// Public API: Aggregates (Management)
// - BrandID/AssigneeID の Name 解決は usecase で実施（best-effort）
// - paging は「商品（ProductBlueprint）単位」
// ============================================================

func (uc *ProductBlueprintReviewUsecase) ListCompanyReviewAggregatesWithNames(
	ctx context.Context,
	companyID string,
	status pbr.ReviewStatus,
	page domcommon.Page,
) (domcommon.PageResult[ProductBlueprintReviewAggregateItem], error) {
	if uc == nil || uc.ReviewRepo == nil || uc.ProductBlueprintRepo == nil {
		return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, pbr.ErrInternal
	}
	if companyID == "" {
		return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, pbr.ErrInternal
	}
	if page.Number <= 0 {
		page.Number = 1
	}
	if page.PerPage <= 0 {
		page.PerPage = 100
	}

	productBlueprints, err := uc.ProductBlueprintRepo.ListByCompanyID(ctx, companyID)
	if err != nil {
		return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, err
	}

	totalCount := len(productBlueprints)
	totalPages := 0
	if page.PerPage > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(page.PerPage)))
	}
	if totalPages < 0 {
		totalPages = 0
	}

	start := (page.Number - 1) * page.PerPage
	if start > totalCount {
		start = totalCount
	}
	end := start + page.PerPage
	if end > totalCount {
		end = totalCount
	}
	paged := productBlueprints[start:end]

	items := make([]ProductBlueprintReviewAggregateItem, 0, len(paged))

	// simple per-request cache
	brandNameCache := make(map[string]string, 16)
	assigneeNameCache := make(map[string]string, 16)

	for _, pb := range paged {
		if pb.ID == "" {
			continue
		}

		sum, e := uc.ReviewRepo.GetProductSummary(ctx, pb.ID, status)
		if e != nil {
			return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, e
		}

		brandName := ""
		if pb.BrandID != "" && uc.BrandGetter != nil {
			if v, ok := brandNameCache[pb.BrandID]; ok {
				brandName = v
			} else {
				if b, err := uc.BrandGetter.GetByID(ctx, pb.BrandID); err == nil {
					brandName = b.Name
				}
				brandNameCache[pb.BrandID] = brandName
			}
		}

		assigneeName := ""
		if pb.AssigneeID != "" && uc.AssigneeNameGetter != nil {
			if v, ok := assigneeNameCache[pb.AssigneeID]; ok {
				assigneeName = v
			} else {
				if n, err := uc.AssigneeNameGetter.GetNameLastFirstByUID(ctx, pb.AssigneeID); err == nil {
					assigneeName = n
				}
				assigneeNameCache[pb.AssigneeID] = assigneeName
			}
		}

		items = append(items, ProductBlueprintReviewAggregateItem{
			ID:                 pb.ID,
			ProductBlueprintID: pb.ID,
			ProductName:        pb.ProductName,
			BrandID:            pb.BrandID,
			BrandName:          brandName,
			AssigneeID:         pb.AssigneeID,
			AssigneeName:       assigneeName,
			Rating1Count:       sum.Rating1Count,
			Rating2Count:       sum.Rating2Count,
			Rating3Count:       sum.Rating3Count,
			Rating4Count:       sum.Rating4Count,
			Rating5Count:       sum.Rating5Count,
			TotalCount:         sum.TotalCount,
			AverageRating:      sum.AverageRating,
		})
	}

	out := domcommon.PageResult[ProductBlueprintReviewAggregateItem]{
		Items:      items,
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
	return out, nil
}

// ============================================================
// Public API: List (for handler)
// ============================================================

func (uc *ProductBlueprintReviewUsecase) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	status pbr.ReviewStatus,
	page domcommon.Page,
) (domcommon.PageResult[pbr.Review], error) {
	if uc == nil || uc.ReviewRepo == nil {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInternal
	}
	return uc.ReviewRepo.ListByProductBlueprintID(ctx, productBlueprintID, status, page)
}

// ============================================================
// Public API: List + AvatarName/Icon (for screen)
// ============================================================
//
//   - ReviewRepo の結果に対して、AvatarRepo.GetNameAndIconByID を使って
//     AvatarName / AvatarIcon を詰めて返す
//   - AvatarRepo 未設定でも一覧自体は返す（name/icon は空）
//   - Avatar 取得失敗は best-effort でスキップ（画面表示優先）
func (uc *ProductBlueprintReviewUsecase) ListByProductBlueprintIDWithAvatar(
	ctx context.Context,
	productBlueprintID string,
	status pbr.ReviewStatus,
	page domcommon.Page,
) (domcommon.PageResult[ProductBlueprintReviewListItem], error) {
	if uc == nil || uc.ReviewRepo == nil {
		return domcommon.PageResult[ProductBlueprintReviewListItem]{}, pbr.ErrInternal
	}

	base, err := uc.ReviewRepo.ListByProductBlueprintID(ctx, productBlueprintID, status, page)
	if err != nil {
		return domcommon.PageResult[ProductBlueprintReviewListItem]{}, err
	}

	items := make([]ProductBlueprintReviewListItem, 0, len(base.Items))
	for _, r := range base.Items {
		name := ""
		icon := ""

		if uc.AvatarRepo != nil && r.AvatarID != "" {
			n, i, e := uc.AvatarRepo.GetNameAndIconByID(ctx, r.AvatarID)
			if e == nil {
				name = n
				icon = i
			}
		}

		items = append(items, ProductBlueprintReviewListItem{
			Review:     r,
			AvatarName: name,
			AvatarIcon: icon,
		})
	}

	// paging情報は base からそのまま引き継ぐ（handler が TotalCount/TotalPages/PerPage を参照するため）
	out := domcommon.PageResult[ProductBlueprintReviewListItem]{
		Items:      items,
		Page:       base.Page,
		PerPage:    base.PerPage,
		TotalCount: base.TotalCount,
		TotalPages: base.TotalPages,
	}
	return out, nil
}

// ============================================================
// Public API: VerifiedPurchase check (for handler)
// ============================================================

// IsVerifiedPurchase exposes verified-purchase check for handlers.
// avatarID: docId=avatarId（WalletRepo.GetByAvatarID のキー）
// productBlueprintID: review target productBlueprintId
func (uc *ProductBlueprintReviewUsecase) IsVerifiedPurchase(
	ctx context.Context,
	avatarID string,
	productBlueprintID string,
) (bool, error) {
	return uc.resolveVerifiedPurchase(ctx, avatarID, productBlueprintID)
}

// ============================================================
// VerifiedPurchase 判定 “query”
// ============================================================
//
// 要件：
// wallets の tokens の mintAddress を取得
// mintAddress から tokens の docId を取得
// docId から products の modelId を取得
// modelId から models の productBlueprintId を取得
// productBlueprintReview の productBlueprintId と一致した場合 VerifiedPurchase=true
//
// 既存 wallet_usecase.go の依存を使って実現：
// - mintAddress 一覧: OnchainReader.ListOwnedTokenMints(walletAddress)
// - mintAddress -> token(docId相当=productId): TokenQuery.ResolveTokenByMintAddress().ProductID
// - productId -> modelId: ProductReader.GetByID(productId).ModelID
// - modelId -> productBlueprintId: ModelProductBlueprintID.GetIDByModelID(modelId)
// - productBlueprintReview の productBlueprintId と一致した場合 VerifiedPurchase=true
func (uc *ProductBlueprintReviewUsecase) resolveVerifiedPurchase(
	ctx context.Context,
	avatarID string, // docId=avatarId（WalletRepo.GetByAvatarID のキー）
	reviewProductBlueprintID string,
) (bool, error) {
	if uc == nil || uc.WalletRepo == nil {
		return false, ErrWalletUsecaseNotConfigured
	}
	if uc.OnchainReader == nil {
		return false, ErrWalletSyncOnchainNotConfigured
	}
	if uc.TokenQuery == nil {
		return false, ErrWalletTokenQueryNotConfigured
	}
	if uc.ProductReader == nil {
		return false, ErrWalletProductReaderNotConfigured
	}
	if uc.ModelProductBlueprintID == nil {
		return false, ErrWalletModelProductBlueprintNotConfigured
	}

	if avatarID == "" {
		return false, ErrWalletSyncAvatarIDEmpty
	}
	if reviewProductBlueprintID == "" {
		return false, productdom.ErrInvalidID
	}

	// 1) docId=avatarId で wallet を取得
	w, err := uc.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return false, err
	}

	// 2) walletAddress から on-chain の mint 一覧（＝wallets.tokens.mintAddress 相当）
	if w.WalletAddress == "" {
		return false, ErrWalletSyncWalletAddressEmpty
	}
	mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, w.WalletAddress)
	if err != nil {
		return false, err
	}
	if len(mints) == 0 {
		return false, nil
	}

	// 3) mintAddress -> token(docId=productId) -> product.modelId -> model.productBlueprintId
	for _, mint := range mints {
		if mint == "" {
			continue
		}

		res, err := uc.TokenQuery.ResolveTokenByMintAddress(ctx, mint)
		if err != nil {
			// 逆引き失敗は「未購入扱い」でスキップ（厳密運用なら return err に変更）
			continue
		}
		productID := res.ProductID
		if productID == "" {
			continue
		}

		p, err := uc.ProductReader.GetByID(ctx, productID)
		if err != nil {
			continue
		}
		modelID := p.ModelID
		if modelID == "" {
			continue
		}

		pbID, _, err := uc.ModelProductBlueprintID.GetIDByModelID(ctx, modelID)
		if err != nil {
			continue
		}
		if pbID == "" {
			continue
		}

		if pbID == reviewProductBlueprintID {
			return true, nil
		}
	}

	return false, nil
}

// ============================================================
// 命名衝突回避：CreateReviewInput → CreateProductBlueprintReviewInput
// ============================================================

type CreateProductBlueprintReviewInput struct {
	ProductBlueprintID string
	AvatarID           string // docId=avatarId 前提

	Rating pbr.Rating
	Title  string
	Body   string

	ReviewedAt time.Time

	CreatedAt  time.Time
	CreatedBy  string
	PublishNow bool
}

func (uc *ProductBlueprintReviewUsecase) CreateProductBlueprintReview(
	ctx context.Context,
	in CreateProductBlueprintReviewInput,
) (pbr.Review, error) {
	if uc == nil || uc.ReviewRepo == nil {
		return pbr.Review{}, pbr.ErrInternal
	}

	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = uc.now().UTC()
	}

	reviewedAt := in.ReviewedAt
	if reviewedAt.IsZero() {
		reviewedAt = createdAt
	}

	entity, err := pbr.New(pbr.NewReviewParams{
		ProductBlueprintID: in.ProductBlueprintID,
		AvatarID:           in.AvatarID,
		Rating:             in.Rating,
		Title:              in.Title,
		Body:               in.Body,
		ReviewedAt:         reviewedAt,
		CreatedAt:          createdAt,
		CreatedBy:          in.CreatedBy,
		PublishNow:         in.PublishNow,
	})
	if err != nil {
		return pbr.Review{}, err
	}

	return uc.ReviewRepo.Create(ctx, entity)
}
