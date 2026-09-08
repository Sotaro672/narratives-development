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
// SearchQuery は title / body を対象とする。
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
// News は Console / Mall のユーザーごとに複製しない。
// 1件の News document を全読者で共有する。
type Repository interface {
	// Create creates one globally distributed News.
	//
	// Expected implementation policy:
	// - News.ID must be used as the persistence document ID.
	// - Existing IDs must return ErrConflict.
	// - The entity must not be modified into reader-specific copies.
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
	// News は system-wide message であるため、読者による絞り込みは行わない。
	// 読者自身の既読状態は ReadRepository から取得する。
	List(
		ctx context.Context,
		filter Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[News], error)
}

// ============================================================
// CreateNewsReadResult
// ============================================================

// CreateNewsReadResult は冪等な既読作成の結果を表す。
//
// Created:
// - true  = 今回初めて既読状態を作成した。
// - false = 同一 News / reader の既読状態が既に存在していた。
//
// NewsRead.ID は NewsID + RecipientType + RecipientID から決定論的に
// 生成されるため、同一読者による同一 News の既読操作は冪等になる。
type CreateNewsReadResult struct {
	Read    NewsRead
	Created bool
}

// ============================================================
// NewsReadRepository
// ============================================================

// ReadRepository は News の既読状態を永続化する。
//
// News 配信時に全 MEMBER / AVATAR 分の document を生成してはならない。
// NewsRead は読者が実際に News を既読にした時点で初めて作成する。
//
// Firestore document には読者を検索可能な形で保存しない。
// RecipientType / RecipientID は NewsRead document ID を導出するためだけに利用する。
//
// document ID:
//
//	NewsID + RecipientType + RecipientID
//	↓
//	SHA-256
//	↓
//	NewsReadID
//
// 永続化する既読情報は基本的に以下だけとする。
//
//	ID
//	NewsID
//	ReadAt
//
// このため「誰が読んだか」を newsReads collection から一覧取得する用途は持たない。
// あくまで現在の読者について対象 News が未読か既読かを判定するためのRepositoryとする。
type ReadRepository interface {
	// CreateIfAbsent creates a NewsRead only when it does not already exist.
	//
	// Expected implementation policy:
	// - NewsRead.ID must be used as the persistence document ID.
	// - RecipientType / RecipientID must not be persisted as searchable fields.
	// - If the same ID already exists, the existing read state must be returned
	//   with Created=false.
	// - Existing ReadAt must not be overwritten by retries.
	// - This operation must be safe to retry.
	CreateIfAbsent(
		ctx context.Context,
		read NewsRead,
	) (CreateNewsReadResult, error)

	// Get returns the current reader's read state for one News.
	//
	// The implementation derives the deterministic NewsReadID from:
	//
	//	newsID + recipientType + recipientID
	//
	// and directly reads that document.
	//
	// RecipientType / RecipientID は検索条件としてFirestoreへ送らず、
	// document ID の生成にだけ使用する。
	//
	// Missing read state must return ErrReadNotFound.
	Get(
		ctx context.Context,
		newsID NewsID,
		recipientType RecipientType,
		recipientID string,
	) (NewsRead, error)

	// GetMany returns the current reader's read states for the specified News.
	//
	// 各 NewsID と recipientType / recipientID から決定論的な NewsReadID を
	// 生成し、その document を直接参照する。
	//
	// RecipientType / RecipientID による collection query や
	// newsReads collection 全件走査を行ってはならない。
	//
	// 未読の News に対応する document は存在しないため結果には含めない。
	// 全件が未読の場合は空 slice を返す。
	//
	// NewsIDs の順序と返却結果の順序が一致することは要求しない。
	GetMany(
		ctx context.Context,
		newsIDs []NewsID,
		recipientType RecipientType,
		recipientID string,
	) ([]NewsRead, error)
}
