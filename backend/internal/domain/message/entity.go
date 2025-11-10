// backend/internal/domain/message/entity.go
package message

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ========================================
// Status
// ========================================

type MessageStatus string

const (
	StatusDraft     MessageStatus = "draft"
	StatusSent      MessageStatus = "sent"
	StatusCanceled  MessageStatus = "canceled"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
)

func IsValidStatus(s MessageStatus) bool {
	switch s {
	case StatusDraft, StatusSent, StatusCanceled, StatusDelivered, StatusRead:
		return true
	default:
		return false
	}
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID           = errors.New("message: invalid id")
	ErrInvalidUserID       = errors.New("message: invalid user id")
	ErrInvalidContent      = errors.New("message: invalid content")
	ErrInvalidStatus       = errors.New("message: invalid status")
	ErrInvalidTime         = errors.New("message: invalid time")
	ErrInvalidImage        = errors.New("message: invalid image")
	ErrInvalidMime         = errors.New("message: invalid mimeType")
	ErrInvalidURL          = errors.New("message: invalid url")
	ErrInvalidParticipants = errors.New("message: invalid participants")
	ErrInvalidSubject      = errors.New("message: invalid subject")
	ErrInvalidTransition   = errors.New("message: invalid status transition")
)

var mimeRe = regexp.MustCompile(`^[a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+$`)

// ========================================
// Value Objects (GCS image reference)
// ========================================

type ImageRef struct {
	// ObjectPath: e.g. "messages/{messageId}/{fileName}" or "gs://bucket/..."
	ObjectPath string
	// Optional: public/signed URL if issued by upper layer
	URL        string
	FileName   string
	FileSize   int64
	MimeType   string
	Width      *int
	Height     *int
	UploadedAt time.Time
}

func (r ImageRef) validate() error {
	if strings.TrimSpace(r.FileName) == "" {
		return fmt.Errorf("%w: empty fileName", ErrInvalidImage)
	}
	if strings.TrimSpace(r.ObjectPath) == "" {
		return fmt.Errorf("%w: empty objectPath", ErrInvalidImage)
	}
	// URL is optional; validate only when provided
	if strings.TrimSpace(r.URL) != "" {
		if _, err := url.ParseRequestURI(r.URL); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidURL, err)
		}
	}
	if r.FileSize < 0 {
		return fmt.Errorf("%w: negative fileSize", ErrInvalidImage)
	}
	if !mimeRe.MatchString(r.MimeType) {
		return ErrInvalidMime
	}
	if r.UploadedAt.IsZero() {
		return fmt.Errorf("%w: uploadedAt is zero", ErrInvalidTime)
	}
	return nil
}

// ========================================
// Entity (Firestore document)
// ========================================

type Message struct {
	ID         string
	SenderID   string
	ReceiverID string
	Content    string
	Status     MessageStatus
	// messageImage は従属エンティティ（GCS保管）なので、Firestore 側には参照情報のみ保持
	Images []ImageRef

	CreatedAt  time.Time
	UpdatedAt  *time.Time
	DeletedAt  *time.Time
	ReadAt     *time.Time
	CanceledAt *time.Time
}

// ========================================
// Factories (Firestore operations aligned)
// ========================================

// CreateDraftMessage: 下書き作成（Firestore: messages に draft で保存）
func CreateDraftMessage(
	id, senderID, receiverID, content string,
	images []ImageRef,
	now time.Time,
) (Message, error) {
	m := Message{
		ID:         strings.TrimSpace(id),
		SenderID:   strings.TrimSpace(senderID),
		ReceiverID: strings.TrimSpace(receiverID),
		Content:    strings.TrimSpace(content),
		Status:     StatusDraft,
		Images:     append([]ImageRef(nil), images...),
		CreatedAt:  now.UTC(),
		UpdatedAt:  nil,
		DeletedAt:  nil,
		ReadAt:     nil,
		CanceledAt: nil,
	}
	if err := m.validate(); err != nil {
		return Message{}, err
	}
	return m, nil
}

// ========================================
// Behavior (state transitions saved via Firestore updates)
// ========================================

