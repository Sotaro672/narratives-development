// backend/internal/application/query/console/transaction_management_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Ports
// ============================================================

// TransactionSettlementReader reads seller-side Settlement records.
//
// TransactionManagementQuery uses Settlement as the source of truth for:
//
// - current Company boundary
// - Account-level Stripe Connect Transfer history
// - Transfer Reversal history
//
// One Payment may have multiple Settlement records because one Company may use
// multiple payout Accounts.
type TransactionSettlementReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]settlementdom.Settlement, error)

	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]settlementdom.Settlement, error)
}

// TransactionOrderReader reads the source Order only when an existing
// Order-specific transaction filter is supplied.
//
// Company boundary and transaction amount must never be derived from Order.
// Settlement is authoritative for those values.
type TransactionOrderReader interface {
	GetByID(
		ctx context.Context,
		orderID string,
	) (orderdom.Order, error)
}

// TransactionCompanyIDResolver resolves the authenticated Console Company.
type TransactionCompanyIDResolver func(
	ctx context.Context,
) string

// ============================================================
// Query
// ============================================================

type TransactionManagementQuery struct {
	settlementReader  TransactionSettlementReader
	orderReader       TransactionOrderReader
	companyIDResolver TransactionCompanyIDResolver
}

type NewTransactionManagementQueryParams struct {
	SettlementReader  TransactionSettlementReader
	OrderReader       TransactionOrderReader
	CompanyIDResolver TransactionCompanyIDResolver
}

func NewTransactionManagementQuery(
	p NewTransactionManagementQueryParams,
) *TransactionManagementQuery {
	return &TransactionManagementQuery{
		settlementReader:  p.SettlementReader,
		orderReader:       p.OrderReader,
		companyIDResolver: p.CompanyIDResolver,
	}
}

// ============================================================
// Internal read model
// ============================================================

type transactionSettlementCandidate struct {
	ID string

	SettlementID string

	OrderID   string
	PaymentID string

	AccountID string

	Type string

	Amount int

	Currency string

	Description string

	Status string

	StripeTransferID string

	StripeTransferReversalID string

	Timestamp time.Time
}

// ============================================================
// Public APIs
// ============================================================

func (q *TransactionManagementQuery) List(
	ctx context.Context,
	filter orderdom.Filter,
	sortSpec common.Sort,
	page common.Page,
) (common.PageResult[querydto.TransactionManagementRowDTO], error) {
	page = NormalizeCommonPage(page)

	if q == nil ||
		q.settlementReader == nil ||
		q.orderReader == nil ||
		q.companyIDResolver == nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{},
			errors.New(
				"TransactionManagementQuery.List: wiring is incomplete (settlementReader/orderReader/companyIDResolver required)",
			)
	}

	companyID := strings.TrimSpace(
		q.companyIDResolver(ctx),
	)
	if companyID == "" {
		return common.PageResult[querydto.TransactionManagementRowDTO]{},
			settlementdom.ErrInvalidCompanyID
	}

	// Settlement transaction history represents actual Stripe Connect balance
	// movements only. A paid=false query therefore cannot contain any rows.
	if filter.Paid != nil && !*filter.Paid {
		return emptyTransactionManagementPage(page), nil
	}

	normalizedSort, err := normalizeTransactionManagementSort(
		sortSpec,
	)
	if err != nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
	}

	settlements, err := q.settlementReader.ListByCompanyID(
		ctx,
		companyID,
	)
	if err != nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
	}

	if settlements == nil {
		settlements = []settlementdom.Settlement{}
	}

	orderCache := make(
		map[string]orderdom.Order,
	)

	candidates := make(
		[]transactionSettlementCandidate,
		0,
		len(settlements),
	)

	for _, settlement := range settlements {
		if settlement.CompanyID != companyID {
			return common.PageResult[querydto.TransactionManagementRowDTO]{},
				errors.New(
					"TransactionManagementQuery.List: settlement company boundary mismatch",
				)
		}

		if transactionRequiresOrderFilter(
			filter,
		) {
			order, exists :=
				orderCache[settlement.OrderID]

			if !exists {
				order, err =
					q.orderReader.GetByID(
						ctx,
						settlement.OrderID,
					)
				if err != nil {
					return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
				}

				orderCache[settlement.OrderID] =
					order
			}

			if !transactionOrderMatchesFilter(
				order,
				filter,
			) {
				continue
			}
		}

		events, err :=
			buildTransactionSettlementEvents(
				settlement,
			)
		if err != nil {
			return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
		}

		for _, event := range events {
			if !transactionTimestampMatchesCreatedFilter(
				event.Timestamp,
				filter,
			) {
				continue
			}

			candidates = append(
				candidates,
				event,
			)
		}
	}

	sortTransactionSettlementCandidates(
		candidates,
		normalizedSort,
	)

	totalCount := len(candidates)
	totalPages := TotalPages(
		totalCount,
		page.PerPage,
	)

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
		rows = append(
			rows,
			buildTransactionSettlementRow(
				candidate,
			),
		)
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
// Settlement events
// ============================================================

