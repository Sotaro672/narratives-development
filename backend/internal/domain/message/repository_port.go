package message

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("message: not found")
	ErrAlreadyExists = errors.New("message: already exists")
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 100
)

type ListFilter struct {
	Limit           int
	BeforeCreatedAt *time.Time
}

type Repository interface {
	Create(ctx context.Context, message Message) error
	FindByID(ctx context.Context, id string) (Message, error)
	Update(ctx context.Context, message Message) error
	Delete(ctx context.Context, id string) error

	ListThread(ctx context.Context, avatarID, peerAvatarID string, filter ListFilter) ([]Message, error)
	ListReceived(ctx context.Context, receiverAvatarID string, filter ListFilter) ([]Message, error)
	ListSent(ctx context.Context, senderAvatarID string, filter ListFilter) ([]Message, error)

	MarkAsRead(ctx context.Context, id string, readAt time.Time) error
}