// SendMessage: draft -> sent（Firestore の status を更新）
func (m *Message) SendMessage(now time.Time) error {
	if m.Status != StatusDraft {
		return ErrInvalidTransition
	}
	if now.IsZero() {
		return ErrInvalidTime
	}
	m.Status = StatusSent
	t := now.UTC()
	m.UpdatedAt = &t
	return nil
}

// CancelMessage: sent -> canceled（Firestore の status を更新しタイムスタンプ付与）
func (m *Message) CancelMessage(now time.Time) error {
	if m.Status != StatusSent {
		return ErrInvalidTransition
	}
	if now.IsZero() {
		return ErrInvalidTime
	}
	m.Status = StatusCanceled
	t := now.UTC()
	m.CanceledAt = &t
	m.UpdatedAt = &t
	return nil
}

// MarkDelivered: sent -> delivered（任意）
func (m *Message) MarkDelivered(now time.Time) error {
	if m.Status != StatusSent {
		return ErrInvalidTransition
	}
	if now.IsZero() {
		return ErrInvalidTime
	}
	m.Status = StatusDelivered
	t := now.UTC()
	m.UpdatedAt = &t
	return nil
}

// MarkRead: delivered -> read（任意）
func (m *Message) MarkRead(at time.Time) error {
	if m.Status != StatusDelivered {
		return ErrInvalidTransition
	}
	if at.IsZero() {
		return ErrInvalidTime
	}
	m.Status = StatusRead
	t := at.UTC()
	m.ReadAt = &t
	m.UpdatedAt = &t
	return nil
}

// TouchUpdated sets/refreshes UpdatedAt.
func (m *Message) TouchUpdated(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidTime
	}
	t := now.UTC()
	m.UpdatedAt = &t
	return nil
}

// MarkDeleted sets DeletedAt.
func (m *Message) MarkDeleted(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidTime
	}
	t := now.UTC()
	m.DeletedAt = &t
	return nil
}

func (m *Message) ClearDeleted() {
	m.DeletedAt = nil
}

// ========================================
func (m Message) validate() error {
	if m.ID == "" {
		return ErrInvalidID
	}
	if m.SenderID == "" || m.ReceiverID == "" {
		return ErrInvalidUserID
	}
	if strings.TrimSpace(m.Content) == "" {
		return ErrInvalidContent
	}
	if !IsValidStatus(m.Status) {
		return ErrInvalidStatus
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidTime
	}
	for _, img := range m.Images {
		if err := img.validate(); err != nil {
			return err
		}
	}
	// Time order
	if m.UpdatedAt != nil && m.UpdatedAt.Before(m.CreatedAt) {
		return fmt.Errorf("%w: updatedAt < createdAt", ErrInvalidTime)
	}
	if m.DeletedAt != nil && m.DeletedAt.Before(m.CreatedAt) {
		return fmt.Errorf("%w: deletedAt < createdAt", ErrInvalidTime)
	}
	if m.ReadAt != nil && m.ReadAt.Before(m.CreatedAt) {
		return fmt.Errorf("%w: readAt < createdAt", ErrInvalidTime)
	}
	if m.CanceledAt != nil && m.CanceledAt.Before(m.CreatedAt) {
		return fmt.Errorf("%w: canceledAt < createdAt", ErrInvalidTime)
	}
	// Status-specific requirements
	if m.Status == StatusRead && m.ReadAt == nil {
		return fmt.Errorf("%w: readAt required when status=read", ErrInvalidTime)
	}
	if m.Status == StatusCanceled && m.CanceledAt == nil {
		return fmt.Errorf("%w: canceledAt required when status=canceled", ErrInvalidTime)
	}
	return nil
}

// ========================================
// DTO helpers (ISO8601 strings)
// ========================================

type MessageDTO struct {
	ID         string        `json:"id"`
	SenderID   string        `json:"senderId"`
	ReceiverID string        `json:"receiverId"`
	Content    string        `json:"content"`
	Status     MessageStatus `json:"status"`
	Images     []ImageRef    `json:"images,omitempty"`
	CreatedAt  string        `json:"createdAt"`
	UpdatedAt  *string       `json:"updatedAt,omitempty"`
	DeletedAt  *string       `json:"deletedAt,omitempty"`
	ReadAt     *string       `json:"readAt,omitempty"`
	CanceledAt *string       `json:"canceledAt,omitempty"`
}

