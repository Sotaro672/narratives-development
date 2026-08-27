// backend/internal/adapters/out/firestore/refund_completion_notification_repository_fs.go
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

	refunddom "narratives/internal/domain/refund"
)

const (
	refundCompletionNotificationDeliveriesCollectionName = "refundCompletionNotificationDeliveries"
	refundCompletionNotificationAttemptError             = "maximum refund completion notification delivery attempts reached"
)

var ErrRefundCompletionNotificationRepositoryNotConfigured = errors.New(
	"refund_completion_notification_repository_fs: not configured",
)

type refundCompletionNotificationDeliveryDocument struct {
	PaymentID string `firestore:"paymentId"`
	OrderID   string `firestore:"orderId"`
	UserID    string `firestore:"userId"`

	StripeRefundID string `firestore:"stripeRefundId"`
	RefundedAmount int    `firestore:"refundedAmount"`

	Status       refunddom.CompletionNotificationStatus `firestore:"status"`
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

type RefundCompletionNotificationRepositoryFS struct {
	Client *firestore.Client
}

func NewRefundCompletionNotificationRepositoryFS(
	client *firestore.Client,
) *RefundCompletionNotificationRepositoryFS {
	return &RefundCompletionNotificationRepositoryFS{
		Client: client,
	}
}

func (r *RefundCompletionNotificationRepositoryFS) deliveriesCol() *firestore.CollectionRef {
	return r.Client.Collection(
		refundCompletionNotificationDeliveriesCollectionName,
	)
}

func (r *RefundCompletionNotificationRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
) (refunddom.CompletionNotificationDelivery, bool, error) {
	if r == nil || r.Client == nil {
		return refunddom.CompletionNotificationDelivery{},
			false,
			ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	normalized, err := delivery.Normalize()
	if err != nil {
		return refunddom.CompletionNotificationDelivery{}, false, err
	}

	deliveryRef := r.deliveriesCol().Doc(normalized.ID)

	var (
		result  refunddom.CompletionNotificationDelivery
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
				existing, err :=
					readRefundCompletionNotificationDeliverySnapshot(
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
					"get refund completion notification delivery %q: %w",
					normalized.ID,
					err,
				)
			}

			if err := tx.Create(
				deliveryRef,
				refundCompletionNotificationDeliveryToDocument(
					normalized,
				),
			); err != nil {
				return fmt.Errorf(
					"create refund completion notification delivery %q: %w",
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
		return refunddom.CompletionNotificationDelivery{},
			false,
			fmt.Errorf(
				"create refund completion notification delivery transaction: %w",
				err,
			)
	}

	return result, created, nil
}

func (r *RefundCompletionNotificationRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (refunddom.CompletionNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return refunddom.CompletionNotificationDelivery{},
			ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return refunddom.CompletionNotificationDelivery{},
			refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	doc, err := r.deliveriesCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return refunddom.CompletionNotificationDelivery{},
				refunddom.ErrNotFound
		}

		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf(
				"get refund completion notification delivery %q: %w",
				id,
				err,
			)
	}

	return readRefundCompletionNotificationDeliverySnapshot(doc)
}

func (r *RefundCompletionNotificationRepositoryFS) ListDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]refunddom.CompletionNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return nil,
			ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	now = now.UTC()

	if limit <= 0 {
		limit = 50
	}

	query := r.deliveriesCol().Where(
		"status",
		"in",
		[]string{
			string(refunddom.CompletionNotificationStatusPending),
			string(refunddom.CompletionNotificationStatusProcessing),
			string(refunddom.CompletionNotificationStatusRetryableFailed),
		},
	)

	iter := query.Documents(ctx)
	defer iter.Stop()

	deliveries := make(
		[]refunddom.CompletionNotificationDelivery,
		0,
	)

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"list refund completion notification deliveries: %w",
				err,
			)
		}

		delivery, err :=
			readRefundCompletionNotificationDeliverySnapshot(
				doc,
			)
		if err != nil {
			return nil, err
		}

		if delivery.AttemptCount >=
			delivery.MaxAttempts {
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
			left :=
				refundCompletionNotificationDueTime(
					deliveries[i],
				)

			right :=
				refundCompletionNotificationDueTime(
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
		deliveries =
			deliveries[:limit]
	}

	return deliveries, nil
}

