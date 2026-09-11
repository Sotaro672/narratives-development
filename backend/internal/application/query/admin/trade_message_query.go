// backend/internal/application/query/admin/trade_message_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	avatardom "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
	tradedom "narratives/internal/domain/trade"
)

var ErrTradeMessageQueryNotConfigured = errors.New(
	"trade message query is not configured",
)

type tradeMessageReader interface {
	ListByTradeID(
		ctx context.Context,
		tradeID string,
		filter tradedom.MessageListFilter,
	) ([]tradedom.Message, error)
}

type tradeMessageAvatarReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}

type tradeMessageReportCaseReader interface {
	ListCases(
		ctx context.Context,
		filter reportdom.CaseFilter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[reportdom.ReportCase], error)
}

type TradeMessageListItem struct {
	ID           string                     `json:"id"`
	TradeID      string                     `json:"tradeId"`
	SenderSide   tradedom.MessageSenderSide `json:"senderSide"`
	SenderType   tradedom.MessageSenderType `json:"senderType"`
	SenderID     string                     `json:"senderId"`
	SenderName   string                     `json:"senderName"`
	Content      string                     `json:"content"`
	Images       []tradedom.MessageImage    `json:"images"`
	ReportCount  int                        `json:"reportCount"`
	ReportCaseID string                     `json:"reportCaseId,omitempty"`
	CreatedAt    time.Time                  `json:"createdAt"`
}

type TradeMessageListResult struct {
	Items      []TradeMessageListItem `json:"items"`
	TotalCount int                    `json:"totalCount"`
}

type TradeMessageQuery struct {
	messageReader    tradeMessageReader
	avatarReader     tradeMessageAvatarReader
	reportCaseReader tradeMessageReportCaseReader
}

func NewTradeMessageQuery(
	messageReader tradeMessageReader,
	avatarReader tradeMessageAvatarReader,
	reportCaseReader tradeMessageReportCaseReader,
) *TradeMessageQuery {
	return &TradeMessageQuery{
		messageReader:    messageReader,
		avatarReader:     avatarReader,
		reportCaseReader: reportCaseReader,
	}
}

func (q *TradeMessageQuery) ListByTradeID(
	ctx context.Context,
	tradeID string,
) (*TradeMessageListResult, error) {
	if q == nil ||
		q.messageReader == nil ||
		q.avatarReader == nil ||
		q.reportCaseReader == nil {
		return nil, ErrTradeMessageQueryNotConfigured
	}
	if tradeID == "" {
		return nil, fmt.Errorf("trade message query: tradeId is empty")
	}

	messages, err := q.messageReader.ListByTradeID(
		ctx,
		tradeID,
		tradedom.MessageListFilter{
			Limit: tradedom.MaxMessageListLimit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"trade message query: list messages by tradeId %s: %w",
			tradeID,
			err,
		)
	}

	reportCases, err := q.listReportCasesByTradeID(ctx, tradeID)
	if err != nil {
		return nil, err
	}

	avatarNames := make(map[string]string)
	items := make([]TradeMessageListItem, 0, len(messages))

	for _, message := range messages {
		senderName := q.resolveSenderName(
			ctx,
			message,
			avatarNames,
		)

		reportCount := 0
		reportCaseID := ""

		if reportCase, ok := reportCases[message.ID]; ok {
			reportCount = reportCase.ReportCount
			reportCaseID = string(reportCase.ID)
		}

		images := message.Images
		if images == nil {
			images = []tradedom.MessageImage{}
		}

		items = append(items, TradeMessageListItem{
			ID:           message.ID,
			TradeID:      message.TradeID,
			SenderSide:   message.SenderSide,
			SenderType:   message.SenderType,
			SenderID:     message.SenderID,
			SenderName:   senderName,
			Content:      message.Content,
			Images:       images,
			ReportCount:  reportCount,
			ReportCaseID: reportCaseID,
			CreatedAt:    message.CreatedAt,
		})
	}

	return &TradeMessageListResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}

func (q *TradeMessageQuery) listReportCasesByTradeID(
	ctx context.Context,
	tradeID string,
) (map[string]reportdom.ReportCase, error) {
	targetType := reportdom.TargetTypeTradeMessage

	filter := reportdom.CaseFilter{
		TargetType:     &targetType,
		TargetParentID: tradeID,
	}

	sortOption := common.Sort{
		Column: "updatedAt",
		Order:  common.SortDesc,
	}

	const perPage = 100

	resultByMessageID := make(map[string]reportdom.ReportCase)

	for pageNumber := 1; ; pageNumber++ {
		result, err := q.reportCaseReader.ListCases(
			ctx,
			filter,
			sortOption,
			common.Page{
				Number:  pageNumber,
				PerPage: perPage,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"trade message query: list report cases by tradeId %s: %w",
				tradeID,
				err,
			)
		}

		for _, reportCase := range result.Items {
			if reportCase.TargetType != reportdom.TargetTypeTradeMessage {
				continue
			}
			if reportCase.TargetParentID != tradeID {
				continue
			}
			if reportCase.TargetID == "" {
				continue
			}

			if _, exists := resultByMessageID[reportCase.TargetID]; exists {
				continue
			}

			resultByMessageID[reportCase.TargetID] = reportCase
		}

		if pageNumber >= result.TotalPages {
			break
		}
	}

	return resultByMessageID, nil
}

func (q *TradeMessageQuery) resolveSenderName(
	ctx context.Context,
	message tradedom.Message,
	avatarNames map[string]string,
) string {
	if message.SenderType == tradedom.MessageSenderTypeSystem ||
		message.SenderSide == tradedom.MessageSenderSideSystem {
		return "AMOL"
	}

	if message.SenderType != tradedom.MessageSenderTypeAvatar ||
		message.SenderID == "" {
		return ""
	}

	if name, ok := avatarNames[message.SenderID]; ok {
		return name
	}

	name := q.resolveAvatarName(ctx, message.SenderID)
	avatarNames[message.SenderID] = name

	return name
}

func (q *TradeMessageQuery) resolveAvatarName(
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
