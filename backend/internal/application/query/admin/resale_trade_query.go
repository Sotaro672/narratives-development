// backend/internal/application/query/admin/resale_trade_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	applicationport "narratives/internal/application/port"
	tradedom "narratives/internal/domain/trade"
)

var ErrResaleTradeQueryNotConfigured = errors.New(
	"resale trade query is not configured",
)

type ResaleTradeListResult struct {
	Items      []tradedom.Trade `json:"items"`
	TotalCount int              `json:"totalCount"`
}

type ResaleTradeQuery struct {
	reader applicationport.AdminResaleTradeReader
}

func NewResaleTradeQuery(
	reader applicationport.AdminResaleTradeReader,
) *ResaleTradeQuery {
	return &ResaleTradeQuery{
		reader: reader,
	}
}

func (q *ResaleTradeQuery) ListByResaleID(
	ctx context.Context,
	resaleID string,
) (*ResaleTradeListResult, error) {
	if q == nil || q.reader == nil {
		return nil, ErrResaleTradeQueryNotConfigured
	}
	if resaleID == "" {
		return nil, fmt.Errorf("resale trade query: resaleId is empty")
	}

	trades, err := q.reader.ListByResaleID(ctx, resaleID)
	if err != nil {
		return nil, fmt.Errorf(
			"resale trade query: list trades by resaleId %s: %w",
			resaleID,
			err,
		)
	}

	if trades == nil {
		trades = []tradedom.Trade{}
	}

	return &ResaleTradeListResult{
		Items:      trades,
		TotalCount: len(trades),
	}, nil
}
