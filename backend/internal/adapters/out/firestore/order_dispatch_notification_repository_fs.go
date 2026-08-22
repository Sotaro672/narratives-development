// backend/internal/adapters/out/firestore/order_dispatch_notification_repository_fs.go
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

	orderdom "narratives/internal/domain/order"
)

const (
	orderDispatchNotificationDeliveriesCollectionName = "orderDispatchNotificationDeliveries"
	orderDispatchNotificationAttemptError             = "maximum dispatch notification delivery attempts reached"
)

var ErrOrderDispatchNotificationRepositoryNotConfigured = errors.New(
	"order_dispatch_notification_repository_fs: not configured",
)

type orderDispatchNotificationItemDocument struct {
	InventoryID        string `firestore:"inventoryId"`
	ListID             string `firestore:"listId"`
	ProductBlueprintID string `firestore:"productBlueprintId"`
	TokenBlueprintID   string `firestore:"tokenBlueprintId"`
	Qty                int    `firestore:"qty"`
}

type orderDispatchNotificationDeliveryDocument struct {
	OrderID   string `firestore:"orderId"`
	CompanyID string `firestore:"companyId"`
	UserID    string `firestore:"userId"`

	Items []orderDispatchNotificationItemDocument `firestore:"items"`

	Status       orderdom.DispatchNotificationStatus `firestore:"status"`
	AttemptCount int                                 `firestore:"attemptCount"`
	MaxAttempts  int                                 `firestore:"maxAttempts"`

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

type OrderDispatchNotificationRepositoryFS struct {
	Client *firestore.Client
}

var _ orderdom.DispatchNotificationRepository = (*OrderDispatchNotificationRepositoryFS)(nil)

func NewOrderDispatchNotificationRepositoryFS(
	client *firestore.Client,
) *OrderDispatchNotificationRepositoryFS {
	return &OrderDispatchNotificationRepositoryFS{
		Client: client,
	}
}

func (r *OrderDispatchNotificationRepositoryFS) deliveriesCol() *firestore.CollectionRef {
	return r.Client.Collection(
		orderDispatchNotificationDeliveriesCollectionName,
	)
}

func (r *OrderDispatchNotificationRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
) (orderdom.DispatchNotificationDelivery, bool, error) {
	if r == nil || r.Client == nil {
		return orderdom.DispatchNotificationDelivery{},
			false,
			ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	normalized, err := delivery.Normalize()
	if err != nil {
		return orderdom.DispatchNotificationDelivery{}, false, err
	}

	deliveryRef := r.deliveriesCol().Doc(normalized.ID)

	var (
		result  orderdom.DispatchNotificationDelivery
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
				existing, err := readOrderDispatchNotificationDeliverySnapshot(
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
					"get order dispatch notification delivery %q: %w",
					normalized.ID,
					err,
				)
			}

			if err := tx.Create(
				deliveryRef,
				orderDispatchNotificationDeliveryToDocument(normalized),
			); err != nil {
				return fmt.Errorf(
					"create order dispatch notification delivery %q: %w",
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
		return orderdom.DispatchNotificationDelivery{},
			false,
			fmt.Errorf(
				"create order dispatch notification delivery transaction: %w",
				err,
			)
	}

	return result, created, nil
}

func (r *OrderDispatchNotificationRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (orderdom.DispatchNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return orderdom.DispatchNotificationDelivery{},
			ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrDispatchNotificationDeliveryIDRequired
	}

	doc, err := r.deliveriesCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return orderdom.DispatchNotificationDelivery{},
				orderdom.ErrNotFound
		}

		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"get order dispatch notification delivery %q: %w",
				id,
				err,
			)
	}

	return readOrderDispatchNotificationDeliverySnapshot(doc)
}

