// backend\internal\application\query\admin\mint_list_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	mintdom "narratives/internal/domain/mint"
)

var ErrMintListQueryNotConfigured = errors.New("mint list query is not configured")

type MintListReader interface {
	ListAll(ctx context.Context) ([]mintdom.Mint, error)
}

type MintListResult struct {
	Items      []mintdom.Mint `json:"items"`
	TotalCount int            `json:"totalCount"`
}

type MintListQuery struct {
	reader MintListReader
}

func NewMintListQuery(reader MintListReader) *MintListQuery {
	return &MintListQuery{reader: reader}
}

func (q *MintListQuery) ListMints(ctx context.Context) (*MintListResult, error) {
	if q == nil || q.reader == nil {
		return nil, ErrMintListQueryNotConfigured
	}

	items, err := q.reader.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mints: %w", err)
	}

	return &MintListResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}
