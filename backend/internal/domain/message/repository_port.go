package message

import (
	"context"
	"errors"
	"time"
)

// Repository は Message ドメインの永続化ポート（契約）です。
// 実装はデータストア技術に依存して構いませんが、ドメイン層からは本インターフェースのみを参照します。
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Message], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Message], error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 取得
	GetByID(ctx context.Context, id string) (Message, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 作成・更新（ドラフト作成と部分更新）
	CreateDraft(ctx context.Context, m Message) (Message, error)
	Patch(ctx context.Context, id string, patch MessagePatch) (Message, error)

	// 状態遷移
	Send(ctx context.Context, id string, at time.Time) (Message, error)        // draft -> sent
	Cancel(ctx context.Context, id string, at time.Time) (Message, error)      // sent -> canceled
	MarkDelivered(ctx context.Context, id string, at time.Time) (Message, error)
	MarkRead(ctx context.Context, id string, at time.Time) (Message, error)

	// 削除
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, m Message) (Message, error)
}

// Message 用の Patch（nil は未更新）
type MessagePatch struct {
	Content   *string
	Images    *[]ImageRef // Firestore 側には参照（ImageRef）を保持（実体は GCS）
	UpdatedAt *time.Time
}

// メッセージのフィルタ/検索条件
type Filter struct {
	SearchQuery string

	SenderID   *string
	ReceiverID *string
	Statuses   []MessageStatus
	UnreadOnly bool

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
}

// 並び替え
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByUpdatedAt SortColumn = "updatedAt"
	SortByStatus    SortColumn = "status"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// =======================
// Threads（会話ビュー）用の契約
// =======================

// ThreadRepository は会話スレッド（MessageThread）向けの永続化ポートです。
type ThreadRepository interface {
	// 一覧取得
	ListThreads(ctx context.Context, filter ThreadFilter, sort ThreadSort, page Page) (PageResult[MessageThread], error)
	ListThreadsByCursor(ctx context.Context, filter ThreadFilter, sort ThreadSort, cpage CursorPage) (CursorPageResult[MessageThread], error)
	CountThreads(ctx context.Context, filter ThreadFilter) (int, error)

	// 取得/保存/削除
	GetThreadByID(ctx context.Context, id string) (MessageThread, error)
	SaveThread(ctx context.Context, t MessageThread) (MessageThread, error)
	DeleteThread(ctx context.Context, id string) error
}

// スレッドのフィルタ
type ThreadFilter struct {
	SearchQuery string

	ParticipantID  *string   // 単一参加者を含むスレッド
	ParticipantIDs []string  // いずれか/全てを含むかの扱いは実装側で定義

	CreatedFrom     *time.Time
	CreatedTo       *time.Time
	UpdatedFrom     *time.Time
	UpdatedTo       *time.Time
	LastMessageFrom *time.Time
	LastMessageTo   *time.Time
}

// スレッドの並び替え
type ThreadSort struct {
	Column ThreadSortColumn
	Order  SortOrder
}

type ThreadSortColumn string

const (
	ThreadSortByLastMessageAt ThreadSortColumn = "lastMessageAt"
	ThreadSortByCreatedAt     ThreadSortColumn = "createdAt"
	ThreadSortByUpdatedAt     ThreadSortColumn = "updatedAt"
)

// ページング
type Page struct {
	Number  int
	PerPage int
}

type PageResult[T any] struct {
	Items      []T
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// カーソルページング
type CursorPage struct {
	After string
	Limit int
}

type CursorPageResult[T any] struct {
	Items      []T
	NextCursor *string
	Limit      int
}

// 契約上の代表的エラー
var (
	ErrNotFound = errors.New("message: not found")
	ErrConflict = errors.New("message: conflict")
	// 汎用の不正値エラー（アダプタ側で利用する場合があるため用意）
	ErrInvalid = errors.New("message: invalid")
)
