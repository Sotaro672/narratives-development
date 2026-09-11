// backend/internal/application/query/admin/resale_trade_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	applicationport "narratives/internal/application/port"
	avatardom "narratives/internal/domain/avatar"
	tradedom "narratives/internal/domain/trade"
)

var ErrResaleTradeQueryNotConfigured = errors.New(
	"resale trade query is not configured",
)

type resaleTradeAvatarReader interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type ResaleTradeListItem struct {
	tradedom.Trade
	BuyerAvatarName string `json:"buyerAvatarName"`
}

type ResaleTradeListResult struct {
	Items      []ResaleTradeListItem `json:"items"`
	TotalCount int                   `json:"totalCount"`
}

type ResaleTradeQuery struct {
	reader       applicationport.AdminResaleTradeReader
	avatarReader resaleTradeAvatarReader
}

func NewResaleTradeQuery(
	reader applicationport.AdminResaleTradeReader,
	avatarReader resaleTradeAvatarReader,
) *ResaleTradeQuery {
	return &ResaleTradeQuery{
		reader:       reader,
		avatarReader: avatarReader,
	}
}

func (q *ResaleTradeQuery) ListByResaleID(
	ctx context.Context,
	resaleID string,
) (*ResaleTradeListResult, error) {
	if q == nil || q.reader == nil || q.avatarReader == nil {
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

	items := make([]ResaleTradeListItem, 0, len(trades))
	avatarNames := make(map[string]string)

	for _, trade := range trades {
		buyerAvatarName, ok := avatarNames[trade.BuyerAvatarID]
		if !ok {
			buyerAvatarName = q.resolveAvatarName(ctx, trade.BuyerAvatarID)
			avatarNames[trade.BuyerAvatarID] = buyerAvatarName
		}

		items = append(items, ResaleTradeListItem{
			Trade:           trade,
			BuyerAvatarName: buyerAvatarName,
		})
	}

	return &ResaleTradeListResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}

func (q *ResaleTradeQuery) resolveAvatarName(
	ctx context.Context,
	avatarID string,
) string {
	if q == nil || q.avatarReader == nil || avatarID == "" {
		return ""
	}

	avatar, err := q.avatarReader.GetByID(ctx, avatarID)
	if err != nil {
		return ""
	}

	return avatar.AvatarName
}
