// backend/internal/adapters/out/firestore/transfer_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	transferdom "narratives/internal/domain/transfer"
)

// ============================================================
// Transfer RepositoryPort (Firestore)
// ============================================================

var (
	ErrTransferRepoNotConfigured  = errors.New("transfer_repo_fs: not configured")
	ErrInvalidTransferAttempt     = errors.New("transfer_repo_fs: attempt is invalid")
	ErrInvalidTransferCounterData = errors.New("transfer_repo_fs: attempt counter is invalid")
	ErrInvalidTransferData        = errors.New("transfer_repo_fs: transfer data is invalid")
)

type TransferRepositoryFS struct {
	Client *firestore.Client
	Now    func() time.Time

	TransfersCollection       string
	AttemptCountersCollection string
}

var _ transferdom.RepositoryPort = (*TransferRepositoryFS)(nil)

func NewTransferRepositoryFS(
	client *firestore.Client,
) *TransferRepositoryFS {
	return &TransferRepositoryFS{
		Client: client,
		Now:    time.Now,
	}
}

func (r *TransferRepositoryFS) transfersCol() *firestore.CollectionRef {
	collection := r.TransfersCollection
	if collection == "" {
		collection = os.Getenv("TRANSFERS_COLLECTION")
	}
	if collection == "" {
		collection = "transfers"
	}

	return r.Client.Collection(collection)
}

func (r *TransferRepositoryFS) countersCol() *firestore.CollectionRef {
	collection := r.AttemptCountersCollection
	if collection == "" {
		collection = os.Getenv(
			"TRANSFER_ATTEMPT_COUNTERS_COLLECTION",
		)
	}
	if collection == "" {
		collection = "transferAttemptCounters"
	}

	return r.Client.Collection(collection)
}

func (r *TransferRepositoryFS) transferDocID(
	productID string,
	attempt int,
) string {
	return productID + "__" + strconv.Itoa(attempt)
}

func (r *TransferRepositoryFS) counterDoc(
	productID string,
) *firestore.DocumentRef {
	return r.countersCol().Doc(productID)
}

func (r *TransferRepositoryFS) GetLatestByProductID(
	ctx context.Context,
	productID string,
) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if productID == "" {
		return nil, ErrInvalidTransferProductID
	}

	iter := r.transfersCol().
		Where("productId", "==", productID).
		OrderBy("attempt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	snap, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, transferdom.ErrNotFound
		}
		return nil, err
	}

	return transferFromSnapshot(snap)
}

func (r *TransferRepositoryFS) GetByProductIDAndAttempt(
	ctx context.Context,
	productID string,
	attempt int,
) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if productID == "" {
		return nil, ErrInvalidTransferProductID
	}
	if attempt <= 0 {
		return nil, ErrInvalidTransferAttempt
	}

	snap, err := r.transfersCol().
		Doc(r.transferDocID(productID, attempt)).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, transferdom.ErrNotFound
		}
		return nil, err
	}
	if snap == nil || !snap.Exists() {
		return nil, transferdom.ErrNotFound
	}

	return transferFromSnapshot(snap)
}

func (r *TransferRepositoryFS) ListByProductID(
	ctx context.Context,
	productID string,
) ([]transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if productID == "" {
		return nil, ErrInvalidTransferProductID
	}

	iter := r.transfersCol().
		Where("productId", "==", productID).
		OrderBy("attempt", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	out := make([]transferdom.Transfer, 0)

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		t, err := transferFromSnapshot(snap)
		if err != nil {
			return nil, err
		}

		out = append(out, *t)
	}

	return out, nil
}

func (r *TransferRepositoryFS) ResolveTransferredAtByMintAddress(
	ctx context.Context,
	mintAddress string,
) (
	transferdom.ResolveTransferredAtByMintAddressResult,
	error,
) {
	if r == nil || r.Client == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			ErrTransferRepoNotConfigured
	}
	if mintAddress == "" {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			transferdom.ErrInvalidMintAddress
	}

	iter := r.transfersCol().
		Where("mintAddress", "==", mintAddress).
		Where(
			"status",
			"==",
			string(transferdom.StatusSucceeded),
		).
		OrderBy("transferredAt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	snap, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return transferdom.ResolveTransferredAtByMintAddressResult{},
				transferdom.ErrNotFound
		}

		return transferdom.ResolveTransferredAtByMintAddressResult{},
			err
	}
	if snap == nil || snap.Ref == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			transferdom.ErrNotFound
	}

	raw := snap.Data()
	if raw == nil {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			transferdom.ErrNotFound
	}

	productID, ok := raw["productId"].(string)
	if !ok || productID == "" {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			ErrInvalidTransferData
	}

	avatarID, ok := raw["avatarId"].(string)
	if !ok || avatarID == "" {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			ErrInvalidTransferData
	}

	rawAttempt, ok := raw["attempt"].(int64)
	if !ok || rawAttempt <= 0 {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			ErrInvalidTransferData
	}

	transferredAt, ok := raw["transferredAt"].(time.Time)
	if !ok || transferredAt.IsZero() {
		return transferdom.ResolveTransferredAtByMintAddressResult{},
			transferdom.ErrInvalidTransferredAt
	}

	return transferdom.ResolveTransferredAtByMintAddressResult{
		ProductID:     productID,
		Attempt:       int(rawAttempt),
		AvatarID:      avatarID,
		MintAddress:   mintAddress,
		TransferredAt: transferredAt.UTC(),
	}, nil
}

