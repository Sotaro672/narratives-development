// backend/internal/adapters/out/firestore/trade_message_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeMessageRepositoryNotConfigured = errors.New("trade_message_repository_fs: not configured")
	ErrInvalidTradeMessageDocumentData     = errors.New("trade_message_repository_fs: invalid message document data")
)

// TradeMessageRepositoryFS implements tradedom.MessageRepository using Firestore.
//
// Firestore:
//
//	trades/{tradeId}/messages/{messageId}
//
// Message creation also updates:
//
//	trades/{tradeId}.lastMessageAt
//	trades/{tradeId}.updatedAt
//
// in the same Firestore transaction.
type TradeMessageRepositoryFS struct {
	Client *firestore.Client
}

var _ tradedom.MessageRepository = (*TradeMessageRepositoryFS)(nil)

func NewTradeMessageRepositoryFS(client *firestore.Client) *TradeMessageRepositoryFS {
	return &TradeMessageRepositoryFS{Client: client}
}

func (r *TradeMessageRepositoryFS) tradesCol() *firestore.CollectionRef {
	return r.Client.Collection("trades")
}

func (r *TradeMessageRepositoryFS) col(tradeID string) *firestore.CollectionRef {
	return r.tradesCol().Doc(tradeID).Collection("messages")
}

// ============================================================
// Create
// ============================================================

func (r *TradeMessageRepositoryFS) Create(ctx context.Context, message tradedom.Message) (tradedom.Message, error) {
	if r == nil || r.Client == nil {
		return tradedom.Message{}, ErrTradeMessageRepositoryNotConfigured
	}
	if message.TradeID == "" {
		return tradedom.Message{}, tradedom.ErrInvalidMessageTradeID
	}

	var messageRef *firestore.DocumentRef
	if message.ID == "" {
		messageRef = r.col(message.TradeID).NewDoc()
		message.ID = messageRef.ID
	} else {
		messageRef = r.col(message.TradeID).Doc(message.ID)
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	} else {
		message.CreatedAt = message.CreatedAt.UTC()
	}

	message.BuyerReadAt = cloneTradeMessageTimePtr(message.BuyerReadAt)
	message.SellerReadAt = cloneTradeMessageTimePtr(message.SellerReadAt)

	if message.Images == nil {
		message.Images = []tradedom.MessageImage{}
	}

	if err := message.ValidateForPersist(); err != nil {
		return tradedom.Message{}, err
	}

	tradeRef := r.tradesCol().Doc(message.TradeID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		tradeSnap, err := tx.Get(tradeRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return tradedom.ErrNotFound
			}
			return err
		}

		trade, err := docToTrade(tradeSnap)
		if err != nil {
			return err
		}
		if trade.Status == tradedom.StatusClosed {
			return tradedom.ErrTradeAlreadyClosed
		}
		if err := trade.MarkMessageActivity(message.CreatedAt); err != nil {
			return err
		}

		existingMessageSnap, err := tx.Get(messageRef)
		if err == nil && existingMessageSnap.Exists() {
			return tradedom.ErrMessageAlreadyExists
		}
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		if err := tx.Create(messageRef, tradeMessageToDoc(message)); err != nil {
			return err
		}

		return tx.Update(tradeRef, []firestore.Update{
			{Path: "lastMessageAt", Value: message.CreatedAt.UTC()},
			{Path: "updatedAt", Value: message.CreatedAt.UTC()},
		})
	})
	if err != nil {
		if errors.Is(err, tradedom.ErrMessageAlreadyExists) || status.Code(err) == codes.AlreadyExists {
			return tradedom.Message{}, tradedom.ErrMessageAlreadyExists
		}
		return tradedom.Message{}, err
	}

	return message, nil
}

// ============================================================
// Read
// ============================================================

func (r *TradeMessageRepositoryFS) GetByID(ctx context.Context, tradeID string, messageID string) (tradedom.Message, error) {
	if r == nil || r.Client == nil {
		return tradedom.Message{}, ErrTradeMessageRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.Message{}, tradedom.ErrInvalidMessageTradeID
	}
	if messageID == "" {
		return tradedom.Message{}, tradedom.ErrInvalidMessageID
	}

	snap, err := r.col(tradeID).Doc(messageID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.Message{}, tradedom.ErrMessageNotFound
		}
		return tradedom.Message{}, err
	}

	message, err := docToTradeMessage(snap)
	if err != nil {
		return tradedom.Message{}, err
	}
	if message.ID != messageID || message.TradeID != tradeID {
		return tradedom.Message{}, tradedom.ErrMessageNotFound
	}

	return message, nil
}

