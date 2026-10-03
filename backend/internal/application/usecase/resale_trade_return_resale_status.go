// backend/internal/application/usecase/resale_trade_return_resale_status.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	resaledom "narratives/internal/domain/resale"
)

var (
	ErrResaleTradeReturnResaleStatusNotConfigured = errors.New(
		"resale trade return resale status: repository is not configured",
	)
	ErrResaleTradeReturnResaleStatusInvalidCompletedAt = errors.New(
		"resale trade return resale status: invalid completedAt",
	)
	ErrResaleTradeReturnResaleStatusConflict = errors.New(
		"resale trade return resale status: resale status conflict",
	)
)

// ensureReturnedResaleSuspended ensures that the Resale associated with a
// completed return is no longer kept in the sold state.
//
// Normal lifecycle:
//
//	listing
//	  -> purchase
//	  -> sold
//	  -> return completed
//	  -> suspended
//
// suspended is idempotently accepted because a previous execution may already
// have updated the Resale before a later operation failed.
//
// listing requires special handling for retry safety. Payment currently marks a
// purchased Resale sold on a best-effort basis, so a legacy or partially failed
// purchase may still leave the Resale listing when the return completes. Such a
// listing is suspended.
//
// However, if the Resale was updated to listing after the return completion
// timestamp, the seller may already have manually reviewed and republished the
// returned item. In that case this retry must not suspend it again.
func ensureReturnedResaleSuspended(
	ctx context.Context,
	resaleRepo resaledom.Repository,
	resaleID string,
	completedAt time.Time,
) (resaledom.Resale, error) {
	if resaleRepo == nil {
		return resaledom.Resale{},
			ErrResaleTradeReturnResaleStatusNotConfigured
	}

	resaleID = strings.TrimSpace(resaleID)
	if resaleID == "" {
		return resaledom.Resale{},
			resaledom.ErrInvalidID
	}

	if completedAt.IsZero() {
		return resaledom.Resale{},
			ErrResaleTradeReturnResaleStatusInvalidCompletedAt
	}

	completedAt = completedAt.UTC()

	item, err := resaleRepo.GetByID(
		ctx,
		resaleID,
	)
	if err != nil {
		return resaledom.Resale{}, err
	}

	if item.ID != resaleID {
		return resaledom.Resale{},
			resaledom.ErrNotFound
	}

	switch item.Status {
	case resaledom.StatusSuspended:
		return item, nil

	case resaledom.StatusSold:
		return suspendReturnedResale(
			ctx,
			resaleRepo,
			item,
			completedAt,
		)

	case resaledom.StatusListing:
		if resaleWasUpdatedAfterReturnCompletion(
			item,
			completedAt,
		) {
			return item, nil
		}

		return suspendReturnedResale(
			ctx,
			resaleRepo,
			item,
			completedAt,
		)

	default:
		return resaledom.Resale{},
			ErrResaleTradeReturnResaleStatusConflict
	}
}

func suspendReturnedResale(
	ctx context.Context,
	resaleRepo resaledom.Repository,
	item resaledom.Resale,
	completedAt time.Time,
) (resaledom.Resale, error) {
	if strings.TrimSpace(item.ID) == "" {
		return resaledom.Resale{},
			resaledom.ErrInvalidID
	}

	if err := item.Suspend(completedAt); err != nil {
		return resaledom.Resale{}, err
	}

	updated, err := resaleRepo.Update(
		ctx,
		item.ID,
		item,
	)
	if err != nil {
		return resaledom.Resale{}, err
	}

	if updated.ID != item.ID ||
		updated.Status != resaledom.StatusSuspended {
		return resaledom.Resale{},
			ErrResaleTradeReturnResaleStatusConflict
	}

	return updated, nil
}

func resaleWasUpdatedAfterReturnCompletion(
	item resaledom.Resale,
	completedAt time.Time,
) bool {
	if item.UpdatedAt == nil ||
		item.UpdatedAt.IsZero() {
		return false
	}

	return item.UpdatedAt.UTC().After(completedAt.UTC())
}
