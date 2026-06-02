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
//   - productName/tokenName は best-effort。
//     pbName/tbName が DI されていれば埋める、されていなければ空で返す（500にしない）。
//   - listReadableId も best-effort。
//     listReadable が DI されていれば listId->readableId を引いて埋める。なければ空で返す（500にしない）。
//   - avatarName も best-effort。
//     avatarNameResolver が DI されていれば avatarId->avatar を引いて avatarName を埋める。
//     なければ空で返す（500にしない）。
//   - model fields も best-effort。
//     modelResolver が DI されていれば modelId(variationID)->apparel/alcohol 表示情報を埋める。なければ空で返す（500にしない）。
//   - category/categoryFields も best-effort。
//     productBlueprintResolver が DI されていれば productBlueprintId->category snapshot/categoryFields を埋める。
//     なければ空で返す（500にしない）。
//
import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	avatardom "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
	invdom "narratives/internal/domain/inventory"
	orderdom "narratives/internal/domain/order"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
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

// InventoryBlueprintResolver resolves productBlueprintId/tokenBlueprintId from inventoryId.
type InventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(ctx context.Context, inventoryID string) (productBlueprintID string, tokenBlueprintID string, err error)
}

// ProductBlueprintNameResolver resolves productName from productBlueprintId.
type ProductBlueprintNameResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// ProductBlueprintResolver resolves category snapshot/categoryFields from productBlueprintId.
type ProductBlueprintResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// TokenBlueprintNameResolver resolves tokenName from tokenBlueprintId.
type TokenBlueprintNameResolver interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
}

// ListReadableIDResolver resolves listId to readableId.
type ListReadableIDResolver interface {
	GetReadableIDByID(ctx context.Context, id string) (string, error)
}

