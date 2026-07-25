// backend/internal/adapters/out/firestore/invitation_delivery_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	invdom "narratives/internal/domain/invitation"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	invitationDeliveriesCollectionName      = "invitationDeliveries"
	invitationDeliveryActivesCollectionName = "invitationDeliveryActives"
	invitationTokenTTL                      = 7 * 24 * time.Hour
	invitationDeliverySupersededError       = "superseded by another active invitation delivery"
	invitationDeliveryAttemptError          = "maximum invitation delivery attempts reached"
	invitationDeliveryTokenError            = "invitation delivery token is not available"
)

type invitationDeliveryDocument struct {
	Token               string                `firestore:"token"`
	MemberID            string                `firestore:"memberId"`
	CompanyID           string                `firestore:"companyId"`
	AssignedBrandIDs    []string              `firestore:"assignedBrands"`
	Permissions         []string              `firestore:"permissions"`
	Email               string                `firestore:"email"`
	Status              invdom.DeliveryStatus `firestore:"status"`
	AttemptCount        int                   `firestore:"attemptCount"`
	MaxAttempts         int                   `firestore:"maxAttempts"`
	LastError           string                `firestore:"lastError"`
	ProviderMessageID   string                `firestore:"providerMessageId"`
	CreatedAt           time.Time             `firestore:"createdAt"`
	UpdatedAt           time.Time             `firestore:"updatedAt"`
	NextAttemptAt       *time.Time            `firestore:"nextAttemptAt,omitempty"`
	ProcessingStartedAt *time.Time            `firestore:"processingStartedAt,omitempty"`
	ProcessingUntil     *time.Time            `firestore:"processingUntil,omitempty"`
	DeliveredAt         *time.Time            `firestore:"deliveredAt,omitempty"`
	FailedAt            *time.Time            `firestore:"failedAt,omitempty"`
}
type invitationDeliveryActiveDocument struct {
	DeliveryID string    `firestore:"deliveryId"`
	Token      string    `firestore:"token"`
	CreatedAt  time.Time `firestore:"createdAt"`
	UpdatedAt  time.Time `firestore:"updatedAt"`
}
type invitationDeliveryRecord struct {
	Ref         *firestore.DocumentRef
	Delivery    invdom.InvitationDelivery
	TokenRef    *firestore.DocumentRef
	Token       invdom.InvitationToken
	TokenExists bool
}

func (r *InvitationTokenRepositoryFS) deliveriesCol() *firestore.CollectionRef {
	return r.Client.Collection(invitationDeliveriesCollectionName)
}
func (r *InvitationTokenRepositoryFS) deliveryActivesCol() *firestore.CollectionRef {
	return r.Client.Collection(invitationDeliveryActivesCollectionName)
}

var _ invdom.DeliveryRepository = (*InvitationTokenRepositoryFS)(nil)

