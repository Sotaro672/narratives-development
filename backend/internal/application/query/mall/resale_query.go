// backend/internal/application/query/mall/resale_query.go
package mall

import (
	"context"
	"errors"
	"sort"
	"time"

	applicationport "narratives/internal/application/port"
	mallshared "narratives/internal/application/query/mall/shared"
	avatardom "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
	resalereview "narratives/internal/domain/resale_review"
)

type ResaleChatListItem struct {
	ResaleID           string                 `json:"resaleId"`
	Status             resaledom.ResaleStatus `json:"status"`
	ProductName        string                 `json:"productName"`
	TokenName          string                 `json:"tokenName"`
	TokenIcon          string                 `json:"tokenIcon"`
	BrandName          string                 `json:"brandName"`
	ImageURL           string                 `json:"imageUrl"`
	Price              int                    `json:"price"`
	LatestComment      *resalereview.Comment  `json:"latestComment,omitempty"`
	CommentCount       int                    `json:"commentCount"`
	UnreadCommentCount int                    `json:"unreadCommentCount"`
	LatestActivityAt   time.Time              `json:"latestActivityAt"`
}

type ResaleQuery struct {
	resaleRepo       resaledom.Repository
	imageRepo        applicationport.ResaleImageLister
	resaleReviewRepo resalereview.RepositoryPort
	displayResolver  mallshared.MallDisplayResolver
	avatarRepo       avatardom.Repository
}

func NewResaleQuery(
	resaleRepo resaledom.Repository,
	imageRepo applicationport.ResaleImageLister,
	resaleReviewRepo resalereview.RepositoryPort,
	displayResolver mallshared.MallDisplayResolver,
	avatarRepo avatardom.Repository,
) *ResaleQuery {
	return &ResaleQuery{
		resaleRepo:       resaleRepo,
		imageRepo:        imageRepo,
		resaleReviewRepo: resaleReviewRepo,
		displayResolver:  displayResolver,
		avatarRepo:       avatarRepo,
	}
}

func (q *ResaleQuery) List(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	page resaledom.Page,
) (resaledom.PageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.PageResult[resaledom.Resale]{}, errors.New("not supported: ResaleQuery.List")
	}

	result, err := q.resaleRepo.List(ctx, filter, sort, page)
	if err != nil {
		return resaledom.PageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *ResaleQuery) ListByCursor(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	cpage resaledom.CursorPage,
) (resaledom.CursorPageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, errors.New("not supported: ResaleQuery.ListByCursor")
	}

	result, err := q.resaleRepo.ListByCursor(ctx, filter, sort, cpage)
	if err != nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *ResaleQuery) GetByID(
	ctx context.Context,
	id string,
) (resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.Resale{}, errors.New("not supported: ResaleQuery.GetByID")
	}

	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	item, err := q.resaleRepo.GetByID(ctx, id)
	if err != nil {
		return resaledom.Resale{}, err
	}

	item = q.enrichResaleForDisplay(ctx, item)

	return item, nil
}

func (q *ResaleQuery) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return nil, errors.New("not supported: ResaleQuery.ListByAvatarID")
	}

	if avatarID == "" {
		return []resaledom.Resale{}, nil
	}

	items, err := q.resaleRepo.ListByAvatarID(ctx, avatarID)
	if err != nil {
		return nil, err
	}

	items = q.enrichResalesForDisplay(ctx, items)

	return items, nil
}

// ListChatItems returns resale comment threads owned by avatarID.
//
// Only resales with at least one visible comment are returned.
// Threads are ordered by the creation time of the latest visible comment.
//
// IsRead is seller-side state, therefore UnreadCommentCount represents
// comments that the resale owner has not yet read.
func (q *ResaleQuery) ListChatItems(
	ctx context.Context,
	avatarID string,
) ([]ResaleChatListItem, error) {
	if q == nil || q.resaleRepo == nil {
		return nil, errors.New("not supported: ResaleQuery.ListChatItems")
	}
	if q.resaleReviewRepo == nil || q.resaleReviewRepo.Comments() == nil {
		return nil, errors.New("not supported: ResaleQuery.ListChatItems.ResaleReviewRepo")
	}

	if avatarID == "" {
		return []ResaleChatListItem{}, nil
	}

	resales, err := q.resaleRepo.ListByAvatarID(ctx, avatarID)
	if err != nil {
		return nil, err
	}

	resales = q.enrichResalesForDisplay(ctx, resales)
	items := make([]ResaleChatListItem, 0, len(resales))

	for _, resale := range resales {
		item, ok, err := q.buildChatListItem(ctx, resale)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].LatestActivityAt.Equal(items[j].LatestActivityAt) {
			return items[i].ResaleID > items[j].ResaleID
		}
		return items[i].LatestActivityAt.After(items[j].LatestActivityAt)
	})

	return items, nil
}

