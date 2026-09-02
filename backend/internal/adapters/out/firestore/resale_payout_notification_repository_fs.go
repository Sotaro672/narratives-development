// backend/internal/adapters/out/firestore/resale_payout_notification_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	applicationport "narratives/internal/application/port"
	bankpayoutdom "narratives/internal/domain/bankPayout"
)

const resalePayoutNotificationDeliveriesCollectionName = "resalePayoutNotificationDeliveries"

var ErrResalePayoutNotificationRepositoryNotConfigured = errors.New(
	"resale_payout_notification_repository_fs: not configured",
)

type resalePayoutNotificationDeliveryDocument struct {
	BankPayoutID      string `firestore:"bankPayoutId"`
	SalesReceivableID string `firestore:"salesReceivableId"`
	OrderID           string `firestore:"orderId"`
	ResaleID          string `firestore:"resaleId"`
	UserID            string `firestore:"userId"`

	Amount   int    `firestore:"amount"`
	Currency string `firestore:"currency"`

	BankName   string `firestore:"bankName"`
	BranchName string `firestore:"branchName"`
	BankLast4  string `firestore:"bankLast4"`

	PaidAt time.Time `firestore:"paidAt"`

	Status       bankpayoutdom.PayoutNotificationStatus `firestore:"status"`
	AttemptCount int                                    `firestore:"attemptCount"`
	MaxAttempts  int                                    `firestore:"maxAttempts"`

	ProviderMessageID string `firestore:"providerMessageId"`
	LastError         string `firestore:"lastError"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`

	NextAttemptAt       *time.Time `firestore:"nextAttemptAt,omitempty"`
	ProcessingStartedAt *time.Time `firestore:"processingStartedAt,omitempty"`
	ProcessingUntil     *time.Time `firestore:"processingUntil,omitempty"`
	DeliveredAt         *time.Time `firestore:"deliveredAt,omitempty"`
	FailedAt            *time.Time `firestore:"failedAt,omitempty"`
}

type ResalePayoutNotificationRepositoryFS struct {
	Client *firestore.Client
}

var _ applicationport.ResalePayoutNotificationRepositoryPort = (*ResalePayoutNotificationRepositoryFS)(nil)

func NewResalePayoutNotificationRepositoryFS(
	client *firestore.Client,
) *ResalePayoutNotificationRepositoryFS {
	return &ResalePayoutNotificationRepositoryFS{
		Client: client,
	}
}

func (r *ResalePayoutNotificationRepositoryFS) deliveriesCol() *firestore.CollectionRef {
	return r.Client.Collection(
		resalePayoutNotificationDeliveriesCollectionName,
	)
}

func (r *ResalePayoutNotificationRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
) (bankpayoutdom.PayoutNotificationDelivery, bool, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			false,
			ErrResalePayoutNotificationRepositoryNotConfigured
	}

	normalized, err := delivery.Normalize()
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{}, false, err
	}

	deliveryRef := r.deliveriesCol().Doc(normalized.ID)

	var (
		result  bankpayoutdom.PayoutNotificationDelivery
		created bool
	)

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			existingDoc, err := tx.Get(deliveryRef)
			if err == nil {
				existing, err := readResalePayoutNotificationDeliverySnapshot(
					existingDoc,
				)
				if err != nil {
					return err
				}

				result = existing
				created = false
				return nil
			}

			if status.Code(err) != codes.NotFound {
				return fmt.Errorf(
					"get resale payout notification delivery %q: %w",
					normalized.ID,
					err,
				)
			}

			if err := tx.Create(
				deliveryRef,
				resalePayoutNotificationDeliveryToDocument(
					normalized,
				),
			); err != nil {
				return fmt.Errorf(
					"create resale payout notification delivery %q: %w",
					normalized.ID,
					err,
				)
			}

			result = normalized
			created = true
			return nil
		},
	)
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			false,
			fmt.Errorf(
				"create resale payout notification delivery transaction: %w",
				err,
			)
	}

	return result, created, nil
}

