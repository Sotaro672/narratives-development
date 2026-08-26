// backend/internal/application/query/mall/inquiry_query.go
package mall

import (
	"context"
	"fmt"
	"time"

	mallshared "narratives/internal/application/query/mall/shared"
	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryListItem は mall の問い合わせ一覧画面向け read model です。
// Inquiry の基本情報に、一覧表示で必要な reply 集約値を付加します。
type InquiryListItem struct {
	inquirydom.Inquiry
	LatestReply      *inquirydom.Reply `json:"latestReply,omitempty"`
	ReplyCount       int               `json:"replyCount"`
	UnreadReplyCount int               `json:"unreadReplyCount"`
	LatestActivityAt time.Time         `json:"latestActivityAt"`
}

// InquiryListResult は mall の問い合わせ一覧 BFF response 用 read model です。
type InquiryListResult struct {
	Items   []InquiryListItem `json:"items"`
	Page    int               `json:"page"`
	PerPage int               `json:"perPage"`
}

// InquiryQuery は mall 側の Inquiry / Reply read model を扱います。
// usecase は command 専用に寄せ、mall 画面で必要な read 処理はこの query service に集約します。
type InquiryQuery struct {
	repo      inquirydom.Repository
	replyRepo inquirydom.ReplyRepository
}

// NewInquiryQuery は InquiryQuery を初期化します。
func NewInquiryQuery(repo inquirydom.Repository, replyRepo inquirydom.ReplyRepository) *InquiryQuery {
	return &InquiryQuery{
		repo:      repo,
		replyRepo: replyRepo,
	}
}

// ListByAvatarID は avatar に紐づく Inquiry domain entity 一覧を取得します。
// avatarID は request body / query から受け取らず、middleware の AvatarContext から解決した値を渡します。
// filter.AvatarID は呼び出し元の値を信用せず、必ず引数 avatarID で上書きします。
func (q *InquiryQuery) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	filter inquirydom.Filter,
	sort inquirydom.Sort,
	page inquirydom.Page,
) (inquirydom.PageResult[inquirydom.Inquiry], error) {
	if q == nil || q.repo == nil {
		return inquirydom.PageResult[inquirydom.Inquiry]{}, fmt.Errorf("mall inquiry query: repository is nil")
	}
	if avatarID == "" {
		return inquirydom.PageResult[inquirydom.Inquiry]{}, inquirydom.ErrInvalidAvatarID
	}

	filter.AvatarID = &avatarID
	page.Number, page.PerPage = mallshared.NormalizeIntPage(page.Number, page.PerPage, 1, 100, 0)

	return q.repo.ListByAvatarID(ctx, avatarID, filter, sort, page)
}

// ListForAvatar は問い合わせ一覧画面用の BFF read model を返します。
// frontend が inquiry ごとに replies API を呼び出して latest reply / reply count / latest activity を
// 組み立てる必要がないよう、この query service で集約します。
func (q *InquiryQuery) ListForAvatar(
	ctx context.Context,
	avatarID string,
	filter inquirydom.Filter,
	sort inquirydom.Sort,
	page inquirydom.Page,
) (InquiryListResult, error) {
	if q == nil || q.replyRepo == nil {
		return InquiryListResult{}, fmt.Errorf("mall inquiry query: reply repository is nil")
	}

	result, err := q.ListByAvatarID(ctx, avatarID, filter, sort, page)
	if err != nil {
		return InquiryListResult{}, err
	}

	items := make([]InquiryListItem, 0, len(result.Items))
	for _, inquiry := range result.Items {
		replies, err := q.ListByInquiryID(ctx, inquiry.ID)
		if err != nil {
			return InquiryListResult{}, err
		}

		latestReply := findLatestInquiryReply(replies)
		unreadReplyCount := countUnreadInquiryRepliesForAvatar(
			replies,
			avatarID,
		)
		latestActivityAt := inquiry.UpdatedAt
		if latestActivityAt.IsZero() {
			latestActivityAt = inquiry.CreatedAt
		}
		if latestReply != nil {
			replyActivityAt := latestReply.CreatedAt
			if latestReply.UpdatedAt != nil && !latestReply.UpdatedAt.IsZero() {
				replyActivityAt = *latestReply.UpdatedAt
			}
			if replyActivityAt.After(latestActivityAt) {
				latestActivityAt = replyActivityAt
			}
		}

		items = append(items, InquiryListItem{
			Inquiry:          inquiry,
			LatestReply:      latestReply,
			ReplyCount:       len(replies),
			UnreadReplyCount: unreadReplyCount,
			LatestActivityAt: latestActivityAt,
		})
	}

	return InquiryListResult{
		Items:   items,
		Page:    page.Number,
		PerPage: page.PerPage,
	}, nil
}

