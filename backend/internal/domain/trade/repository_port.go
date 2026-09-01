// backend/internal/domain/trade/repository_port.go
package trade

import (
	"context"
	"errors"
	"time"
)

// ============================================================
// Contract errors
// ============================================================

var (
	ErrNotFound      = errors.New("trade: not found")
	ErrConflict      = errors.New("trade: conflict")
	ErrAlreadyExists = errors.New("trade: already exists")

	ErrMessageNotFound      = errors.New("trade: message not found")
	ErrMessageAlreadyExists = errors.New("trade: message already exists")
)

// ============================================================
// Trade repository
// ============================================================

// Repository is the repository port for Trade aggregate roots.
//
// Firestore:
//
//	trades/{tradeId}
//
// One Trade belongs to exactly one Resale Order item:
//
//	orderId + orderItemIndex -> Trade
//
// Trade does not own Order cancellation, return, refund, dispatch, transfer,
// or settlement state. Those states remain authoritative in their respective
// domains.
//
// Trade is intentionally not physically deleted because it represents
// transaction communication history.
type Repository interface {
	// GetByID retrieves one Trade by document ID.
	GetByID(ctx context.Context, id string) (Trade, error)

	// GetByOrderItem retrieves the Trade associated with one Order item.
	//
	// There must be at most one Trade for:
	//
	//	orderId + orderItemIndex
	//
	// Implementations should return ErrNotFound when no Trade exists and
	// ErrConflict if persisted data contains multiple matching Trades.
	GetByOrderItem(ctx context.Context, orderID string, orderItemIndex int) (Trade, error)

	// ListByAvatarID returns Trades in which avatarID participates as either
	// buyer or Resale seller.
	//
	// Expected participant conditions:
	//   - buyerAvatarId == avatarID
	//   - sellerAvatarId == avatarID
	//
	// Implementations must merge both roles without returning duplicate Trades.
	// Returned Trades should belong only to Avatar-to-Avatar Resale transactions.
	//
	// Ordering is not part of the repository contract. The application/query
	// layer should determine presentation ordering using LastMessageAt,
	// UpdatedAt, or CreatedAt as appropriate.
	ListByAvatarID(ctx context.Context, avatarID string) ([]Trade, error)

	// Create persists one Trade.
	//
	// Trade creation is expected to be idempotent at the application layer.
	// When a Trade with the same ID already exists, implementations should
	// return ErrAlreadyExists.
	//
	// Implementations must also prevent more than one Trade from representing
	// the same orderId + orderItemIndex.
	Create(ctx context.Context, trade Trade) (Trade, error)

	// Update persists mutable Trade state.
	//
	// Expected mutable fields:
	//   - Status
	//   - UpdatedAt
	//   - LastMessageAt
	//
	// Immutable identity fields must not be changed:
	//   - ID
	//   - OrderID
	//   - OrderItemIndex
	//   - BuyerAvatarID
	//   - SellerType
	//   - SellerCompanyID
	//   - SellerBrandID
	//   - SellerAvatarID
	//   - CreatedAt
	Update(ctx context.Context, id string, trade Trade) (Trade, error)
}

// ============================================================
// Message list filter
// ============================================================

const (
	DefaultMessageListLimit = 50
	MaxMessageListLimit     = 100
)

// MessageListFilter controls pagination of messages under one Trade.
//
// Messages should normally be ordered by CreatedAt.
//
// BeforeCreatedAt is useful when loading older messages by scrolling upward.
// AfterCreatedAt can be used when polling or synchronizing newer messages.
type MessageListFilter struct {
	Limit int

	BeforeCreatedAt *time.Time
	AfterCreatedAt  *time.Time
}

// ============================================================
// Message repository
// ============================================================

// MessageRepository is the repository port for Trade messages.
//
// Firestore:
//
//	trades/{tradeId}/messages/{messageId}
//
// Trade messages are treated as immutable transaction communication records.
// Editing or physical deletion is intentionally not part of this contract.
//
// Read state is maintained independently for buyer and seller using:
//
//	buyerReadAt
//	sellerReadAt
type MessageRepository interface {
	// Create persists one Trade message.
	//
	// Implementations should return ErrMessageAlreadyExists if a message with
	// the same tradeId + messageId already exists.
	Create(ctx context.Context, message Message) (Message, error)

	// GetByID retrieves one message from the Trade messages subcollection.
	GetByID(ctx context.Context, tradeID string, messageID string) (Message, error)

	// ListByTradeID returns messages belonging to one Trade.
	//
	// Repository implementations must scope by tradeID and must never query
	// messages globally using messageID alone.
	ListByTradeID(ctx context.Context, tradeID string, filter MessageListFilter) ([]Message, error)

	// MarkReadByBuyer marks seller/system-originated messages as read by the buyer.
	//
	// Implementations must not mark buyer-originated messages as newly read.
	//
	//	buyerReadAt = readAt
	MarkReadByBuyer(ctx context.Context, tradeID string, readAt time.Time) error

	// MarkReadBySeller marks buyer/system-originated messages as read by the seller.
	//
	// Implementations must not mark seller-originated messages as newly read.
	//
	//	sellerReadAt = readAt
	MarkReadBySeller(ctx context.Context, tradeID string, readAt time.Time) error

	// CountUnreadForBuyer counts messages that should be presented as unread
	// to the buyer.
	//
	// Expected condition:
	//   - senderSide != buyer
	//   - buyerReadAt == nil
	CountUnreadForBuyer(ctx context.Context, tradeID string) (int, error)

	// CountUnreadForSeller counts messages that should be presented as unread
	// to the seller.
	//
	// Expected condition:
	//   - senderSide != seller
	//   - sellerReadAt == nil
	CountUnreadForSeller(ctx context.Context, tradeID string) (int, error)
}
