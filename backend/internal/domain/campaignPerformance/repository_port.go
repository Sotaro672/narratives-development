// backend\internal\domain\campaignPerformance\repository_port.go
package campaignPerformance

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
type CampaignPerformancePatch struct {
	Impressions   *int
	Clicks        *int
	Conversions   *int
	Purchases     *int
	LastUpdatedAt *time.Time
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	// 絞り込み
	CampaignID  *string
	CampaignIDs []string

	// 数値レンジ
	ImpressionsMin *int
	ImpressionsMax *int
	ClicksMin      *int
	ClicksMax      *int
	ConversionsMin *int
	ConversionsMax *int
	PurchasesMin   *int
	PurchasesMax   *int

	// 日付レンジ（lastUpdatedAt）
	LastUpdatedFrom *time.Time
	LastUpdatedTo   *time.Time
}

// 共通型エイリアス（インフラ非依存）
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
	ErrNotFound = errors.New("campaignPerformance: not found")
	ErrConflict = errors.New("campaignPerformance: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[CampaignPerformance], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[CampaignPerformance], error)

	// 取得
	GetByID(ctx context.Context, id string) (CampaignPerformance, error)
	GetByCampaignID(ctx context.Context, campaignID string, sort Sort, page Page) (PageResult[CampaignPerformance], error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, cp CampaignPerformance) (CampaignPerformance, error)
	Update(ctx context.Context, id string, patch CampaignPerformancePatch) (CampaignPerformance, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, cp CampaignPerformance, opts *SaveOptions) (CampaignPerformance, error)
}
