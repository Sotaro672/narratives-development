// backend/internal/application/query/console/order_management_query.go
package query

//
// 機能: OrderManagementQuery (console)
//   - currentCompany 境界（inventory_query 相当）で許可された inventoryId のみを対象に
//     Order.Items[].InventoryID をフラットに列挙する
//   - order lister の ListByInventoryIDs を使い、allowed inventoryId に紐づく order を取得し、
//     allowed items を集約してから再ページングする
//
// 目的:
// - order テーブルの items に記載された inventoryId を、company 境界に従って安全に一覧できるようにする
//
// ✅ DI整合のための方針:
//   - Query側の port は domain/order.Filter / common.Sort / common.Page を引数に取る。
//   - currentCompany 境界のため、OrderLister は List ではなく ListByInventoryIDs を要求する。
//
// ✅ 重要:
//   - productName/tokenName は best-effort。
//     pbName/tbName が DI されていれば埋める、されていなければ空で返す（500にしない）。
//   - listReadableId も best-effort。
//     listReadable が DI されていれば listId->readableId を引いて埋める。なければ空で返す（500にしない）。
//   - avatarName も best-effort。
//     avatarNameResolver が DI されていれば avatarId->avatarName を引いて埋める。なければ空で返す（500にしない）。
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
	common "narratives/internal/domain/common"
	invdom "narratives/internal/domain/inventory"
	orderdom "narratives/internal/domain/order"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// Ports (read-only)
// ============================================================

// OrderLister lists orders for console query processing.
// It must support company-bound inventory filtering.
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

// ✅ inventoryId から productBlueprintId / tokenBlueprintId を引ける read-only port
type InventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(ctx context.Context, inventoryID string) (productBlueprintID string, tokenBlueprintID string, err error)
}

// ✅ productBlueprintId -> ProductBlueprint（best-effort）
// productName を取得するために使う。
type ProductBlueprintNameResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// ✅ productBlueprintId -> ProductBlueprint（best-effort）
// category snapshot / categoryFields を取得するために使う。
type ProductBlueprintResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// ✅ tokenBlueprintId -> tokenName（best-effort）
// tokenBlueprint.RepositoryPort の GetNameByID を想定
type TokenBlueprintNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// ✅ listId -> readableId（best-effort）
type ListReadableIDResolver interface {
	GetReadableIDByID(ctx context.Context, id string) (string, error)
}

// ✅ avatarId -> avatarName（best-effort）
// avatar.RepositoryPort の GetNameByID を想定（今回追加した port）
type AvatarNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// ✅ modelId(variationID) -> apparel/alcohol 表示情報（best-effort）
type ModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) resolver.ModelResolved
}

// ============================================================
// DTO
// ============================================================

// OrderItemInventoryRowDTO
// - Order.Items をフラット化した 1行 DTO
// - UI はこれをテーブル表示すればよい
type OrderItemInventoryRowDTO struct {
	OrderID string `json:"orderId"`

	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	// resolved from avatarId (best-effort)
	AvatarName string `json:"avatarName,omitempty"`

	// order-level
	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	// item-level
	InventoryID string `json:"inventoryId"`

	// resolved from inventoryId
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// resolved from IDs (best-effort)
	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	// UIへは listId ではなく readableId を渡す
	// - listId 自体が必要なら別フィールドで追加してもよいが、要件に従い置き換える
	ListReadableID string `json:"listReadableId,omitempty"`

	// productBlueprint category snapshot / categoryFields
	CategoryID     string         `json:"categoryId,omitempty"`
	CategoryCode   string         `json:"categoryCode,omitempty"`
	CategoryNameJa string         `json:"categoryNameJa,omitempty"`
	CategoryNameEn string         `json:"categoryNameEn,omitempty"`
	CategoryKind   string         `json:"categoryKind,omitempty"`
	CategoryPath   []string       `json:"categoryPath,omitempty"`
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	// model
	ModelID string `json:"modelId,omitempty"`

	// model resolved fields (best-effort)
	Kind        string `json:"kind,omitempty"`
	Size        string `json:"size,omitempty"`
	Color       string `json:"color,omitempty"`
	RGB         string `json:"rgb,omitempty"`
	ModelNumber string `json:"modelNumber,omitempty"`

	// alcohol model fields
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	Qty   int `json:"qty,omitempty"`
	Price int `json:"price,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"` // RFC3339(UTC)
}