func (r *TradeMessageRepositoryFS) ListByTradeID(ctx context.Context, tradeID string, filter tradedom.MessageListFilter) ([]tradedom.Message, error) {
	if r == nil || r.Client == nil {
		return nil, ErrTradeMessageRepositoryNotConfigured
	}
	if tradeID == "" {
		return nil, tradedom.ErrInvalidMessageTradeID
	}
	if err := r.ensureTradeExists(ctx, tradeID); err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = tradedom.DefaultMessageListLimit
	}
	if limit > tradedom.MaxMessageListLimit {
		limit = tradedom.MaxMessageListLimit
	}

	var before *time.Time
	if filter.BeforeCreatedAt != nil {
		if filter.BeforeCreatedAt.IsZero() {
			return nil, tradedom.ErrInvalidMessageCreatedAt
		}
		value := filter.BeforeCreatedAt.UTC()
		before = &value
	}

	var after *time.Time
	if filter.AfterCreatedAt != nil {
		if filter.AfterCreatedAt.IsZero() {
			return nil, tradedom.ErrInvalidMessageCreatedAt
		}
		value := filter.AfterCreatedAt.UTC()
		after = &value
	}

	it := r.col(tradeID).Documents(ctx)
	defer it.Stop()

	messages := make([]tradedom.Message, 0)

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		message, err := docToTradeMessage(snap)
		if err != nil {
			return nil, err
		}
		if message.TradeID != tradeID {
			return nil, ErrInvalidTradeMessageDocumentData
		}
		if before != nil && !message.CreatedAt.Before(*before) {
			continue
		}
		if after != nil && !message.CreatedAt.After(*after) {
			continue
		}

		messages = append(messages, message)
	}

	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})

	if len(messages) <= limit {
		return messages, nil
	}

	if after != nil {
		result := make([]tradedom.Message, limit)
		copy(result, messages[:limit])
		return result, nil
	}

	start := len(messages) - limit
	result := make([]tradedom.Message, limit)
	copy(result, messages[start:])
	return result, nil
}

// ============================================================
// Read state
// ============================================================

func (r *TradeMessageRepositoryFS) MarkReadByBuyer(ctx context.Context, tradeID string, readAt time.Time) error {
	return r.markRead(ctx, tradeID, tradedom.MessageSenderSideBuyer, readAt)
}

func (r *TradeMessageRepositoryFS) MarkReadBySeller(ctx context.Context, tradeID string, readAt time.Time) error {
	return r.markRead(ctx, tradeID, tradedom.MessageSenderSideSeller, readAt)
}

func (r *TradeMessageRepositoryFS) markRead(ctx context.Context, tradeID string, readerSide tradedom.MessageSenderSide, readAt time.Time) error {
	if r == nil || r.Client == nil {
		return ErrTradeMessageRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.ErrInvalidMessageTradeID
	}
	if readAt.IsZero() {
		switch readerSide {
		case tradedom.MessageSenderSideBuyer:
			return tradedom.ErrInvalidBuyerReadAt
		case tradedom.MessageSenderSideSeller:
			return tradedom.ErrInvalidSellerReadAt
		default:
			return tradedom.ErrInvalidMessageSenderSide
		}
	}
	if readerSide != tradedom.MessageSenderSideBuyer && readerSide != tradedom.MessageSenderSideSeller {
		return tradedom.ErrInvalidMessageSenderSide
	}
	if err := r.ensureTradeExists(ctx, tradeID); err != nil {
		return err
	}

	readAt = readAt.UTC()

	it := r.col(tradeID).Documents(ctx)
	defer it.Stop()

	bulk := r.Client.BulkWriter(ctx)
	defer bulk.End()

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		message, err := docToTradeMessage(snap)
		if err != nil {
			return err
		}
		if message.TradeID != tradeID {
			return ErrInvalidTradeMessageDocumentData
		}
		if message.SenderSide == readerSide {
			continue
		}
		if message.CreatedAt.After(readAt) {
			continue
		}

		switch readerSide {
		case tradedom.MessageSenderSideBuyer:
			if message.BuyerReadAt != nil {
				continue
			}
			if _, err := bulk.Update(snap.Ref, []firestore.Update{{Path: "buyerReadAt", Value: readAt}}); err != nil {
				return err
			}

		case tradedom.MessageSenderSideSeller:
			if message.SellerReadAt != nil {
				continue
			}
			if _, err := bulk.Update(snap.Ref, []firestore.Update{{Path: "sellerReadAt", Value: readAt}}); err != nil {
				return err
			}
		}
	}

	return nil
}

// ============================================================
// Unread count
// ============================================================

func (r *TradeMessageRepositoryFS) CountUnreadForBuyer(ctx context.Context, tradeID string) (int, error) {
	return r.countUnread(ctx, tradeID, tradedom.MessageSenderSideBuyer)
}

func (r *TradeMessageRepositoryFS) CountUnreadForSeller(ctx context.Context, tradeID string) (int, error) {
	return r.countUnread(ctx, tradeID, tradedom.MessageSenderSideSeller)
}

