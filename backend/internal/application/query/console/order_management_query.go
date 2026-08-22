// backend/internal/application/query/console/order_management_query.go
package query

//
// 機能: OrderManagementQuery (console)
//   - currentCompany 境界（inventory_query 相当）で許可された inventoryId のみを対象に
//     Order.Items[].InventoryID をフラットに列挙する
//   - order lister の ListByInventoryIDs を使い、orders を取得する
//   - allowed items を集約してから item 単位で再ページングする
//
// 目的:
// - order テーブルの items に記載された inventoryId を、company 境界に従って安全に一覧できるようにする
//
// ✅ DI整合のための方針:
//   - Query側の port は domain/order.Filter / common.Sort / common.Page を引数に取る。
//   - currentCompany 境界のため、OrderLister は List ではなく ListByInventoryIDs を要求する。
//   - company-bound inventory filtering は OrderManagementQuery 側で item 単位に適用する。
//
// ✅ 重要:
//   - productName/category/categoryFields は best-effort。
//     productBlueprintResolver が DI されていれば productBlueprintId->ProductBlueprint を1回だけ取得し、
//     productName と productBlueprintCategoryPath/categoryFields を埋める。
//     なければ空で返す（500にしない）。
//   - tokenName は best-effort。
//     tbName が DI されていれば埋める、されていなければ空で返す（500にしない）。
//   - listReadableId も best-effort。
//     listReadable が DI されていれば listId->readableId を引いて埋める。なければ空で返す（500にしない）。
//   - userName も best-effort。
//     userNameResolver が DI されていれば userId->user を引いて userName を埋める。
//     なければ空で返す（500にしない）。
//   - model fields も best-effort。
//     modelResolver が DI されていれば modelId(variationID)->apparel/alcohol 表示情報を埋める。なければ空で返す（500にしない）。
//
import (
	"context"
	"errors"
	"strconv"
	"time"

	applicationport "narratives/internal/application/port"
	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	common "narratives/internal/domain/common"
	invdom "narratives/internal/domain/inventory"
	orderdom "narratives/internal/domain/order"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// Ports (read-only)
// ============================================================

// OrderLister lists orders for console query processing.
//
// NOTE:
// Company-bound inventory filtering is applied by OrderManagementQuery at item level.
type OrderLister interface {
	ListByInventoryIDs(
		ctx context.Context,
		allowedInventoryIDs map[string]struct{},
		filter orderdom.Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[orderdom.Order], error)
}

type InventoryRowsLister interface {
	ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error)
}

type OrderManagementUserNameResolver interface {
	ResolveUserName(ctx context.Context, userID string) string
}

// ============================================================
// DTO
// ============================================================

// OrderItemInventoryRowDTO is a flattened order item row for console UI.
type OrderItemInventoryRowDTO struct {
	OrderID string `json:"orderId"`

	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	UserName string `json:"userName,omitempty"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	InventoryID string `json:"inventoryId"`

	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	ListReadableID string `json:"listReadableId,omitempty"`

	ProductBlueprintCategoryPath []string       `json:"productBlueprintCategoryPath,omitempty"`
	CategoryFields               map[string]any `json:"categoryFields,omitempty"`

	ModelID string `json:"modelId,omitempty"`

	Kind        string `json:"kind,omitempty"`
	Size        string `json:"size,omitempty"`
	Color       string `json:"color,omitempty"`
	RGB         string `json:"rgb,omitempty"`
	ModelNumber string `json:"modelNumber,omitempty"`

	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	Qty   int `json:"qty,omitempty"`
	Price int `json:"price,omitempty"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"` // RFC3339(UTC)
}

// ============================================================
// Query
// ============================================================

type OrderManagementQuery struct {
	lister       OrderLister
	invRows      InventoryRowsLister        // REQUIRED
	invBlueprint InventoryBlueprintResolver // REQUIRED

	productBlueprint applicationport.ProductBlueprintGetter
	tbName           applicationport.TokenBlueprintGetter
	listReadable     ListReadableIDReader
	userNameResolver OrderManagementUserNameResolver
	modelResolver    ModelResolver
}

