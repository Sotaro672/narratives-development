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
	ErrInvalidImageFileURL = errors.New(
		"news: invalid image file url",
	)
	ErrInvalidImageObjectPath = errors.New(
		"news: invalid image object path",
	)
	ErrInvalidImageFileName = errors.New(
		"news: invalid image file name",
	)
	ErrInvalidImageMimeType = errors.New(
		"news: invalid image mime type",
	)
	ErrInvalidImageFileSize = errors.New(
		"news: invalid image file size",
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
	if string(id) == "" {
		return ErrInvalidID
	}
	return nil
}

type NewsReadID string

func (id NewsReadID) Validate() error {
	if string(id) == "" {
		return ErrInvalidReadID
	}
	return nil
}

// ============================================================
// RecipientType
// ============================================================

// RecipientType は News の既読判定主体を表す。
// Console は MEMBER、Mall は AVATAR 単位で既読状態を判定する。
//
// RecipientType 自体を NewsRead の永続化フィールドとして保存する必要はない。
// NewsID + RecipientType + RecipientID から決定論的な NewsReadID を生成するために
// アプリケーション内部でのみ利用する。
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
// NewsImage
// ============================================================

// NewsImage は News に掲載する任意のメイン画像を表す。
//
// 画像本体は Firebase Storage に保存し、News document には表示・管理に
// 必要なメタデータのみ保持する。
//
// 既存 News との後方互換性を維持するため、News.Image は nil を許容する。
type NewsImage struct {
	FileURL    string `json:"fileUrl"`
	ObjectPath string `json:"objectPath"`
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	FileSize   int64  `json:"fileSize"`
	Alt        string `json:"alt,omitempty"`
}

// NewNewsImage creates News image metadata.
func NewNewsImage(
	fileURL string,
	objectPath string,
	fileName string,
	mimeType string,
	fileSize int64,
	alt string,
) (NewsImage, error) {
	image := NewsImage{
		FileURL:    fileURL,
		ObjectPath: objectPath,
		FileName:   fileName,
		MimeType:   strings.ToLower(mimeType),
		FileSize:   fileSize,
		Alt:        alt,
	}

	if err := image.Validate(); err != nil {
		return NewsImage{}, err
	}

	return image, nil
}

// Validate validates NewsImage invariants.
func (i NewsImage) Validate() error {
	if i.FileURL == "" {
		return ErrInvalidImageFileURL
	}

	if i.ObjectPath == "" {
		return ErrInvalidImageObjectPath
	}

	if i.FileName == "" {
		return ErrInvalidImageFileName
	}

	switch strings.ToLower(i.MimeType) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return ErrInvalidImageMimeType
	}

	if i.FileSize <= 0 {
		return ErrInvalidImageFileSize
	}

	return nil
}

// normalizeNewsImage returns a validated copy of image metadata.
func normalizeNewsImage(
	image *NewsImage,
) (*NewsImage, error) {
	if image == nil {
		return nil, nil
	}

	normalized, err := NewNewsImage(
		image.FileURL,
		image.ObjectPath,
		image.FileName,
		image.MimeType,
		image.FileSize,
		image.Alt,
	)
	if err != nil {
		return nil, err
	}

	return &normalized, nil
}

// ============================================================
// News
// ============================================================

// News は AMOL Admin から Console / Mall の全ユーザーへ配信する
// システム共通のお知らせを表す。
//
// News 自体は受信者ごとに複製しない。
// 1件の News document を全ユーザーが共有する。
//
// Image は任意で、画像本体は Firebase Storage、画像メタデータは
// News document に保持する。
//
// 現在の配信方針では Admin から作成された News は即時公開されるため、
// Published フラグは持たず PublishedAt を必須とする。
type News struct {
	ID NewsID `json:"id"`

	Title string `json:"title"`
	Body  string `json:"body"`

	Image *NewsImage `json:"image,omitempty"`

	PublishedAt time.Time `json:"publishedAt"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

// New は公開済みのシステム News を生成する。
func New(
	id NewsID,
	title string,
	body string,
	image *NewsImage,
	publishedAt time.Time,
	createdAt time.Time,
	createdBy string,
) (News, error) {
	normalizedImage, err := normalizeNewsImage(image)
	if err != nil {
		return News{}, err
	}

	n := News{
		ID:          id,
		Title:       title,
		Body:        body,
		Image:       normalizedImage,
		PublishedAt: canonicalTime(publishedAt),
		CreatedAt:   canonicalTime(createdAt),
		CreatedBy:   createdBy,
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

	if n.Title == "" {
		return ErrInvalidTitle
	}

	if n.Body == "" {
		return ErrInvalidBody
	}

	if n.Image != nil {
		if err := n.Image.Validate(); err != nil {
			return err
		}
	}

	if n.CreatedBy == "" {
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

// NewsRead は現在の読者に対する News の既読状態を表す。
//
// News 配信時に全ユーザー分の NewsRead を作成しない。
// 読者が実際に News を既読にした時点で初めて既読 document を作成する。
//
// RecipientType / RecipientID は永続化するための情報ではない。
// NewsID + RecipientType + RecipientID から決定論的な NewsReadID を生成し、
// その document が存在するかどうかによって現在の読者が未読か既読かを判定する。
//
// Firestore に永続化する既読情報は基本的に以下だけでよい。
//
//	ID
//	NewsID
//	ReadAt
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

	// RecipientType / RecipientID は NewsReadID の導出および
	// アプリケーション内部での既読判定にのみ使用する。
	// Firestore document のフィールドとして保存する必要はない。
	RecipientType RecipientType `json:"-"`
	RecipientID   string        `json:"-"`

	ReadAt time.Time `json:"readAt"`
}

// NewRead creates a read-state value for the current reader.
//
// ID は NewsID + RecipientType + RecipientID から決定論的に生成されるため、
// 同一読者・同一 News の既読処理を冪等に扱える。
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
		RecipientID:   recipientID,
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

	if r.RecipientID == "" {
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

// BuildNewsReadID returns a stable document ID for the current reader's read state.
//
// RecipientType / RecipientID 自体を Firestore に保存せず、
// NewsID + RecipientType + RecipientID を hash 化した値だけを
// NewsRead document ID として使用する。
//
// このため、newsReads collection から「誰が読んだか」を直接取得するための
// recipientId / recipientType フィールドは保持しない。
//
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
