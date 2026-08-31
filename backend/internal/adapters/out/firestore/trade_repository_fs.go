// backend/internal/adapters/out/firestore/trade_repository_fs.go
package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeRepositoryNotConfigured = errors.New("trade_repository_fs: not configured")
	ErrInvalidTradeDocumentData     = errors.New("trade_repository_fs: invalid trade document data")
)

// TradeRepositoryFS implements tradedom.Repository using Firestore.
//
// Collections:
//
//	trades/{tradeId}
//
// orderId + orderItemIndex uniqueness:
//
//	tradeOrderItemKeys/{sha256(orderId:itemIndex)}
//
// tradeOrderItemKeys is an infrastructure-only uniqueness projection.
// Trade remains the source of truth for transaction-participant data.
type TradeRepositoryFS struct {
	Client *firestore.Client
}

var _ tradedom.Repository = (*TradeRepositoryFS)(nil)

func NewTradeRepositoryFS(client *firestore.Client) *TradeRepositoryFS {
	return &TradeRepositoryFS{Client: client}
}

func (r *TradeRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("trades")
}

func (r *TradeRepositoryFS) orderItemKeysCol() *firestore.CollectionRef {
	return r.Client.Collection("tradeOrderItemKeys")
}

func (r *TradeRepositoryFS) orderItemKeyDoc(orderID string, orderItemIndex int) *firestore.DocumentRef {
	raw := fmt.Sprintf("%s:%d", orderID, orderItemIndex)
	sum := sha256.Sum256([]byte(raw))
	id := hex.EncodeToString(sum[:])
	return r.orderItemKeysCol().Doc(id)
}

// ============================================================
// Read
// ============================================================

func (r *TradeRepositoryFS) GetByID(ctx context.Context, id string) (tradedom.Trade, error) {
	if r == nil || r.Client == nil {
		return tradedom.Trade{}, ErrTradeRepositoryNotConfigured
	}
	if id == "" {
		return tradedom.Trade{}, tradedom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.Trade{}, tradedom.ErrNotFound
		}
		return tradedom.Trade{}, err
	}

	trade, err := docToTrade(snap)
	if err != nil {
		return tradedom.Trade{}, err
	}
	if trade.ID != id {
		return tradedom.Trade{}, ErrInvalidTradeDocumentData
	}

	return trade, nil
}

func (r *TradeRepositoryFS) GetByOrderItem(ctx context.Context, orderID string, orderItemIndex int) (tradedom.Trade, error) {
	if r == nil || r.Client == nil {
		return tradedom.Trade{}, ErrTradeRepositoryNotConfigured
	}
	if orderID == "" || orderItemIndex < 0 {
		return tradedom.Trade{}, tradedom.ErrNotFound
	}

	keyRef := r.orderItemKeyDoc(orderID, orderItemIndex)
	keySnap, err := keyRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.Trade{}, tradedom.ErrNotFound
		}
		return tradedom.Trade{}, err
	}

	var keyDoc tradeOrderItemKeyDoc
	if err := keySnap.DataTo(&keyDoc); err != nil {
		return tradedom.Trade{}, err
	}
	if keyDoc.OrderID != orderID || keyDoc.OrderItemIndex != orderItemIndex || keyDoc.TradeID == "" {
		return tradedom.Trade{}, ErrInvalidTradeDocumentData
	}

	trade, err := r.GetByID(ctx, keyDoc.TradeID)
	if err != nil {
		return tradedom.Trade{}, err
	}
	if trade.OrderID != orderID || trade.OrderItemIndex != orderItemIndex {
		return tradedom.Trade{}, ErrInvalidTradeDocumentData
	}

	return trade, nil
}

// ============================================================
// Create
// ============================================================

