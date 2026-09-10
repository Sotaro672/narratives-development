// backend/internal/application/query/admin/mint_list_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	mintdom "narratives/internal/domain/mint"
)

var ErrMintListQueryNotConfigured = errors.New("mint list query is not configured")

type MintListReader interface {
	ListAll(ctx context.Context) ([]mintdom.Mint, error)
}

type MintNameResolver interface {
	ResolveBrandName(ctx context.Context, brandID string) string
	ResolveTokenName(ctx context.Context, tokenBlueprintID string) string
}

type MintListItem struct {
	BrandName          string             `json:"brandName"`
	TokenBlueprintName string             `json:"tokenBlueprintName"`
	ProductCount       int                `json:"productCount"`
	Status             mintdom.MintStatus `json:"status"`
	CreatedAt          time.Time          `json:"createdAt"`
	RequestedBy        string             `json:"requestedBy,omitempty"`
	MintedAt           *time.Time         `json:"mintedAt,omitempty"`
	ScheduledBurnDate  *time.Time         `json:"scheduledBurnDate,omitempty"`
}

type MintListResult struct {
	Items      []MintListItem `json:"items"`
	TotalCount int            `json:"totalCount"`
}

type MintListQuery struct {
	reader       MintListReader
	nameResolver MintNameResolver
}

func NewMintListQuery(reader MintListReader, nameResolver MintNameResolver) *MintListQuery {
	return &MintListQuery{
		reader:       reader,
		nameResolver: nameResolver,
	}
}

func (q *MintListQuery) ListMints(ctx context.Context) (*MintListResult, error) {
	if q == nil || q.reader == nil || q.nameResolver == nil {
		return nil, ErrMintListQueryNotConfigured
	}

	mints, err := q.reader.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mints: %w", err)
	}

	items := make([]MintListItem, 0, len(mints))
	brandNames := make(map[string]string)
	tokenNames := make(map[string]string)

	for _, mint := range mints {
		brandName, ok := brandNames[mint.BrandID]
		if !ok {
			brandName = q.nameResolver.ResolveBrandName(ctx, mint.BrandID)
			brandNames[mint.BrandID] = brandName
		}

		tokenName, ok := tokenNames[mint.TokenBlueprintID]
		if !ok {
			tokenName = q.nameResolver.ResolveTokenName(ctx, mint.TokenBlueprintID)
			tokenNames[mint.TokenBlueprintID] = tokenName
		}

		items = append(items, MintListItem{
			BrandName:          brandName,
			TokenBlueprintName: tokenName,
			ProductCount:       len(mint.Products),
			Status:             mint.Status,
			CreatedAt:          mint.CreatedAt,
			RequestedBy:        mint.RequestedBy,
			MintedAt:           mint.MintedAt,
			ScheduledBurnDate:  mint.ScheduledBurnDate,
		})
	}

	return &MintListResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}