func (r *TransferRepositoryFS) CreateAttempt(
	ctx context.Context,
	in transferdom.CreateAttemptInput,
) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}

	counterRef := r.counterDoc(in.ProductID)
	var created transferdom.Transfer

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			counterSnap, err := tx.Get(counterRef)

			var attempt int
			counterExists := true

			if err != nil {
				if status.Code(err) != codes.NotFound {
					return err
				}

				counterExists = false
				attempt = 1
			} else {
				if counterSnap == nil ||
					!counterSnap.Exists() {
					return ErrInvalidTransferCounterData
				}

				raw := counterSnap.Data()
				if raw == nil {
					return ErrInvalidTransferCounterData
				}

				nextAttempt, ok :=
					raw["nextAttempt"].(int64)
				if !ok || nextAttempt <= 0 {
					return ErrInvalidTransferCounterData
				}

				attempt = int(nextAttempt)
			}

			t, err := in.NewTransfer(attempt)
			if err != nil {
				return err
			}

			doc, err := transferDocument(
				t,
				t.CreatedAt,
			)
			if err != nil {
				return err
			}

			transferRef := r.transfersCol().
				Doc(
					r.transferDocID(
						t.ProductID,
						t.Attempt,
					),
				)

			if counterExists {
				if err := tx.Set(
					counterRef,
					map[string]any{
						"productId": t.ProductID,
						"nextAttempt": int64(
							t.Attempt + 1,
						),
						"updatedAt": t.CreatedAt,
					},
					firestore.MergeAll,
				); err != nil {
					return err
				}
			} else {
				if err := tx.Create(
					counterRef,
					map[string]any{
						"productId":     t.ProductID,
						"nextAttempt":   int64(2),
						"updatedAt":     t.CreatedAt,
						"initializedAt": t.CreatedAt,
					},
				); err != nil {
					return err
				}
			}

			if err := tx.Create(
				transferRef,
				doc,
			); err != nil {
				return err
			}

			created = t
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *TransferRepositoryFS) Save(
	ctx context.Context,
	t transferdom.Transfer,
) (*transferdom.Transfer, error) {
	if r == nil ||
		r.Client == nil ||
		r.Now == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}

	now := r.Now().UTC()
	doc, err := transferDocument(t, now)
	if err != nil {
		return nil, err
	}

	if t.Status == transferdom.StatusSucceeded {
		doc["transferredAt"] = now
	}

	_, err = r.transfersCol().
		Doc(r.transferDocID(t.ProductID, t.Attempt)).
		Set(ctx, doc)
	if err != nil {
		return nil, err
	}

	saved := t
	return &saved, nil
}

func (r *TransferRepositoryFS) Patch(
	ctx context.Context,
	productID string,
	attempt int,
	patch transferdom.TransferPatch,
) (*transferdom.Transfer, error) {
	if r == nil ||
		r.Client == nil ||
		r.Now == nil {
		return nil, ErrTransferRepoNotConfigured
	}
	if productID == "" {
		return nil, ErrInvalidTransferProductID
	}
	if attempt <= 0 {
		return nil, ErrInvalidTransferAttempt
	}

	ref := r.transfersCol().
		Doc(r.transferDocID(productID, attempt))

	var updated transferdom.Transfer

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snap, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return transferdom.ErrNotFound
				}
				return err
			}
			if snap == nil || !snap.Exists() {
				return transferdom.ErrNotFound
			}

			t, err := transferFromSnapshot(snap)
			if err != nil {
				return err
			}
			if err := t.ApplyPatch(patch); err != nil {
				return err
			}

			now := r.Now().UTC()
			fields := map[string]any{
				"updatedAt": now,
			}

			if patch.Status != nil {
				fields["status"] = t.Status
				if t.Status ==
					transferdom.StatusSucceeded {
					fields["transferredAt"] = now
				}
			}
			if patch.ErrorType != nil {
				fields["errorType"] = t.ErrorType
			}
			if patch.ErrorMsg != nil {
				fields["errorMsg"] = t.ErrorMsg
			}
			if patch.TxSignature != nil {
				fields["txSignature"] = t.TxSignature
			}
			if patch.MintAddress != nil {
				fields["mintAddress"] = t.MintAddress
			}
			if patch.ToWalletAddress != nil {
				fields["toWalletAddress"] =
					t.ToWalletAddress
			}

			if err := tx.Set(
				ref,
				fields,
				firestore.MergeAll,
			); err != nil {
				return err
			}

			updated = *t
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// ============================================================
// Transfer helpers
// ============================================================