type NewOrderManagementQueryParams struct {
	Lister       OrderLister
	InvRows      InventoryRowsLister        // REQUIRED
	InvBlueprint InventoryBlueprintResolver // REQUIRED

	ProductBlueprint applicationport.ProductBlueprintGetter
	TBName           applicationport.TokenBlueprintGetter
	ListReadable     ListReadableIDReader
	UserName         OrderManagementUserNameResolver
	ModelResolver    ModelResolver
}

func NewOrderManagementQuery(p NewOrderManagementQueryParams) *OrderManagementQuery {
	return &OrderManagementQuery{
		lister:           p.Lister,
		invRows:          p.InvRows,
		invBlueprint:     p.InvBlueprint,
		productBlueprint: p.ProductBlueprint,
		tbName:           p.TBName,
		listReadable:     p.ListReadable,
		userNameResolver: p.UserName,
		modelResolver:    p.ModelResolver,
	}
}

// ============================================================
// Public APIs
// ============================================================

func (q *OrderManagementQuery) AllowedInventoryIDSet(
	ctx context.Context,
) (map[string]struct{}, error) {
	if q == nil || q.invRows == nil {
		return nil, errors.New(
			"OrderManagementQuery.AllowedInventoryIDSet: invRows is required",
		)
	}

	return AllowedInventoryIDSetFromContext(
		ctx,
		q.invRows,
	)
}

