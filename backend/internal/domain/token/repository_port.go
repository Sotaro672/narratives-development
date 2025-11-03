package token

import (
	"context"
	"errors"
	"time"
)

// ========================================
// 入出力（契約のみ）
// ========================================

type CreateTokenInput struct {
	MintAddress   string `json:"mintAddress"`
	MintRequestID string `json:"mintRequestId"`
	Owner         string `json:"owner"`
}

type UpdateTokenInput struct {
	MintRequestID *string `json:"mintRequestId,omitempty"`
	Owner         *string `json:"owner,omitempty"`
}

// ========================================
// 検索条件/ソート/ページング（契約のみ）
// ========================================

type Filter struct {
	// 識別子
	MintAddresses   []string
	MintRequestIDs  []string
	Owners          []string
	MintAddressLike string // 部分一致（実装依存）

	// 期間（実装側の保持フィールドに合わせて任意で利用）
	MintedFrom         *time.Time
	MintedTo           *time.Time
	LastTransferredFrom *time.Time
	LastTransferredTo   *time.Time
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByMintAddress       SortColumn = "mintAddress"
	SortByMintRequestID     SortColumn = "mintRequestId"
	SortByOwner             SortColumn = "owner"
	SortByMintedAt          SortColumn = "mintedAt"
	SortByLastTransferredAt SortColumn = "lastTransferredAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []Token
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ========================================
// 統計（任意契約）
// ========================================

type TokenStats struct {
	TotalTokens        int
	UniqueOwners       int
	UniqueMintRequests int
	ByMintRequest      map[string]int
	ByOwner            map[string]int
	TopOwners          []struct {
		Owner string
		Count int
	}
	TopMintRequests []struct {
		MintRequestID string
		Count         int
	}
}

// ========================================
// Repository Port（契約のみ）
// ========================================

type RepositoryPort interface {
	// 取得系
	GetByMintAddress(ctx context.Context, mintAddress string) (Token, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)
	GetByOwner(ctx context.Context, owner string) ([]Token, error)
	GetByMintRequest(ctx context.Context, mintRequestID string) ([]Token, error)

	// 変更系
	Create(ctx context.Context, in CreateTokenInput) (Token, error)
	Update(ctx context.Context, mintAddress string, in UpdateTokenInput) (Token, error)
	Delete(ctx context.Context, mintAddress string) error

	// ドメイン操作
	Transfer(ctx context.Context, mintAddress, newOwner string) (Token, error)
	Burn(ctx context.Context, mintAddress string) error

	// 統計（任意）
	GetStats(ctx context.Context) (TokenStats, error)

	// （任意）トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// 管理（任意）
	Reset(ctx context.Context) error
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("token: not found")
	ErrConflict = errors.New("token: conflict")
)