func (r *TradeRepositoryFS) Create(ctx context.Context, trade tradedom.Trade) (tradedom.Trade, error) {
	if r == nil || r.Client == nil {
		return tradedom.Trade{}, ErrTradeRepositoryNotConfigured
	}
	if trade.OrderID == "" || trade.OrderItemIndex < 0 {
		return tradedom.Trade{}, tradedom.ErrNotFound
	}

	now := time.Now().UTC()

	var tradeRef *firestore.DocumentRef
	if trade.ID == "" {
		tradeRef = r.col().NewDoc()
		trade.ID = tradeRef.ID
	} else {
		tradeRef = r.col().Doc(trade.ID)
	}

	if trade.CreatedAt.IsZero() {
		trade.CreatedAt = now
	} else {
		trade.CreatedAt = trade.CreatedAt.UTC()
	}
	if trade.UpdatedAt.IsZero() {
		trade.UpdatedAt = trade.CreatedAt
	} else {
		trade.UpdatedAt = trade.UpdatedAt.UTC()
	}
	if trade.LastMessageAt != nil {
		value := trade.LastMessageAt.UTC()
		trade.LastMessageAt = &value
	}
	if trade.Status == "" {
		trade.Status = tradedom.StatusActive
	}

	if err := trade.ValidateForPersist(); err != nil {
		return tradedom.Trade{}, err
	}

	keyRef := r.orderItemKeyDoc(trade.OrderID, trade.OrderItemIndex)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		keySnap, err := tx.Get(keyRef)
		if err == nil && keySnap.Exists() {
			return tradedom.ErrAlreadyExists
		}
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		tradeSnap, err := tx.Get(tradeRef)
		if err == nil && tradeSnap.Exists() {
			return tradedom.ErrAlreadyExists
		}
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(tradeRef, tradeToDoc(trade)); err != nil {
			return err
		}
		if err := tx.Create(keyRef, tradeOrderItemKeyDoc{
			OrderID:        trade.OrderID,
			OrderItemIndex: trade.OrderItemIndex,
			TradeID:        trade.ID,
			CreatedAt:      trade.CreatedAt.UTC(),
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, tradedom.ErrAlreadyExists) || status.Code(err) == codes.AlreadyExists {
			return tradedom.Trade{}, tradedom.ErrAlreadyExists
		}
		return tradedom.Trade{}, err
	}

	return trade, nil
}

// ============================================================
// Update
// ============================================================

func (r *TradeRepositoryFS) Update(ctx context.Context, id string, trade tradedom.Trade) (tradedom.Trade, error) {
	if r == nil || r.Client == nil {
		return tradedom.Trade{}, ErrTradeRepositoryNotConfigured
	}
	if id == "" {
		return tradedom.Trade{}, tradedom.ErrNotFound
	}

	ref := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return tradedom.ErrNotFound
			}
			return err
		}

		current, err := docToTrade(snap)
		if err != nil {
			return err
		}
		if !sameTradeIdentity(current, trade, id) {
			return tradedom.ErrConflict
		}

		updated := current
		updated.Status = trade.Status
		updated.LastMessageAt = cloneTradeTimePtr(trade.LastMessageAt)

		if trade.UpdatedAt.IsZero() {
			updated.UpdatedAt = time.Now().UTC()
		} else {
			updated.UpdatedAt = trade.UpdatedAt.UTC()
		}
		if updated.LastMessageAt != nil && updated.UpdatedAt.Before(*updated.LastMessageAt) {
			updated.UpdatedAt = updated.LastMessageAt.UTC()
		}
		if err := updated.ValidateForPersist(); err != nil {
			return err
		}

		return tx.Set(ref, tradeToDoc(updated))
	})
	if err != nil {
		return tradedom.Trade{}, err
	}

	return r.GetByID(ctx, id)
}

// ============================================================
// Firestore documents
// ============================================================