func (q *OrderManagementQuery) ListItemInventoryRows(
	ctx context.Context,
	filter orderdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[OrderItemInventoryRowDTO], error) {
	page = NormalizeCommonPage(page)

	if q == nil || q.lister == nil || q.invRows == nil || q.invBlueprint == nil {
		return common.PageResult[OrderItemInventoryRowDTO]{}, errors.New("OrderManagementQuery.ListItemInventoryRows: wiring is incomplete (lister/invRows/invBlueprint required)")
	}

	allowedSet, err := AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		return common.PageResult[OrderItemInventoryRowDTO]{}, err
	}
	if len(allowedSet) == 0 {
		return common.PageResult[OrderItemInventoryRowDTO]{
			Items:      []OrderItemInventoryRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	allowedAll := make([]OrderItemInventoryRowDTO, 0, page.PerPage)

	type bt struct {
		pb string
		tb string
	}
	blueprintCache := map[string]bt{}

	productBlueprintCache := map[string]pbdom.ProductBlueprint{}
	tbNameCache := map[string]string{}
	listReadableCache := map[string]string{}
	userNameCache := map[string]string{}
	modelCache := map[string]resolver.ModelResolved{}

	resolveBlueprint := func(invID string) (string, string, error) {
		if invID == "" {
			return "", "", invdom.ErrInvalidMintID
		}

		if v, ok := blueprintCache[invID]; ok {
			return v.pb, v.tb, nil
		}

		pbID, tbID, e := q.invBlueprint.ResolveBlueprintIDsByInventoryID(ctx, invID)
		if e != nil {
			return "", "", e
		}

		blueprintCache[invID] = bt{pb: pbID, tb: tbID}
		return pbID, tbID, nil
	}

	resolveProductBlueprint := func(pbID string) (pbdom.ProductBlueprint, error) {
		if q.productBlueprint == nil || pbID == "" {
			return pbdom.ProductBlueprint{}, nil
		}

		if v, ok := productBlueprintCache[pbID]; ok {
			return v, nil
		}

		pb, e := q.productBlueprint.GetByID(ctx, pbID)
		if e != nil {
			return pbdom.ProductBlueprint{}, e
		}

		productBlueprintCache[pbID] = pb
		return pb, nil
	}

	resolveTokenName := func(tbID string) (string, error) {
		if q.tbName == nil || tbID == "" {
			return "", nil
		}

		if v, ok := tbNameCache[tbID]; ok {
			return v, nil
		}

		tb, e := q.tbName.GetByID(ctx, tbID)
		if e != nil {
			return "", e
		}
		if tb == nil {
			tbNameCache[tbID] = ""
			return "", nil
		}

		name := tb.Name
		tbNameCache[tbID] = name
		return name, nil
	}

	resolveListReadableID := func(listID string) (string, error) {
		if q.listReadable == nil || listID == "" {
			return "", nil
		}

		if v, ok := listReadableCache[listID]; ok {
			return v, nil
		}

		readable, e := q.listReadable.GetReadableIDByID(ctx, listID)
		if e != nil {
			return "", e
		}

		listReadableCache[listID] = readable
		return readable, nil
	}

	resolveUserName := func(userID string) string {
		if q.userNameResolver == nil || userID == "" {
			return ""
		}

		if v, ok := userNameCache[userID]; ok {
			return v
		}

		name := q.userNameResolver.ResolveUserName(
			ctx,
			userID,
		)

		userNameCache[userID] = name
		return name
	}

	resolveModel := func(modelID string) resolver.ModelResolved {
		if q.modelResolver == nil || modelID == "" {
			return resolver.ModelResolved{}
		}

		if v, ok := modelCache[modelID]; ok {
			return v
		}

		resolved := q.modelResolver.ResolveModelResolved(ctx, modelID)
		modelCache[modelID] = resolved
		return resolved
	}

	const maxScanPages = 500
	srcPage := 1

	for {
		if srcPage > maxScanPages {
			break
		}

		pr, e := q.lister.ListByInventoryIDs(
			ctx,
			allowedSet,
			filter,
			sort,
			common.Page{Number: srcPage, PerPage: page.PerPage},
		)
		if e != nil {
			return common.PageResult[OrderItemInventoryRowDTO]{}, e
		}
		if pr.Items == nil {
			pr.Items = []orderdom.Order{}
		}

		for _, ord := range pr.Items {
			orderID := NonEmpty(ord.ID, "(missing order id)")

			createdAt := ""
			if !ord.CreatedAt.IsZero() {
				createdAt = ord.CreatedAt.UTC().Format(time.RFC3339)
			}

			userID := ord.UserID
			avatarID := ord.AvatarID
			cartID := ord.CartID

			userName := resolveUserName(userID)

			for _, it := range ord.Items {
				invID := it.InventoryID
				if !InventoryAllowed(allowedSet, invID) {
					continue
				}

				pbID, tbID, e2 := resolveBlueprint(invID)
				if e2 != nil {
					return common.PageResult[OrderItemInventoryRowDTO]{}, e2
				}

				productName := ""

				var productBlueprintCategoryPath []string
				var categoryFields map[string]any

				if pbID != "" {
					pb, ePB := resolveProductBlueprint(pbID)
					if ePB != nil {
						return common.PageResult[OrderItemInventoryRowDTO]{}, ePB
					}

					productName = pb.ProductName

					if len(pb.ProductBlueprintCategoryPath) > 0 {
						productBlueprintCategoryPath = append(
							[]string(nil),
							pb.ProductBlueprintCategoryPath...,
						)
					}

					if len(pb.CategoryFields) > 0 {
						categoryFields = make(map[string]any, len(pb.CategoryFields))
						for k, v := range pb.CategoryFields {
							if k == "" {
								continue
							}
							categoryFields[k] = v
						}
					}
				}

				tokenName := ""
				if tbID != "" {
					n, e4 := resolveTokenName(tbID)
					if e4 != nil {
						return common.PageResult[OrderItemInventoryRowDTO]{}, e4
					}
					tokenName = n
				}

				listReadableID := ""
				if it.ListID != "" {
					n, e5 := resolveListReadableID(it.ListID)
					if e5 == nil {
						listReadableID = n
					}
				}

				kind := ""
				size := ""
				color := ""
				rgb := ""
				modelNumber := ""
				var volumeValue *int
				volumeUnit := ""

				if it.ModelID != "" {
					mr := resolveModel(it.ModelID)

					kind = mr.Kind
					modelNumber = mr.ModelNumber

					if mr.Kind == "apparel" {
						size = mr.Size
						color = mr.Color

						if mr.RGB != nil {
							rgb = strconv.Itoa(*mr.RGB)
						}
					}

					if mr.Kind == "alcohol" {
						volumeValue = mr.VolumeValue
						volumeUnit = mr.VolumeUnit
					}
				}

				transferredAt := ""
				if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
					transferredAt = it.TransferredAt.UTC().Format(time.RFC3339)
				}

				allowedAll = append(allowedAll, OrderItemInventoryRowDTO{
					OrderID: orderID,

					UserID:   userID,
					AvatarID: avatarID,
					CartID:   cartID,

					UserName: userName,

					Paid:      ord.Paid,
					CreatedAt: createdAt,

					InventoryID:        invID,
					ProductBlueprintID: pbID,
					TokenBlueprintID:   tbID,
					ProductName:        productName,
					TokenName:          tokenName,

					ListReadableID: listReadableID,

					ProductBlueprintCategoryPath: productBlueprintCategoryPath,
					CategoryFields:               categoryFields,

					ModelID: it.ModelID,

					Kind:        kind,
					Size:        size,
					Color:       color,
					RGB:         rgb,
					ModelNumber: modelNumber,

					VolumeValue: volumeValue,
					VolumeUnit:  volumeUnit,

					Qty:   it.Qty,
					Price: it.Price,

					IsCancelled:  it.IsCancelled,
					IsDispatched: it.IsDispatched,

					Transferred:   it.Transferred,
					TransferredAt: transferredAt,
				})
			}
		}

		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else if len(pr.Items) < page.PerPage {
			break
		}

		srcPage++
	}

	totalCount := len(allowedAll)
	tp := TotalPages(totalCount, page.PerPage)

	start := (page.Number - 1) * page.PerPage
	if start < 0 {
		start = 0
	}
	if start >= totalCount {
		return common.PageResult[OrderItemInventoryRowDTO]{
			Items:      []OrderItemInventoryRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: totalCount,
			TotalPages: tp,
		}, nil
	}

	end := MinInt(start+page.PerPage, totalCount)

	return common.PageResult[OrderItemInventoryRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

func (q *OrderManagementQuery) CountUndispatchedOrders(
	ctx context.Context,
) (int, error) {
	if q == nil || q.lister == nil || q.invRows == nil {
		return 0, errors.New("OrderManagementQuery.CountUndispatchedOrders: wiring is incomplete (lister/invRows required)")
	}

	allowedSet, err := AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		return 0, err
	}
	if len(allowedSet) == 0 {
		return 0, nil
	}

	orderIDs := make(map[string]struct{})

	const (
		maxScanPages = 500
		perPage      = 200
	)

	srcPage := 1

	for {
		if srcPage > maxScanPages {
			break
		}

		pr, err := q.lister.ListByInventoryIDs(
			ctx,
			allowedSet,
			orderdom.Filter{},
			common.Sort{},
			common.Page{
				Number:  srcPage,
				PerPage: perPage,
			},
		)
		if err != nil {
			return 0, err
		}
		if pr.Items == nil {
			pr.Items = []orderdom.Order{}
		}

		for _, ord := range pr.Items {
			if ord.ID == "" {
				continue
			}

			for _, item := range ord.Items {
				if !InventoryAllowed(
					allowedSet,
					item.InventoryID,
				) {
					continue
				}

				if item.IsCancelled {
					continue
				}

				if item.IsDispatched {
					continue
				}

				orderIDs[ord.ID] = struct{}{}
				break
			}
		}

		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else if len(pr.Items) < perPage {
			break
		}

		srcPage++
	}

	return len(orderIDs), nil
}
