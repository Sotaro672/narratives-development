// backend/internal/application/usecase/productBlueprintReview_usecase.go
package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	memberdom "narratives/internal/domain/member"
	pbdomain "narratives/internal/domain/productBlueprint"
	pbr "narratives/internal/domain/productBlueprintReview"
)

// Avatar 取得
type AvatarGetter interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// Brand 取得
type BrandGetter interface {
	GetByID(ctx context.Context, brandID string) (branddom.Brand, error)
}

// handler/画面へ渡す DTO（Review + AvatarName/Icon を同梱）
type ProductBlueprintReviewListItem struct {
	pbr.Review

	AvatarName string `json:"AvatarName"`
	AvatarIcon string `json:"AvatarIcon"`
}

// management 用: aggregate + BrandName + AssigneeName（PascalCase JSON）
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
	BrandGetter BrandGetter

	// assigneeId は member の Firestore docId として保存されている前提。
	// そのため AssigneeName 解決では GetByUID ではなく GetByID を使う。
	MemberRepo memberdom.Repository

	// verified purchase 判定
	ownedProductResolver OwnedProductResolver

	// avatarId -> Avatar
	// 実体は avatar.Repository を注入して使う想定
	AvatarRepo AvatarGetter

	now func() time.Time
}

func NewProductBlueprintReviewUsecase(
	reviewRepo pbr.Repository,
	productBlueprintRepo pbdomain.Repository,
	brandGetter BrandGetter,
	memberRepo memberdom.Repository,
	ownedProductResolver OwnedProductResolver,
	avatarRepo AvatarGetter,
	now func() time.Time,
) *ProductBlueprintReviewUsecase {
	if now == nil {
		now = time.Now
	}

	return &ProductBlueprintReviewUsecase{
		ReviewRepo:           reviewRepo,
		ProductBlueprintRepo: productBlueprintRepo,
		BrandGetter:          brandGetter,
		MemberRepo:           memberRepo,
		ownedProductResolver: ownedProductResolver,
		AvatarRepo:           avatarRepo,
		now:                  now,
	}
}

// ============================================================
// Public API: Aggregates (Management)
// - BrandID の Name 解決は usecase で実施（best-effort）
// - AssigneeID は ProductBlueprintRepo.GetByID(ctx, pb.ID) の戻り値から取得する
// - AssigneeID は member の docId 前提で、MemberRepo.GetByID(ctx, assigneeID) により名前解決する
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
	memberNameCache := make(map[string]string, 16)

	for _, pb := range paged {
		if pb.ID == "" {
			continue
		}

		// AssigneeID は GetByID の戻り値を正として扱う。
		livePB, err := uc.ProductBlueprintRepo.GetByID(ctx, pb.ID)
		if err != nil {
			return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, err
		}

		sum, err := uc.ReviewRepo.GetProductSummary(ctx, livePB.ID, status)
		if err != nil {
			return domcommon.PageResult[ProductBlueprintReviewAggregateItem]{}, err
		}

		brandName := ""
		if livePB.BrandID != "" && uc.BrandGetter != nil {
			if value, ok := brandNameCache[livePB.BrandID]; ok {
				brandName = value
			} else {
				if brand, err := uc.BrandGetter.GetByID(ctx, livePB.BrandID); err == nil {
					brandName = brand.Name
				}

				brandNameCache[livePB.BrandID] = brandName
			}
		}

		assigneeName := "-"
		if livePB.AssigneeID != "" {
			if value, ok := memberNameCache[livePB.AssigneeID]; ok {
				assigneeName = value
			} else {
				assigneeName = uc.resolveAssigneeNameByMemberID(
					ctx,
					livePB.AssigneeID,
				)

				memberNameCache[livePB.AssigneeID] = assigneeName
			}
		}

		items = append(items, ProductBlueprintReviewAggregateItem{
			ID:                 livePB.ID,
			ProductBlueprintID: livePB.ID,
			ProductName:        livePB.ProductName,
			BrandID:            livePB.BrandID,
			BrandName:          brandName,
			AssigneeID:         livePB.AssigneeID,
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

// resolveAssigneeNameByMemberID resolves assigneeName from member Firestore docId.
//
// NOTE:
// ProductBlueprint.AssigneeID は Firebase Auth UID ではなく member の Firestore docId。
// そのため GetByUID ではなく GetByID を使う。
func (uc *ProductBlueprintReviewUsecase) resolveAssigneeNameByMemberID(
	ctx context.Context,
	memberID string,
) string {
	if memberID == "" {
		return ""
	}

	if uc.MemberRepo == nil {
		return memberID
	}

	rec, err := uc.MemberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, memberdom.ErrNotFound) {
			return memberID
		}

		return memberID
	}

	name := memberdom.FormatLastFirst(
		rec.Member.LastName,
		rec.Member.FirstName,
	)

	if name == "" {
		return memberID
	}

	return name
}

// ============================================================
// Public API: List + AvatarName/Icon
// ============================================================
//
//   - ReviewRepo の結果に対して、AvatarRepo.GetByID を使って
//     AvatarName / AvatarIcon を詰めて返す
//   - AvatarRepo 未設定でも一覧自体は返す（name/icon は空）
//   - Avatar 取得失敗は best-effort でスキップ（画面表示優先）
func (uc *ProductBlueprintReviewUsecase) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	status pbr.ReviewStatus,
	page domcommon.Page,
) (domcommon.PageResult[ProductBlueprintReviewListItem], error) {
	if uc == nil || uc.ReviewRepo == nil {
		return domcommon.PageResult[ProductBlueprintReviewListItem]{}, pbr.ErrInternal
	}

	base, err := uc.ReviewRepo.ListByProductBlueprintID(
		ctx,
		productBlueprintID,
		status,
		page,
	)
	if err != nil {
		return domcommon.PageResult[ProductBlueprintReviewListItem]{}, err
	}

	items := make([]ProductBlueprintReviewListItem, 0, len(base.Items))
	for _, review := range base.Items {
		name := ""
		icon := ""

		if uc.AvatarRepo != nil && review.AvatarID != "" {
			avatar, err := uc.AvatarRepo.GetByID(ctx, review.AvatarID)
			if err == nil {
				name = avatar.AvatarName
				if avatar.AvatarIcon != nil {
					icon = *avatar.AvatarIcon
				}
			}
		}

		items = append(items, ProductBlueprintReviewListItem{
			Review:     review,
			AvatarName: name,
			AvatarIcon: icon,
		})
	}

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
// avatarID: docId=avatarId
// productBlueprintID: review target productBlueprintId
func (uc *ProductBlueprintReviewUsecase) IsVerifiedPurchase(
	ctx context.Context,
	avatarID string,
	productBlueprintID string,
) (bool, error) {
	if uc == nil || uc.ownedProductResolver == nil {
		return false, pbr.ErrInternal
	}

	return uc.ownedProductResolver.HasOwnedProductBlueprint(
		ctx,
		avatarID,
		productBlueprintID,
	)
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