func (r *RefundCompletionNotificationRepositoryFS) Claim(
	ctx context.Context,
	id string,
	now time.Time,
	processingUntil time.Time,
) (refunddom.CompletionNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return refunddom.CompletionNotificationDelivery{},
			ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return refunddom.CompletionNotificationDelivery{},
			refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	now = now.UTC()
	processingUntil = processingUntil.UTC()

	deliveryRef :=
		r.deliveriesCol().Doc(id)

	var (
		result       refunddom.CompletionNotificationDelivery
		committedErr error
	)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(
				deliveryRef,
			)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return refunddom.ErrNotFound
				}

				return fmt.Errorf(
					"get refund completion notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err :=
				readRefundCompletionNotificationDeliverySnapshot(
					deliveryDoc,
				)
			if err != nil {
				return err
			}

			if delivery.IsTerminal() {
				return refunddom.ErrCompletionNotificationNotClaimable
			}

			if delivery.AttemptCount >=
				delivery.MaxAttempts {
				failed, err :=
					delivery.MarkFailed(
						refundCompletionNotificationAttemptError,
						now,
					)
				if err != nil {
					return err
				}

				if err := tx.Set(
					deliveryRef,
					refundCompletionNotificationDeliveryToDocument(
						failed,
					),
				); err != nil {
					return fmt.Errorf(
						"fail refund completion notification delivery %q: %w",
						id,
						err,
					)
				}

				committedErr =
					refunddom.ErrCompletionNotificationAttemptLimit

				return nil
			}

			claimed, err :=
				delivery.Claim(
					now,
					processingUntil,
				)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				refundCompletionNotificationDeliveryToDocument(
					claimed,
				),
			); err != nil {
				return fmt.Errorf(
					"claim refund completion notification delivery %q: %w",
					id,
					err,
				)
			}

			result = claimed
			return nil
		},
	)
	if err != nil {
		return refunddom.CompletionNotificationDelivery{},
			mapRefundCompletionNotificationRepositoryError(
				"claim refund completion notification delivery",
				err,
			)
	}

	if committedErr != nil {
		return refunddom.CompletionNotificationDelivery{},
			committedErr
	}

	return result, nil
}

func (r *RefundCompletionNotificationRepositoryFS) MarkDelivered(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	providerMessageID string,
	deliveredAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	deliveredAt = deliveredAt.UTC()
	deliveryRef :=
		r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(
				deliveryRef,
			)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return refunddom.ErrNotFound
				}

				return fmt.Errorf(
					"get refund completion notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err :=
				readRefundCompletionNotificationDeliverySnapshot(
					deliveryDoc,
				)
			if err != nil {
				return err
			}

			if delivery.Status ==
				refunddom.CompletionNotificationStatusDelivered {
				return nil
			}

			if delivery.AttemptCount !=
				expectedAttemptCount ||
				delivery.Status !=
					refunddom.CompletionNotificationStatusProcessing {
				return refunddom.ErrCompletionNotificationNotClaimable
			}

			delivered, err :=
				delivery.MarkDelivered(
					providerMessageID,
					deliveredAt,
				)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				refundCompletionNotificationDeliveryToDocument(
					delivered,
				),
			); err != nil {
				return fmt.Errorf(
					"mark refund completion notification delivery %q delivered: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapRefundCompletionNotificationRepositoryError(
			"mark refund completion notification delivery delivered",
			err,
		)
	}

	return nil
}

func (r *RefundCompletionNotificationRepositoryFS) MarkRetryableFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	nextAttemptAt = nextAttemptAt.UTC()
	failedAt = failedAt.UTC()

	deliveryRef :=
		r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(
				deliveryRef,
			)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return refunddom.ErrNotFound
				}

				return fmt.Errorf(
					"get refund completion notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err :=
				readRefundCompletionNotificationDeliverySnapshot(
					deliveryDoc,
				)
			if err != nil {
				return err
			}

			if delivery.Status ==
				refunddom.CompletionNotificationStatusRetryableFailed &&
				delivery.AttemptCount ==
					expectedAttemptCount {
				return nil
			}

			if delivery.AttemptCount !=
				expectedAttemptCount ||
				delivery.Status !=
					refunddom.CompletionNotificationStatusProcessing {
				return refunddom.ErrCompletionNotificationNotClaimable
			}

			retryable, err :=
				delivery.MarkRetryableFailed(
					lastError,
					nextAttemptAt,
					failedAt,
				)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				refundCompletionNotificationDeliveryToDocument(
					retryable,
				),
			); err != nil {
				return fmt.Errorf(
					"mark refund completion notification delivery %q retryable failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapRefundCompletionNotificationRepositoryError(
			"mark refund completion notification delivery retryable failed",
			err,
		)
	}

	return nil
}