// buildTransactionSettlementEvents converts one Account-level Settlement into
// actual Company cash-movement events.
//
// transferred:
//
//	AMOL Stripe Platform -> Company Stripe Connected Account
//	receive / TransferAmount / TransferredAt
//
// reversed:
//
//  1. original receive event
//  2. Company Stripe Connected Account -> AMOL Stripe Platform
//     send / TransferAmount / ReversedAt
//
// pending / ready / transferring / failed / canceled Settlements are not
// transaction history because no completed Stripe balance movement exists.
func buildTransactionSettlementEvents(
	settlement settlementdom.Settlement,
) ([]transactionSettlementCandidate, error) {
	switch settlement.Status {
	case settlementdom.StatusTransferred,
		settlementdom.StatusReversed:

	default:
		return []transactionSettlementCandidate{}, nil
	}

	if strings.TrimSpace(
		settlement.ID,
	) == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement id is empty",
			)
	}

	if strings.TrimSpace(
		settlement.OrderID,
	) == "" ||
		strings.TrimSpace(
			settlement.PaymentID,
		) == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement order/payment id is empty",
			)
	}

	if strings.TrimSpace(
		settlement.AccountID,
	) == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement account id is empty",
			)
	}

	if settlement.TransferAmount <= 0 {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement transfer amount is invalid",
			)
	}

	currency := strings.ToUpper(
		strings.TrimSpace(
			settlement.Currency,
		),
	)
	if currency == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement currency is empty",
			)
	}

	stripeTransferID := strings.TrimSpace(
		settlement.StripeTransferID,
	)
	if stripeTransferID == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement Stripe Transfer id is empty",
			)
	}

	if settlement.TransferredAt == nil ||
		settlement.TransferredAt.IsZero() {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement transferredAt is invalid",
			)
	}

	receive := transactionSettlementCandidate{
		ID: settlement.ID +
			"_receive",

		SettlementID: settlement.ID,

		OrderID:   settlement.OrderID,
		PaymentID: settlement.PaymentID,

		AccountID: settlement.AccountID,

		Type: querydto.TransactionTypeReceive,

		Amount: settlement.TransferAmount,

		Currency: currency,

		Description: "売上精算",

		Status: string(
			settlementdom.StatusTransferred,
		),

		StripeTransferID: stripeTransferID,

		StripeTransferReversalID: "",

		Timestamp: settlement.TransferredAt.UTC(),
	}

	events := []transactionSettlementCandidate{
		receive,
	}

	if settlement.Status !=
		settlementdom.StatusReversed {
		return events, nil
	}

	stripeTransferReversalID :=
		strings.TrimSpace(
			settlement.StripeTransferReversalID,
		)
	if stripeTransferReversalID == "" {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement Stripe Transfer Reversal id is empty",
			)
	}

	if settlement.ReversedAt == nil ||
		settlement.ReversedAt.IsZero() {
		return nil,
			errors.New(
				"TransactionManagementQuery.List: settlement reversedAt is invalid",
			)
	}

	send := transactionSettlementCandidate{
		ID: settlement.ID +
			"_send",

		SettlementID: settlement.ID,

		OrderID:   settlement.OrderID,
		PaymentID: settlement.PaymentID,

		AccountID: settlement.AccountID,

		Type: querydto.TransactionTypeSend,

		Amount: settlement.TransferAmount,

		Currency: currency,

		Description: "売上精算取消",

		Status: string(
			settlementdom.StatusReversed,
		),

		StripeTransferID: stripeTransferID,

		StripeTransferReversalID: stripeTransferReversalID,

		Timestamp: settlement.ReversedAt.UTC(),
	}

	events = append(
		events,
		send,
	)

	return events, nil
}

