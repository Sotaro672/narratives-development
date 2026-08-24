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
	paymentdom "narratives/internal/domain/payment"
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
// - Company-attributable gross amount
// - Stripe PaymentIntent relationship
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

// TransactionOrderReader reads the source Order only for display metadata and
// existing Order filter compatibility.
//
// Company boundary and transaction amount must never be derived from Order
// inventory ownership here. Settlement is authoritative for those values.
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

type transactionSettlementGroupKey struct {
	OrderID   string
	PaymentID string
}

type transactionSettlementCandidate struct {
	Order orderdom.Order

	OrderID   string
	PaymentID string

	StripePaymentIntentID string

	GrossAmount int

	SettlementCreatedAt time.Time

	IsMultiCompanyOrder bool
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

	// Settlement exists only after a succeeded Payment.
	// Therefore a paid=false query cannot contain any Settlement transaction.
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

	groups, err := buildTransactionSettlementGroups(
		settlements,
		companyID,
	)
	if err != nil {
		return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
	}

	if len(groups) == 0 {
		return emptyTransactionManagementPage(page), nil
	}

	orderCache := make(
		map[string]orderdom.Order,
		len(groups),
	)

	multiCompanyCache := make(
		map[string]bool,
		len(groups),
	)

	candidates := make(
		[]transactionSettlementCandidate,
		0,
		len(groups),
	)

	for _, group := range groups {
		if !transactionSettlementMatchesCreatedFilter(
			group.SettlementCreatedAt,
			filter,
		) {
			continue
		}

		order, exists := orderCache[group.OrderID]
		if !exists {
			order, err = q.orderReader.GetByID(
				ctx,
				group.OrderID,
			)
			if err != nil {
				return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
			}

			orderCache[group.OrderID] = order
		}

		if !transactionOrderMatchesFilter(
			order,
			filter,
		) {
			continue
		}

		isMultiCompanyOrder, exists :=
			multiCompanyCache[group.PaymentID]

		if !exists {
			isMultiCompanyOrder, err =
				q.isMultiCompanyPayment(
					ctx,
					group.PaymentID,
					companyID,
				)
			if err != nil {
				return common.PageResult[querydto.TransactionManagementRowDTO]{}, err
			}

			multiCompanyCache[group.PaymentID] =
				isMultiCompanyOrder
		}

		candidates = append(
			candidates,
			transactionSettlementCandidate{
				Order: order,

				OrderID:   group.OrderID,
				PaymentID: group.PaymentID,

				StripePaymentIntentID: group.StripePaymentIntentID,

				GrossAmount: group.GrossAmount,

				SettlementCreatedAt: group.SettlementCreatedAt,

				IsMultiCompanyOrder: isMultiCompanyOrder,
			},
		)
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
// Settlement grouping
// ============================================================

// buildTransactionSettlementGroups groups Account-level Settlements into one
// Company-level transaction row.
//
// Example:
//
// Payment P1
//
//	Company A
//	  Account A1 -> Settlement 1 Gross=5,000
//	  Account A2 -> Settlement 2 Gross=3,000
//
// TransactionManagement:
//
//	Company A / Payment P1 / Gross=8,000
//
// Therefore Console does not accidentally display duplicate Order rows merely
// because one Company has multiple payout Accounts.
func buildTransactionSettlementGroups(
	settlements []settlementdom.Settlement,
	companyID string,
) ([]transactionSettlementCandidate, error) {
	groupMap := make(
		map[transactionSettlementGroupKey]*transactionSettlementCandidate,
	)

	maxInt := int(^uint(0) >> 1)

	for _, settlement := range settlements {
		if settlement.CompanyID != companyID {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement company boundary mismatch",
				)
		}

		if settlement.OrderID == "" ||
			settlement.PaymentID == "" {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement order/payment id is empty",
				)
		}

		if settlement.GrossAmount <= 0 {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement gross amount is invalid",
				)
		}

		if settlement.CreatedAt.IsZero() {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement createdAt is invalid",
				)
		}

		key := transactionSettlementGroupKey{
			OrderID:   settlement.OrderID,
			PaymentID: settlement.PaymentID,
		}

		group, exists := groupMap[key]
		if !exists {
			groupMap[key] =
				&transactionSettlementCandidate{
					OrderID:   settlement.OrderID,
					PaymentID: settlement.PaymentID,

					StripePaymentIntentID: settlement.StripePaymentIntentID,

					GrossAmount: settlement.GrossAmount,

					SettlementCreatedAt: settlement.CreatedAt.UTC(),
				}

			continue
		}

		if group.StripePaymentIntentID !=
			settlement.StripePaymentIntentID {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement Stripe PaymentIntent mismatch",
				)
		}

		if group.GrossAmount >
			maxInt-settlement.GrossAmount {
			return nil,
				errors.New(
					"TransactionManagementQuery.List: settlement gross amount overflow",
				)
		}

		group.GrossAmount +=
			settlement.GrossAmount

		if settlement.CreatedAt.Before(
			group.SettlementCreatedAt,
		) {
			group.SettlementCreatedAt =
				settlement.CreatedAt.UTC()
		}
	}

	result := make(
		[]transactionSettlementCandidate,
		0,
		len(groupMap),
	)

	for _, group := range groupMap {
		result = append(
			result,
			*group,
		)
	}

	return result, nil
}

