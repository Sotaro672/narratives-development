// backend/internal/application/usecase/news_detail.go
package usecase

import (
	"context"

	newsdom "narratives/internal/domain/news"
)

// GetNewsForMember returns one News item together with the authenticated
// Console member's read state.
func (u *NewsUsecase) GetNewsForMember(
	ctx context.Context,
	newsID newsdom.NewsID,
	memberID string,
) (NewsRecipientItem, error) {
	return u.getNewsForRecipient(
		ctx,
		newsID,
		newsdom.RecipientTypeMember,
		memberID,
	)
}

func (u *NewsUsecase) getNewsForRecipient(
	ctx context.Context,
	newsID newsdom.NewsID,
	recipientType newsdom.RecipientType,
	recipientID string,
) (NewsRecipientItem, error) {
	if err := u.ensureNewsRepository(); err != nil {
		return NewsRecipientItem{}, err
	}
	if err := u.ensureNewsReadRepository(); err != nil {
		return NewsRecipientItem{}, err
	}
	if err := validateNewsRecipient(recipientType, recipientID); err != nil {
		return NewsRecipientItem{}, err
	}
	if newsID == "" {
		return NewsRecipientItem{}, newsdom.ErrInvalidID
	}

	entity, err := u.newsRepo.GetByID(ctx, newsID)
	if err != nil {
		return NewsRecipientItem{}, err
	}

	reads, err := u.newsReadRepo.GetMany(
		ctx,
		[]newsdom.NewsID{newsID},
		recipientType,
		recipientID,
	)
	if err != nil {
		return NewsRecipientItem{}, err
	}

	item := NewsRecipientItem{
		News:   entity,
		IsRead: false,
		ReadAt: nil,
	}

	for _, read := range reads {
		if read.NewsID != newsID {
			continue
		}

		readAt := read.ReadAt
		item.IsRead = true
		item.ReadAt = &readAt
		break
	}

	return item, nil
}
