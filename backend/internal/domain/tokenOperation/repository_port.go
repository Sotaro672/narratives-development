// backend/internal/domain/tokenOperation/repository_port.go
package tokenOperation

import (
	"context"
	"errors"
	"time"
)

// ========================================
// ドメインエンティティ
// ========================================

type OperationalToken struct {
	ID               string
	TokenBlueprintID string
	AssigneeID       string

	// 拡張/表示用プロパティ（サーバー応答に含まれる想定）
	TokenName    string
	Symbol       string
	BrandID      string
	AssigneeName string
	BrandName    string

	// クライアント更新で付与される想定のプロパティ
	Name      string
	Status    string // e.g. "operational"
	UpdatedAt time.Time
	UpdatedBy string
}

type TokenContent struct {
	ID          string
	TokenID     string
	Type        string // "image" | "video"
	URL         string
	Description string
	PublishedBy string
	CreatedAt   time.Time
}

// 任意: 商品詳細（必要最小限）
type ProductDetail struct {
	ID          string
	Name        string
	Description string
	// 必要に応じてフィールドを追加
}

// ========================================
// 入出力DTO
// ========================================

type CreateOperationalTokenData struct {
	TokenBlueprintID string
	AssigneeID       string
}

type UpdateOperationalTokenData struct {
	AssigneeID *string
	Name       *string
	Symbol     *string
	Status     *string
	UpdatedBy  string
}

// ホルダー
type Holder struct {
	ID            string
	TokenID       string
	WalletAddress string
	Balance       string
	UpdatedAt     time.Time
}

type HolderSearchParams struct {
	TokenID string
	Query   string
	Limit   int
	Offset  int
}

// 更新履歴
type TokenUpdateHistory struct {
	ID         string
	TokenID    string
	Event      string
	AssigneeID string
	Note       string
	CreatedAt  time.Time
}

type TokenUpdateHistorySearchParams struct {
	TokenID string
	Limit   int
	Offset  int
}

// ========================================
// サーバーAPI通信用のDTO（必要に応じて使用）
// ========================================

type ServerTokenOperationResponse struct {
	ID               string `json:"id"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
	AssigneeID       string `json:"assigneeId"`
	TokenName        string `json:"tokenName"`
	Symbol           string `json:"symbol"`
	BrandID          string `json:"brandId"`
	AssigneeName     string `json:"assigneeName"`
	BrandName        string `json:"brandName"`
}

type ServerTokenOperationCreateRequest struct {
	TokenBlueprintID string `json:"tokenBlueprintId"`
	AssigneeID       string `json:"assigneeId"`
}

type ServerTokenOperationUpdateRequest struct {
	AssigneeID string `json:"assigneeId"`
}

// 共通エラー（ドメイン）
var (
	ErrNotFound = errors.New("tokenOperation: not found")
	ErrConflict = errors.New("tokenOperation: conflict")
)

// ========================================
// Repository Port
// ========================================

type RepositoryPort interface {
	// 取得系
	GetOperationalTokens(ctx context.Context) ([]*OperationalToken, error)
	GetOperationalTokenByID(ctx context.Context, id string) (*OperationalToken, error)

	// 変更系
	CreateOperationalToken(ctx context.Context, in CreateOperationalTokenData) (*OperationalToken, error)
	UpdateOperationalToken(ctx context.Context, id string, in UpdateOperationalTokenData) (*OperationalToken, error)

	// ホルダー
	GetHoldersByTokenID(ctx context.Context, params HolderSearchParams) (holders []*Holder, total int, err error)

	// 更新履歴
	GetTokenUpdateHistory(ctx context.Context, params TokenUpdateHistorySearchParams) ([]*TokenUpdateHistory, error)

	// コンテンツ
	GetTokenContents(ctx context.Context, tokenID string) ([]*TokenContent, error)
	AddTokenContent(ctx context.Context, tokenID string, typ, url, description, publishedBy string) (*TokenContent, error)
	DeleteTokenContent(ctx context.Context, contentID string) error

	// 商品詳細
	GetProductDetailByID(ctx context.Context, productID string) (*ProductDetail, error)

	// リセット（モック/管理オペレーション）
	ResetTokenOperations(ctx context.Context) error

	// 任意: トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