// （任意）inventoryId だけ欲しい画面向け（distinct）
type InventoryIDDTO struct {
	InventoryID string `json:"inventoryId"`
}

// ============================================================
// Query
// ============================================================

type OrderManagementQuery struct {
	lister       OrderLister
	invRows      InventoryRowsLister        // REQUIRED
	invBlueprint InventoryBlueprintResolver // REQUIRED

	// optional (best-effort)
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

	// optional
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
	page = normalizePage(page)

	// optional は required 扱いしない。
	if q == nil || q.lister == nil || q.invRows == nil || q.invBlueprint == nil {
		return common.PageResult[OrderItemInventoryRowDTO]{}, errors.New("OrderManagementQuery.ListItemInventoryRows: wiring is incomplete (lister/invRows/invBlueprint required)")
	}

	// 原因特定用: listReadable の実体型をログ出力（interface の中身確認）
	// - Cloud Run で意図しない実装が DI されている場合、ここで即判別できる。
	log.Printf("[OrderManagementQuery] DEBUG listReadable resolver type=%T", q.listReadable)

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
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

	// inventoryId -> (pbID,tbID) cache
	type bt struct {
		pb string
		tb string
	}
	blueprintCache := map[string]bt{}

	// productBlueprintId -> productName cache
	pbNameCache := map[string]string{}

	// productBlueprintId -> ProductBlueprint cache
	productBlueprintCache := map[string]pbdom.ProductBlueprint{}

	// tokenBlueprintId -> tokenName cache
	tbNameCache := map[string]string{}

	// listId -> readableId cache
	listReadableCache := map[string]string{}

	// avatarId -> avatarName cache
	avatarNameCache := map[string]string{}

	// modelId -> resolved cache (best-effort)
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
		// optional
		if q.pbName == nil {
			return "", nil
		}

		if pbID == "" {
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
		// optional
		if q.productBlueprint == nil {
			return pbdom.ProductBlueprint{}, nil
		}

		if pbID == "" {
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
		// optional
		if q.tbName == nil {
			return "", nil
		}

		if tbID == "" {
			return "", nil
		}
		if v, ok := tbNameCache[tbID]; ok {
			return v, nil
		}
		name, e := q.tbName.GetNameByID(ctx, tbID)
		if e != nil {
			return "", e
		}
		tbNameCache[tbID] = name
		return name, nil
	}

	resolveListReadableID := func(listID string) (string, error) {
		// optional
		if q.listReadable == nil {
			return "", nil
		}

		if listID == "" {
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
		// optional
		if q.avatarNameResolver == nil {
			return "", nil
		}

		if avatarID == "" {
			return "", nil
		}
		if v, ok := avatarNameCache[avatarID]; ok {
			return v, nil
		}

		name, e := q.avatarNameResolver.GetNameByID(ctx, avatarID)
		if e != nil {
			return "", e
		}
		avatarNameCache[avatarID] = name
		return name, nil
	}

	resolveModel := func(modelID string) resolver.ModelResolved {
		// optional
		if q.modelResolver == nil || modelID == "" {
			return resolver.ModelResolved{}
		}
		if v, ok := modelCache[modelID]; ok {
			return v
		}
		resolved := q.modelResolver.ResolveModelResolved(ctx, modelID) // 取れない場合はゼロ値
		modelCache[modelID] = resolved
		return resolved
	}

	// スキャン上限
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
			orderID := nonEmpty(ord.ID, "(missing order id)")

			createdAt := ""
			if !ord.CreatedAt.IsZero() {
				createdAt = ord.CreatedAt.UTC().Format(time.RFC3339)
			}

			userID := ord.UserID
			avatarID := ord.AvatarID
			cartID := ord.CartID

			// avatarId -> avatarName (best-effort, order-level)
			avatarName := ""
			if avatarID != "" {
				n, e0 := resolveAvatarName(avatarID)
				if e0 != nil {
					log.Printf("[OrderManagementQuery] ERROR GetNameByID failed avatarId=%q err=%v", avatarID, e0)
					return common.PageResult[OrderItemInventoryRowDTO]{}, e0
				}
				avatarName = n
			}

			for _, it := range ord.Items {
				invID := it.InventoryID
				if !inventoryAllowed(allowedSet, invID) {
					continue
				}

				// inventoryId -> pb/tb
				pbID, tbID, e2 := resolveBlueprint(invID)
				if e2 != nil {
					log.Printf("[OrderManagementQuery] ERROR ResolveBlueprintIDsByInventoryID failed inventoryId=%q err=%v", invID, e2)
					return common.PageResult[OrderItemInventoryRowDTO]{}, e2
				}

				// productBlueprint category snapshot / categoryFields (best-effort)
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

				// names (best-effort)
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
						log.Printf("[OrderManagementQuery] ERROR GetNameByID failed tokenBlueprintId=%q err=%v", tbID, e4)
						return common.PageResult[OrderItemInventoryRowDTO]{}, e4
					}
					tokenName = n
				}

				// listId -> readableId (best-effort)
				// - 失敗しても 500 にしない（listReadableId は空のまま返す）
				listReadableID := ""
				if it.ListID != "" {
					n, e5 := resolveListReadableID(it.ListID)
					if e5 != nil {
						log.Printf("[OrderManagementQuery] WARN GetReadableIDByID failed listId=%q err=%v", it.ListID, e5)
					} else {
						listReadableID = n
					}
				}

				// model fields (best-effort)
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

		// 終端判定
		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else {
			if len(pr.Items) < page.PerPage {
				break
			}
		}

		srcPage++
	}

	// item単位で再ページング
	totalCount := len(allowedAll)
	tp := totalPages(totalCount, page.PerPage)

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
	end := minInt(start+page.PerPage, totalCount)

	return common.PageResult[OrderItemInventoryRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

// ListDistinctInventoryIDs
func (q *OrderManagementQuery) ListDistinctInventoryIDs(
	ctx context.Context,
	filter orderdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[InventoryIDDTO], error) {
	pr, err := q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		return common.PageResult[InventoryIDDTO]{}, err
	}

	seen := map[string]struct{}{}
	out := make([]InventoryIDDTO, 0, len(pr.Items))
	for _, row := range pr.Items {
		id := row.InventoryID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, InventoryIDDTO{InventoryID: id})
	}

	return common.PageResult[InventoryIDDTO]{
		Items:      out,
		Page:       pr.Page,
		PerPage:    pr.PerPage,
		TotalCount: len(out),
		TotalPages: totalPages(len(out), pr.PerPage),
	}, nil
}

// ============================================================
// local helpers
// ============================================================

func allowedInventoryIDSetFromContext(ctx context.Context, invRows InventoryRowsLister) (map[string]struct{}, error) {
	if invRows == nil {
		return nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, r := range rows {
		pbID := r.ProductBlueprintID
		tbID := r.TokenBlueprintID
		if pbID == "" || tbID == "" {
			continue
		}
		invID := pbID + "__" + tbID
		set[invID] = struct{}{}
	}
	return set, nil
}

func inventoryAllowed(set map[string]struct{}, inventoryID string) bool {
	if len(set) == 0 {
		return false
	}
	id := inventoryID
	if id == "" {
		return false
	}
	_, ok := set[id]
	return ok
}

func normalizePage(p common.Page) common.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	return p
}

func totalPages(totalCount int, perPage int) int {
	if perPage <= 0 || totalCount <= 0 {
		return 0
	}
	return (totalCount + perPage - 1) / perPage
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nonEmpty(v string, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
