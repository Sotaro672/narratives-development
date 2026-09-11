// backend/internal/adapters/out/firestore/trade_message_stats_reader_fs.go
package firestore

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	applicationport "narratives/internal/application/port"
	tradedom "narratives/internal/domain/trade"
)

var ErrTradeMessageStatsReaderNotConfigured = errors.New(
	"trade_message_stats_reader_fs: not configured",
)

const tradeMessageReportTargetType = "TRADE_MESSAGE"

type TradeMessageStatsReaderFS struct {
	Client *firestore.Client
}

var _ applicationport.AdminTradeMessageStatsReader = (*TradeMessageStatsReaderFS)(nil)

func NewTradeMessageStatsReaderFS(
	client *firestore.Client,
) *TradeMessageStatsReaderFS {
	return &TradeMessageStatsReaderFS{
		Client: client,
	}
}

func (r *TradeMessageStatsReaderFS) messageCol(
	tradeID string,
) *firestore.CollectionRef {
	return r.Client.
		Collection("trades").
		Doc(tradeID).
		Collection("messages")
}

func (r *TradeMessageStatsReaderFS) reportCaseCol() *firestore.CollectionRef {
	return r.Client.Collection(defaultReportCaseCollection)
}

func (r *TradeMessageStatsReaderFS) GetByTradeID(
	ctx context.Context,
	tradeID string,
) (applicationport.AdminTradeMessageStats, error) {
	if r == nil || r.Client == nil {
		return applicationport.AdminTradeMessageStats{},
			ErrTradeMessageStatsReaderNotConfigured
	}
	if tradeID == "" {
		return applicationport.AdminTradeMessageStats{},
			tradedom.ErrInvalidMessageTradeID
	}

	commentCount, err := r.countComments(ctx, tradeID)
	if err != nil {
		return applicationport.AdminTradeMessageStats{}, err
	}

	reportCount, err := r.countReports(ctx, tradeID)
	if err != nil {
		return applicationport.AdminTradeMessageStats{}, err
	}

	return applicationport.AdminTradeMessageStats{
		CommentCount: commentCount,
		ReportCount:  reportCount,
	}, nil
}

func (r *TradeMessageStatsReaderFS) countComments(
	ctx context.Context,
	tradeID string,
) (int, error) {
	iter := r.messageCol(tradeID).
		Query.
		Select("senderSide").
		Documents(ctx)
	defer iter.Stop()

	count := 0

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}

		var doc struct {
			SenderSide string `firestore:"senderSide"`
		}
		if err := snap.DataTo(&doc); err != nil {
			return 0, err
		}

		switch tradedom.MessageSenderSide(doc.SenderSide) {
		case tradedom.MessageSenderSideBuyer,
			tradedom.MessageSenderSideSeller:
			count++
		}
	}

	return count, nil
}

func (r *TradeMessageStatsReaderFS) countReports(
	ctx context.Context,
	tradeID string,
) (int, error) {
	query := r.reportCaseCol().
		Query.
		Where("targetType", "==", tradeMessageReportTargetType).
		Where("targetParentId", "==", tradeID).
		Select("reportCount")

	iter := query.Documents(ctx)
	defer iter.Stop()

	total := 0

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}

		var doc struct {
			ReportCount int `firestore:"reportCount"`
		}
		if err := snap.DataTo(&doc); err != nil {
			return 0, err
		}
		if doc.ReportCount < 0 {
			continue
		}

		total += doc.ReportCount
	}

	return total, nil
}
