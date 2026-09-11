// backend/internal/application/port/admin_trade_message_stats_reader.go
package port

import "context"

// AdminTradeMessageStats represents aggregate message statistics for one Trade.
//
// CommentCount counts user-authored Trade messages.
// System messages are excluded.
//
// ReportCount is the sum of report counts for all reportable Trade messages
// belonging to the Trade.
type AdminTradeMessageStats struct {
	CommentCount int
	ReportCount  int
}

// AdminTradeMessageStatsReader provides Admin-side aggregate statistics for
// Trade messages without exposing message contents.
type AdminTradeMessageStatsReader interface {
	GetByTradeID(
		ctx context.Context,
		tradeID string,
	) (AdminTradeMessageStats, error)
}
