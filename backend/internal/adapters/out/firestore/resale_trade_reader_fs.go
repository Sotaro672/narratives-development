// backend/internal/adapters/out/firestore/resale_trade_reader_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	orderdom "narratives/internal/domain/order"
	tradedom "narratives/internal/domain/trade"
)

var ErrResaleTradeReaderNotConfigured = errors.New(
	"resale_trade_reader_fs: not configured",
)

type ResaleTradeReaderFS struct {
	Client    *firestore.Client
	tradeRepo tradedom.Repository
}

func NewResaleTradeReaderFS(client *firestore.Client) *ResaleTradeReaderFS {
	if client == nil {
		return &ResaleTradeReaderFS{}
	}

	return &ResaleTradeReaderFS{
		Client:    client,
		tradeRepo: NewTradeRepositoryFS(client),
	}
}

func (r *ResaleTradeReaderFS) transferItemsCol() *firestore.CollectionRef {
	return r.Client.Collection("orderTransferItems")
}

func (r *ResaleTradeReaderFS) ListByResaleID(
	ctx context.Context,
	resaleID string,
) ([]tradedom.Trade, error) {
	if r == nil || r.Client == nil || r.tradeRepo == nil {
		return nil, ErrResaleTradeReaderNotConfigured
	}
	if resaleID == "" {
		return nil, orderdom.ErrNotFound
	}

	iter := r.transferItemsCol().
		Where("resaleId", "==", resaleID).
		Documents(ctx)
	defer iter.Stop()

	tradesByID := make(map[string]tradedom.Trade)

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		projection, err := orderTransferItemFromSnapshot(snap)
		if err != nil {
			return nil, err
		}

		if projection.ItemType != orderdom.OrderItemTypeResale {
			continue
		}
		if projection.ResaleID != resaleID {
			return nil, ErrTransferItemProjectionMismatch
		}

		trade, err := r.tradeRepo.GetByOrderItem(
			ctx,
			projection.OrderID,
			projection.ItemIndex,
		)
		if err != nil {
			if errors.Is(err, tradedom.ErrNotFound) {
				continue
			}
			return nil, err
		}

		if trade.OrderID != projection.OrderID ||
			trade.OrderItemIndex != projection.ItemIndex {
			return nil, ErrInvalidTradeDocumentData
		}
		if trade.SellerType != tradedom.SellerTypeAvatar {
			return nil, ErrInvalidTradeDocumentData
		}

		tradesByID[trade.ID] = trade
	}

	trades := make([]tradedom.Trade, 0, len(tradesByID))
	for _, trade := range tradesByID {
		trades = append(trades, trade)
	}

	sort.Slice(trades, func(i, j int) bool {
		left := trades[i]
		right := trades[j]

		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		return left.ID < right.ID
	})

	return trades, nil
}