// CreateOrReuseInvitationDeliveryは、Memberごとの有効なdeliveryとtokenを
// 1件に集約します。
//
// pending、processing、retryable_failed、または未使用のdeliveredが
// 既に存在する場合は、そのdeliveryとtokenを再利用します。
//
// 有効なdeliveryが存在しない場合は、tokenとdeliveryを同一transactionで
// 新規作成します。
func (r *InvitationTokenRepositoryFS) CreateOrReuseInvitationDelivery(
	ctx context.Context,
	info invdom.InvitationInfo,
) (invdom.InvitationDelivery, error) {
	if r == nil || r.Client == nil {
		return invdom.InvitationDelivery{},
			errors.New("firestore client is nil")
	}
	normalizedInfo, err := info.Normalize()
	if err != nil {
		return invdom.InvitationDelivery{}, err
	}
	newDeliveryRef := r.deliveriesCol().NewDoc()
	newRawToken := r.col().NewDoc().ID
	newTokenRef := r.col().Doc(
		invitationTokenDocumentID(newRawToken),
	)
	activeRef := r.deliveryActivesCol().Doc(
		normalizedInfo.MemberID,
	)
	var result invdom.InvitationDelivery
	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			now := time.Now().UTC()
			activeDocument, activeExists, err :=
				getInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
				)
			if err != nil {
				return err
			}
			records, err :=
				r.listMemberInvitationDeliveriesInTransaction(
					tx,
					normalizedInfo.MemberID,
				)
			if err != nil {
				return err
			}
			sort.SliceStable(
				records,
				func(i int, j int) bool {
					left := records[i].Delivery.UpdatedAt
					right := records[j].Delivery.UpdatedAt
					if left.Equal(right) {
						return records[i].Delivery.CreatedAt.After(
							records[j].Delivery.CreatedAt,
						)
					}
					return left.After(right)
				},
			)
			selectedIndex := -1
			if activeExists {
				for index := range records {
					if records[index].Delivery.ID !=
						activeDocument.DeliveryID {
						continue
					}
					if records[index].Delivery.Token !=
						activeDocument.Token {
						continue
					}
					if reusableInvitationDeliveryRecord(
						records[index],
						normalizedInfo,
						now,
					) {
						selectedIndex = index
					}
					break
				}
			}
			if selectedIndex < 0 {
				for index := range records {
					if reusableInvitationDeliveryRecord(
						records[index],
						normalizedInfo,
						now,
					) {
						selectedIndex = index
						break
					}
				}
			}
			for index := range records {
				if index == selectedIndex {
					continue
				}
				if !revocableInvitationDeliveryRecord(
					records[index],
					now,
				) {
					continue
				}
				if err := revokeInvitationDeliveryRecord(
					tx,
					records[index],
					invitationDeliverySupersededError,
					now,
				); err != nil {
					return err
				}
			}
			if selectedIndex >= 0 {
				selected := records[selectedIndex]
				updatedDelivery, err :=
					updateInvitationDeliverySnapshot(
						selected.Delivery,
						normalizedInfo,
						now,
					)
				if err != nil {
					return err
				}
				updatedToken, err :=
					updateInvitationTokenSnapshot(
						selected.Token,
						updatedDelivery.ID,
						normalizedInfo,
						now,
					)
				if err != nil {
					return err
				}
				if err := tx.Set(
					selected.Ref,
					invitationDeliveryToDocument(
						updatedDelivery,
					),
				); err != nil {
					return fmt.Errorf(
						"update invitation delivery %q: %w",
						updatedDelivery.ID,
						err,
					)
				}
				if err := tx.Set(
					selected.TokenRef,
					invitationTokenToDocument(
						updatedToken,
						now,
					),
				); err != nil {
					return fmt.Errorf(
						"update invitation token %q: %w",
						updatedToken.Token,
						err,
					)
				}
				activeCreatedAt :=
					invitationDeliveryActiveCreatedAt(
						activeDocument,
						activeExists &&
							activeDocument.DeliveryID ==
								updatedDelivery.ID,
						now,
					)
				if err := tx.Set(
					activeRef,
					invitationDeliveryActiveDocument{
						DeliveryID: updatedDelivery.ID,
						Token:      updatedDelivery.Token,
						CreatedAt:  activeCreatedAt,
						UpdatedAt:  now,
					},
				); err != nil {
					return fmt.Errorf(
						"save active invitation delivery for member %q: %w",
						normalizedInfo.MemberID,
						err,
					)
				}
				result = updatedDelivery
				return nil
			}
			newDelivery, err := invdom.NewInvitationDelivery(
				newDeliveryRef.ID,
				newRawToken,
				normalizedInfo,
				now,
				invdom.DefaultInvitationDeliveryMaxAttempts,
			)
			if err != nil {
				return fmt.Errorf(
					"create invitation delivery entity: %w",
					err,
				)
			}
			expiresAt := now.Add(invitationTokenTTL)
			newToken, err := invdom.NewInvitationToken(
				newRawToken,
				newDeliveryRef.ID,
				normalizedInfo,
				now,
				&expiresAt,
			)
			if err != nil {
				return fmt.Errorf(
					"create invitation token entity: %w",
					err,
				)
			}
			if err := tx.Create(
				newDeliveryRef,
				invitationDeliveryToDocument(newDelivery),
			); err != nil {
				return fmt.Errorf(
					"create invitation delivery %q: %w",
					newDeliveryRef.ID,
					err,
				)
			}
			if err := tx.Create(
				newTokenRef,
				invitationTokenToDocument(
					newToken,
					now,
				),
			); err != nil {
				return fmt.Errorf(
					"create invitation token %q: %w",
					newTokenRef.ID,
					err,
				)
			}
			if err := tx.Set(
				activeRef,
				invitationDeliveryActiveDocument{
					DeliveryID: newDeliveryRef.ID,
					Token:      newRawToken,
					CreatedAt:  now,
					UpdatedAt:  now,
				},
			); err != nil {
				return fmt.Errorf(
					"save active invitation delivery for member %q: %w",
					normalizedInfo.MemberID,
					err,
				)
			}
			result = newDelivery
			return nil
		},
	)
	if err != nil {
		return invdom.InvitationDelivery{}, fmt.Errorf(
			"create or reuse invitation delivery transaction: %w",
			err,
		)
	}
	return result, nil
}

