// backend/internal/application/query/console/transaction_management_query.go
package query

import (
	"context"
	"errors"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

// ============================================================
// Ports
// ============================================================

// TransactionPaymentReader reads the Payment associated with an Order.
//
// PaymentID must be the same value as Order.ID.
type TransactionPaymentReader interface {
	GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error)
}

// ============================================================
// Query
// ============================================================

type TransactionManagementQuery struct {
	lister        OrderLister
	invRows       InventoryRowsLister
	paymentReader TransactionPaymentReader
}

type NewTransactionManagementQueryParams struct {
	Lister        OrderLister
	InvRows       InventoryRowsLister
	PaymentReader TransactionPaymentReader
}

func NewTransactionManagementQuery(
	p NewTransactionManagementQueryParams,
) *TransactionManagementQuery {
	return &TransactionManagementQuery{
		lister:        p.Lister,
		invRows:       p.InvRows,
		paymentReader: p.PaymentReader,
	}
}

// ============================================================
// Internal read model
// ============================================================

type transactionOrderCandidate struct {
	Order               orderdom.Order
	OrderAmount         int
	IsMultiCompanyOrder bool
}

// ============================================================
// Public APIs
// ============================================================

func (q *TransactionManagementQuery) List(
	ctx context.Context,
	filter orderdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[querydto.TransactionManagementRowDTO], error) {
	page = NormalizeCommonPage(page)

	if q == nil ||
		q.lister == nil ||
		q.invRows == nil ||
		q.paymentReader == nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{},
			errors.New(
				"TransactionManagementQuery.List: wiring is incomplete (lister/invRows/paymentReader required)",
			)
	}

	allowedSet, err := AllowedInventoryIDSetFromContext(
		ctx,
		q.invRows,
	)
	if err != nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
	}

	if len(allowedSet) == 0 {
		return emptyTransactionManagementPage(page), nil
	}

	candidates := make([]transactionOrderCandidate, 0, page.PerPage)

	const (
		maxScanPages  = 500
		sourcePerPage = 200
	)

	srcPage := 1

	for {
		if srcPage > maxScanPages {
			break
		}

		result, err := q.lister.ListByInventoryIDs(
			ctx,
			allowedSet,
			filter,
			sort,
			common.Page{
				Number:  srcPage,
				PerPage: sourcePerPage,
			},
		)
		if err != nil {
			return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
		}

		if result.Items == nil {
			result.Items = []orderdom.Order{}
		}

		for _, order := range result.Items {
			companyOrder, belongsToCompany, isMultiCompanyOrder, err :=
				buildCurrentCompanyOrder(
					order,
					allowedSet,
				)
			if err != nil {
				return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
			}

			if !belongsToCompany {
				continue
			}

			orderAmount, err := orderdom.CalculatePaymentAmount(
				companyOrder,
			)
			if err != nil {
				return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
			}

			candidates = append(
				candidates,
				transactionOrderCandidate{
					Order:               order,
					OrderAmount:         orderAmount,
					IsMultiCompanyOrder: isMultiCompanyOrder,
				},
			)
		}

		if len(result.Items) == 0 {
			break
		}

		if result.TotalPages > 0 {
			if srcPage >= result.TotalPages {
				break
			}
		} else if len(result.Items) < sourcePerPage {
			break
		}

		srcPage++
	}

	totalCount := len(candidates)
	totalPages := TotalPages(totalCount, page.PerPage)

	start := (page.Number - 1) * page.PerPage
	if start < 0 {
		start = 0
	}

	if start >= totalCount {
		return common.PageResult[querydto.TransactionManagementRowDTO]{
			Items:      []querydto.TransactionManagementRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}

	end := MinInt(
		start+page.PerPage,
		totalCount,
	)

	pageCandidates := candidates[start:end]

	rows := make(
		[]querydto.TransactionManagementRowDTO,
		0,
		len(pageCandidates),
	)

	for _, candidate := range pageCandidates {
		row, err := q.buildRow(
			ctx,
			candidate,
		)
		if err != nil {
			return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
		}

		rows = append(rows, row)
	}

	return common.PageResult[querydto.TransactionManagementRowDTO]{
		Items:      rows,
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// ============================================================
// Row builder
// ============================================================

func (q *TransactionManagementQuery) buildRow(
	ctx context.Context,
	candidate transactionOrderCandidate,
) (querydto.TransactionManagementRowDTO, error) {
	order := candidate.Order

	orderCreatedAt := formatTransactionTime(
		order.CreatedAt,
	)

	row := querydto.TransactionManagementRowDTO{
		OrderID:             order.ID,
		CreatedAt:           orderCreatedAt,
		OrderCreatedAt:      orderCreatedAt,
		Paid:                order.Paid,
		OrderAmount:         candidate.OrderAmount,
		PaymentStatus:       querydto.TransactionPaymentStatusUnpaid,
		IsMultiCompanyOrder: candidate.IsMultiCompanyOrder,
	}

	payment, err := q.paymentReader.GetByPaymentID(
		ctx,
		order.ID,
	)
	if err != nil {
		if errors.Is(
			err,
			paymentdom.ErrNotFound,
		) {
			return row, nil
		}

		return querydto.TransactionManagementRowDTO{}, err
	}

	if payment == nil {
		return row, nil
	}

	row.PaymentID = payment.PaymentID
	row.PaymentStatus = string(payment.Status)
	row.StripePaymentIntentID = payment.StripePaymentIntentID
	row.PaymentCreatedAt = formatTransactionTime(
		payment.CreatedAt,
	)

	if row.PaymentCreatedAt != "" {
		row.CreatedAt = row.PaymentCreatedAt
	}

	// Payment.Amount represents the complete customer charge.
	//
	// It can be exposed directly only when every Order item belongs to the
	// current Company. For a multi-company Order, the Company-specific
	// payment amount must come from the future Stripe Connect allocation.
	if !candidate.IsMultiCompanyOrder {
		paymentAmount := payment.Amount
		row.PaymentAmount = &paymentAmount

		amountMatched := payment.Amount == candidate.OrderAmount
		row.AmountMatched = &amountMatched
	}

	return row, nil
}

// ============================================================
// Current Company Order projection
// ============================================================

// buildCurrentCompanyOrder creates an Order snapshot containing only the
// current Company's inventory-bound items.
//
// The returned Order is used only for CalculatePaymentAmount.
// It is never persisted.
//
// belongsToCompany:
//   - true when at least one Order item belongs to the current Company.
//
// isMultiCompanyOrder:
//   - true when the source Order contains an item or shipping quote item
//     outside the current Company's inventory boundary.
func buildCurrentCompanyOrder(
	order orderdom.Order,
	allowedSet map[string]struct{},
) (
	companyOrder orderdom.Order,
	belongsToCompany bool,
	isMultiCompanyOrder bool,
	err error,
) {
	if len(allowedSet) == 0 {
		return orderdom.Order{}, false, false, nil
	}

	companyItems := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(order.Items),
	)

	hasForeignItem := false

	for _, item := range order.Items {
		if InventoryAllowed(
			allowedSet,
			item.InventoryID,
		) {
			companyItems = append(companyItems, item)
			continue
		}

		hasForeignItem = true
	}

	if len(companyItems) == 0 {
		return orderdom.Order{}, false, false, nil
	}

	companyShippingItems := make(
		[]orderdom.ShippingQuoteItemSnapshot,
		0,
		len(order.ShippingQuoteSnapshot.Items),
	)

	companyShippingAmount := 0
	hasForeignShippingItem := false
	maxInt := int(^uint(0) >> 1)

	for _, item := range order.ShippingQuoteSnapshot.Items {
		if !InventoryAllowed(
			allowedSet,
			item.InventoryID,
		) {
			hasForeignShippingItem = true
			continue
		}

		if item.Amount < 0 {
			return orderdom.Order{},
				false,
				false,
				orderdom.ErrInvalidPaymentAmount
		}

		if companyShippingAmount > maxInt-item.Amount {
			return orderdom.Order{},
				false,
				false,
				orderdom.ErrInvalidPaymentAmount
		}

		companyShippingAmount += item.Amount
		companyShippingItems = append(
			companyShippingItems,
			item,
		)
	}

	if len(companyShippingItems) == 0 {
		return orderdom.Order{},
			false,
			false,
			orderdom.ErrInvalidPaymentAmount
	}

	companyOrder = order
	companyOrder.Items = companyItems
	companyOrder.ShippingQuoteSnapshot.Items = companyShippingItems
	companyOrder.ShippingQuoteSnapshot.Amount = companyShippingAmount

	isMultiCompanyOrder =
		hasForeignItem ||
			hasForeignShippingItem

	return companyOrder, true, isMultiCompanyOrder, nil
}

// ============================================================
// Helpers
// ============================================================

func emptyTransactionManagementPage(
	page common.Page,
) common.PageResult[querydto.TransactionManagementRowDTO] {
	return common.PageResult[querydto.TransactionManagementRowDTO]{
		Items:      []querydto.TransactionManagementRowDTO{},
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: 0,
		TotalPages: 0,
	}
}

func formatTransactionTime(
	value time.Time,
) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}