func (r *OrderDispatchNotificationRepositoryFS) ListDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]orderdom.DispatchNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return nil, ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	now = now.UTC()

	if limit <= 0 {
		limit = 50
	}

	query := r.deliveriesCol().Where(
		"status",
		"in",
		[]string{
			string(orderdom.DispatchNotificationStatusPending),
			string(orderdom.DispatchNotificationStatusProcessing),
			string(orderdom.DispatchNotificationStatusRetryableFailed),
		},
	)

	iter := query.Documents(ctx)
	defer iter.Stop()

	deliveries := make(
		[]orderdom.DispatchNotificationDelivery,
		0,
	)

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"list order dispatch notification deliveries: %w",
				err,
			)
		}

		delivery, err := readOrderDispatchNotificationDeliverySnapshot(doc)
		if err != nil {
			return nil, err
		}

		if delivery.AttemptCount >= delivery.MaxAttempts {
			continue
		}

		if !delivery.IsDue(now) {
			continue
		}

		deliveries = append(deliveries, delivery)
	}

	sort.SliceStable(
		deliveries,
		func(i int, j int) bool {
			left := orderDispatchNotificationDueTime(deliveries[i])
			right := orderDispatchNotificationDueTime(deliveries[j])

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

func (r *OrderDispatchNotificationRepositoryFS) Claim(
	ctx context.Context,
	id string,
	now time.Time,
	processingUntil time.Time,
) (orderdom.DispatchNotificationDelivery, error) {
	if r == nil || r.Client == nil {
		return orderdom.DispatchNotificationDelivery{},
			ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrDispatchNotificationDeliveryIDRequired
	}

	now = now.UTC()
	processingUntil = processingUntil.UTC()

	deliveryRef := r.deliveriesCol().Doc(id)

	var (
		result       orderdom.DispatchNotificationDelivery
		committedErr error
	)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return orderdom.ErrNotFound
				}

				return fmt.Errorf(
					"get order dispatch notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readOrderDispatchNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.IsTerminal() {
				return orderdom.ErrDispatchNotificationNotClaimable
			}

			if delivery.AttemptCount >= delivery.MaxAttempts {
				failed, err := delivery.MarkFailed(
					orderDispatchNotificationAttemptError,
					now,
				)
				if err != nil {
					return err
				}

				if err := tx.Set(
					deliveryRef,
					orderDispatchNotificationDeliveryToDocument(failed),
				); err != nil {
					return fmt.Errorf(
						"fail order dispatch notification delivery %q: %w",
						id,
						err,
					)
				}

				committedErr = orderdom.ErrDispatchNotificationAttemptLimit
				return nil
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
				orderDispatchNotificationDeliveryToDocument(claimed),
			); err != nil {
				return fmt.Errorf(
					"claim order dispatch notification delivery %q: %w",
					id,
					err,
				)
			}

			result = claimed
			return nil
		},
	)
	if err != nil {
		return orderdom.DispatchNotificationDelivery{},
			mapOrderDispatchNotificationRepositoryError(
				"claim order dispatch notification delivery",
				err,
			)
	}

	if committedErr != nil {
		return orderdom.DispatchNotificationDelivery{}, committedErr
	}

	return result, nil
}

func (r *OrderDispatchNotificationRepositoryFS) MarkDelivered(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	providerMessageID string,
	deliveredAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.ErrDispatchNotificationDeliveryIDRequired
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
					return orderdom.ErrNotFound
				}

				return fmt.Errorf(
					"get order dispatch notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readOrderDispatchNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status == orderdom.DispatchNotificationStatusDelivered {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status != orderdom.DispatchNotificationStatusProcessing {
				return orderdom.ErrDispatchNotificationNotClaimable
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
				orderDispatchNotificationDeliveryToDocument(delivered),
			); err != nil {
				return fmt.Errorf(
					"mark order dispatch notification delivery %q delivered: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapOrderDispatchNotificationRepositoryError(
			"mark order dispatch notification delivery delivered",
			err,
		)
	}

	return nil
}

func (r *OrderDispatchNotificationRepositoryFS) MarkRetryableFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.ErrDispatchNotificationDeliveryIDRequired
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
					return orderdom.ErrNotFound
				}

				return fmt.Errorf(
					"get order dispatch notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readOrderDispatchNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status ==
				orderdom.DispatchNotificationStatusRetryableFailed &&
				delivery.AttemptCount == expectedAttemptCount {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status != orderdom.DispatchNotificationStatusProcessing {
				return orderdom.ErrDispatchNotificationNotClaimable
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
				orderDispatchNotificationDeliveryToDocument(retryable),
			); err != nil {
				return fmt.Errorf(
					"mark order dispatch notification delivery %q retryable failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapOrderDispatchNotificationRepositoryError(
			"mark order dispatch notification delivery retryable failed",
			err,
		)
	}

	return nil
}

func (r *OrderDispatchNotificationRepositoryFS) MarkFailed(
	ctx context.Context,
	id string,
	expectedAttemptCount int,
	lastError string,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return ErrOrderDispatchNotificationRepositoryNotConfigured
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.ErrDispatchNotificationDeliveryIDRequired
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
					return orderdom.ErrNotFound
				}

				return fmt.Errorf(
					"get order dispatch notification delivery %q: %w",
					id,
					err,
				)
			}

			delivery, err := readOrderDispatchNotificationDeliverySnapshot(
				deliveryDoc,
			)
			if err != nil {
				return err
			}

			if delivery.Status == orderdom.DispatchNotificationStatusDelivered {
				return orderdom.ErrDispatchNotificationStatusInvalid
			}

			if delivery.Status == orderdom.DispatchNotificationStatusFailed {
				return nil
			}

			if delivery.AttemptCount != expectedAttemptCount {
				return orderdom.ErrDispatchNotificationNotClaimable
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
				orderDispatchNotificationDeliveryToDocument(failed),
			); err != nil {
				return fmt.Errorf(
					"mark order dispatch notification delivery %q failed: %w",
					id,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		return mapOrderDispatchNotificationRepositoryError(
			"mark order dispatch notification delivery failed",
			err,
		)
	}

	return nil
}