// ListDueInvitationDeliveriesは、処理時刻を迎えたdeliveryを返します。
func (r *InvitationTokenRepositoryFS) ListDueInvitationDeliveries(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]invdom.InvitationDelivery, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	now = now.UTC()
	if limit <= 0 {
		limit = 50
	}
	query := r.deliveriesCol().
		Where(
			"status",
			"in",
			[]string{
				string(invdom.InvitationDeliveryStatusPending),
				string(invdom.InvitationDeliveryStatusProcessing),
				string(invdom.InvitationDeliveryStatusRetryableFailed),
			},
		)
	iter := query.Documents(ctx)
	defer iter.Stop()
	deliveries := make([]invdom.InvitationDelivery, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"list invitation deliveries: %w",
				err,
			)
		}
		delivery, err := readInvitationDeliverySnapshot(doc)
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
			left := invitationDeliveryDueTime(
				deliveries[i],
			)
			right := invitationDeliveryDueTime(
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

// ClaimInvitationDeliveryは、送信対象deliveryをprocessingへ変更します。
func (r *InvitationTokenRepositoryFS) ClaimInvitationDelivery(
	ctx context.Context,
	deliveryID string,
	now time.Time,
	processingUntil time.Time,
) (invdom.InvitationDelivery, error) {
	if r == nil || r.Client == nil {
		return invdom.InvitationDelivery{},
			errors.New("firestore client is nil")
	}
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return invdom.InvitationDelivery{},
			invdom.ErrInvitationDeliveryIDRequired
	}
	now = now.UTC()
	processingUntil = processingUntil.UTC()
	deliveryRef := r.deliveriesCol().Doc(deliveryID)
	var (
		result       invdom.InvitationDelivery
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
					return invdom.ErrInvitationDeliveryNotFound
				}
				return fmt.Errorf(
					"get invitation delivery %q: %w",
					deliveryID,
					err,
				)
			}
			delivery, err :=
				readInvitationDeliverySnapshot(deliveryDoc)
			if err != nil {
				return err
			}
			tokenRef := r.col().Doc(
				invitationTokenDocumentID(delivery.Token),
			)
			token, tokenExists, err :=
				getInvitationTokenInTransaction(
					tx,
					tokenRef,
					delivery.Token,
				)
			if err != nil {
				return err
			}
			activeRef := r.deliveryActivesCol().Doc(
				delivery.MemberID,
			)
			activeDocument, activeExists, err :=
				getInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
				)
			if err != nil {
				return err
			}
			if delivery.IsTerminal() {
				return invdom.ErrInvitationDeliveryNotClaimable
			}
			if !tokenExists ||
				!availableInvitationDeliveryToken(
					token,
					delivery,
					now,
				) {
				if err := failInvitationDeliveryInTransaction(
					tx,
					deliveryRef,
					delivery,
					tokenRef,
					token,
					tokenExists,
					activeRef,
					activeDocument,
					activeExists,
					invitationDeliveryTokenError,
					now,
				); err != nil {
					return err
				}
				committedErr =
					invdom.ErrInvitationDeliveryNotClaimable
				return nil
			}
			if delivery.AttemptCount >= delivery.MaxAttempts {
				if err := failInvitationDeliveryInTransaction(
					tx,
					deliveryRef,
					delivery,
					tokenRef,
					token,
					true,
					activeRef,
					activeDocument,
					activeExists,
					invitationDeliveryAttemptError,
					now,
				); err != nil {
					return err
				}
				committedErr =
					invdom.ErrInvitationDeliveryAttemptLimit
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
				invitationDeliveryToDocument(claimed),
			); err != nil {
				return fmt.Errorf(
					"claim invitation delivery %q: %w",
					deliveryID,
					err,
				)
			}
			result = claimed
			return nil
		},
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryNotFound,
		):
			return invdom.InvitationDelivery{},
				invdom.ErrInvitationDeliveryNotFound
		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryNotClaimable,
		):
			return invdom.InvitationDelivery{},
				invdom.ErrInvitationDeliveryNotClaimable
		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryAttemptLimit,
		):
			return invdom.InvitationDelivery{},
				invdom.ErrInvitationDeliveryAttemptLimit
		default:
			return invdom.InvitationDelivery{}, fmt.Errorf(
				"claim invitation delivery transaction: %w",
				err,
			)
		}
	}
	if committedErr != nil {
		return invdom.InvitationDelivery{}, committedErr
	}
	return result, nil
}