func (r *ResalePayoutNotificationRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (bankpayoutdom.PayoutNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			ErrResalePayoutNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return bankpayoutdom.PayoutNotificationDelivery{},
			bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	doc, err := r.deliveriesCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return bankpayoutdom.PayoutNotificationDelivery{},
				bankpayoutdom.ErrNotFound
		}

		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf(
				"get resale payout notification delivery %q: %w",
				id,
				err,
			)
	}

	return readResalePayoutNotificationDeliverySnapshot(doc)
}

func (r *ResalePayoutNotificationRepositoryFS) ListDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]bankpayoutdom.PayoutNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return nil,
			ErrResalePayoutNotificationRepositoryNotConfigured
	}

	now = now.UTC()

	if limit <= 0 {
		limit = 50
	}

	query := r.deliveriesCol().Where(
		"status",
		"in",
		[]string{
			string(bankpayoutdom.PayoutNotificationStatusPending),
			string(bankpayoutdom.PayoutNotificationStatusProcessing),
			string(bankpayoutdom.PayoutNotificationStatusRetryableFailed),
		},
	)

	iter := query.Documents(ctx)
	defer iter.Stop()

	deliveries := make(
		[]bankpayoutdom.PayoutNotificationDelivery,
		0,
	)

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"list resale payout notification deliveries: %w",
				err,
			)
		}

		delivery, err := readResalePayoutNotificationDeliverySnapshot(
			doc,
		)
		if err != nil {
			return nil, err
		}

		if delivery.AttemptCount >= delivery.MaxAttempts {
			continue
		}

		if !delivery.IsDue(now) {
			continue
		}

		deliveries = append(
			deliveries,
			delivery,
		)
	}

	sort.SliceStable(
		deliveries,
		func(i int, j int) bool {
			left := resalePayoutNotificationDueTime(
				deliveries[i],
			)
			right := resalePayoutNotificationDueTime(
				deliveries[j],
			)

			if left.Equal(right) {
				return deliveries[i].CreatedAt.Before(
					deliveries[j].CreatedAt,
				)
			}

			return left.Before(right)
		},
	)

	if len(deliveries) > limit {
		deliveries = deliveries[:limit]
	}

	return deliveries, nil
}

func (r *ResalePayoutNotificationRepositoryFS) Claim(
	ctx context.Context,
	id string,
	now time.Time,
	processingUntil time.Time,
) (bankpayoutdom.PayoutNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			ErrResalePayoutNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return bankpayoutdom.PayoutNotificationDelivery{},
			bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	now = now.UTC()
	processingUntil = processingUntil.UTC()

	deliveryRef := r.deliveriesCol().Doc(id)

	var result bankpayoutdom.PayoutNotificationDelivery

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return bankpayoutdom.ErrNotFound
				}

				return fmt.Errorf(
					"get resale payout notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readResalePayoutNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.IsTerminal() {
				return bankpayoutdom.ErrPayoutNotificationNotClaimable
			}

			if delivery.AttemptCount >= delivery.MaxAttempts {
				return bankpayoutdom.ErrPayoutNotificationAttemptLimit
			}

			claimed, err := delivery.Claim(
				now,
				processingUntil,
			)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				resalePayoutNotificationDeliveryToDocument(
					claimed,
				),
			); err != nil {
				return fmt.Errorf(
					"claim resale payout notification delivery %q: %w",
					id,
					err,
				)
			}

			result = claimed
			return nil
		},
	)
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			mapResalePayoutNotificationRepositoryError(
				"claim resale payout notification delivery",
				err,
			)
	}

	return result, nil
}