func orderDispatchNotificationDeliveryToDocument(
	delivery orderdom.DispatchNotificationDelivery,
) orderDispatchNotificationDeliveryDocument {
	items := make(
		[]orderDispatchNotificationItemDocument,
		0,
		len(delivery.Items),
	)

	for _, item := range delivery.Items {
		items = append(
			items,
			orderDispatchNotificationItemDocument{
				InventoryID:        item.InventoryID,
				ListID:             item.ListID,
				ProductBlueprintID: item.ProductBlueprintID,
				TokenBlueprintID:   item.TokenBlueprintID,
				Qty:                item.Qty,
			},
		)
	}

	return orderDispatchNotificationDeliveryDocument{
		OrderID:           delivery.OrderID,
		CompanyID:         delivery.CompanyID,
		UserID:            delivery.UserID,
		Items:             items,
		Status:            delivery.Status,
		AttemptCount:      delivery.AttemptCount,
		MaxAttempts:       delivery.MaxAttempts,
		ProviderMessageID: delivery.ProviderMessageID,
		LastError:         delivery.LastError,
		CreatedAt:         delivery.CreatedAt.UTC(),
		UpdatedAt:         delivery.UpdatedAt.UTC(),
		NextAttemptAt: copyOrderDispatchNotificationTimePointer(
			delivery.NextAttemptAt,
		),
		ProcessingStartedAt: copyOrderDispatchNotificationTimePointer(
			delivery.ProcessingStartedAt,
		),
		ProcessingUntil: copyOrderDispatchNotificationTimePointer(
			delivery.ProcessingUntil,
		),
		DeliveredAt: copyOrderDispatchNotificationTimePointer(
			delivery.DeliveredAt,
		),
		FailedAt: copyOrderDispatchNotificationTimePointer(
			delivery.FailedAt,
		),
	}
}

func readOrderDispatchNotificationDeliverySnapshot(
	doc *firestore.DocumentSnapshot,
) (orderdom.DispatchNotificationDelivery, error) {
	if doc == nil {
		return orderdom.DispatchNotificationDelivery{},
			errors.New(
				"order dispatch notification delivery document snapshot is nil",
			)
	}

	var stored orderDispatchNotificationDeliveryDocument
	if err := doc.DataTo(&stored); err != nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"decode order dispatch notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	items := make(
		[]orderdom.DispatchNotificationItem,
		0,
		len(stored.Items),
	)

	for _, item := range stored.Items {
		items = append(
			items,
			orderdom.DispatchNotificationItem{
				InventoryID:        item.InventoryID,
				ListID:             item.ListID,
				ProductBlueprintID: item.ProductBlueprintID,
				TokenBlueprintID:   item.TokenBlueprintID,
				Qty:                item.Qty,
			},
		)
	}

	delivery := orderdom.DispatchNotificationDelivery{
		ID:                  doc.Ref.ID,
		OrderID:             stored.OrderID,
		CompanyID:           stored.CompanyID,
		UserID:              stored.UserID,
		Items:               items,
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
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"normalize order dispatch notification delivery %q: %w",
				doc.Ref.ID,
				err,
			)
	}

	return normalized, nil
}

func orderDispatchNotificationDueTime(
	delivery orderdom.DispatchNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case orderdom.DispatchNotificationStatusProcessing:
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

func copyOrderDispatchNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}

func mapOrderDispatchNotificationRepositoryError(
	operation string,
	err error,
) error {
	switch {
	case errors.Is(err, orderdom.ErrNotFound):
		return orderdom.ErrNotFound

	case errors.Is(
		err,
		orderdom.ErrDispatchNotificationNotClaimable,
	):
		return orderdom.ErrDispatchNotificationNotClaimable

	case errors.Is(
		err,
		orderdom.ErrDispatchNotificationAttemptLimit,
	):
		return orderdom.ErrDispatchNotificationAttemptLimit

	case errors.Is(
		err,
		orderdom.ErrDispatchNotificationStatusInvalid,
	):
		return orderdom.ErrDispatchNotificationStatusInvalid

	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
