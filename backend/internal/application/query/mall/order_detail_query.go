// backend/internal/application/query/mall/order_detail_query.go
package mall

import (
	"context"
	"errors"
	"time"

	orderdetaildto "narratives/internal/application/query/mall/dto"
	mallshared "narratives/internal/application/query/mall/shared"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

var (
	ErrOrderDetailQueryNotConfigured = errors.New(
		"mall order detail query: not configured",
	)
)

type OrderDetailPaymentReader interface {
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*paymentdom.Payment, error)
}

type OrderDetailQuery struct {
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver
	displayResolver            mallshared.MallDisplayResolver
	paymentReader              OrderDetailPaymentReader
}

func NewOrderDetailQuery(
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver,
	displayResolver mallshared.MallDisplayResolver,
	paymentReader OrderDetailPaymentReader,
) *OrderDetailQuery {
	return &OrderDetailQuery{
		inventoryBlueprintResolver: inventoryBlueprintResolver,
		displayResolver:            displayResolver,
		paymentReader:              paymentReader,
	}
}

func (q *OrderDetailQuery) EnrichOrderDetail(
	ctx context.Context,
	in orderdom.Order,
) (orderdetaildto.OrderDetail, error) {
	if q == nil ||
		q.inventoryBlueprintResolver == nil ||
		q.displayResolver == nil {
		return orderdetaildto.OrderDetail{},
			ErrOrderDetailQueryNotConfigured
	}

	amountSummary :=
		orderdom.PaymentAmountSummary{}

	hasActiveItem := false

	for _, item := range in.Items {
		if !item.IsCancelled {
			hasActiveItem = true
			break
		}
	}

	if hasActiveItem {
		resolvedAmountSummary, err :=
			orderdom.CalculatePaymentAmountSummary(
				in,
			)
		if err != nil {
			return orderdetaildto.OrderDetail{},
				err
		}

		amountSummary =
			resolvedAmountSummary
	}

	out := orderdetaildto.OrderDetail{
		ID:       in.ID,
		UserID:   in.UserID,
		AvatarID: in.AvatarID,
		CartID:   in.CartID,

		ShippingQuoteSnapshot: cloneOrderDetailShippingQuote(
			in.ShippingQuoteSnapshot,
		),

		SubtotalAmount: amountSummary.SubtotalAmount,
		ShippingAmount: amountSummary.ShippingAmount,
		ConsumptionTax: amountSummary.ConsumptionTax,
		TotalAmount:    amountSummary.TotalAmount,

		Paid: in.Paid,

		RefundStatus: string(
			paymentdom.RefundStatusNone,
		),

		Items: make(
			[]orderdetaildto.OrderDetailItem,
			0,
			len(in.Items),
		),
	}

	if !in.CreatedAt.IsZero() {
		out.CreatedAt = in.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	// An unpaid Order may legitimately have no Payment document yet.
	//
	// Once Order.Paid is true, Payment becomes authoritative for Refund state.
	// Refund lifecycle is intentionally independent from PaymentIntent status:
	// a successfully refunded Payment still has Payment.Status=succeeded.
	if in.Paid {
		if q.paymentReader == nil {
			return orderdetaildto.OrderDetail{},
				ErrOrderDetailQueryNotConfigured
		}

		payment, err := q.paymentReader.GetByPaymentID(
			ctx,
			in.ID,
		)
		if err != nil {
			return orderdetaildto.OrderDetail{}, err
		}

		if payment == nil ||
			payment.PaymentID != in.ID {
			return orderdetaildto.OrderDetail{},
				paymentdom.ErrNotFound
		}

		refundStatus := payment.RefundStatus
		if refundStatus == "" {
			// Legacy Payment documents created before Refund fields existed are
			// interpreted as not refunded.
			refundStatus =
				paymentdom.RefundStatusNone
		}

		if !paymentdom.IsValidRefundStatus(
			refundStatus,
		) {
			return orderdetaildto.OrderDetail{},
				paymentdom.ErrInvalidRefundStatus
		}

		out.RefundStatus =
			string(refundStatus)

		out.RefundedAmount =
			payment.RefundedAmount

		if payment.RefundedAt != nil &&
			!payment.RefundedAt.IsZero() {
			out.RefundedAt =
				payment.RefundedAt.
					UTC().
					Format(time.RFC3339Nano)
		}
	}

	blueprintCache := make(map[string]historyBlueprintIDs)
	productBlueprintCache := make(map[string]mallshared.ProductBlueprintDisplay)
	tokenBlueprintCache := make(map[string]mallshared.TokenBlueprintDisplay)
	brandCache := make(map[string]mallshared.BrandDisplay)
	modelByIDCache := make(map[string]mallshared.ModelDisplay)
	modelByProductIDCache := make(map[string]mallshared.ModelDisplay)

	for _, sourceItem := range in.Items {
		item := orderdetaildto.OrderDetailItem{
			ItemType: string(sourceItem.Type),

			ModelID:     sourceItem.ModelID,
			InventoryID: sourceItem.InventoryID,
			ListID:      sourceItem.ListID,
			ResaleID:    sourceItem.ResaleID,

			ProductID: sourceItem.ProductID,

			ProductBlueprintID: sourceItem.ProductBlueprintID,
			TokenBlueprintID:   sourceItem.TokenBlueprintID,

			BrandID: sourceItem.BrandID,

			ProductBlueprintCategoryPath: append(
				[]string(nil),
				sourceItem.ProductBlueprintCategoryPath...,
			),
			ConsumptionTaxRate: sourceItem.ConsumptionTaxRate,

			Qty:   sourceItem.Qty,
			Price: sourceItem.Price,

			IsCancelled:       sourceItem.IsCancelled,
			IsDispatched:      sourceItem.IsDispatched,
			IsReturnRequested: sourceItem.IsReturnRequested,

			Transferred: sourceItem.Transferred,
		}

		if sourceItem.ReturnRequestedAt != nil &&
			!sourceItem.ReturnRequestedAt.IsZero() {
			item.ReturnRequestedAt = sourceItem.ReturnRequestedAt.
				UTC().
				Format(time.RFC3339Nano)
		}

		if sourceItem.TokenTransferVerifiedAt != nil &&
			!sourceItem.TokenTransferVerifiedAt.IsZero() {
			item.TokenTransferVerifiedAt = sourceItem.TokenTransferVerifiedAt.
				UTC().
				Format(time.RFC3339Nano)
		}

		if sourceItem.TransferredAt != nil &&
			!sourceItem.TransferredAt.IsZero() {
			item.TransferredAt = sourceItem.TransferredAt.
				UTC().
				Format(time.RFC3339Nano)
		}

		blueprintIDs := historyBlueprintIDs{
			ProductBlueprintID: sourceItem.ProductBlueprintID,
			TokenBlueprintID:   sourceItem.TokenBlueprintID,
		}

		if sourceItem.InventoryID != "" {
			cachedBlueprintIDs, ok :=
				blueprintCache[sourceItem.InventoryID]

			if ok {
				blueprintIDs = mergeHistoryBlueprintIDs(
					blueprintIDs,
					cachedBlueprintIDs,
				)
			} else {
				productBlueprintID,
					tokenBlueprintID,
					err :=
					q.inventoryBlueprintResolver.
						ResolveBlueprintIDsByInventoryID(
							ctx,
							sourceItem.InventoryID,
						)

				if err == nil {
					resolvedBlueprintIDs := historyBlueprintIDs{
						ProductBlueprintID: productBlueprintID,
						TokenBlueprintID:   tokenBlueprintID,
					}

					blueprintCache[sourceItem.InventoryID] =
						resolvedBlueprintIDs

					blueprintIDs = mergeHistoryBlueprintIDs(
						blueprintIDs,
						resolvedBlueprintIDs,
					)
				}
			}
		}

		modelInfo := mallshared.ModelDisplay{}

		if sourceItem.ModelID != "" {
			cachedModel, ok :=
				modelByIDCache[sourceItem.ModelID]

			if ok {
				modelInfo = cachedModel
			} else {
				resolvedModel, err :=
					q.displayResolver.
						ResolveModelByModelID(
							ctx,
							sourceItem.ModelID,
						)

				if err == nil {
					modelInfo = resolvedModel
					modelByIDCache[sourceItem.ModelID] =
						resolvedModel
				}
			}
		} else if sourceItem.ProductID != "" {
			cachedModel, ok :=
				modelByProductIDCache[sourceItem.ProductID]

			if ok {
				modelInfo = cachedModel
			} else {
				resolvedModel, err :=
					q.displayResolver.
						ResolveModelByProductID(
							ctx,
							sourceItem.ProductID,
						)

				if err == nil {
					modelInfo = resolvedModel
					modelByProductIDCache[sourceItem.ProductID] =
						resolvedModel
				}
			}
		}

		if item.ModelID == "" &&
			modelInfo.ModelID != "" {
			item.ModelID = modelInfo.ModelID
		}

		if blueprintIDs.ProductBlueprintID == "" &&
			modelInfo.ProductBlueprintID != "" {
			blueprintIDs.ProductBlueprintID =
				modelInfo.ProductBlueprintID
		}

		if blueprintIDs.ProductBlueprintID != "" {
			item.ProductBlueprintID =
				blueprintIDs.ProductBlueprintID
		}

		if blueprintIDs.TokenBlueprintID != "" {
			item.TokenBlueprintID =
				blueprintIDs.TokenBlueprintID
		}

		if blueprintIDs.ProductBlueprintID != "" {
			productBlueprintInfo, ok :=
				productBlueprintCache[blueprintIDs.ProductBlueprintID]

			if !ok {
				resolvedInfo, err :=
					q.displayResolver.
						ResolveProductBlueprintInfo(
							ctx,
							blueprintIDs.ProductBlueprintID,
						)

				if err == nil {
					productBlueprintInfo = resolvedInfo
					productBlueprintCache[blueprintIDs.ProductBlueprintID] =
						resolvedInfo
				}
			}

			if productBlueprintInfo.ProductName != "" {
				item.ProductName =
					productBlueprintInfo.ProductName
			}

			if item.BrandID == "" &&
				productBlueprintInfo.BrandID != "" {
				item.BrandID =
					productBlueprintInfo.BrandID
			}
		}

		if blueprintIDs.TokenBlueprintID != "" {
			tokenBlueprintInfo, ok :=
				tokenBlueprintCache[blueprintIDs.TokenBlueprintID]

			if !ok {
				resolvedInfo, err :=
					q.displayResolver.
						ResolveTokenBlueprintInfo(
							ctx,
							blueprintIDs.TokenBlueprintID,
						)

				if err == nil {
					tokenBlueprintInfo = resolvedInfo
					tokenBlueprintCache[blueprintIDs.TokenBlueprintID] =
						resolvedInfo
				}
			}

			if tokenBlueprintInfo.TokenName != "" {
				item.TokenName =
					tokenBlueprintInfo.TokenName
			}

			if tokenBlueprintInfo.TokenIcon != "" {
				item.TokenIcon =
					tokenBlueprintInfo.TokenIcon
			}

			if item.BrandID == "" &&
				tokenBlueprintInfo.BrandID != "" {
				item.BrandID =
					tokenBlueprintInfo.BrandID
			}
		}

		if item.BrandID != "" {
			brandInfo, ok :=
				brandCache[item.BrandID]

			if !ok {
				resolvedInfo, err :=
					q.displayResolver.
						ResolveBrandInfo(
							ctx,
							item.BrandID,
						)

				if err == nil {
					brandInfo = resolvedInfo
					brandCache[item.BrandID] =
						resolvedInfo
				}
			}

			if brandInfo.BrandName != "" {
				item.BrandName =
					brandInfo.BrandName
			}

			if brandInfo.BrandIcon != "" {
				item.BrandIcon =
					brandInfo.BrandIcon
			}
		}

		applyModelDisplayToOrderDetailItem(
			&item,
			modelInfo,
		)

		out.Items = append(
			out.Items,
			item,
		)
	}

	return out, nil
}

func applyModelDisplayToOrderDetailItem(
	item *orderdetaildto.OrderDetailItem,
	model mallshared.ModelDisplay,
) {
	if item == nil {
		return
	}

	if model.ModelID != "" {
		item.ModelID = model.ModelID
	}

	if item.ProductBlueprintID == "" &&
		model.ProductBlueprintID != "" {
		item.ProductBlueprintID =
			model.ProductBlueprintID
	}

	item.Kind = model.Kind
	item.ModelNumber = model.ModelNumber
	item.Size = model.Size

	if model.ColorName != "" ||
		model.ColorRGB != 0 {
		item.Color = &orderdetaildto.OrderDetailColor{
			Name: model.ColorName,
			RGB:  model.ColorRGB,
		}
	}

	item.Measurements =
		cloneOrderDetailMeasurements(
			model.Measurements,
		)

	item.VolumeValue =
		cloneOrderDetailIntPointer(
			model.VolumeValue,
		)

	item.VolumeUnit = model.VolumeUnit
}

func cloneOrderDetailMeasurements(
	in map[string]int,
) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(
		map[string]int,
		len(in),
	)

	for key, value := range in {
		out[key] = value
	}

	return out
}

func cloneOrderDetailIntPointer(
	in *int,
) *int {
	if in == nil {
		return nil
	}

	value := *in

	return &value
}

func cloneOrderDetailShippingQuote(
	in orderdom.ShippingQuoteSnapshot,
) orderdom.ShippingQuoteSnapshot {
	out := orderdom.ShippingQuoteSnapshot{
		Amount:   in.Amount,
		Currency: in.Currency,
	}

	if len(in.Items) == 0 {
		out.Items =
			[]orderdom.ShippingQuoteItemSnapshot{}

		return out
	}

	out.Items = make(
		[]orderdom.ShippingQuoteItemSnapshot,
		len(in.Items),
	)

	copy(
		out.Items,
		in.Items,
	)

	return out
}