func (r *ResalePayoutNotificationRepositoryFS) MarkDelivered(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	providerMessageID string,
	deliveredAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrResalePayoutNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	deliveredAt = deliveredAt.UTC()
	deliveryRef := r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return bankpayoutdom.ErrNotFound
				}

				return fmt.Errorf(
					"get resale payout notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readResalePayoutNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status ==
				bankpayoutdom.PayoutNotificationStatusDelivered {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status != bankpayoutdom.PayoutNotificationStatusProcessing {
				return bankpayoutdom.ErrPayoutNotificationNotClaimable
			}

			delivered, err := delivery.MarkDelivered(
				providerMessageID,
				deliveredAt,
			)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				resalePayoutNotificationDeliveryToDocument(
					delivered,
				),
			); err != nil {
				return fmt.Errorf(
					"mark resale payout notification delivery %q delivered: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapResalePayoutNotificationRepositoryError(
			"mark resale payout notification delivery delivered",
			err,
		)
	}

	return nil
}

func (r *ResalePayoutNotificationRepositoryFS) MarkRetryableFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrResalePayoutNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	nextAttemptAt = nextAttemptAt.UTC()
	failedAt = failedAt.UTC()

	deliveryRef := r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return bankpayoutdom.ErrNotFound
				}

				return fmt.Errorf(
					"get resale payout notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readResalePayoutNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status ==
				bankpayoutdom.PayoutNotificationStatusRetryableFailed &&
				delivery.AttemptCount == expectedAttemptCount {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status != bankpayoutdom.PayoutNotificationStatusProcessing {
				return bankpayoutdom.ErrPayoutNotificationNotClaimable
			}

			retryable, err := delivery.MarkRetryableFailed(
				lastError,
				nextAttemptAt,
				failedAt,
			)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				resalePayoutNotificationDeliveryToDocument(
					retryable,
				),
			); err != nil {
				return fmt.Errorf(
					"mark resale payout notification delivery %q retryable failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapResalePayoutNotificationRepositoryError(
			"mark resale payout notification delivery retryable failed",
			err,
		)
	}

	return nil
}

func (r *ResalePayoutNotificationRepositoryFS) MarkFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrResalePayoutNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	failedAt = failedAt.UTC()
	deliveryRef := r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return bankpayoutdom.ErrNotFound
				}

				return fmt.Errorf(
					"get resale payout notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readResalePayoutNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status ==
				bankpayoutdom.PayoutNotificationStatusDelivered {
				return bankpayoutdom.ErrPayoutNotificationStatusInvalid
			}

			if delivery.Status ==
				bankpayoutdom.PayoutNotificationStatusFailed {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount {
				return bankpayoutdom.ErrPayoutNotificationNotClaimable
			}

			failed, err := delivery.MarkFailed(
				lastError,
				failedAt,
			)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				resalePayoutNotificationDeliveryToDocument(
					failed,
				),
			); err != nil {
				return fmt.Errorf(
					"mark resale payout notification delivery %q failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapResalePayoutNotificationRepositoryError(
			"mark resale payout notification delivery failed",
			err,
		)
	}

	return nil
}

func resalePayoutNotificationDeliveryToDocument(
	delivery bankpayoutdom.PayoutNotificationDelivery,
) resalePayoutNotificationDeliveryDocument {
	return resalePayoutNotificationDeliveryDocument{
		BankPayoutID:      delivery.BankPayoutID,
		SalesReceivableID: delivery.SalesReceivableID,
		OrderID:           delivery.OrderID,
		ResaleID:          delivery.ResaleID,
		UserID:            delivery.UserID,

		Amount:   delivery.Amount,
		Currency: delivery.Currency,

		BankName:   delivery.BankName,
		BranchName: delivery.BranchName,
		BankLast4:  delivery.BankLast4,

		PaidAt: delivery.PaidAt.UTC(),

		Status:       delivery.Status,
		AttemptCount: delivery.AttemptCount,
		MaxAttempts:  delivery.MaxAttempts,

		ProviderMessageID: delivery.ProviderMessageID,
		LastError:         delivery.LastError,

		CreatedAt: delivery.CreatedAt.UTC(),
		UpdatedAt: delivery.UpdatedAt.UTC(),

		NextAttemptAt: copyResalePayoutNotificationTimePointer(
			delivery.NextAttemptAt,
		),
		ProcessingStartedAt: copyResalePayoutNotificationTimePointer(
			delivery.ProcessingStartedAt,
		),
		ProcessingUntil: copyResalePayoutNotificationTimePointer(
			delivery.ProcessingUntil,
		),
		DeliveredAt: copyResalePayoutNotificationTimePointer(
			delivery.DeliveredAt,
		),
		FailedAt: copyResalePayoutNotificationTimePointer(
			delivery.FailedAt,
		),
	}
}

