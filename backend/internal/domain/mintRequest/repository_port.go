package mintrequest

import (
	"context"
	"errors"
	"time"
)

// RepositoryPort は MintRequest の永続化契約（ドメイン層）です。
// 具体的なデータストア技術には依存しません。
type RepositoryPort interface {
	// 基本CRUD
	GetByID(ctx context.Context, id string) (*MintRequest, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Create(ctx context.Context, in CreateMintRequest) (*MintRequest, error)
	Update(ctx context.Context, id string, patch UpdateMintRequest) (*MintRequest, error)
	Delete(ctx context.Context, id string) error

	// 便利系
	Count(ctx context.Context, filter Filter) (int, error)

	// テスト/開発向け
	Reset(ctx context.Context) error
}

// Create 用入力（最小限の必須項目 + 任意の計画値）
type CreateMintRequest struct {
	TokenBlueprintID string
	ProductionID     string
	MintQuantity     int
	BurnDate         *time.Time // nil 可（未設定）
	CreatedBy        string     // 監査用
}

// 部分更新（nil は更新しない）
// 注意: ドメイン制約（例: planning 以外での TokenBlueprintID 変更不可など）は
// アプリケーションサービス層で検証される前提です。
type UpdateMintRequest struct {
	Status           *MintRequestStatus
	TokenBlueprintID *string
	MintQuantity     *int
	BurnDate         *time.Time

	// 状態遷移関連（requested/minted の監査用フィールド）
	RequestedBy *string
	RequestedAt  *time.Time
	MintedAt    *time.Time

	// 監査
	UpdatedBy string      // 更新者（必須）
	DeletedAt *time.Time  // 論理削除
	DeletedBy *string     // 論理削除者
}

// 一覧フィルタ
type Filter struct {
	ProductionID     string
	TokenBlueprintID string
	Statuses         []MintRequestStatus

	RequestedBy string

	// 時間範囲
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	RequestFrom  *time.Time
	RequestTo    *time.Time
	MintedFrom    *time.Time
	MintedTo      *time.Time
	BurnFrom      *time.Time
	BurnTo        *time.Time

	// 論理削除
	Deleted *bool // nil: 全件, true: 削除済のみ, false: 未削除のみ
}

// ソート
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt   SortColumn = "createdAt"
	SortByUpdatedAt   SortColumn = "updatedAt"
	SortByBurnDate    SortColumn = "burnDate"
	SortByMintedAt    SortColumn = "mintedAt"
	SortByRequestedAt SortColumn = "requestedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ページング
type Page struct {
	Number  int
	PerPage int
}

// ページング結果
type PageResult struct {
	Items      []MintRequest
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// 代表的なリポジトリエラー
var (
	ErrNotFound = errors.New("mintRequest: not found")
	ErrConflict = errors.New("mintRequest: conflict")
)