// MarkInvitationDeliveryDeliveredは、deliveryをdeliveredへ変更し、
// token.deliveredAtを同一transactionで更新します。
func (r *InvitationTokenRepositoryFS) MarkInvitationDeliveryDelivered(
	ctx context.Context,
	deliveryID string,
	expectedAttemptCount int,
	providerMessageID string,
	deliveredAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return invdom.ErrInvitationDeliveryIDRequired
	}
	deliveredAt = deliveredAt.UTC()
	deliveryRef := r.deliveriesCol().Doc(deliveryID)
	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return invdom.ErrInvitationDeliveryNotFound
				}
				return fmt.Errorf(
					"get invitation delivery %q: %w",
					deliveryID,
					err,
				)
			}
			delivery, err :=
				readInvitationDeliverySnapshot(deliveryDoc)
			if err != nil {
				return err
			}
			tokenRef := r.col().Doc(
				invitationTokenDocumentID(delivery.Token),
			)
			token, tokenExists, err :=
				getInvitationTokenInTransaction(
					tx,
					tokenRef,
					delivery.Token,
				)
			if err != nil {
				return err
			}
			if !tokenExists {
				return invdom.ErrInvitationTokenNotFound
			}
			activeRef := r.deliveryActivesCol().Doc(
				delivery.MemberID,
			)
			activeDocument, activeExists, err :=
				getInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
				)
			if err != nil {
				return err
			}
			if delivery.Status ==
				invdom.InvitationDeliveryStatusDelivered {
				if token.DeliveredAt == nil {
					token.DeliveredAt =
						copyInvitationTimePointer(
							&deliveredAt,
						)
					token.UpdatedAt =
						copyInvitationTimePointer(
							&deliveredAt,
						)
					if err := tx.Set(
						tokenRef,
						invitationTokenToDocument(
							token,
							deliveredAt,
						),
					); err != nil {
						return err
					}
				}
				return saveInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
					activeDocument,
					activeExists,
					delivery,
					deliveredAt,
				)
			}
			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status !=
					invdom.InvitationDeliveryStatusProcessing {
				return invdom.ErrInvitationDeliveryNotClaimable
			}
			if token.IsUsed() ||
				token.IsRevoked() ||
				token.IsExpired(deliveredAt) {
				return invdom.ErrInvitationDeliveryNotClaimable
			}
			delivered, err := delivery.MarkDelivered(
				providerMessageID,
				deliveredAt,
			)
			if err != nil {
				return err
			}
			token.DeliveredAt =
				copyInvitationTimePointer(&deliveredAt)
			token.UpdatedAt =
				copyInvitationTimePointer(&deliveredAt)
			if err := tx.Set(
				deliveryRef,
				invitationDeliveryToDocument(delivered),
			); err != nil {
				return fmt.Errorf(
					"mark invitation delivery %q delivered: %w",
					deliveryID,
					err,
				)
			}
			if err := tx.Set(
				tokenRef,
				invitationTokenToDocument(
					token,
					deliveredAt,
				),
			); err != nil {
				return fmt.Errorf(
					"activate invitation token %q: %w",
					token.Token,
					err,
				)
			}
			return saveInvitationDeliveryActiveInTransaction(
				tx,
				activeRef,
				activeDocument,
				activeExists,
				delivered,
				deliveredAt,
			)
		},
	)
	if err != nil {
		return mapInvitationDeliveryRepositoryError(
			"mark invitation delivery delivered",
			err,
		)
	}
	return nil
}