func (r *RefundCompletionNotificationRepositoryFS) MarkFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrRefundCompletionNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	failedAt = failedAt.UTC()
	deliveryRef :=
		r.deliveriesCol().Doc(id)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(
				deliveryRef,
			)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return refunddom.ErrNotFound
				}

				return fmt.Errorf(
					"get refund completion notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err :=
				readRefundCompletionNotificationDeliverySnapshot(
					deliveryDoc,
				)
			if err != nil {
				return err
			}

			if delivery.Status ==
				refunddom.CompletionNotificationStatusDelivered {
				return refunddom.ErrCompletionNotificationStatusInvalid
			}

			if delivery.Status ==
				refunddom.CompletionNotificationStatusFailed {
				return nil
			}

			if delivery.AttemptCount !=
				expectedAttemptCount {
				return refunddom.ErrCompletionNotificationNotClaimable
			}

			failed, err :=
				delivery.MarkFailed(
					lastError,
					failedAt,
				)
			if err != nil {
				return err
			}

			if err := tx.Set(
				deliveryRef,
				refundCompletionNotificationDeliveryToDocument(
					failed,
				),
			); err != nil {
				return fmt.Errorf(
					"mark refund completion notification delivery %q failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapRefundCompletionNotificationRepositoryError(
			"mark refund completion notification delivery failed",
			err,
		)
	}

	return nil
}

func refundCompletionNotificationDeliveryToDocument(
	delivery refunddom.CompletionNotificationDelivery,
) refundCompletionNotificationDeliveryDocument {
	return refundCompletionNotificationDeliveryDocument{
		PaymentID:         delivery.PaymentID,
		OrderID:           delivery.OrderID,
		UserID:            delivery.UserID,
		StripeRefundID:    delivery.StripeRefundID,
		RefundedAmount:    delivery.RefundedAmount,
		Status:            delivery.Status,
		AttemptCount:      delivery.AttemptCount,
		MaxAttempts:       delivery.MaxAttempts,
		ProviderMessageID: delivery.ProviderMessageID,
		LastError:         delivery.LastError,
		CreatedAt:         delivery.CreatedAt.UTC(),
		UpdatedAt:         delivery.UpdatedAt.UTC(),

		NextAttemptAt: copyRefundCompletionNotificationTimePointer(
			delivery.NextAttemptAt,
		),
		ProcessingStartedAt: copyRefundCompletionNotificationTimePointer(
			delivery.ProcessingStartedAt,
		),
		ProcessingUntil: copyRefundCompletionNotificationTimePointer(
			delivery.ProcessingUntil,
		),
		DeliveredAt: copyRefundCompletionNotificationTimePointer(
			delivery.DeliveredAt,
		),
		FailedAt: copyRefundCompletionNotificationTimePointer(
			delivery.FailedAt,
		),
	}
}

func readRefundCompletionNotificationDeliverySnapshot(
	doc *firestore.DocumentSnapshot,
) (refunddom.CompletionNotificationDelivery, error) {
	if doc == nil {
		return refunddom.CompletionNotificationDelivery{},
			errors.New(
				"refund completion notification delivery document snapshot is nil",
			)
	}

	var stored refundCompletionNotificationDeliveryDocument

	if err := doc.DataTo(&stored); err != nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf(
				"decode refund completion notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	delivery := refunddom.CompletionNotificationDelivery{
		ID:             doc.Ref.ID,
		PaymentID:      stored.PaymentID,
		OrderID:        stored.OrderID,
		UserID:         stored.UserID,
		StripeRefundID: stored.StripeRefundID,
		RefundedAmount: stored.RefundedAmount,

		Status:       stored.Status,
		AttemptCount: stored.AttemptCount,
		MaxAttempts:  stored.MaxAttempts,

		ProviderMessageID: stored.ProviderMessageID,
		LastError:         stored.LastError,

		CreatedAt: stored.CreatedAt,
		UpdatedAt: stored.UpdatedAt,

		NextAttemptAt:       stored.NextAttemptAt,
		ProcessingStartedAt: stored.ProcessingStartedAt,
		ProcessingUntil:     stored.ProcessingUntil,
		DeliveredAt:         stored.DeliveredAt,
		FailedAt:            stored.FailedAt,
	}

	normalized, err :=
		delivery.Normalize()
	if err != nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf(
				"normalize refund completion notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	return normalized, nil
}

func refundCompletionNotificationDueTime(
	delivery refunddom.CompletionNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case refunddom.CompletionNotificationStatusProcessing:
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

func copyRefundCompletionNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil ||
		value.IsZero() {
		return nil
	}

	normalized := value.UTC()

	return &normalized
}

func mapRefundCompletionNotificationRepositoryError(
	operation string,
	err error,
) error {
	switch {
	case errors.Is(
		err,
		refunddom.ErrNotFound,
	):
		return refunddom.ErrNotFound

	case errors.Is(
		err,
		refunddom.ErrCompletionNotificationNotClaimable,
	):
		return refunddom.ErrCompletionNotificationNotClaimable

	case errors.Is(
		err,
		refunddom.ErrCompletionNotificationAttemptLimit,
	):
		return refunddom.ErrCompletionNotificationAttemptLimit

	case errors.Is(
		err,
		refunddom.ErrCompletionNotificationStatusInvalid,
	):
		return refunddom.ErrCompletionNotificationStatusInvalid

	default:
		return fmt.Errorf(
			"%s: %w",
			operation,
			err,
		)
	}
}
