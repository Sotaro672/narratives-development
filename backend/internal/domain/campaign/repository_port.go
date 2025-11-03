package campaign

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignActive    CampaignStatus = "active"
	CampaignPaused    CampaignStatus = "paused"
	CampaignScheduled CampaignStatus = "scheduled"
	CampaignCompleted CampaignStatus = "completed"
)

// CreateCampaignInput - 作成入力
type CreateCampaignInput struct {
	BrandID       string         `json:"brandId"`
	Name          string         `json:"name,omitempty"`
	Description   *string        `json:"description,omitempty"`
	Status        CampaignStatus `json:"status"`
	AssigneeID    *string        `json:"assigneeId,omitempty"`
	ImageID       *string        `json:"imageId,omitempty"`
	PerformanceID *string        `json:"performanceId,omitempty"`
	StartDate     time.Time      `json:"startDate"`
	EndDate       time.Time      `json:"endDate"`
}

// UpdateCampaignInput - 更新入力（部分更新）
type UpdateCampaignInput struct {
	BrandID       *string         `json:"brandId,omitempty"`
	Name          *string         `json:"name,omitempty"`
	Description   *string         `json:"description,omitempty"`
	Status        *CampaignStatus `json:"status,omitempty"`
	AssigneeID    *string         `json:"assigneeId,omitempty"`
	ImageID       *string         `json:"imageId,omitempty"`
	PerformanceID *string         `json:"performanceId,omitempty"`
	StartDate     *time.Time      `json:"startDate,omitempty"`
	EndDate       *time.Time      `json:"endDate,omitempty"`
}

// ========================================
// Patch（部分更新）: nil のフィールドは更新しない
// ========================================
type CampaignPatch struct {
	Name           *string
	BrandID        *string
	AssigneeID     *string
	ListID         *string
	Status         *CampaignStatus
	Budget         *float64
	Spent          *float64
	StartDate      *time.Time
	EndDate        *time.Time
	TargetAudience *string
	AdType         *AdType
	Headline       *string
	Description    *string
	PerformanceID  *string
	ImageID        *string
	UpdatedBy      *string
	UpdatedAt      *time.Time
	DeletedAt      *time.Time
	DeletedBy      *string
}

// ========================================
// フィルタ/検索条件（実装側で適宜解釈）
// ========================================
type Filter struct {
	// フリーテキスト検索（name, description, targetAudience, headline など）
	SearchQuery string

	// 絞り込み
	BrandID     *string
	BrandIDs    []string
	AssigneeID  *string
	AssigneeIDs []string
	ListID      *string
	ListIDs     []string

	Statuses []CampaignStatus
	AdTypes  []AdType

	// 数値レンジ
	BudgetMin *float64
	BudgetMax *float64
	SpentMin  *float64
	SpentMax  *float64

	// 日付レンジ
	StartFrom   *time.Time
	StartTo     *time.Time
	EndFrom     *time.Time
	EndTo       *time.Time
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	// 関連IDの有無
	HasPerformanceID *bool
	HasImageID       *bool

	// 作成者フィルタ
	CreatedBy *string

	// 論理削除の tri-state（nil: 全件 / true: 削除済のみ / false: 未削除のみ）
	Deleted *bool
}

// ========================================
// 共通型エイリアス（インフラ非依存）
// ========================================
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("campaign: not found")
	ErrConflict = errors.New("campaign: conflict")
)

// ========================================
// Repository ポート（契約）
// ========================================
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Campaign], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Campaign], error)

	// 取得
	GetByID(ctx context.Context, id string) (Campaign, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, c Campaign) (Campaign, error)
	Update(ctx context.Context, id string, patch CampaignPatch) (Campaign, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, c Campaign, opts *SaveOptions) (Campaign, error)
}