// MarkInvitationDeliveryRetryableFailedは、deliveryを再試行可能な失敗へ
// 変更します。tokenは失効させません。
func (r *InvitationTokenRepositoryFS) MarkInvitationDeliveryRetryableFailed(
	ctx context.Context,
	deliveryID string,
	expectedAttemptCount int,
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return invdom.ErrInvitationDeliveryIDRequired
	}
	deliveryRef := r.deliveriesCol().Doc(deliveryID)
	nextAttemptAt = nextAttemptAt.UTC()
	failedAt = failedAt.UTC()
	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return invdom.ErrInvitationDeliveryNotFound
				}
				return err
			}
			delivery, err :=
				readInvitationDeliverySnapshot(deliveryDoc)
			if err != nil {
				return err
			}
			tokenRef := r.col().Doc(
				invitationTokenDocumentID(delivery.Token),
			)
			token, tokenExists, err :=
				getInvitationTokenInTransaction(
					tx,
					tokenRef,
					delivery.Token,
				)
			if err != nil {
				return err
			}
			if !tokenExists ||
				token.IsUsed() ||
				token.IsRevoked() ||
				token.IsExpired(failedAt) {
				return invdom.ErrInvitationDeliveryNotClaimable
			}
			activeRef := r.deliveryActivesCol().Doc(
				delivery.MemberID,
			)
			activeDocument, activeExists, err :=
				getInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
				)
			if err != nil {
				return err
			}
			if delivery.Status ==
				invdom.InvitationDeliveryStatusRetryableFailed &&
				delivery.AttemptCount == expectedAttemptCount {
				return nil
			}
			if delivery.AttemptCount != expectedAttemptCount ||
				delivery.Status !=
					invdom.InvitationDeliveryStatusProcessing {
				return invdom.ErrInvitationDeliveryNotClaimable
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
				invitationDeliveryToDocument(retryable),
			); err != nil {
				return fmt.Errorf(
					"mark invitation delivery %q retryable failed: %w",
					deliveryID,
					err,
				)
			}
			return saveInvitationDeliveryActiveInTransaction(
				tx,
				activeRef,
				activeDocument,
				activeExists,
				retryable,
				failedAt,
			)
		},
	)
	if err != nil {
		return mapInvitationDeliveryRepositoryError(
			"mark invitation delivery retryable failed",
			err,
		)
	}
	return nil
}

