// backend/internal/domain/wallet/repository_port.go
package wallet

import (
	"context"
	"errors"
	"time"
)

// Note: Wallet はドメインエンティティです（entity.go 等で定義されている想定）。
// このファイルでは再定義しません。

// ========================================
// 外部DTO（API/ストレージ境界）
// ========================================

type WalletDTO struct {
	WalletAddress string       `json:"walletAddress"`
	Tokens        []string     `json:"tokens"`
	Status        WalletStatus `json:"status"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	LastUpdatedAt time.Time    `json:"lastUpdatedAt"`
}

// ========================================
// 入出力DTO（UseCase/Service -> Repository）
// ========================================

type CreateWalletInput struct {
	WalletAddress string        `json:"walletAddress"`
	Tokens        []string      `json:"tokens,omitempty"`
	Status        *WalletStatus `json:"status,omitempty"`

	// nil の場合は実装側で補完可（通常は現在時刻）
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt,omitempty"`
}

type UpdateWalletInput struct {
	// walletAddress はURL等で指定される想定
	Tokens        *[]string     `json:"tokens,omitempty"`       // 完全置換
	AddTokens     []string      `json:"addTokens,omitempty"`    // 追加
	RemoveTokens  []string      `json:"removeTokens,omitempty"` // 削除
	Status        *WalletStatus `json:"status,omitempty"`
	UpdatedAt     *time.Time    `json:"updatedAt,omitempty"`
	LastUpdatedAt *time.Time    `json:"lastUpdatedAt,omitempty"` // 任意で上書き（通常はトークン変更時に実装側で更新）
}

// バッチ関連
type BatchWalletRequest struct {
	WalletAddresses []string `json:"walletAddresses"`
	IncludeDefaults bool     `json:"includeDefaults"`
}

type BatchWalletResponse struct {
	Wallets  []*Wallet `json:"wallets"`
	NotFound []string  `json:"notFound"`
}

type BatchWalletUpdate struct {
	WalletAddress string                 `json:"walletAddress"`
	Data          map[string]interface{} `json:"data"` // Partial<Omit<Wallet, 'walletAddress'>>
}

type BatchWalletUpdateResponse struct {
	Succeeded []*Wallet `json:"succeeded"`
	Failed    []struct {
		WalletAddress string `json:"walletAddress"`
		Error         string `json:"error"`
	} `json:"failed"`
}

type BatchTokenAddRequest struct {
	WalletAddress string   `json:"walletAddress"`
	MintAddresses []string `json:"mintAddresses"`
}

type BatchTokenRemoveRequest struct {
	WalletAddress string   `json:"walletAddress"`
	MintAddresses []string `json:"mintAddresses"`
}

// ========================================
// フィルター・ソート・ページネーション
// ========================================

type TokenTier string

const (
	TierWhale  TokenTier = "whale"
	TierLarge  TokenTier = "large"
	TierMedium TokenTier = "medium"
	TierSmall  TokenTier = "small"
	TierEmpty  TokenTier = "empty"
)