// AvatarNameResolver resolves avatar from avatarId.
type AvatarNameResolver interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// ModelResolver resolves modelId(variationID) to display fields.
type ModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) resolver.ModelResolved
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

	AvatarName string `json:"avatarName,omitempty"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	InventoryID string `json:"inventoryId"`

	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	ListReadableID string `json:"listReadableId,omitempty"`

	CategoryID     string         `json:"categoryId,omitempty"`
	CategoryCode   string         `json:"categoryCode,omitempty"`
	CategoryNameJa string         `json:"categoryNameJa,omitempty"`
	CategoryNameEn string         `json:"categoryNameEn,omitempty"`
	CategoryKind   string         `json:"categoryKind,omitempty"`
	CategoryPath   []string       `json:"categoryPath,omitempty"`
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

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

	pbName             ProductBlueprintNameResolver
	productBlueprint   ProductBlueprintResolver
	tbName             TokenBlueprintNameResolver
	listReadable       ListReadableIDResolver
	avatarNameResolver AvatarNameResolver
	modelResolver      ModelResolver
}

type NewOrderManagementQueryParams struct {
	Lister       OrderLister
	InvRows      InventoryRowsLister        // REQUIRED
	InvBlueprint InventoryBlueprintResolver // REQUIRED

	PBName           ProductBlueprintNameResolver
	ProductBlueprint ProductBlueprintResolver
	TBName           TokenBlueprintNameResolver
	ListReadable     ListReadableIDResolver
	AvatarName       AvatarNameResolver
	ModelResolver    ModelResolver
}

func NewOrderManagementQuery(p NewOrderManagementQueryParams) *OrderManagementQuery {
	return &OrderManagementQuery{
		lister:             p.Lister,
		invRows:            p.InvRows,
		invBlueprint:       p.InvBlueprint,
		pbName:             p.PBName,
		productBlueprint:   p.ProductBlueprint,
		tbName:             p.TBName,
		listReadable:       p.ListReadable,
		avatarNameResolver: p.AvatarName,
		modelResolver:      p.ModelResolver,
	}
}

// ============================================================
// Public APIs
// ============================================================

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

	log.Printf("[OrderManagementQuery] DEBUG listReadable resolver type=%T", q.listReadable)

	allowedSet, err := AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[OrderManagementQuery] ERROR company boundary (inventory_query) failed: %v", err)
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

	pbNameCache := map[string]string{}
	productBlueprintCache := map[string]pbdom.ProductBlueprint{}
	tbNameCache := map[string]string{}
	listReadableCache := map[string]string{}
	avatarNameCache := map[string]string{}
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

	resolveProductName := func(pbID string) (string, error) {
		if q.pbName == nil || pbID == "" {
			return "", nil
		}
		if v, ok := pbNameCache[pbID]; ok {
			return v, nil
		}

		pb, e := q.pbName.GetByID(ctx, pbID)
		if e != nil {
			return "", e
		}

		name := pb.ProductName
		pbNameCache[pbID] = name
		return name, nil
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

	resolveAvatarName := func(avatarID string) (string, error) {
		if q.avatarNameResolver == nil || avatarID == "" {
			return "", nil
		}
		if v, ok := avatarNameCache[avatarID]; ok {
			return v, nil
		}

		avatar, e := q.avatarNameResolver.GetByID(ctx, avatarID)
		if e != nil {
			return "", e
		}

		name := avatar.AvatarName
		avatarNameCache[avatarID] = name
		return name, nil
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
			log.Printf("[OrderManagementQuery] WARN scan page limit reached (max=%d). results may be truncated.", maxScanPages)
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
			log.Printf("[OrderManagementQuery] ERROR lister.ListByInventoryIDs failed (scan page=%d): %v", srcPage, e)
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

			avatarName := ""
			if avatarID != "" {
				n, e0 := resolveAvatarName(avatarID)
				if e0 != nil {
					log.Printf("[OrderManagementQuery] ERROR Avatar.GetByID failed avatarId=%q err=%v", avatarID, e0)
					return common.PageResult[OrderItemInventoryRowDTO]{}, e0
				}
				avatarName = n
			}

			for _, it := range ord.Items {
				invID := it.InventoryID
				if !InventoryAllowed(allowedSet, invID) {
					continue
				}

				pbID, tbID, e2 := resolveBlueprint(invID)
				if e2 != nil {
					log.Printf("[OrderManagementQuery] ERROR ResolveBlueprintIDsByInventoryID failed inventoryId=%q err=%v", invID, e2)
					return common.PageResult[OrderItemInventoryRowDTO]{}, e2
				}

				categoryID := ""
				categoryCode := ""
				categoryNameJa := ""
				categoryNameEn := ""
				categoryKind := ""
				var categoryPath []string
				var categoryFields map[string]any

				if pbID != "" {
					pb, ePB := resolveProductBlueprint(pbID)
					if ePB != nil {
						log.Printf("[OrderManagementQuery] ERROR ProductBlueprint.GetByID failed productBlueprintId=%q err=%v", pbID, ePB)
						return common.PageResult[OrderItemInventoryRowDTO]{}, ePB
					}

					categoryID = pb.ProductBlueprintCategory.ID
					categoryCode = pb.ProductBlueprintCategory.Code
					categoryNameJa = pb.ProductBlueprintCategory.NameJa
					categoryNameEn = pb.ProductBlueprintCategory.NameEn
					categoryKind = string(pb.ProductBlueprintCategory.Kind)

					if len(pb.ProductBlueprintCategory.Path) > 0 {
						categoryPath = append([]string(nil), pb.ProductBlueprintCategory.Path...)
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

				productName := ""
				if pbID != "" {
					n, e3 := resolveProductName(pbID)
					if e3 != nil {
						log.Printf("[OrderManagementQuery] ERROR ProductBlueprint.GetByID failed productBlueprintId=%q err=%v", pbID, e3)
						return common.PageResult[OrderItemInventoryRowDTO]{}, e3
					}
					productName = n
				}

				tokenName := ""
				if tbID != "" {
					n, e4 := resolveTokenName(tbID)
					if e4 != nil {
						log.Printf("[OrderManagementQuery] ERROR TokenBlueprint.GetByID failed tokenBlueprintId=%q err=%v", tbID, e4)
						return common.PageResult[OrderItemInventoryRowDTO]{}, e4
					}
					tokenName = n
				}

				listReadableID := ""
				if it.ListID != "" {
					n, e5 := resolveListReadableID(it.ListID)
					if e5 != nil {
						log.Printf("[OrderManagementQuery] WARN GetReadableIDByID failed listId=%q err=%v", it.ListID, e5)
					} else {
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

					AvatarName: avatarName,

					Paid:      ord.Paid,
					CreatedAt: createdAt,

					InventoryID:        invID,
					ProductBlueprintID: pbID,
					TokenBlueprintID:   tbID,
					ProductName:        productName,
					TokenName:          tokenName,

					ListReadableID: listReadableID,

					CategoryID:     categoryID,
					CategoryCode:   categoryCode,
					CategoryNameJa: categoryNameJa,
					CategoryNameEn: categoryNameEn,
					CategoryKind:   categoryKind,
					CategoryPath:   categoryPath,
					CategoryFields: categoryFields,

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
