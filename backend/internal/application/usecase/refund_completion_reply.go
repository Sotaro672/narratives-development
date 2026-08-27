// backend/internal/application/usecase/refund_completion_reply.go
package usecase

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	refunddom "narratives/internal/domain/refund"
)

const (
	refundCompletionReplyIDPrefix = "refund_completed_"
)

var (
	ErrRefundCompletionReplyInvalidRefund = errors.New(
		"refund completion reply: invalid refund",
	)

	ErrRefundCompletionReplyNotFinanciallyCompleted = errors.New(
		"refund completion reply: refund is not financially completed",
	)

	ErrRefundCompletionReplyAmountMismatch = errors.New(
		"refund completion reply: refund amount does not match breakdown",
	)
)

// refundCompletionReplyID returns the deterministic Inquiry reply ID used for
// one completed item-level Refund.
//
// The Refund ID itself is deterministic per Order item:
//
//	{orderId}_{orderItemIndex}
//
// Therefore:
//
//	refund_completed_{refundId}
//
// is also deterministic and can be used to prevent duplicate completion replies
// when a return receipt operation is retried.
func refundCompletionReplyID(
	refundID string,
) string {
	return refundCompletionReplyIDPrefix +
		refundID
}

// buildRefundCompletionReplyContent builds the purchaser-facing Inquiry reply
// after one item-level Refund has completed financially.
//
// Refund is the authoritative source of every displayed amount.
//
// Displayed refund breakdown:
//
//	MerchandiseAmount
//	MerchandiseTaxAmount
//	OutboundShippingAmount
//	OutboundShippingTaxAmount
//
// ReturnShippingAmount and ReturnShippingTaxAmount are intentionally excluded.
//
// They represent an additional company-side burden and are not part of the
// purchaser Stripe Refund amount.
//
// This function must only be called after:
//
//	refund.IsFinanciallyCompleted() == true
func buildRefundCompletionReplyContent(
	refund refunddom.Refund,
) (string, error) {
	if err := refund.Validate(); err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrRefundCompletionReplyInvalidRefund,
				err,
			)
	}

	if !refund.IsFinanciallyCompleted() {
		return "",
			ErrRefundCompletionReplyNotFinanciallyCompleted
	}

	if refund.Currency !=
		refunddom.CurrencyJPY {
		return "",
			ErrRefundCompletionReplyInvalidRefund
	}

	breakdownAmount, err :=
		addRefundCompletionReplyAmounts(
			refund.MerchandiseAmount,
			refund.MerchandiseTaxAmount,
			refund.OutboundShippingAmount,
			refund.OutboundShippingTaxAmount,
		)
	if err != nil {
		return "", err
	}

	if breakdownAmount !=
		refund.RefundAmount {
		return "",
			ErrRefundCompletionReplyAmountMismatch
	}

	var builder strings.Builder

	builder.WriteString(
		"返金処理が完了しました。\n\n",
	)

	builder.WriteString(
		"返金額：",
	)
	builder.WriteString(
		formatRefundCompletionReplyJPY(
			refund.RefundAmount,
		),
	)
	builder.WriteString(
		"\n\n内訳\n",
	)

	builder.WriteString(
		"商品代金：",
	)
	builder.WriteString(
		formatRefundCompletionReplyJPY(
			refund.MerchandiseAmount,
		),
	)

	builder.WriteString(
		"\n商品消費税：",
	)
	builder.WriteString(
		formatRefundCompletionReplyJPY(
			refund.MerchandiseTaxAmount,
		),
	)

	if refund.OutboundShippingAmount > 0 {
		builder.WriteString(
			"\nご購入時配送料：",
		)
		builder.WriteString(
			formatRefundCompletionReplyJPY(
				refund.OutboundShippingAmount,
			),
		)
	}

	if refund.OutboundShippingTaxAmount > 0 {
		builder.WriteString(
			"\n配送料消費税：",
		)
		builder.WriteString(
			formatRefundCompletionReplyJPY(
				refund.OutboundShippingTaxAmount,
			),
		)
	}

	return builder.String(), nil
}

func addRefundCompletionReplyAmounts(
	amounts ...int,
) (int, error) {
	maxInt := int(^uint(0) >> 1)

	total := 0

	for _, amount := range amounts {
		if amount < 0 {
			return 0,
				ErrRefundCompletionReplyInvalidRefund
		}

		if total >
			maxInt-amount {
			return 0,
				ErrRefundCompletionReplyInvalidRefund
		}

		total += amount
	}

	return total, nil
}

func formatRefundCompletionReplyJPY(
	amount int,
) string {
	if amount == 0 {
		return "¥0"
	}

	negative := amount < 0

	var value uint64
	if negative {
		value =
			uint64(-(amount + 1)) + 1
	} else {
		value = uint64(amount)
	}

	text := strconv.FormatUint(
		value,
		10,
	)

	if len(text) > 3 {
		var builder strings.Builder

		firstGroupLength :=
			len(text) % 3
		if firstGroupLength == 0 {
			firstGroupLength = 3
		}

		builder.WriteString(
			text[:firstGroupLength],
		)

		for i :=
			firstGroupLength; i < len(text); i += 3 {
			builder.WriteByte(',')
			builder.WriteString(
				text[i : i+3],
			)
		}

		text = builder.String()
	}

	if negative {
		return "-¥" + text
	}

	return "¥" + text
}
