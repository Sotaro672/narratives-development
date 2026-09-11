// backend/internal/application/port/admin_resale_trade_reader.go
package port

import (
	"context"

	tradedom "narratives/internal/domain/trade"
)

// AdminResaleTradeReader provides Admin-side read access to Trades
// associated with a Resale.
//
// A single resaleId may be associated with multiple Trades over time.
// Implementations must therefore return all matching Trades.
//
// This reader intentionally exposes Trade entities only.
// Trade messages under:
//
//	trades/{tradeId}/messages/{messageId}
//
// are outside this port's responsibility and must not be loaded.
type AdminResaleTradeReader interface {
	ListByResaleID(
		ctx context.Context,
		resaleID string,
	) ([]tradedom.Trade, error)
}