func transferFromSnapshot(
	snap *firestore.DocumentSnapshot,
) (*transferdom.Transfer, error) {
	if snap == nil ||
		snap.Ref == nil ||
		!snap.Exists() {
		return nil, transferdom.ErrNotFound
	}

	raw := snap.Data()
	if raw == nil {
		return nil, ErrInvalidTransferData
	}

	rawAttempt, ok := raw["attempt"].(int64)
	if !ok || rawAttempt <= 0 {
		return nil, ErrInvalidTransferData
	}

	productID, ok := raw["productId"].(string)
	if !ok || productID == "" {
		return nil, ErrInvalidTransferData
	}

	orderID, ok := raw["orderId"].(string)
	if !ok || orderID == "" {
		return nil, ErrInvalidTransferData
	}

	avatarID, ok := raw["avatarId"].(string)
	if !ok || avatarID == "" {
		return nil, ErrInvalidTransferData
	}

	mintAddress, ok := raw["mintAddress"].(string)
	if !ok || mintAddress == "" {
		return nil, ErrInvalidTransferData
	}

	toWalletAddress, ok :=
		raw["toWalletAddress"].(string)
	if !ok || toWalletAddress == "" {
		return nil, ErrInvalidTransferData
	}

	rawStatus, ok := raw["status"].(string)
	if !ok || rawStatus == "" {
		return nil, ErrInvalidTransferData
	}

	createdAt, ok := raw["createdAt"].(time.Time)
	if !ok || createdAt.IsZero() {
		return nil, ErrInvalidTransferData
	}

	t := transferdom.Transfer{
		Attempt:         int(rawAttempt),
		ProductID:       productID,
		OrderID:         orderID,
		AvatarID:        avatarID,
		MintAddress:     mintAddress,
		ToWalletAddress: toWalletAddress,
		Status:          transferdom.Status(rawStatus),
		CreatedAt:       createdAt.UTC(),
	}

	if value, exists := raw["txSignature"]; exists &&
		value != nil {
		txSignature, ok := value.(string)
		if !ok {
			return nil, ErrInvalidTransferData
		}
		t.TxSignature = &txSignature
	}

	if value, exists := raw["errorType"]; exists &&
		value != nil {
		rawErrorType, ok := value.(string)
		if !ok {
			return nil, ErrInvalidTransferData
		}

		errorType := transferdom.ErrorType(
			rawErrorType,
		)
		t.ErrorType = &errorType
	}

	if value, exists := raw["errorMsg"]; exists &&
		value != nil {
		errorMsg, ok := value.(string)
		if !ok {
			return nil, ErrInvalidTransferData
		}
		t.ErrorMsg = &errorMsg
	}

	if err := t.Validate(); err != nil {
		return nil, err
	}

	return &t, nil
}

func transferDocument(
	t transferdom.Transfer,
	updatedAt time.Time,
) (map[string]any, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if updatedAt.IsZero() {
		return nil, transferdom.ErrInvalidCreatedAt
	}

	doc := map[string]any{
		"attempt":         int64(t.Attempt),
		"avatarId":        t.AvatarID,
		"createdAt":       t.CreatedAt.UTC(),
		"errorMsg":        t.ErrorMsg,
		"errorType":       t.ErrorType,
		"mintAddress":     t.MintAddress,
		"orderId":         t.OrderID,
		"productId":       t.ProductID,
		"status":          t.Status,
		"toWalletAddress": t.ToWalletAddress,
		"txSignature":     t.TxSignature,
		"updatedAt":       updatedAt.UTC(),
	}

	if share, ok := parseShareTransferRef(
		t.OrderID,
	); ok {
		if share.ProductID != t.ProductID {
			return nil,
				transferdom.ErrInvalidProductID
		}

		doc["transferKind"] = "share"
		doc["shareRef"] = t.OrderID
		doc["fromAvatarId"] = share.FromAvatarID
		doc["toAvatarId"] = share.ToAvatarID
		doc["receiverAvatarId"] = t.AvatarID
	} else {
		doc["transferKind"] = "order"
	}

	return doc, nil
}

// ============================================================
// Share transfer reference
// ============================================================

type shareTransferRef struct {
	FromAvatarID string
	ToAvatarID   string
	ProductID    string
}

func parseShareTransferRef(
	ref string,
) (shareTransferRef, bool) {
	if ref == "" {
		return shareTransferRef{}, false
	}

	parts := strings.Split(ref, ":")
	if len(parts) != 4 {
		return shareTransferRef{}, false
	}
	if parts[0] != "share" {
		return shareTransferRef{}, false
	}
	if parts[1] == "" ||
		parts[2] == "" ||
		parts[3] == "" {
		return shareTransferRef{}, false
	}

	return shareTransferRef{
		FromAvatarID: parts[1],
		ToAvatarID:   parts[2],
		ProductID:    parts[3],
	}, true
}
