// backend/internal/application/query/mall/inquiry_query.go
package mall

import (
	"context"
	"fmt"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryQuery は mall 側の Inquiry / Reply read model を扱います。
//
// usecase は command 専用に寄せるため、mall 画面で必要な read 処理は
// この query service に集約します。
//
// 期待する reply 一覧取得フロー:
//
//  1. ListByAvatarID
//     avatarId に紐づく Inquiry 一覧を取得し、対象 inquiryId が avatar のものか確認する
//
//  2. GetByID
//     対象 Inquiry の現在状態を取得する
//
//  3. ListByInquiryID
//     inquiries/{inquiryId}/replies/{replyId} を取得する
type InquiryQuery struct {
	repo      inquirydom.Repository
	replyRepo inquirydom.ReplyRepository
}

// NewInquiryQuery は InquiryQuery を初期化します。
func NewInquiryQuery(
	repo inquirydom.Repository,
	replyRepo inquirydom.ReplyRepository,
) *InquiryQuery {
	return &InquiryQuery{
		repo:      repo,
		replyRepo: replyRepo,
	}
}

// ListByAvatarID は avatar に紐づく Inquiry 一覧を取得します。
//
// Mall 側のチャット一覧 / 問い合わせ一覧で使います。
// avatarID は request body / query から受け取らず、middleware の AvatarContext から解決した値を渡します。
//
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

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return inquirydom.PageResult[inquirydom.Inquiry]{}, inquirydom.ErrInvalidAvatarID
	}

	filter.AvatarID = &avatarID

	if page.Number <= 0 {
		page.Number = 1
	}

	if page.PerPage <= 0 {
		page.PerPage = 100
	}

	return q.repo.ListByAvatarID(ctx, avatarID, filter, sort, page)
}

// GetByID は Inquiry を取得します。
//
// command 処理前の現在状態取得など、domain entity が必要な場合に使います。
func (q *InquiryQuery) GetByID(
	ctx context.Context,
	id string,
) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("mall inquiry query: repository is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

// GetByIDForAvatar は avatar 所有確認込みで Inquiry を取得します。
//
// ListByAvatarID で avatar scope を確認した後、GetByID で現在状態を取得します。
// 取得結果の AvatarID も念のため確認します。
func (q *InquiryQuery) GetByIDForAvatar(
	ctx context.Context,
	id string,
	avatarID string,
) (inquirydom.Inquiry, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	avatarID = strings.TrimSpace(avatarID)
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
		if strings.TrimSpace(item.ID) == id {
			found = true
			break
		}
	}

	if !found {
		return inquirydom.Inquiry{}, inquirydom.ErrInquiryForbidden
	}

	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	if strings.TrimSpace(inq.AvatarID) != avatarID {
		return inquirydom.Inquiry{}, inquirydom.ErrInquiryForbidden
	}

	return inq, nil
}

// ListByInquiryID は Inquiry の reply subcollection を取得します。
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
func (q *InquiryQuery) ListByInquiryID(
	ctx context.Context,
	inquiryID string,
) ([]inquirydom.Reply, error) {
	if q == nil || q.replyRepo == nil {
		return nil, fmt.Errorf("mall inquiry query: reply repository is nil")
	}

	inquiryID = strings.TrimSpace(inquiryID)
	if inquiryID == "" {
		return nil, inquirydom.ErrInvalidReplyInquiryID
	}

	return q.replyRepo.ListByInquiryID(ctx, inquiryID)
}

// ListRepliesByInquiryIDForAvatar は avatar 所有確認込みで reply 一覧を取得します。
//
// 処理順:
//
//  1. ListByAvatarID
//  2. GetByID
//  3. ListByInquiryID
//
// handler 側で reply 一覧を返す場合は、この method を呼びます。
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