// ============================================================
// Row builder
// ============================================================

func buildTransactionSettlementRow(
	candidate transactionSettlementCandidate,
) querydto.TransactionManagementRowDTO {
	return querydto.TransactionManagementRowDTO{
		ID: candidate.ID,

		SettlementID: candidate.SettlementID,

		OrderID: candidate.OrderID,

		PaymentID: candidate.PaymentID,

		AccountID: candidate.AccountID,

		Type: candidate.Type,

		Amount: candidate.Amount,

		Currency: candidate.Currency,

		Description: candidate.Description,

		Status: candidate.Status,

		StripeTransferID: candidate.StripeTransferID,

		StripeTransferReversalID: candidate.StripeTransferReversalID,

		Timestamp: formatTransactionTime(
			candidate.Timestamp,
		),
	}
}

// ============================================================
// Filter
// ============================================================

// transactionTimestampMatchesCreatedFilter applies the existing transaction
// createdAt filter to the actual transaction event timestamp.
//
// receive uses Settlement.TransferredAt.
// send uses Settlement.ReversedAt.
func transactionTimestampMatchesCreatedFilter(
	timestamp time.Time,
	filter orderdom.Filter,
) bool {
	if filter.CreatedFrom != nil &&
		timestamp.Before(
			filter.CreatedFrom.UTC(),
		) {
		return false
	}

	if filter.CreatedTo != nil &&
		timestamp.After(
			filter.CreatedTo.UTC(),
		) {
		return false
	}

	return true
}

// transactionRequiresOrderFilter reports whether the existing /transactions
// query contains a filter that requires loading the source Order.
//
// paid and createdAt are handled directly from Settlement transaction events.
func transactionRequiresOrderFilter(
	filter orderdom.Filter,
) bool {
	return filter.ID != "" ||
		filter.UserID != "" ||
		filter.AvatarID != "" ||
		filter.CartID != "" ||
		filter.ShippingSnapshot != nil ||
		transactionHasItemFilter(
			filter,
		)
}

// transactionOrderMatchesFilter preserves the existing Order-specific filter
// behavior.
//
// Order is used only for filtering. It is not used to calculate transaction
// amount, direction, Account ownership, or Company ownership.
func transactionOrderMatchesFilter(
	order orderdom.Order,
	filter orderdom.Filter,
) bool {
	if filter.ID != "" &&
		order.ID != filter.ID {
		return false
	}

	if filter.UserID != "" &&
		order.UserID != filter.UserID {
		return false
	}

	if filter.AvatarID != "" &&
		order.AvatarID != filter.AvatarID {
		return false
	}

	if filter.CartID != "" &&
		order.CartID != filter.CartID {
		return false
	}

	if filter.Paid != nil &&
		!*filter.Paid {
		return false
	}

	if filter.ShippingSnapshot != nil &&
		!transactionShippingSnapshotEqual(
			order.ShippingSnapshot,
			*filter.ShippingSnapshot,
		) {
		return false
	}

	if !transactionHasItemFilter(filter) {
		return true
	}

	for _, item := range order.Items {
		if transactionOrderItemMatchesFilter(
			item,
			filter,
		) {
			return true
		}
	}

	return false
}