// CountUnreadCommentsByAvatarID returns the total number of visible unread
// comments across all resales owned by avatarID.
//
// This method intentionally does not load display information or latest
// comments because it is intended for the chat badge counter.
func (q *ResaleQuery) CountUnreadCommentsByAvatarID(
	ctx context.Context,
	avatarID string,
) (int, error) {
	if q == nil || q.resaleRepo == nil {
		return 0, errors.New("not supported: ResaleQuery.CountUnreadCommentsByAvatarID")
	}
	if q.resaleReviewRepo == nil || q.resaleReviewRepo.Comments() == nil {
		return 0, errors.New("not supported: ResaleQuery.CountUnreadCommentsByAvatarID.ResaleReviewRepo")
	}

	if avatarID == "" {
		return 0, nil
	}

	resales, err := q.resaleRepo.ListByAvatarID(ctx, avatarID)
	if err != nil {
		return 0, err
	}

	visible := false
	unread := false
	total := 0

	for _, resale := range resales {
		result, err := q.resaleReviewRepo.Comments().List(
			ctx,
			resalereview.FilterComment{
				ResaleID: resale.ID,
				Deleted:  &visible,
				IsRead:   &unread,
			},
			common.Sort{
				Column: "createdAt",
				Order:  common.SortDesc,
			},
			common.Page{
				Number:  1,
				PerPage: 1,
			},
		)
		if err != nil {
			return 0, err
		}

		total += result.TotalCount
	}

	return total, nil
}

func (q *ResaleQuery) ListImages(
	ctx context.Context,
	resaleID string,
) ([]resaledom.ResaleImage, error) {
	if q == nil || q.imageRepo == nil {
		return nil, errors.New("not supported: ResaleQuery.ListImages")
	}

	if resaleID == "" {
		return nil, resaledom.ErrInvalidConditionImageResaleID
	}

	return q.imageRepo.ListByResaleID(ctx, resaleID)
}

func (q *ResaleQuery) buildChatListItem(
	ctx context.Context,
	resale resaledom.Resale,
) (ResaleChatListItem, bool, error) {
	if q == nil || q.resaleReviewRepo == nil || q.resaleReviewRepo.Comments() == nil {
		return ResaleChatListItem{}, false, errors.New("not supported: ResaleQuery.buildChatListItem")
	}
	if resale.ID == "" {
		return ResaleChatListItem{}, false, resaledom.ErrInvalidID
	}

	visible := false

	latestResult, err := q.resaleReviewRepo.Comments().List(
		ctx,
		resalereview.FilterComment{
			ResaleID: resale.ID,
			Deleted:  &visible,
		},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  1,
			PerPage: 1,
		},
	)
	if err != nil {
		return ResaleChatListItem{}, false, err
	}

	if latestResult.TotalCount == 0 || len(latestResult.Items) == 0 {
		return ResaleChatListItem{}, false, nil
	}

	latestComment := latestResult.Items[0]
	unread := false

	unreadResult, err := q.resaleReviewRepo.Comments().List(
		ctx,
		resalereview.FilterComment{
			ResaleID: resale.ID,
			Deleted:  &visible,
			IsRead:   &unread,
		},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  1,
			PerPage: 1,
		},
	)
	if err != nil {
		return ResaleChatListItem{}, false, err
	}

	return ResaleChatListItem{
		ResaleID:           resale.ID,
		Status:             resale.Status,
		ProductName:        resale.ProductName,
		TokenName:          resale.TokenName,
		TokenIcon:          resale.TokenIcon,
		BrandName:          resale.BrandName,
		ImageURL:           resale.ImageURL,
		Price:              resale.Price,
		LatestComment:      &latestComment,
		CommentCount:       latestResult.TotalCount,
		UnreadCommentCount: unreadResult.TotalCount,
		LatestActivityAt:   latestComment.CreatedAt,
	}, true, nil
}

func (q *ResaleQuery) enrichResalesForDisplay(
	ctx context.Context,
	items []resaledom.Resale,
) []resaledom.Resale {
	return q.newDisplayEnricher().enrichResalesForDisplay(ctx, items)
}

func (q *ResaleQuery) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	return q.newDisplayEnricher().enrichResaleForDisplay(ctx, item)
}

func (q *ResaleQuery) newDisplayEnricher() *resaleDisplayEnricher {
	if q == nil {
		return newResaleDisplayEnricher(resaleDisplayEnricherConfig{})
	}

	return newResaleDisplayEnricher(resaleDisplayEnricherConfig{
		displayResolver: q.displayResolver,
		avatarRepo:      q.avatarRepo,

		// ResaleQuery の表示補完:
		// - avatarName/avatarIcon を補完する
		// - resale image は ListImages API 側で取得する
		// - tokenBlueprint.IconURL を ImageURL fallback として使う
		includeAvatar:               true,
		includeImage:                false,
		useTokenIconAsImageFallback: true,
	})
}
