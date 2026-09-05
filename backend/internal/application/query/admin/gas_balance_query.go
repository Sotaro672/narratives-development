// backend/internal/application/query/admin/gas_balance_query.go
package query

import (
	"context"
	"errors"
	"fmt"
)

var ErrGasBalanceQueryNotConfigured = errors.New("gas balance query is not configured")

// GasBalanceResult は Admin のガス管理画面へ公開する Reserve Wallet の残高情報です。
type GasBalanceResult struct {
	Cluster         string  `json:"cluster"`
	Address         string  `json:"address"`
	BalanceLamports string  `json:"balanceLamports"`
	BalanceSOL      float64 `json:"balanceSol"`
}

// GasBalanceFetcher は Reserve Wallet の現在残高を取得する read-only executor です。
// application/query -> infra/solana の直接依存を避けるため、DI 層で MintClient.GetReserveBalance をラップして注入します。
//
// IMPORTANT:
//   - Reserve から SOL を送金しない
//   - Fee Payer を補充しない
//   - Mint を実行しない
//   - Merkle Tree / Core Collection を作成しない
type GasBalanceFetcher func(ctx context.Context) (*GasBalanceResult, error)

type GasBalanceQuery struct {
	fetch GasBalanceFetcher
}

func NewGasBalanceQuery(fetch GasBalanceFetcher) *GasBalanceQuery {
	return &GasBalanceQuery{fetch: fetch}
}

// GetGasBalance は solana-bubblegum service が管理する Reserve Wallet の現在残高を取得します。
func (q *GasBalanceQuery) GetGasBalance(ctx context.Context) (*GasBalanceResult, error) {
	if q == nil || q.fetch == nil {
		return nil, ErrGasBalanceQueryNotConfigured
	}

	result, err := q.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reserve wallet gas balance: %w", err)
	}
	if result == nil {
		return nil, errors.New("gas balance fetcher returned nil result")
	}
	if result.Cluster == "" {
		return nil, errors.New("gas balance cluster is empty")
	}
	if result.Address == "" {
		return nil, errors.New("gas balance address is empty")
	}
	if result.BalanceLamports == "" {
		return nil, errors.New("gas balance balanceLamports is empty")
	}

	return result, nil
}