func (m Message) ToDTO() MessageDTO {
	toPtr := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.UTC().Format(time.RFC3339)
		return &s
	}
	return MessageDTO{
		ID:         m.ID,
		SenderID:   m.SenderID,
		ReceiverID: m.ReceiverID,
		Content:    m.Content,
		Status:     m.Status,
		Images:     append([]ImageRef(nil), m.Images...),
		CreatedAt:  m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  toPtr(m.UpdatedAt),
		DeletedAt:  toPtr(m.DeletedAt),
		ReadAt:     toPtr(m.ReadAt),
		CanceledAt: toPtr(m.CanceledAt),
	}
}

// ========================================
// Threads (conversation view)
// ========================================

// MessageThread は会話スレッド（参加者間の最新メッセージ要約等）を表します。
// Firestore 側では messages とは別コレクション/ビューで保持されることがあります。
type MessageThread struct {
	ID              string         `json:"id"`
	ParticipantIDs  []string       `json:"participantIds"`         // 参加者（通常は2名）
	LastMessageID   string         `json:"lastMessageId"`          // 最新メッセージID
	LastMessageAt   time.Time      `json:"lastMessageAt"`          // 最新メッセージ時刻
	LastMessageText string         `json:"lastMessageText"`        // 一部抜粋（サマリ）
	UnreadCounts    map[string]int `json:"unreadCounts,omitempty"` // 参加者ごとの未読数
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       *time.Time     `json:"updatedAt,omitempty"`
}

// NewMessageThread は最小限の情報でスレッドを組み立てます。
func NewMessageThread(id string, participants []string, lastID string, lastAt time.Time, summary string, now time.Time) (MessageThread, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MessageThread{}, ErrInvalidID
	}
	ps := dedupTrim(participants)
	if len(ps) == 0 {
		return MessageThread{}, ErrInvalidParticipants
	}
	if strings.TrimSpace(lastID) == "" || lastAt.IsZero() {
		return MessageThread{}, ErrInvalidTime
	}
	return MessageThread{
		ID:              id,
		ParticipantIDs:  ps,
		LastMessageID:   strings.TrimSpace(lastID),
		LastMessageAt:   lastAt.UTC(),
		LastMessageText: strings.TrimSpace(summary),
		UnreadCounts:    map[string]int{},
		CreatedAt:       now.UTC(),
		UpdatedAt:       nil,
	}, nil
}

// ========================================
// Helpers
// ========================================

// 未使用ヘルパーを削除（parseTime, normalizeTimePtr, contains）して lint を解消

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// ========================================
// SQL DDL
// ========================================
const MessagesTableDDL = `
-- Migration: Initialize messages table

BEGIN;

CREATE TABLE IF NOT EXISTS messages (
  id           UUID        PRIMARY KEY,
  sender_id    UUID        NOT NULL,
  receiver_id  UUID        NOT NULL,
  content      TEXT        NOT NULL,
  status       TEXT        NOT NULL CHECK (status IN ('draft','sent','canceled','delivered','read')),
  created_at   TIMESTAMPTZ NOT NULL,
  updated_at   TIMESTAMPTZ NULL,
  deleted_at   TIMESTAMPTZ NULL,
  read_at      TIMESTAMPTZ NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_messages_non_empty CHECK (char_length(trim(content)) > 0),

  -- Time order
  CONSTRAINT chk_messages_time_order CHECK (
    (updated_at IS NULL OR updated_at >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
    AND (read_at IS NULL OR read_at >= created_at)
  ),

  -- Foreign keys to members
  CONSTRAINT fk_messages_sender   FOREIGN KEY (sender_id)   REFERENCES members (id) ON DELETE RESTRICT,
  CONSTRAINT fk_messages_receiver FOREIGN KEY (receiver_id) REFERENCES members (id) ON DELETE RESTRICT
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_messages_sender_id    ON messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id  ON messages (receiver_id);
CREATE INDEX IF NOT EXISTS idx_messages_status       ON messages (status);
CREATE INDEX IF NOT EXISTS idx_messages_created_at   ON messages (created_at);
CREATE INDEX IF NOT EXISTS idx_messages_updated_at   ON messages (updated_at);
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at   ON messages (deleted_at);
CREATE INDEX IF NOT EXISTS idx_messages_read_at      ON messages (read_at);

COMMIT;
`