type TokenTierDefinition struct {
	Tier        TokenTier `json:"tier"`
	DisplayName string    `json:"displayName"`
	MinTokens   int       `json:"minTokens"`
	MaxTokens   *int      `json:"maxTokens,omitempty"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
}

type WalletFilter struct {
	SearchQuery   string         `json:"searchQuery,omitempty"` // 部分一致: walletAddress など実装依存
	HasTokensOnly bool           `json:"hasTokensOnly,omitempty"`
	MinTokenCount *int           `json:"minTokenCount,omitempty"`
	MaxTokenCount *int           `json:"maxTokenCount,omitempty"`
	TokenIDs      []string       `json:"tokenIds,omitempty"` // 所持トークンに含まれるウォレット
	Tiers         []TokenTier    `json:"tiers,omitempty"`
	Statuses      []WalletStatus `json:"statuses,omitempty"`

	LastUpdatedAfter  *time.Time `json:"lastUpdatedAfter,omitempty"`
	LastUpdatedBefore *time.Time `json:"lastUpdatedBefore,omitempty"`
	CreatedAfter      *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore     *time.Time `json:"createdBefore,omitempty"`
	UpdatedAfter      *time.Time `json:"updatedAfter,omitempty"`
	UpdatedBefore     *time.Time `json:"updatedBefore,omitempty"`
}

type WalletSortConfig struct {
	// "walletAddress" | "tokenCount" | "lastUpdatedAt" | "createdAt" | "updatedAt" | "status"
	Column string `json:"column"`
	// "asc" | "desc"
	Order string `json:"order"`
}

type WalletPaginationOptions struct {
	Page         int `json:"page"`
	ItemsPerPage int `json:"itemsPerPage"`
}

type WalletSearchOptions struct {
	Filter     *WalletFilter            `json:"filter,omitempty"`
	Sort       *WalletSortConfig        `json:"sort,omitempty"`
	Pagination *WalletPaginationOptions `json:"pagination,omitempty"`
}

type WalletPaginationResult struct {
	Wallets         []*Wallet `json:"wallets"`
	TotalPages      int       `json:"totalPages"`
	TotalCount      int       `json:"totalCount"`
	CurrentPage     int       `json:"currentPage"`
	ItemsPerPage    int       `json:"itemsPerPage"`
	HasNextPage     bool      `json:"hasNextPage"`
	HasPreviousPage bool      `json:"hasPreviousPage"`
}

// ========================================
// 統計・分析
// ========================================

type WalletStats struct {
	TotalWallets           int     `json:"totalWallets"`
	WalletsWithTokens      int     `json:"walletsWithTokens"`
	WalletsWithoutTokens   int     `json:"walletsWithoutTokens"`
	TotalTokens            int     `json:"totalTokens"`
	AverageTokensPerWallet float64 `json:"averageTokensPerWallet"`
	MedianTokensPerWallet  float64 `json:"medianTokensPerWallet"`
	TopHolderTokenCount    int     `json:"topHolderTokenCount"`
	UniqueTokenTypes       int     `json:"uniqueTokenTypes"`
}

type TokenDistribution struct {
	Tier       TokenTier `json:"tier"`
	Count      int       `json:"count"`
	Percentage float64   `json:"percentage"`
}

type TokenHoldingStats struct {
	TokenID       string `json:"tokenId"`
	HolderCount   int    `json:"holderCount"`
	TotalHoldings int    `json:"totalHoldings"`
	TopHolders    []struct {
		WalletAddress string `json:"walletAddress"`
		TokenCount    int    `json:"tokenCount"`
		Rank          int    `json:"rank"`
	} `json:"topHolders"`
}

// ランキング
type TopWalletInfo struct {
	*Wallet
	Rank       int                 `json:"rank"`
	TokenCount int                 `json:"tokenCount"`
	TierInfo   TokenTierDefinition `json:"tierInfo"`
}

type WalletRankingRequest struct {
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	TokenID *string `json:"tokenId,omitempty"` // 特定トークンのホルダーランキング
}

type WalletRankingResponse struct {
	Rankings []TopWalletInfo `json:"rankings"`
	Total    int             `json:"total"`
}

// トークンホルダー
type TokenHolder struct {
	WalletAddress string    `json:"walletAddress"`
	TokenCount    int       `json:"tokenCount"`
	Percentage    *float64  `json:"percentage,omitempty"`
	Tier          TokenTier `json:"tier"`
}

// ========================================
// Repository Port（契約のみ）
// ========================================

type RepositoryPort interface {
	// 取得系
	GetAllWallets(ctx context.Context) ([]*Wallet, error)
	GetWalletByAddress(ctx context.Context, walletAddress string) (*Wallet, error)

	// 検索・一覧
	SearchWallets(ctx context.Context, opts WalletSearchOptions) (WalletPaginationResult, error)

	// 変更系
	CreateWallet(ctx context.Context, in CreateWalletInput) (*Wallet, error)
	UpdateWallet(ctx context.Context, walletAddress string, in UpdateWalletInput) (*Wallet, error)
	DeleteWallet(ctx context.Context, walletAddress string) error

	// トークン操作（ドメイン操作の明示メソッド）
	AddTokenToWallet(ctx context.Context, walletAddress, mintAddress string) (*Wallet, error)
	RemoveTokenFromWallet(ctx context.Context, walletAddress, mintAddress string) (*Wallet, error)
	AddTokensToWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*Wallet, error)
	RemoveTokensFromWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*Wallet, error)

	// バッチ
	GetWalletsBatch(ctx context.Context, req BatchWalletRequest) (BatchWalletResponse, error)
	UpdateWalletsBatch(ctx context.Context, updates []BatchWalletUpdate) (BatchWalletUpdateResponse, error)

	// 統計・分析
	GetWalletStats(ctx context.Context) (WalletStats, error)
	GetTokenDistribution(ctx context.Context) ([]TokenDistribution, error)
	GetTokenHoldingStats(ctx context.Context, tokenID string) (TokenHoldingStats, error)
	GetWalletRanking(ctx context.Context, req WalletRankingRequest) (WalletRankingResponse, error)
	GetTokenHolders(ctx context.Context, tokenID string, limit int) ([]TokenHolder, error)

	// 管理（開発用）
	ResetWallets(ctx context.Context) error

	// 任意: トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("wallet: not found")
	ErrConflict = errors.New("wallet: conflict")
)
