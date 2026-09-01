// backend/internal/domain/trade/entity.go
package trade

import (
	"errors"
	"strings"
	"time"
)

// ============================================================
// Types
// ============================================================

type Status string
type SellerType string
type MessageSenderSide string
type MessageSenderType string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

const (
	SellerTypeCompany SellerType = "company"
	SellerTypeAvatar  SellerType = "avatar"
)

const (
	MessageSenderSideBuyer  MessageSenderSide = "buyer"
	MessageSenderSideSeller MessageSenderSide = "seller"
	MessageSenderSideSystem MessageSenderSide = "system"
)

const (
	MessageSenderTypeAvatar MessageSenderType = "avatar"
	MessageSenderTypeSystem MessageSenderType = "system"
)

// ============================================================
// Policy
// ============================================================

const (
	MaxReferenceIDLength           = 256
	MaxMessageContentLength        = 5000
	MaxMessageImageCount           = 10
	MaxImageFileNameLength         = 255
	MaxImageFileURLLength          = 4096
	MaxImageObjectPathLength       = 1024
	MaxImageMIMETypeLength         = 128
	MaxMessageImageSize      int64 = 20 * 1024 * 1024
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidID              = errors.New("trade: invalid id")
	ErrInvalidOrderID         = errors.New("trade: invalid orderId")
	ErrInvalidOrderItemIndex  = errors.New("trade: invalid orderItemIndex")
	ErrInvalidBuyerAvatarID   = errors.New("trade: invalid buyerAvatarId")
	ErrInvalidSellerType      = errors.New("trade: invalid sellerType")
	ErrInvalidSellerCompanyID = errors.New(
		"trade: invalid sellerCompanyId",
	)
	ErrInvalidSellerBrandID = errors.New(
		"trade: invalid sellerBrandId",
	)
	ErrInvalidSellerAvatarID = errors.New(
		"trade: invalid sellerAvatarId",
	)
	ErrBuyerAndSellerMustDiffer = errors.New(
		"trade: buyer and seller avatar must differ",
	)
	ErrInvalidStatus        = errors.New("trade: invalid status")
	ErrInvalidCreatedAt     = errors.New("trade: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("trade: invalid updatedAt")
	ErrInvalidLastMessageAt = errors.New("trade: invalid lastMessageAt")
	ErrTradeAlreadyClosed   = errors.New("trade: already closed")

	ErrInvalidMessageID         = errors.New("trade: invalid message id")
	ErrInvalidMessageTradeID    = errors.New("trade: invalid message tradeId")
	ErrInvalidMessageSenderSide = errors.New(
		"trade: invalid message senderSide",
	)
	ErrInvalidMessageSenderType = errors.New(
		"trade: invalid message senderType",
	)
	ErrInvalidMessageSenderID = errors.New(
		"trade: invalid message senderId",
	)
	ErrInvalidMessageContent = errors.New(
		"trade: invalid message content",
	)
	ErrMessageContentOrImageRequired = errors.New(
		"trade: message content or image is required",
	)
	ErrInvalidMessageCreatedAt = errors.New(
		"trade: invalid message createdAt",
	)
	ErrInvalidBuyerReadAt = errors.New(
		"trade: invalid buyerReadAt",
	)
	ErrInvalidSellerReadAt = errors.New(
		"trade: invalid sellerReadAt",
	)

	ErrTooManyMessageImages = errors.New(
		"trade: too many message images",
	)
	ErrInvalidMessageImageFileName = errors.New(
		"trade: invalid message image fileName",
	)
	ErrInvalidMessageImageFileURL = errors.New(
		"trade: invalid message image fileUrl",
	)
	ErrInvalidMessageImageObjectPath = errors.New(
		"trade: invalid message image objectPath",
	)
	ErrInvalidMessageImageFileSize = errors.New(
		"trade: invalid message image fileSize",
	)
	ErrInvalidMessageImageMIMEType = errors.New(
		"trade: invalid message image mimeType",
	)
	ErrDuplicateMessageImage = errors.New(
		"trade: duplicate message image objectPath",
	)
	ErrInvalidMessageSenderCombination = errors.New(
		"trade: invalid message sender combination",
	)
)

// ============================================================
// Trade
// ============================================================

// Trade represents one buyer-seller transaction conversation.
//
// A Trade is created for one Order item.
// Order state remains authoritative for cancellation, dispatch, return,
// token transfer, refund, and settlement state.
//
// Seller identity is snapshotted from OrderItemSnapshot.SellerSnapshot:
//   - list sale: sellerType=company, sellerCompanyId and sellerBrandId
//   - resale sale: sellerType=avatar, sellerAvatarId
//
// Firestore:
//
//	trades/{tradeId}
type Trade struct {
	ID             string `json:"id"`
	OrderID        string `json:"orderId"`
	OrderItemIndex int    `json:"orderItemIndex"`

	BuyerAvatarID string `json:"buyerAvatarId"`

	SellerType      SellerType `json:"sellerType"`
	SellerCompanyID string     `json:"sellerCompanyId,omitempty"`
	SellerBrandID   string     `json:"sellerBrandId,omitempty"`
	SellerAvatarID  string     `json:"sellerAvatarId,omitempty"`

	Status Status `json:"status"`

	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	LastMessageAt *time.Time `json:"lastMessageAt,omitempty"`
}

func (t Trade) GetID() string {
	return t.ID
}

// NewForCreate creates a Trade before persistence.
//
// ID may be empty when the repository generates it. The application layer may
// also provide a deterministic ID derived from orderId + orderItemIndex.
//
// CreatedAt / UpdatedAt may be zero because the repository can fill them.
func NewForCreate(
	id string,
	orderID string,
	orderItemIndex int,
	buyerAvatarID string,
	sellerType SellerType,
	sellerCompanyID string,
	sellerBrandID string,
	sellerAvatarID string,
) (Trade, error) {
	t := Trade{
		ID:              strings.TrimSpace(id),
		OrderID:         strings.TrimSpace(orderID),
		OrderItemIndex:  orderItemIndex,
		BuyerAvatarID:   strings.TrimSpace(buyerAvatarID),
		SellerType:      sellerType,
		SellerCompanyID: strings.TrimSpace(sellerCompanyID),
		SellerBrandID:   strings.TrimSpace(sellerBrandID),
		SellerAvatarID:  strings.TrimSpace(sellerAvatarID),
		Status:          StatusActive,
	}

	if err := t.ValidateForCreate(); err != nil {
		return Trade{}, err
	}

	return t, nil
}

// NewCompanyTradeForCreate creates a Trade for a primary List sale.
func NewCompanyTradeForCreate(
	id string,
	orderID string,
	orderItemIndex int,
	buyerAvatarID string,
	sellerCompanyID string,
	sellerBrandID string,
) (Trade, error) {
	return NewForCreate(
		id,
		orderID,
		orderItemIndex,
		buyerAvatarID,
		SellerTypeCompany,
		sellerCompanyID,
		sellerBrandID,
		"",
	)
}

// NewAvatarTradeForCreate creates a Trade for a Resale transaction.
func NewAvatarTradeForCreate(
	id string,
	orderID string,
	orderItemIndex int,
	buyerAvatarID string,
	sellerAvatarID string,
) (Trade, error) {
	return NewForCreate(
		id,
		orderID,
		orderItemIndex,
		buyerAvatarID,
		SellerTypeAvatar,
		"",
		"",
		sellerAvatarID,
	)
}

func (t *Trade) MarkMessageActivity(at time.Time) error {
	if t == nil {
		return nil
	}
	if at.IsZero() {
		return ErrInvalidLastMessageAt
	}

	at = at.UTC()
	if !t.CreatedAt.IsZero() && at.Before(t.CreatedAt) {
		return ErrInvalidLastMessageAt
	}
	if t.LastMessageAt != nil && at.Before(*t.LastMessageAt) {
		return ErrInvalidLastMessageAt
	}

	t.LastMessageAt = &at
	t.UpdatedAt = at
	return nil
}

func (t *Trade) Close(at time.Time) error {
	if t == nil {
		return nil
	}
	if t.Status == StatusClosed {
		return ErrTradeAlreadyClosed
	}
	if at.IsZero() {
		return ErrInvalidUpdatedAt
	}

	at = at.UTC()
	if !t.CreatedAt.IsZero() && at.Before(t.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	t.Status = StatusClosed
	t.UpdatedAt = at
	return nil
}

func (t Trade) ValidateForCreate() error {
	if t.ID != "" && !isValidReferenceID(t.ID) {
		return ErrInvalidID
	}
	if !isValidReferenceID(t.OrderID) {
		return ErrInvalidOrderID
	}
	if t.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}
	if !isValidReferenceID(t.BuyerAvatarID) {
		return ErrInvalidBuyerAvatarID
	}
	if err := validateSeller(t); err != nil {
		return err
	}
	if t.Status != "" && !IsValidStatus(t.Status) {
		return ErrInvalidStatus
	}
	if !t.CreatedAt.IsZero() && !t.UpdatedAt.IsZero() && t.UpdatedAt.Before(t.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if t.LastMessageAt != nil {
		if t.LastMessageAt.IsZero() {
			return ErrInvalidLastMessageAt
		}
		if !t.CreatedAt.IsZero() && t.LastMessageAt.Before(t.CreatedAt) {
			return ErrInvalidLastMessageAt
		}
	}

	return nil
}

func (t Trade) ValidateForPersist() error {
	if !isValidReferenceID(t.ID) {
		return ErrInvalidID
	}
	if !isValidReferenceID(t.OrderID) {
		return ErrInvalidOrderID
	}
	if t.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}
	if !isValidReferenceID(t.BuyerAvatarID) {
		return ErrInvalidBuyerAvatarID
	}
	if err := validateSeller(t); err != nil {
		return err
	}
	if !IsValidStatus(t.Status) {
		return ErrInvalidStatus
	}
	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if t.UpdatedAt.IsZero() || t.UpdatedAt.Before(t.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if t.LastMessageAt != nil {
		if t.LastMessageAt.IsZero() || t.LastMessageAt.Before(t.CreatedAt) {
			return ErrInvalidLastMessageAt
		}
		if t.LastMessageAt.After(t.UpdatedAt) {
			return ErrInvalidLastMessageAt
		}
	}

	return nil
}

func validateSeller(t Trade) error {
	if !IsValidSellerType(t.SellerType) {
		return ErrInvalidSellerType
	}

	switch t.SellerType {
	case SellerTypeCompany:
		if !isValidReferenceID(t.SellerCompanyID) {
			return ErrInvalidSellerCompanyID
		}
		if !isValidReferenceID(t.SellerBrandID) {
			return ErrInvalidSellerBrandID
		}
		if t.SellerAvatarID != "" {
			return ErrInvalidSellerAvatarID
		}

	case SellerTypeAvatar:
		if !isValidReferenceID(t.SellerAvatarID) {
			return ErrInvalidSellerAvatarID
		}
		if t.SellerCompanyID != "" {
			return ErrInvalidSellerCompanyID
		}
		if t.SellerBrandID != "" {
			return ErrInvalidSellerBrandID
		}
		if t.BuyerAvatarID == t.SellerAvatarID {
			return ErrBuyerAndSellerMustDiffer
		}
	}

	return nil
}

// ============================================================
// Trade Message
// ============================================================

// Message represents one chat or system timeline message.
//
// Firestore:
//
//	trades/{tradeId}/messages/{messageId}
type Message struct {
	ID      string `json:"id"`
	TradeID string `json:"tradeId"`

	SenderSide MessageSenderSide `json:"senderSide"`
	SenderType MessageSenderType `json:"senderType"`
	SenderID   string            `json:"senderId"`

	Content string         `json:"content,omitempty"`
	Images  []MessageImage `json:"images,omitempty"`

	BuyerReadAt  *time.Time `json:"buyerReadAt,omitempty"`
	SellerReadAt *time.Time `json:"sellerReadAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

type MessageImage struct {
	FileName   string `json:"fileName"`
	FileURL    string `json:"fileUrl"`
	ObjectPath string `json:"objectPath"`
	FileSize   int64  `json:"fileSize"`
	MIMEType   string `json:"mimeType"`
}

func (m Message) GetID() string {
	return m.ID
}

// NewMessageForCreate creates a user-originated Trade message.
//
// ID and CreatedAt may be empty/zero before repository persistence.
func NewMessageForCreate(
	id string,
	tradeID string,
	senderSide MessageSenderSide,
	senderType MessageSenderType,
	senderID string,
	content string,
	images []MessageImage,
) (Message, error) {
	m := Message{
		ID:         strings.TrimSpace(id),
		TradeID:    strings.TrimSpace(tradeID),
		SenderSide: senderSide,
		SenderType: senderType,
		SenderID:   strings.TrimSpace(senderID),
		Content:    strings.TrimSpace(content),
		Images:     cloneMessageImages(images),
	}

	if m.Images == nil {
		m.Images = []MessageImage{}
	}

	if err := m.ValidateForCreate(); err != nil {
		return Message{}, err
	}

	return m, nil
}

// NewSystemMessageForCreate creates an application-generated timeline message.
func NewSystemMessageForCreate(
	id string,
	tradeID string,
	content string,
) (Message, error) {
	return NewMessageForCreate(
		id,
		tradeID,
		MessageSenderSideSystem,
		MessageSenderTypeSystem,
		"system",
		content,
		nil,
	)
}

func (m *Message) MarkReadByBuyer(at time.Time) error {
	if m == nil {
		return nil
	}
	if at.IsZero() {
		return ErrInvalidBuyerReadAt
	}

	at = at.UTC()
	if !m.CreatedAt.IsZero() && at.Before(m.CreatedAt) {
		return ErrInvalidBuyerReadAt
	}

	if m.BuyerReadAt != nil && !at.After(*m.BuyerReadAt) {
		return nil
	}

	m.BuyerReadAt = &at
	return nil
}

func (m *Message) MarkReadBySeller(at time.Time) error {
	if m == nil {
		return nil
	}
	if at.IsZero() {
		return ErrInvalidSellerReadAt
	}

	at = at.UTC()
	if !m.CreatedAt.IsZero() && at.Before(m.CreatedAt) {
		return ErrInvalidSellerReadAt
	}

	if m.SellerReadAt != nil && !at.After(*m.SellerReadAt) {
		return nil
	}

	m.SellerReadAt = &at
	return nil
}

func (m Message) ValidateForCreate() error {
	if m.ID != "" && !isValidReferenceID(m.ID) {
		return ErrInvalidMessageID
	}
	if !isValidReferenceID(m.TradeID) {
		return ErrInvalidMessageTradeID
	}
	if err := validateMessageSender(m); err != nil {
		return err
	}
	if err := validateMessageContent(m.Content, m.Images); err != nil {
		return err
	}
	if err := validateMessageImages(m.Images); err != nil {
		return err
	}
	if m.BuyerReadAt != nil {
		if m.BuyerReadAt.IsZero() {
			return ErrInvalidBuyerReadAt
		}
		if !m.CreatedAt.IsZero() && m.BuyerReadAt.Before(m.CreatedAt) {
			return ErrInvalidBuyerReadAt
		}
	}
	if m.SellerReadAt != nil {
		if m.SellerReadAt.IsZero() {
			return ErrInvalidSellerReadAt
		}
		if !m.CreatedAt.IsZero() && m.SellerReadAt.Before(m.CreatedAt) {
			return ErrInvalidSellerReadAt
		}
	}

	return nil
}

func (m Message) ValidateForPersist() error {
	if !isValidReferenceID(m.ID) {
		return ErrInvalidMessageID
	}
	if !isValidReferenceID(m.TradeID) {
		return ErrInvalidMessageTradeID
	}
	if err := validateMessageSender(m); err != nil {
		return err
	}
	if err := validateMessageContent(m.Content, m.Images); err != nil {
		return err
	}
	if err := validateMessageImages(m.Images); err != nil {
		return err
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidMessageCreatedAt
	}
	if m.BuyerReadAt != nil && (m.BuyerReadAt.IsZero() || m.BuyerReadAt.Before(m.CreatedAt)) {
		return ErrInvalidBuyerReadAt
	}
	if m.SellerReadAt != nil && (m.SellerReadAt.IsZero() || m.SellerReadAt.Before(m.CreatedAt)) {
		return ErrInvalidSellerReadAt
	}

	return nil
}

func validateMessageSender(m Message) error {
	if !IsValidMessageSenderSide(m.SenderSide) {
		return ErrInvalidMessageSenderSide
	}
	if !IsValidMessageSenderType(m.SenderType) {
		return ErrInvalidMessageSenderType
	}
	if !isValidReferenceID(m.SenderID) {
		return ErrInvalidMessageSenderID
	}

	switch m.SenderSide {
	case MessageSenderSideBuyer:
		if m.SenderType != MessageSenderTypeAvatar {
			return ErrInvalidMessageSenderCombination
		}

	case MessageSenderSideSeller:
		if m.SenderType != MessageSenderTypeAvatar {
			return ErrInvalidMessageSenderCombination
		}

	case MessageSenderSideSystem:
		if m.SenderType != MessageSenderTypeSystem ||
			m.SenderID != "system" {
			return ErrInvalidMessageSenderCombination
		}
	}

	return nil
}

func validateMessageContent(
	content string,
	images []MessageImage,
) error {
	if len([]rune(content)) > MaxMessageContentLength {
		return ErrInvalidMessageContent
	}
	if strings.TrimSpace(content) == "" && len(images) == 0 {
		return ErrMessageContentOrImageRequired
	}

	return nil
}

func validateMessageImages(images []MessageImage) error {
	if len(images) > MaxMessageImageCount {
		return ErrTooManyMessageImages
	}

	objectPaths := make(map[string]struct{}, len(images))
	for _, image := range images {
		if image.FileName == "" ||
			len([]rune(image.FileName)) > MaxImageFileNameLength {
			return ErrInvalidMessageImageFileName
		}
		if image.FileURL == "" ||
			len([]rune(image.FileURL)) > MaxImageFileURLLength {
			return ErrInvalidMessageImageFileURL
		}
		if image.ObjectPath == "" ||
			len([]rune(image.ObjectPath)) > MaxImageObjectPathLength {
			return ErrInvalidMessageImageObjectPath
		}
		if _, exists := objectPaths[image.ObjectPath]; exists {
			return ErrDuplicateMessageImage
		}
		objectPaths[image.ObjectPath] = struct{}{}

		if image.FileSize <= 0 ||
			image.FileSize > MaxMessageImageSize {
			return ErrInvalidMessageImageFileSize
		}
		if !isAllowedImageMIMEType(image.MIMEType) {
			return ErrInvalidMessageImageMIMEType
		}
	}

	return nil
}

// ============================================================
// Validation helpers
// ============================================================

func IsValidStatus(status Status) bool {
	switch status {
	case StatusActive, StatusClosed:
		return true
	default:
		return false
	}
}

func IsValidSellerType(sellerType SellerType) bool {
	switch sellerType {
	case SellerTypeCompany, SellerTypeAvatar:
		return true
	default:
		return false
	}
}

func IsValidMessageSenderSide(side MessageSenderSide) bool {
	switch side {
	case MessageSenderSideBuyer,
		MessageSenderSideSeller,
		MessageSenderSideSystem:
		return true
	default:
		return false
	}
}

func IsValidMessageSenderType(senderType MessageSenderType) bool {
	switch senderType {
	case MessageSenderTypeAvatar,
		MessageSenderTypeSystem:
		return true
	default:
		return false
	}
}

func isValidReferenceID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > MaxReferenceIDLength {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if strings.Contains(value, "/") || strings.Contains(value, "://") {
		return false
	}

	return true
}

func isAllowedImageMIMEType(value string) bool {
	if value == "" || len([]rune(value)) > MaxImageMIMETypeLength {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/jpeg",
		"image/png",
		"image/webp",
		"image/gif":
		return true
	default:
		return false
	}
}

func cloneMessageImages(images []MessageImage) []MessageImage {
	if len(images) == 0 {
		return []MessageImage{}
	}

	out := make([]MessageImage, len(images))
	copy(out, images)
	return out
}