// ============================================================
// Multi-company detection
// ============================================================

// isMultiCompanyPayment determines multi-company state from Settlements.
//
// This intentionally does not inspect inventory ownership.
//
// If one Payment has Settlements belonging to more than one Company, the
// customer paid for sellers from multiple Companies in the same checkout.
func (q *TransactionManagementQuery) isMultiCompanyPayment(
	ctx context.Context,
	paymentID string,
	currentCompanyID string,
) (bool, error) {
	settlements, err :=
		q.settlementReader.ListByPaymentID(
			ctx,
			paymentID,
		)
	if err != nil {
		return false, err
	}

	for _, settlement := range settlements {
		if settlement.CompanyID != "" &&
			settlement.CompanyID != currentCompanyID {
			return true, nil
		}
	}

	return false, nil
}

// ============================================================
// Row builder
// ============================================================

func buildTransactionSettlementRow(
	candidate transactionSettlementCandidate,
) querydto.TransactionManagementRowDTO {
	settlementCreatedAt :=
		formatTransactionTime(
			candidate.SettlementCreatedAt,
		)

	orderCreatedAt :=
		formatTransactionTime(
			candidate.Order.CreatedAt,
		)

	companyGrossAmount :=
		candidate.GrossAmount

	// AmountMatched now means that the amount displayed as the current
	// Company's transaction amount is backed directly by persisted
	// Settlement.GrossAmount rather than reconstructed from Order snapshots.
	amountMatched := true

	return querydto.TransactionManagementRowDTO{
		OrderID: candidate.OrderID,

		PaymentID: candidate.PaymentID,

		CreatedAt: settlementCreatedAt,

		OrderCreatedAt: orderCreatedAt,

		Paid: true,

		// Existing DTO field names are retained for frontend compatibility.
		//
		// Both values now represent the amount attributable to the current
		// Company according to persisted Settlement records.
		OrderAmount: companyGrossAmount,

		PaymentAmount: &companyGrossAmount,

		// Settlement is created only for a succeeded Payment.
		PaymentStatus: string(paymentdom.StatusSucceeded),

		StripePaymentIntentID: candidate.StripePaymentIntentID,

		IsMultiCompanyOrder: candidate.IsMultiCompanyOrder,

		AmountMatched: &amountMatched,
	}
}

// ============================================================
// Filter
// ============================================================

// transactionSettlementMatchesCreatedFilter applies the existing transaction
// createdAt filter to Settlement.CreatedAt.
//
// TransactionManagement is now Settlement-based, therefore the primary
// transaction timestamp is the Settlement creation time rather than the Order
// creation time.
func transactionSettlementMatchesCreatedFilter(
	createdAt time.Time,
	filter orderdom.Filter,
) bool {
	if filter.CreatedFrom != nil &&
		createdAt.Before(
			filter.CreatedFrom.UTC(),
		) {
		return false
	}

	if filter.CreatedTo != nil &&
		createdAt.After(
			filter.CreatedTo.UTC(),
		) {
		return false
	}

	return true
}

// transactionOrderMatchesFilter preserves the existing Order-specific filter
// behavior.
//
// Order is used only for filtering/display metadata. It is not used to
// calculate transaction amount or Company ownership.
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
					SettlementCreatedAt

			right :=
				candidates[j].
					SettlementCreatedAt

			if left.Equal(right) {
				if candidates[i].PaymentID ==
					candidates[j].PaymentID {
					return candidates[i].OrderID <
						candidates[j].OrderID
				}

				if sortSpec.Order ==
					common.SortAsc {
					return candidates[i].PaymentID <
						candidates[j].PaymentID
				}

				return candidates[i].PaymentID >
					candidates[j].PaymentID
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