func readResalePayoutNotificationDeliverySnapshot(
	doc *firestore.DocumentSnapshot,
) (bankpayoutdom.PayoutNotificationDelivery, error) {
	if doc == nil || doc.Ref == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			errors.New(
				"resale payout notification delivery document snapshot is nil",
			)
	}

	var stored resalePayoutNotificationDeliveryDocument

	if err := doc.DataTo(&stored); err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf(
				"decode resale payout notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	delivery := bankpayoutdom.PayoutNotificationDelivery{
		ID:                  doc.Ref.ID,
		BankPayoutID:        stored.BankPayoutID,
		SalesReceivableID:   stored.SalesReceivableID,
		OrderID:             stored.OrderID,
		ResaleID:            stored.ResaleID,
		UserID:              stored.UserID,
		Amount:              stored.Amount,
		Currency:            stored.Currency,
		BankName:            stored.BankName,
		BranchName:          stored.BranchName,
		BankLast4:           stored.BankLast4,
		PaidAt:              stored.PaidAt,
		Status:              stored.Status,
		AttemptCount:        stored.AttemptCount,
		MaxAttempts:         stored.MaxAttempts,
		ProviderMessageID:   stored.ProviderMessageID,
		LastError:           stored.LastError,
		CreatedAt:           stored.CreatedAt,
		UpdatedAt:           stored.UpdatedAt,
		NextAttemptAt:       stored.NextAttemptAt,
		ProcessingStartedAt: stored.ProcessingStartedAt,
		ProcessingUntil:     stored.ProcessingUntil,
		DeliveredAt:         stored.DeliveredAt,
		FailedAt:            stored.FailedAt,
	}

	normalized, err := delivery.Normalize()
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf(
				"normalize resale payout notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	return normalized, nil
}

func resalePayoutNotificationDueTime(
	delivery bankpayoutdom.PayoutNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case bankpayoutdom.PayoutNotificationStatusProcessing:
		if delivery.ProcessingUntil != nil {
			return delivery.ProcessingUntil.UTC()
		}

	default:
		if delivery.NextAttemptAt != nil {
			return delivery.NextAttemptAt.UTC()
		}
	}

	return delivery.CreatedAt.UTC()
}

func copyResalePayoutNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}

func mapResalePayoutNotificationRepositoryError(
	operation string,
	err error,
) error {
	switch {
	case errors.Is(
		err,
		bankpayoutdom.ErrNotFound,
	):
		return bankpayoutdom.ErrNotFound

	case errors.Is(
		err,
		bankpayoutdom.ErrPayoutNotificationNotClaimable,
	):
		return bankpayoutdom.ErrPayoutNotificationNotClaimable

	case errors.Is(
		err,
		bankpayoutdom.ErrPayoutNotificationAttemptLimit,
	):
		return bankpayoutdom.ErrPayoutNotificationAttemptLimit

	case errors.Is(
		err,
		bankpayoutdom.ErrPayoutNotificationStatusInvalid,
	):
		return bankpayoutdom.ErrPayoutNotificationStatusInvalid

	default:
		return fmt.Errorf(
			"%s: %w",
			operation,
			err,
		)
	}
}
