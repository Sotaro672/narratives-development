// backend/internal/domain/news/repository_port.go
package news

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// ============================================================
// Repository errors
// ============================================================

var (
	ErrNotFound = errors.New(
		"news: not found",
	)
	ErrConflict = errors.New(
		"news: conflict",
	)
	ErrReadNotFound = errors.New(
		"news: read state not found",
	)
)

// ============================================================
// News filter
// ============================================================

// Filter は News 一覧取得用の検索条件。
// SearchQuery は title / body を対象とする想定。
// Created は News.CreatedAt に対する期間条件。
// PublishedAt は News.PublishedAt に対する期間条件。
type Filter struct {
	common.FilterCommon `json:",inline"`

	PublishedAt common.TimeRange `json:"publishedAt"`
}

// ============================================================
// News sort
// ============================================================

// AllowedSortColumns は News 一覧で許可するソート列。
var AllowedSortColumns = map[string]struct{}{
	"createdAt":   {},
	"publishedAt": {},
	"title":       {},
}

// ============================================================
// NewsRepository
// ============================================================

// Repository は AMOL Admin が配信する News 本体を永続化する。
//
// News は Console / Mall の各ユーザーごとには複製しない。
// 1件の News を全受信者で共有し、受信者固有の既読状態は
// ReadRepository で管理する。
type Repository interface {
	// Create creates one globally distributed News.
	//
	// Expected implementation policy:
	// - News.ID must be used as the persistence document ID.
	// - Existing IDs must return ErrConflict.
	// - The entity must not be modified into recipient-specific copies.
	Create(
		ctx context.Context,
		entity News,
	) (News, error)

	// GetByID returns one News.
	//
	// Expected implementation policy:
	// - Empty or invalid newsID must return a domain validation error.
	// - Missing News must return ErrNotFound.
	GetByID(
		ctx context.Context,
		newsID NewsID,
	) (News, error)

	// List returns News ordered and paginated according to sort/page.
	//
	// All News are system-wide messages, so recipient filtering must not be
	// performed here. Recipient-specific read state belongs to ReadRepository.
	List(
		ctx context.Context,
		filter Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[News], error)
}

// ============================================================
// NewsRead filter
// ============================================================

// ReadFilter は NewsRead 一覧取得用の検索条件。
//
// RecipientType / RecipientID の組み合わせで、Console Member または
// Mall Avatar の既読レコードだけを取得できる。
//
// NewsIDs が指定された場合、その News 群に対する既読状態だけを取得する。
// 通知一覧取得時に、取得した News page と既読状態を突き合わせる用途を想定する。
type ReadFilter struct {
	RecipientType *RecipientType `json:"recipientType"`
	RecipientID   string         `json:"recipientId"`

	NewsIDs []NewsID `json:"newsIds,omitempty"`

	ReadAt common.TimeRange `json:"readAt"`
}

// ============================================================
// NewsRead sort
// ============================================================

var AllowedReadSortColumns = map[string]struct{}{
	"readAt": {},
}

// ============================================================
// CreateNewsReadResult
// ============================================================

// CreateNewsReadResult は冪等な既読作成の結果を表す。
//
// Created:
// - true  = 今回初めて既読レコードを作成した。
// - false = 同一 News / recipient の既読レコードが既に存在していた。
//
// NewsRead.ID は NewsID + RecipientType + RecipientID から決定論的に
// 生成されるため、CreateIfAbsent は同一操作の再試行に対して冪等でなければならない。
type CreateNewsReadResult struct {
	Read    NewsRead
	Created bool
}

// ============================================================
// NewsReadRepository
// ============================================================

// ReadRepository は News の受信者ごとの既読状態を永続化する。
//
// News 配信時に全 MEMBER / AVATAR 分の document を生成してはならない。
// NewsRead は受信者が実際に News を既読にした時点で初めて作成する。
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
type ReadRepository interface {
	// CreateIfAbsent creates a NewsRead only when it does not already exist.
	//
	// Expected implementation policy:
	// - NewsRead.ID must be used as the persistence document ID.
	// - If the same ID already exists, the existing record must be returned
	//   with Created=false.
	// - Existing ReadAt must not be overwritten by retries.
	// - This operation must be safe to retry.
	CreateIfAbsent(
		ctx context.Context,
		read NewsRead,
	) (CreateNewsReadResult, error)

	// Get returns the read state for one News and recipient.
	//
	// The repository should derive or validate the deterministic NewsRead ID
	// from newsID / recipientType / recipientID.
	//
	// Missing read state must return ErrReadNotFound.
	Get(
		ctx context.Context,
		newsID NewsID,
		recipientType RecipientType,
		recipientID string,
	) (NewsRead, error)

	// List returns recipient read states matching the filter.
	//
	// Main use cases:
	// - obtain all read states for one recipient;
	// - obtain read states only for News IDs currently displayed;
	// - obtain TotalCount for unread-counter calculation.
	List(
		ctx context.Context,
		filter ReadFilter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[NewsRead], error)
}