// MarkInvitationDeliveryFailedは、deliveryを最終失敗へ変更し、
// tokenを同一transactionで失効させます。
func (r *InvitationTokenRepositoryFS) MarkInvitationDeliveryFailed(
	ctx context.Context,
	deliveryID string,
	expectedAttemptCount int,
	lastError string,
	failedAt time.Time,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return invdom.ErrInvitationDeliveryIDRequired
	}
	deliveryRef := r.deliveriesCol().Doc(deliveryID)
	failedAt = failedAt.UTC()
	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			deliveryDoc, err := tx.Get(deliveryRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return invdom.ErrInvitationDeliveryNotFound
				}
				return err
			}
			delivery, err :=
				readInvitationDeliverySnapshot(deliveryDoc)
			if err != nil {
				return err
			}
			tokenRef := r.col().Doc(
				invitationTokenDocumentID(delivery.Token),
			)
			token, tokenExists, err :=
				getInvitationTokenInTransaction(
					tx,
					tokenRef,
					delivery.Token,
				)
			if err != nil {
				return err
			}
			activeRef := r.deliveryActivesCol().Doc(
				delivery.MemberID,
			)
			activeDocument, activeExists, err :=
				getInvitationDeliveryActiveInTransaction(
					tx,
					activeRef,
				)
			if err != nil {
				return err
			}
			if delivery.Status ==
				invdom.InvitationDeliveryStatusDelivered {
				return invdom.ErrInvitationDeliveryStatusInvalid
			}
			if delivery.Status !=
				invdom.InvitationDeliveryStatusFailed &&
				delivery.AttemptCount != expectedAttemptCount {
				return invdom.ErrInvitationDeliveryNotClaimable
			}
			failed := delivery
			if delivery.Status !=
				invdom.InvitationDeliveryStatusFailed {
				failed, err = delivery.MarkFailed(
					lastError,
					failedAt,
				)
				if err != nil {
					return err
				}
				if err := tx.Set(
					deliveryRef,
					invitationDeliveryToDocument(failed),
				); err != nil {
					return fmt.Errorf(
						"mark invitation delivery %q failed: %w",
						deliveryID,
						err,
					)
				}
			}
			if tokenExists && !token.IsUsed() {
				token.RevokedAt =
					copyInvitationTimePointer(&failedAt)
				token.UpdatedAt =
					copyInvitationTimePointer(&failedAt)
				if err := tx.Set(
					tokenRef,
					invitationTokenToDocument(
						token,
						failedAt,
					),
				); err != nil {
					return fmt.Errorf(
						"revoke invitation token %q: %w",
						token.Token,
						err,
					)
				}
			}
			if activeExists &&
				activeDocument.DeliveryID == deliveryID {
				if err := tx.Delete(activeRef); err != nil {
					return fmt.Errorf(
						"delete active invitation delivery for member %q: %w",
						delivery.MemberID,
						err,
					)
				}
			}
			return nil
		},
	)
	if err != nil {
		return mapInvitationDeliveryRepositoryError(
			"mark invitation delivery failed",
			err,
		)
	}
	return nil
}
func (r *InvitationTokenRepositoryFS) listMemberInvitationDeliveriesInTransaction(
	tx *firestore.Transaction,
	memberID string,
) ([]invitationDeliveryRecord, error) {
	query := r.deliveriesCol().
		Where("memberId", "==", memberID)
	iter := tx.Documents(query)
	defer iter.Stop()
	records := make([]invitationDeliveryRecord, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"list invitation deliveries for member %q: %w",
				memberID,
				err,
			)
		}
		delivery, err :=
			readInvitationDeliverySnapshot(doc)
		if err != nil {
			return nil, err
		}
		tokenRef := r.col().Doc(
			invitationTokenDocumentID(delivery.Token),
		)
		token, tokenExists, err :=
			getInvitationTokenInTransaction(
				tx,
				tokenRef,
				delivery.Token,
			)
		if err != nil {
			return nil, err
		}
		records = append(
			records,
			invitationDeliveryRecord{
				Ref:         doc.Ref,
				Delivery:    delivery,
				TokenRef:    tokenRef,
				Token:       token,
				TokenExists: tokenExists,
			},
		)
	}
	return records, nil
}
func reusableInvitationDeliveryRecord(
	record invitationDeliveryRecord,
	info invdom.InvitationInfo,
	now time.Time,
) bool {
	if !record.TokenExists {
		return false
	}
	delivery := record.Delivery
	token := record.Token
	if token.DeliveryID != delivery.ID ||
		token.Token != delivery.Token ||
		token.MemberID != info.MemberID ||
		token.CompanyID != info.CompanyID ||
		token.Email != info.Email {
		return false
	}
	if token.IsUsed() ||
		token.IsRevoked() ||
		token.IsExpired(now) {
		return false
	}
	switch delivery.Status {
	case invdom.InvitationDeliveryStatusPending,
		invdom.InvitationDeliveryStatusProcessing,
		invdom.InvitationDeliveryStatusRetryableFailed:
		return !token.IsDelivered()
	case invdom.InvitationDeliveryStatusDelivered:
		return token.IsDelivered()
	default:
		return false
	}
}
func revocableInvitationDeliveryRecord(
	record invitationDeliveryRecord,
	now time.Time,
) bool {
	if !record.TokenExists {
		return false
	}
	token := record.Token
	return !token.IsUsed() &&
		!token.IsRevoked() &&
		!token.IsExpired(now)
}
func revokeInvitationDeliveryRecord(
	tx *firestore.Transaction,
	record invitationDeliveryRecord,
	reason string,
	now time.Time,
) error {
	token := record.Token
	token.RevokedAt = copyInvitationTimePointer(&now)
	token.UpdatedAt = copyInvitationTimePointer(&now)
	if err := tx.Set(
		record.TokenRef,
		invitationTokenToDocument(token, now),
	); err != nil {
		return fmt.Errorf(
			"revoke invitation token %q: %w",
			token.Token,
			err,
		)
	}
	if record.Delivery.Status ==
		invdom.InvitationDeliveryStatusDelivered ||
		record.Delivery.Status ==
			invdom.InvitationDeliveryStatusFailed {
		return nil
	}
	failed, err := record.Delivery.MarkFailed(
		reason,
		now,
	)
	if err != nil {
		return err
	}
	if err := tx.Set(
		record.Ref,
		invitationDeliveryToDocument(failed),
	); err != nil {
		return fmt.Errorf(
			"fail invitation delivery %q: %w",
			record.Delivery.ID,
			err,
		)
	}
	return nil
}
func updateInvitationDeliverySnapshot(
	delivery invdom.InvitationDelivery,
	info invdom.InvitationInfo,
	now time.Time,
) (invdom.InvitationDelivery, error) {
	delivery.MemberID = info.MemberID
	delivery.CompanyID = info.CompanyID
	delivery.Email = info.Email
	delivery.AssignedBrandIDs = append(
		[]string(nil),
		info.AssignedBrandIDs...,
	)
	delivery.Permissions = append(
		[]string(nil),
		info.Permissions...,
	)
	delivery.UpdatedAt = now.UTC()
	return delivery.Normalize()
}
func updateInvitationTokenSnapshot(
	token invdom.InvitationToken,
	deliveryID string,
	info invdom.InvitationInfo,
	now time.Time,
) (invdom.InvitationToken, error) {
	updated, err := invdom.NewInvitationToken(
		token.Token,
		deliveryID,
		info,
		token.CreatedAt,
		token.ExpiresAt,
	)
	if err != nil {
		return invdom.InvitationToken{}, err
	}
	updated.DeliveredAt =
		copyInvitationTimePointer(token.DeliveredAt)
	updated.UsedAt =
		copyInvitationTimePointer(token.UsedAt)
	updated.RevokedAt =
		copyInvitationTimePointer(token.RevokedAt)
	updated.UpdatedAt =
		copyInvitationTimePointer(&now)
	return updated, nil
}
func availableInvitationDeliveryToken(
	token invdom.InvitationToken,
	delivery invdom.InvitationDelivery,
	now time.Time,
) bool {
	if token.Token != delivery.Token ||
		token.DeliveryID != delivery.ID ||
		token.MemberID != delivery.MemberID {
		return false
	}
	return !token.IsUsed() &&
		!token.IsRevoked() &&
		!token.IsExpired(now)
}
func failInvitationDeliveryInTransaction(
	tx *firestore.Transaction,
	deliveryRef *firestore.DocumentRef,
	delivery invdom.InvitationDelivery,
	tokenRef *firestore.DocumentRef,
	token invdom.InvitationToken,
	tokenExists bool,
	activeRef *firestore.DocumentRef,
	activeDocument invitationDeliveryActiveDocument,
	activeExists bool,
	lastError string,
	failedAt time.Time,
) error {
	if delivery.Status !=
		invdom.InvitationDeliveryStatusFailed &&
		delivery.Status !=
			invdom.InvitationDeliveryStatusDelivered {
		failed, err := delivery.MarkFailed(
			lastError,
			failedAt,
		)
		if err != nil {
			return err
		}
		if err := tx.Set(
			deliveryRef,
			invitationDeliveryToDocument(failed),
		); err != nil {
			return err
		}
	}
	if tokenExists && !token.IsUsed() {
		token.RevokedAt =
			copyInvitationTimePointer(&failedAt)
		token.UpdatedAt =
			copyInvitationTimePointer(&failedAt)
		if err := tx.Set(
			tokenRef,
			invitationTokenToDocument(
				token,
				failedAt,
			),
		); err != nil {
			return err
		}
	}
	if activeExists &&
		activeDocument.DeliveryID == delivery.ID {
		if err := tx.Delete(activeRef); err != nil {
			return err
		}
	}
	return nil
}
func saveInvitationDeliveryActiveInTransaction(
	tx *firestore.Transaction,
	activeRef *firestore.DocumentRef,
	current invitationDeliveryActiveDocument,
	exists bool,
	delivery invdom.InvitationDelivery,
	updatedAt time.Time,
) error {
	createdAt := invitationDeliveryActiveCreatedAt(
		current,
		exists &&
			current.DeliveryID == delivery.ID,
		updatedAt,
	)
	if err := tx.Set(
		activeRef,
		invitationDeliveryActiveDocument{
			DeliveryID: delivery.ID,
			Token:      delivery.Token,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt.UTC(),
		},
	); err != nil {
		return fmt.Errorf(
			"save active invitation delivery %q: %w",
			delivery.ID,
			err,
		)
	}
	return nil
}
func invitationDeliveryActiveCreatedAt(
	current invitationDeliveryActiveDocument,
	preserve bool,
	fallback time.Time,
) time.Time {
	if preserve && !current.CreatedAt.IsZero() {
		return current.CreatedAt.UTC()
	}
	return fallback.UTC()
}
func getInvitationDeliveryActiveInTransaction(
	tx *firestore.Transaction,
	ref *firestore.DocumentRef,
) (
	invitationDeliveryActiveDocument,
	bool,
	error,
) {
	doc, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invitationDeliveryActiveDocument{},
				false,
				nil
		}
		return invitationDeliveryActiveDocument{},
			false,
			fmt.Errorf(
				"get active invitation delivery %q: %w",
				ref.ID,
				err,
			)
	}
	var stored invitationDeliveryActiveDocument
	if err := doc.DataTo(&stored); err != nil {
		return invitationDeliveryActiveDocument{},
			false,
			fmt.Errorf(
				"decode active invitation delivery %q: %w",
				ref.ID,
				err,
			)
	}
	stored.DeliveryID =
		strings.TrimSpace(stored.DeliveryID)
	stored.Token =
		strings.TrimSpace(stored.Token)
	if stored.DeliveryID == "" || stored.Token == "" {
		return invitationDeliveryActiveDocument{},
			false,
			fmt.Errorf(
				"active invitation delivery %q is invalid",
				ref.ID,
			)
	}
	if !stored.CreatedAt.IsZero() {
		stored.CreatedAt = stored.CreatedAt.UTC()
	}
	if !stored.UpdatedAt.IsZero() {
		stored.UpdatedAt = stored.UpdatedAt.UTC()
	}
	return stored, true, nil
}
func getInvitationTokenInTransaction(
	tx *firestore.Transaction,
	ref *firestore.DocumentRef,
	rawToken string,
) (invdom.InvitationToken, bool, error) {
	doc, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.InvitationToken{},
				false,
				nil
		}
		return invdom.InvitationToken{},
			false,
			fmt.Errorf(
				"get invitation token %q: %w",
				ref.ID,
				err,
			)
	}
	token, err := readInvitationTokenSnapshot(
		doc,
		rawToken,
	)
	if err != nil {
		return invdom.InvitationToken{}, false, err
	}
	return token, true, nil
}
func invitationDeliveryToDocument(
	delivery invdom.InvitationDelivery,
) invitationDeliveryDocument {
	return invitationDeliveryDocument{
		Token:     delivery.Token,
		MemberID:  delivery.MemberID,
		CompanyID: delivery.CompanyID,
		AssignedBrandIDs: append(
			[]string(nil),
			delivery.AssignedBrandIDs...,
		),
		Permissions: append(
			[]string(nil),
			delivery.Permissions...,
		),
		Email:             delivery.Email,
		Status:            delivery.Status,
		AttemptCount:      delivery.AttemptCount,
		MaxAttempts:       delivery.MaxAttempts,
		LastError:         delivery.LastError,
		ProviderMessageID: delivery.ProviderMessageID,
		CreatedAt:         delivery.CreatedAt.UTC(),
		UpdatedAt:         delivery.UpdatedAt.UTC(),
		NextAttemptAt: copyInvitationTimePointer(
			delivery.NextAttemptAt,
		),
		ProcessingStartedAt: copyInvitationTimePointer(
			delivery.ProcessingStartedAt,
		),
		ProcessingUntil: copyInvitationTimePointer(
			delivery.ProcessingUntil,
		),
		DeliveredAt: copyInvitationTimePointer(
			delivery.DeliveredAt,
		),
		FailedAt: copyInvitationTimePointer(
			delivery.FailedAt,
		),
	}
}
func readInvitationDeliverySnapshot(
	doc *firestore.DocumentSnapshot,
) (invdom.InvitationDelivery, error) {
	if doc == nil {
		return invdom.InvitationDelivery{},
			errors.New(
				"invitation delivery document snapshot is nil",
			)
	}
	var stored invitationDeliveryDocument
	if err := doc.DataTo(&stored); err != nil {
		return invdom.InvitationDelivery{}, fmt.Errorf(
			"decode invitation delivery %q: %w",
			doc.Ref.ID,
			err,
		)
	}
	delivery := invdom.InvitationDelivery{
		ID:                  doc.Ref.ID,
		Token:               stored.Token,
		MemberID:            stored.MemberID,
		CompanyID:           stored.CompanyID,
		AssignedBrandIDs:    stored.AssignedBrandIDs,
		Permissions:         stored.Permissions,
		Email:               stored.Email,
		Status:              stored.Status,
		AttemptCount:        stored.AttemptCount,
		MaxAttempts:         stored.MaxAttempts,
		LastError:           stored.LastError,
		ProviderMessageID:   stored.ProviderMessageID,
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
		return invdom.InvitationDelivery{}, fmt.Errorf(
			"normalize invitation delivery %q: %w",
			doc.Ref.ID,
			err,
		)
	}
	return normalized, nil
}
func invitationDeliveryDueTime(
	delivery invdom.InvitationDelivery,
) time.Time {
	switch delivery.Status {
	case invdom.InvitationDeliveryStatusProcessing:
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
func mapInvitationDeliveryRepositoryError(
	operation string,
	err error,
) error {
	switch {
	case errors.Is(
		err,
		invdom.ErrInvitationDeliveryNotFound,
	):
		return invdom.ErrInvitationDeliveryNotFound
	case errors.Is(
		err,
		invdom.ErrInvitationDeliveryNotClaimable,
	):
		return invdom.ErrInvitationDeliveryNotClaimable
	case errors.Is(
		err,
		invdom.ErrInvitationDeliveryAttemptLimit,
	):
		return invdom.ErrInvitationDeliveryAttemptLimit
	case errors.Is(
		err,
		invdom.ErrInvitationDeliveryStatusInvalid,
	):
		return invdom.ErrInvitationDeliveryStatusInvalid
	case errors.Is(
		err,
		invdom.ErrInvitationTokenNotFound,
	):
		return invdom.ErrInvitationTokenNotFound
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