// GetByID は Inquiry を取得します。
// command 処理前の現在状態取得など、domain entity が必要な場合に使います。
func (q *InquiryQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("mall inquiry query: repository is nil")
	}
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

// GetByIDForAvatar は avatar 所有確認込みで Inquiry を取得します。
// ListByAvatarID で avatar scope を確認した後、GetByID で現在状態を取得します。
// 取得結果の AvatarID も念のため確認します。
func (q *InquiryQuery) GetByIDForAvatar(
	ctx context.Context,
	id string,
	avatarID string,
) (inquirydom.Inquiry, error) {
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	if avatarID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidAvatarID
	}

	filter := inquirydom.Filter{
		IDs: []string{id},
	}

	result, err := q.ListByAvatarID(
		ctx,
		avatarID,
		filter,
		inquirydom.Sort{},
		inquirydom.Page{
			Number:  1,
			PerPage: 1,
		},
	)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	found := false
	for _, item := range result.Items {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		return inquirydom.Inquiry{}, inquirydom.ErrInquiryForbidden
	}

	inquiry, err := q.GetByID(ctx, id)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}
	if inquiry.AvatarID != avatarID {
		return inquirydom.Inquiry{}, inquirydom.ErrInquiryForbidden
	}

	return inquiry, nil
}

// ListByInquiryID は Inquiry の reply subcollection を取得します。
// 保存先: inquiries/{inquiryId}/replies/{replyId}
func (q *InquiryQuery) ListByInquiryID(ctx context.Context, inquiryID string) ([]inquirydom.Reply, error) {
	if q == nil || q.replyRepo == nil {
		return nil, fmt.Errorf("mall inquiry query: reply repository is nil")
	}
	if inquiryID == "" {
		return nil, inquirydom.ErrInvalidReplyInquiryID
	}

	replies, err := q.replyRepo.ListByInquiryID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if replies == nil {
		return []inquirydom.Reply{}, nil
	}

	return replies, nil
}

// ListRepliesByInquiryIDForAvatar は avatar 所有確認込みで reply 一覧を取得します。
// 処理順は ListByAvatarID -> GetByID -> ListByInquiryID です。
func (q *InquiryQuery) ListRepliesByInquiryIDForAvatar(
	ctx context.Context,
	inquiryID string,
	avatarID string,
) ([]inquirydom.Reply, error) {
	if _, err := q.GetByIDForAvatar(ctx, inquiryID, avatarID); err != nil {
		return nil, err
	}

	return q.ListByInquiryID(ctx, inquiryID)
}

// countUnreadInquiryRepliesForAvatar は avatar が受け取った未読 reply 数を返します。
// avatar 自身が送信した reply は未読件数に含めません。
func countUnreadInquiryRepliesForAvatar(
	replies []inquirydom.Reply,
	avatarID string,
) int {
	count := 0

	for _, reply := range replies {
		if reply.IsRead {
			continue
		}

		if reply.SenderType == inquirydom.ReplySenderTypeAvatar &&
			reply.SenderID == avatarID {
			continue
		}

		count++
	}

	return count
}

// findLatestInquiryReply は reply 一覧から最終更新日時が最も新しい reply を返します。
func findLatestInquiryReply(replies []inquirydom.Reply) *inquirydom.Reply {
	if len(replies) == 0 {
		return nil
	}

	latestIndex := 0
	latestAt := inquiryReplyActivityAt(replies[0])

	for index := 1; index < len(replies); index++ {
		currentAt := inquiryReplyActivityAt(replies[index])
		if currentAt.After(latestAt) {
			latestIndex = index
			latestAt = currentAt
		}
	}

	latest := replies[latestIndex]
	return &latest
}

// inquiryReplyActivityAt は reply の一覧表示上の最新日時を返します。
func inquiryReplyActivityAt(reply inquirydom.Reply) time.Time {
	if reply.UpdatedAt != nil && !reply.UpdatedAt.IsZero() {
		return *reply.UpdatedAt
	}
	return reply.CreatedAt
}