type tradeDoc struct {
	ID             string `firestore:"id"`
	OrderID        string `firestore:"orderId"`
	OrderItemIndex int    `firestore:"orderItemIndex"`

	BuyerAvatarID string `firestore:"buyerAvatarId"`

	SellerType      string `firestore:"sellerType"`
	SellerCompanyID string `firestore:"sellerCompanyId,omitempty"`
	SellerBrandID   string `firestore:"sellerBrandId,omitempty"`
	SellerAvatarID  string `firestore:"sellerAvatarId,omitempty"`

	Status string `firestore:"status"`

	CreatedAt     time.Time  `firestore:"createdAt"`
	UpdatedAt     time.Time  `firestore:"updatedAt"`
	LastMessageAt *time.Time `firestore:"lastMessageAt,omitempty"`
}

type tradeOrderItemKeyDoc struct {
	OrderID        string    `firestore:"orderId"`
	OrderItemIndex int       `firestore:"orderItemIndex"`
	TradeID        string    `firestore:"tradeId"`
	CreatedAt      time.Time `firestore:"createdAt"`
}

func tradeToDoc(trade tradedom.Trade) tradeDoc {
	return tradeDoc{
		ID:              trade.ID,
		OrderID:         trade.OrderID,
		OrderItemIndex:  trade.OrderItemIndex,
		BuyerAvatarID:   trade.BuyerAvatarID,
		SellerType:      string(trade.SellerType),
		SellerCompanyID: trade.SellerCompanyID,
		SellerBrandID:   trade.SellerBrandID,
		SellerAvatarID:  trade.SellerAvatarID,
		Status:          string(trade.Status),
		CreatedAt:       trade.CreatedAt.UTC(),
		UpdatedAt:       trade.UpdatedAt.UTC(),
		LastMessageAt:   cloneTradeTimePtr(trade.LastMessageAt),
	}
}

func docToTrade(snap *firestore.DocumentSnapshot) (tradedom.Trade, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return tradedom.Trade{}, tradedom.ErrNotFound
	}

	var doc tradeDoc
	if err := snap.DataTo(&doc); err != nil {
		return tradedom.Trade{}, err
	}

	trade := tradedom.Trade{
		ID:              doc.ID,
		OrderID:         doc.OrderID,
		OrderItemIndex:  doc.OrderItemIndex,
		BuyerAvatarID:   doc.BuyerAvatarID,
		SellerType:      tradedom.SellerType(doc.SellerType),
		SellerCompanyID: doc.SellerCompanyID,
		SellerBrandID:   doc.SellerBrandID,
		SellerAvatarID:  doc.SellerAvatarID,
		Status:          tradedom.Status(doc.Status),
		CreatedAt:       doc.CreatedAt.UTC(),
		UpdatedAt:       doc.UpdatedAt.UTC(),
		LastMessageAt:   cloneTradeTimePtr(doc.LastMessageAt),
	}

	if trade.ID != snap.Ref.ID {
		return tradedom.Trade{}, fmt.Errorf("trade %s: %w: document id mismatch", snap.Ref.ID, ErrInvalidTradeDocumentData)
	}
	if err := trade.ValidateForPersist(); err != nil {
		return tradedom.Trade{}, fmt.Errorf("trade %s: %w: %v", snap.Ref.ID, ErrInvalidTradeDocumentData, err)
	}

	return trade, nil
}

// ============================================================
// Helpers
// ============================================================

func sameTradeIdentity(current tradedom.Trade, incoming tradedom.Trade, targetID string) bool {
	if incoming.ID != targetID || current.ID != targetID {
		return false
	}

	return current.OrderID == incoming.OrderID &&
		current.OrderItemIndex == incoming.OrderItemIndex &&
		current.BuyerAvatarID == incoming.BuyerAvatarID &&
		current.SellerType == incoming.SellerType &&
		current.SellerCompanyID == incoming.SellerCompanyID &&
		current.SellerBrandID == incoming.SellerBrandID &&
		current.SellerAvatarID == incoming.SellerAvatarID &&
		current.CreatedAt.Equal(incoming.CreatedAt)
}

func cloneTradeTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