func transactionHasItemFilter(
	filter orderdom.Filter,
) bool {
	return filter.ModelID != "" ||
		filter.InventoryID != "" ||
		filter.ListID != "" ||
		filter.ItemType != "" ||
		filter.ResaleID != "" ||
		filter.ProductID != "" ||
		filter.ProductBlueprintID != "" ||
		filter.TokenBlueprintID != "" ||
		filter.BrandID != "" ||
		filter.IsCancelled != nil ||
		filter.IsDispatched != nil ||
		filter.Transferred != nil
}

func transactionOrderItemMatchesFilter(
	item orderdom.OrderItemSnapshot,
	filter orderdom.Filter,
) bool {
	if filter.ModelID != "" &&
		item.ModelID != filter.ModelID {
		return false
	}

	if filter.InventoryID != "" &&
		item.InventoryID != filter.InventoryID {
		return false
	}

	if filter.ListID != "" &&
		item.ListID != filter.ListID {
		return false
	}

	if filter.ItemType != "" &&
		item.Type != filter.ItemType {
		return false
	}

	if filter.ResaleID != "" &&
		item.ResaleID != filter.ResaleID {
		return false
	}

	if filter.ProductID != "" &&
		item.ProductID != filter.ProductID {
		return false
	}

	if filter.ProductBlueprintID != "" &&
		item.ProductBlueprintID !=
			filter.ProductBlueprintID {
		return false
	}

	if filter.TokenBlueprintID != "" &&
		item.TokenBlueprintID !=
			filter.TokenBlueprintID {
		return false
	}

	if filter.BrandID != "" &&
		item.BrandID != filter.BrandID {
		return false
	}

	if filter.IsCancelled != nil &&
		item.IsCancelled !=
			*filter.IsCancelled {
		return false
	}

	if filter.IsDispatched != nil &&
		item.IsDispatched !=
			*filter.IsDispatched {
		return false
	}

	if filter.Transferred != nil &&
		item.Transferred !=
			*filter.Transferred {
		return false
	}

	return true
}

func transactionShippingSnapshotEqual(
	left orderdom.ShippingSnapshot,
	right orderdom.ShippingSnapshot,
) bool {
	return left.ZipCode == right.ZipCode &&
		left.State == right.State &&
		left.City == right.City &&
		left.Street == right.Street &&
		left.Street2 == right.Street2 &&
		left.Country == right.Country
}

// ============================================================
// Sort
// ============================================================

func normalizeTransactionManagementSort(
	sortSpec common.Sort,
) (common.Sort, error) {
	if sortSpec.Column == "" {
		sortSpec.Column =
			orderdom.SortByCreatedAt
	}

	if sortSpec.Column !=
		orderdom.SortByCreatedAt {
		return common.Sort{},
			errors.New(
				"TransactionManagementQuery.List: unsupported sort column",
			)
	}

	if sortSpec.Order == "" {
		sortSpec.Order =
			common.SortDesc
	}

	if sortSpec.Order != common.SortAsc &&
		sortSpec.Order != common.SortDesc {
		return common.Sort{},
			errors.New(
				"TransactionManagementQuery.List: invalid sort order",
			)
	}

	return sortSpec, nil
}

func sortTransactionSettlementCandidates(
	candidates []transactionSettlementCandidate,
	sortSpec common.Sort,
) {
	sort.SliceStable(
		candidates,
		func(i, j int) bool {
			left :=
				candidates[i].
					Timestamp

			right :=
				candidates[j].
					Timestamp

			if left.Equal(right) {
				if sortSpec.Order ==
					common.SortAsc {
					return candidates[i].ID <
						candidates[j].ID
				}

				return candidates[i].ID >
					candidates[j].ID
			}

			if sortSpec.Order ==
				common.SortAsc {
				return left.Before(right)
			}

			return left.After(right)
		},
	)
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
