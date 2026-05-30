// backend/internal/domain/product/repository_port.go
package product

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ===============================
// Create/Update inputs (contracts)
// ===============================

// CreateProductInput - 作成入力（id/updatedAtはリポジトリ側で採番/付与してよい）
type CreateProductInput struct {
	ModelID          string            `json:"modelId"`
	ProductionID     string            `json:"productionId"`
	InspectionResult *InspectionResult `json:"inspectionResult,omitempty"` // nilなら notYet
	ConnectedToken   *string           `json:"connectedToken,omitempty"`

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	PrintedBy   *string    `json:"printedBy,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`
	InspectedBy *string    `json:"inspectedBy,omitempty"`

	UpdatedBy string `json:"updatedBy"`
}

// UpdateProductInput - 部分更新（nilは未更新）
type UpdateProductInput struct {
	ModelID          *string           `json:"modelId,omitempty"`
	ProductionID     *string           `json:"productionId,omitempty"`
	InspectionResult *InspectionResult `json:"inspectionResult,omitempty"`
	ConnectedToken   *string           `json:"connectedToken,omitempty"` // 空文字→nilにする等の扱いは実装側判断

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	PrintedBy   *string    `json:"printedBy,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`
	InspectedBy *string    `json:"inspectedBy,omitempty"`

	UpdatedBy *string `json:"updatedBy,omitempty"`
}

// UpdateInspectionInput - 検査更新（専用オペレーションが必要な場合）
type UpdateInspectionInput struct {
	InspectionResult InspectionResult `json:"inspectionResult"`
	InspectedBy      string           `json:"inspectedBy"`
	InspectedAt      *time.Time       `json:"inspectedAt,omitempty"`
}

// ConnectTokenInput - トークン接続/切断 (TokenID=nil で切断)
type ConnectTokenInput struct {
	TokenID *string `json:"tokenId"`
}

// ===============================
// Query contracts
// ===============================

// Filter - 検索条件
type Filter struct {
	common.FilterCommon

	ID           string `json:"id,omitempty"`
	ModelID      string `json:"modelId,omitempty"`
	ProductionID string `json:"productionId,omitempty"`

	InspectionResults []InspectionResult `json:"inspectionResults,omitempty"`
	HasToken          *bool              `json:"hasToken,omitempty"` // nil=全件, true=トークンあり, false=なし
	TokenID           string             `json:"tokenId,omitempty"`

	Printed   TimeRange `json:"printed"`
	Inspected TimeRange `json:"inspected"`
}

// TimeRange は product ドメイン固有の日時条件
type TimeRange struct {
	From *time.Time `json:"from"`
	To   *time.Time `json:"to"`
}

// ===============================
// Repository Port
// ===============================

type Repository interface {
	GetByID(ctx context.Context, id string) (Product, error)
	Create(ctx context.Context, entity Product) (Product, error)
	ListByProductionID(ctx context.Context, productionID string) ([]Product, error)
}

// 共通エラー
var (
	ErrNotFound = errors.New("product: not found")
	ErrConflict = errors.New("product: conflict")
)
