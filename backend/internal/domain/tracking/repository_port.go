// backend/internal/domain/tracking/repository_port.go
package tracking

import (
	"context"
)

// ========================================
// 外部DTO（API/ストレージとの境界）
// ========================================

type TrackingDTO struct {
	ID                  string  `json:"id"`
	OrderID             string  `json:"orderId"`
	Carrier             string  `json:"carrier"`
	TrackingNumber      string  `json:"trackingNumber"`
	SpecialInstructions *string `json:"specialInstructions,omitempty"`
	CreatedAt           string  `json:"createdAt"` // ISO8601 文字列想定
	UpdatedAt           string  `json:"updatedAt"` // ISO8601 文字列想定
}

// ========================================
// 入出力DTO（UseCase/Service -> Repository）
// ========================================

type CreateTrackingInput struct {
	OrderID             string  `json:"orderId"`
	Carrier             string  `json:"carrier"`
	TrackingNumber      string  `json:"trackingNumber"`
	SpecialInstructions *string `json:"specialInstructions,omitempty"`
}

type UpdateTrackingInput struct {
	Carrier             *string `json:"carrier,omitempty"`
	TrackingNumber      *string `json:"trackingNumber,omitempty"`
	SpecialInstructions *string `json:"specialInstructions,omitempty"`
}

// ========================================
// Repository Port
// ========================================

type RepositoryPort interface {
	// 取得系
	GetAllTrackings(ctx context.Context) ([]*Tracking, error)
	GetTrackingByID(ctx context.Context, id string) (*Tracking, error)
	GetTrackingsByOrderID(ctx context.Context, orderID string) ([]*Tracking, error)

	// 変更系
	CreateTracking(ctx context.Context, in CreateTrackingInput) (*Tracking, error)
	UpdateTracking(ctx context.Context, id string, in UpdateTrackingInput) (*Tracking, error)
	DeleteTracking(ctx context.Context, id string) error

	// 管理（開発用）
	ResetTrackings(ctx context.Context) error

	// 任意: トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