func (r *TradeMessageRepositoryFS) countUnread(ctx context.Context, tradeID string, readerSide tradedom.MessageSenderSide) (int, error) {
	if r == nil || r.Client == nil {
		return 0, ErrTradeMessageRepositoryNotConfigured
	}
	if tradeID == "" {
		return 0, tradedom.ErrInvalidMessageTradeID
	}
	if readerSide != tradedom.MessageSenderSideBuyer && readerSide != tradedom.MessageSenderSideSeller {
		return 0, tradedom.ErrInvalidMessageSenderSide
	}
	if err := r.ensureTradeExists(ctx, tradeID); err != nil {
		return 0, err
	}

	it := r.col(tradeID).Documents(ctx)
	defer it.Stop()

	count := 0

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}

		message, err := docToTradeMessage(snap)
		if err != nil {
			return 0, err
		}
		if message.TradeID != tradeID {
			return 0, ErrInvalidTradeMessageDocumentData
		}
		if message.SenderSide == readerSide {
			continue
		}

		switch readerSide {
		case tradedom.MessageSenderSideBuyer:
			if message.BuyerReadAt == nil {
				count++
			}
		case tradedom.MessageSenderSideSeller:
			if message.SellerReadAt == nil {
				count++
			}
		}
	}

	return count, nil
}

// ============================================================
// Firestore documents
// ============================================================

type tradeMessageDoc struct {
	ID      string `firestore:"id"`
	TradeID string `firestore:"tradeId"`

	SenderSide string `firestore:"senderSide"`
	SenderType string `firestore:"senderType"`
	SenderID   string `firestore:"senderId"`

	Content string                 `firestore:"content,omitempty"`
	Images  []tradeMessageImageDoc `firestore:"images,omitempty"`

	BuyerReadAt  *time.Time `firestore:"buyerReadAt,omitempty"`
	SellerReadAt *time.Time `firestore:"sellerReadAt,omitempty"`

	CreatedAt time.Time `firestore:"createdAt"`
}

type tradeMessageImageDoc struct {
	FileName   string `firestore:"fileName"`
	FileURL    string `firestore:"fileUrl"`
	ObjectPath string `firestore:"objectPath"`
	FileSize   int64  `firestore:"fileSize"`
	MIMEType   string `firestore:"mimeType"`
}

func tradeMessageToDoc(message tradedom.Message) tradeMessageDoc {
	images := make([]tradeMessageImageDoc, 0, len(message.Images))

	for _, image := range message.Images {
		images = append(images, tradeMessageImageDoc{
			FileName:   image.FileName,
			FileURL:    image.FileURL,
			ObjectPath: image.ObjectPath,
			FileSize:   image.FileSize,
			MIMEType:   image.MIMEType,
		})
	}

	return tradeMessageDoc{
		ID:           message.ID,
		TradeID:      message.TradeID,
		SenderSide:   string(message.SenderSide),
		SenderType:   string(message.SenderType),
		SenderID:     message.SenderID,
		Content:      message.Content,
		Images:       images,
		BuyerReadAt:  cloneTradeMessageTimePtr(message.BuyerReadAt),
		SellerReadAt: cloneTradeMessageTimePtr(message.SellerReadAt),
		CreatedAt:    message.CreatedAt.UTC(),
	}
}

func docToTradeMessage(snap *firestore.DocumentSnapshot) (tradedom.Message, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return tradedom.Message{}, tradedom.ErrMessageNotFound
	}

	var doc tradeMessageDoc
	if err := snap.DataTo(&doc); err != nil {
		return tradedom.Message{}, err
	}

	images := make([]tradedom.MessageImage, 0, len(doc.Images))
	for _, image := range doc.Images {
		images = append(images, tradedom.MessageImage{
			FileName:   image.FileName,
			FileURL:    image.FileURL,
			ObjectPath: image.ObjectPath,
			FileSize:   image.FileSize,
			MIMEType:   image.MIMEType,
		})
	}

	message := tradedom.Message{
		ID:           doc.ID,
		TradeID:      doc.TradeID,
		SenderSide:   tradedom.MessageSenderSide(doc.SenderSide),
		SenderType:   tradedom.MessageSenderType(doc.SenderType),
		SenderID:     doc.SenderID,
		Content:      doc.Content,
		Images:       images,
		BuyerReadAt:  cloneTradeMessageTimePtr(doc.BuyerReadAt),
		SellerReadAt: cloneTradeMessageTimePtr(doc.SellerReadAt),
		CreatedAt:    doc.CreatedAt.UTC(),
	}

	if message.Images == nil {
		message.Images = []tradedom.MessageImage{}
	}

	if err := message.ValidateForPersist(); err != nil {
		return tradedom.Message{}, fmt.Errorf("trade message %s/%s: %w: %v", message.TradeID, message.ID, ErrInvalidTradeMessageDocumentData, err)
	}

	return message, nil
}

// ============================================================
// Helpers
// ============================================================

func (r *TradeMessageRepositoryFS) ensureTradeExists(ctx context.Context, tradeID string) error {
	snap, err := r.tradesCol().Doc(tradeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.ErrNotFound
		}
		return err
	}
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return tradedom.ErrNotFound
	}
	if _, err := docToTrade(snap); err != nil {
		return err
	}

	return nil
}

func cloneTradeMessageTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
