// backend/internal/domain/productBlueprintReview/repository_port.go
package productBlueprintReview

import (
	"context"
	"time"

	domcommon "narratives/internal/domain/common"
)

// ======================================
// Filter / Sort / Paging
// ======================================

// Filter は一覧検索用フィルタ。
// 共通フィルタ（検索文字列・作成/更新期間）に加えて、口コミドメイン固有の条件を拡張。
type Filter struct {
	domcommon.FilterCommon

	// 対象商品（必須にしたい場合はアプリ層で強制してOK）
	ProductBlueprintID *string `json:"productBlueprintId"`

	// 投稿者で絞る（マイレビュー一覧等）
	AvatarID *string `json:"avatarId"`

	// 掲載ステータス
	Status *ReviewStatus `json:"status"`

	// 星評価での絞り込み（単一値）
	Rating *Rating `json:"rating"`

	// 星評価の範囲指定（1..5）
	RatingMin *Rating `json:"ratingMin"`
	RatingMax *Rating `json:"ratingMax"`

	// 投稿日（ReviewedAt）の範囲
	Reviewed domcommon.TimeRange `json:"reviewed"`
}

// Patch は部分更新用。
// 口コミ編集・ステータス変更・モデレーション理由など、更新対象を絞って扱う。
// ※ 投票系は専用メソッド（Increment）に分ける設計を推奨。
type Patch struct {
	Title  *string `json:"title"`
	Body   *string `json:"body"`
	Rating *Rating `json:"rating"`

	// ステータス / モデレーション
	Status           *ReviewStatus `json:"status"`
	ModerationReason *string       `json:"moderationReason"`

	// 更新監査
	UpdatedAt *time.Time `json:"updatedAt"`
	UpdatedBy *string    `json:"updatedBy"`
}

// AllowedSortColumns は repository 実装側で Sort.Column をバリデートするための許可カラム。
// （DB実装でカラム名が変わる場合は adapter 側でマッピングしてもOK）
var AllowedSortColumns = map[string]struct{}{
	"createdAt":    {},
	"updatedAt":    {},
	"reviewedAt":   {},
	"rating":       {},
	"helpfulVotes": {},
	"totalVotes":   {},
}

// ======================================
// Repository Port
// ======================================

// Repository は productBlueprintReview ドメインのリポジトリポート。
// 共通の CRUD + List に加えて、口コミ特有の取得/集計/投票操作を追加で定義。
type Repository interface {
	domcommon.Repository[Review, Filter, Patch]

	// 商品単位での新着レビュー（Amazonの「新しい順」相当を作りやすい）
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		status ReviewStatus,
		page domcommon.Page,
	) (domcommon.PageResult[Review], error)

	// 集計（商品詳細の「星◯◯個、レビュー数」用）
	GetProductSummary(
		ctx context.Context,
		productBlueprintID string,
		status ReviewStatus,
	) (ProductReviewSummary, error)

	// 投票（役に立った / 役に立たなかった）
	IncrementHelpful(ctx context.Context, reviewID string) (Review, error)
	IncrementNotHelpful(ctx context.Context, reviewID string) (Review, error)
}

// ======================================
// Summary DTO
// ======================================

// ProductReviewSummary は商品単位の口コミ集計。
// - 平均評価
// - 件数
// - 星別分布
type ProductReviewSummary struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	Status             string `json:"status"`

	TotalCount    int     `json:"totalCount"`
	AverageRating float64 `json:"averageRating"`

	// 5..1 の件数（表示で使いやすい）
	Rating5Count int `json:"rating5Count"`
	Rating4Count int `json:"rating4Count"`
	Rating3Count int `json:"rating3Count"`
	Rating2Count int `json:"rating2Count"`
	Rating1Count int `json:"rating1Count"`
}
