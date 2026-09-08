// backend/internal/domain/news/entity.go
package news

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidID = errors.New(
		"news: invalid id",
	)
	ErrInvalidTitle = errors.New(
		"news: invalid title",
	)
	ErrInvalidBody = errors.New(
		"news: invalid body",
	)
	ErrInvalidCreatedBy = errors.New(
		"news: invalid created by",
	)
	ErrInvalidCreatedAt = errors.New(
		"news: invalid created at",
	)
	ErrInvalidPublishedAt = errors.New(
		"news: invalid published at",
	)
	ErrPublishedBeforeCreated = errors.New(
		"news: published at is before created at",
	)

	ErrInvalidReadID = errors.New(
		"news: invalid read id",
	)
	ErrInvalidNewsID = errors.New(
		"news: invalid news id",
	)
	ErrInvalidRecipientType = errors.New(
		"news: invalid recipient type",
	)
	ErrInvalidRecipientID = errors.New(
		"news: invalid recipient id",
	)
	ErrInvalidReadAt = errors.New(
		"news: invalid read at",
	)
)

// ============================================================
// ID
// ============================================================

type NewsID string

func (id NewsID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidID
	}
	return nil
}

type NewsReadID string

func (id NewsReadID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidReadID
	}
	return nil
}

// ============================================================
// RecipientType
// ============================================================

// RecipientType は News の既読主体を表す。
// Console は MEMBER、Mall は AVATAR 単位で既読状態を管理する。
type RecipientType string

const (
	RecipientTypeMember RecipientType = "MEMBER"
	RecipientTypeAvatar RecipientType = "AVATAR"
)

func (t RecipientType) Validate() error {
	switch t {
	case RecipientTypeMember, RecipientTypeAvatar:
		return nil
	default:
		return ErrInvalidRecipientType
	}
}

// ============================================================
// News
// ============================================================

// News は AMOL Admin から Console / Mall の全ユーザーへ配信する
// システム共通のお知らせを表す。
//
// News 自体は受信者ごとに複製しない。
// 1件の News document を全ユーザーが共有し、既読状態のみ NewsRead として
// recipient ごとに保持する。
//
// 現在の配信方針では Admin から作成された News は即時公開されるため、
// Published フラグは持たず PublishedAt を必須とする。
type News struct {
	ID NewsID `json:"id"`

	Title string `json:"title"`
	Body  string `json:"body"`

	PublishedAt time.Time `json:"publishedAt"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

// New は公開済みのシステム News を生成する。
func New(
	id NewsID,
	title string,
	body string,
	publishedAt time.Time,
	createdAt time.Time,
	createdBy string,
) (News, error) {
	n := News{
		ID:          id,
		Title:       strings.TrimSpace(title),
		Body:        strings.TrimSpace(body),
		PublishedAt: canonicalTime(publishedAt),
		CreatedAt:   canonicalTime(createdAt),
		CreatedBy:   strings.TrimSpace(createdBy),
	}

	if err := n.Validate(); err != nil {
		return News{}, err
	}

	return n, nil
}

// Validate validates News invariants.
func (n News) Validate() error {
	if err := n.ID.Validate(); err != nil {
		return err
	}

	if strings.TrimSpace(n.Title) == "" {
		return ErrInvalidTitle
	}

	if strings.TrimSpace(n.Body) == "" {
		return ErrInvalidBody
	}

	if strings.TrimSpace(n.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	if n.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if n.PublishedAt.IsZero() {
		return ErrInvalidPublishedAt
	}

	if n.PublishedAt.Before(n.CreatedAt) {
		return ErrPublishedBeforeCreated
	}

	return nil
}

// ============================================================
// NewsRead
// ============================================================

// NewsRead は News に対する受信者ごとの既読状態を表す。
//
// News 作成時には NewsRead を全ユーザー分 fan-out しない。
// ユーザーが実際に News を既読にした時点で初めて作成する。
//
// Console:
//
//	RecipientType = MEMBER
//	RecipientID   = memberId
//
// Mall:
//
//	RecipientType = AVATAR
//	RecipientID   = avatarId
type NewsRead struct {
	ID NewsReadID `json:"id"`

	NewsID NewsID `json:"newsId"`

	RecipientType RecipientType `json:"recipientType"`
	RecipientID   string        `json:"recipientId"`

	ReadAt time.Time `json:"readAt"`
}

// NewRead creates a read-state record for one recipient.
//
// ID は NewsID + RecipientType + RecipientID から決定論的に生成されるため、
// 同一 recipient に対する既読処理を冪等に扱える。
func NewRead(
	newsID NewsID,
	recipientType RecipientType,
	recipientID string,
	readAt time.Time,
) (NewsRead, error) {
	id, err := BuildNewsReadID(
		newsID,
		recipientType,
		recipientID,
	)
	if err != nil {
		return NewsRead{}, err
	}

	r := NewsRead{
		ID:            id,
		NewsID:        newsID,
		RecipientType: recipientType,
		RecipientID:   strings.TrimSpace(recipientID),
		ReadAt:        canonicalTime(readAt),
	}

	if err := r.Validate(); err != nil {
		return NewsRead{}, err
	}

	return r, nil
}

// Validate validates NewsRead invariants.
func (r NewsRead) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}

	if err := r.NewsID.Validate(); err != nil {
		return ErrInvalidNewsID
	}

	if err := r.RecipientType.Validate(); err != nil {
		return err
	}

	if strings.TrimSpace(r.RecipientID) == "" {
		return ErrInvalidRecipientID
	}

	if r.ReadAt.IsZero() {
		return ErrInvalidReadAt
	}

	expectedID, err := BuildNewsReadID(
		r.NewsID,
		r.RecipientType,
		r.RecipientID,
	)
	if err != nil {
		return err
	}

	if r.ID != expectedID {
		return ErrInvalidReadID
	}

	return nil
}

// ============================================================
// NewsRead ID
// ============================================================

// BuildNewsReadID returns a stable document ID for a recipient's read state.
//
// recipientID をそのまま Firestore document ID に使用せず hash 化することで、
// ID に "/" 等が含まれる可能性を排除する。
// 同一 News / RecipientType / RecipientID からは常に同じ ID が生成される。
func BuildNewsReadID(
	newsID NewsID,
	recipientType RecipientType,
	recipientID string,
) (NewsReadID, error) {
	if err := newsID.Validate(); err != nil {
		return "", ErrInvalidNewsID
	}

	if err := recipientType.Validate(); err != nil {
		return "", err
	}

	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return "", ErrInvalidRecipientID
	}

	source := string(newsID) +
		"|" +
		string(recipientType) +
		"|" +
		recipientID

	sum := sha256.Sum256([]byte(source))

	return NewsReadID(
		"news_read_" + hex.EncodeToString(sum[:]),
	), nil
}

// ============================================================
// Time
// ============================================================

// canonicalTime normalizes timestamps to the precision persisted by Firestore.
// ドメイン上で生成した値と Firestore 復元後の値を比較しても揺れないようにする。
func canonicalTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC().Truncate(time.Microsecond)
}
