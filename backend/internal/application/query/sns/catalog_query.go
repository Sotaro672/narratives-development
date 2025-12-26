// backend/internal/application/query/sns/catalog_query.go
package sns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs (for catalog.dart)
// ============================================================

type SNSCatalogDTO struct {
	List SNSCatalogListDTO `json:"list"`

	Inventory      *SNSCatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string                  `json:"inventoryError,omitempty"`

	ProductBlueprint      *SNSCatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                         `json:"productBlueprintError,omitempty"`

	ModelVariations      []SNSCatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                        `json:"modelVariationsError,omitempty"`
}

type SNSCatalogListDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	// linkage (catalog.dart uses these)
	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

// ✅ inventory stock model value (same shape as SNS inventory response)
type SNSCatalogInventoryModelStockDTO struct {
	Products map[string]bool `json:"products"`
}

type SNSCatalogInventoryDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	// (optional) inventory handler has this; keep it compatible
	ModelIDs []string `json:"modelIds,omitempty"`

	// ✅ NEW: stock を扱えるようにする（products を含める）
	// key=modelId
	Stock map[string]SNSCatalogInventoryModelStockDTO `json:"stock"`
}

type SNSCatalogProductBlueprintDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`

	ItemType string `json:"itemType"`
	Fit      string `json:"fit"`
	Material string `json:"material"`

	// domain 側が float64 のため、0 の場合は omitempty で落ちる
	Weight float64 `json:"weight,omitempty"`

	Printed bool `json:"printed"`

	QualityAssurance []string `json:"qualityAssurance"`
	ProductIDTagType string   `json:"productIdTagType"`
}

type SNSCatalogModelVariationDTO struct {
	ID                 string             `json:"id"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	ModelNumber        string             `json:"modelNumber"`
	Size               string             `json:"size"`
	Color              SNSCatalogColorDTO `json:"color"`
	Measurements       map[string]int     `json:"measurements,omitempty"`

	// ✅ NEW: 在庫の「件数（key数）」だけ画面へ渡す
	StockKeys int `json:"stockKeys,omitempty"`
}

type SNSCatalogColorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"`
}

// ============================================================
// Ports (minimal contracts for this query)
// ============================================================

// InventoryRepository returns already-shaped buyer-facing inventory DTO.
// ✅ stock(products) を含んだ shape を返す前提
type InventoryRepository interface {
	GetByID(ctx context.Context, id string) (*SNSCatalogInventoryDTO, error)
	GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*SNSCatalogInventoryDTO, error)
}

// ✅ OPTIONAL: Stock の “key集合” を domain Mint.Stock(=Products) から復元するための追加口
// - 既存実装を壊さないため、InventoryRepo が実装している場合だけ利用する
type InventoryMintStockSource interface {
	GetMintByID(ctx context.Context, id string) (invdom.Mint, error)
	GetMintByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (invdom.Mint, error)
}

type ProductBlueprintRepository interface {
	GetByID(ctx context.Context, id string) (*pbdom.ProductBlueprint, error)
}

// ============================================================
// Query
// ============================================================

type SNSCatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository

	ModelRepo modeldom.RepositoryPort
}

func NewSNSCatalogQuery(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
) *SNSCatalogQuery {
	return &SNSCatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		ModelRepo:     modelRepo,
	}
}

// GetByListID builds catalog payload by listId.
// - list must be status=listing, otherwise ErrNotFound.
// - inventory/product/model are best-effort; failures populate "*Error" fields.
func (q *SNSCatalogQuery) GetByListID(ctx context.Context, listID string) (SNSCatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return SNSCatalogDTO{}, errors.New("sns catalog query: list repo is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return SNSCatalogDTO{}, ldom.ErrNotFound
	}

	reqID := fmt.Sprintf("%d", time.Now().UnixNano())
	start := time.Now()

	log.Printf("[sns_catalog] start reqId=%s listId=%q", reqID, listID)

	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		log.Printf("[sns_catalog] list get error reqId=%s listId=%q err=%v", reqID, listID, err)
		return SNSCatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		log.Printf("[sns_catalog] list not listing reqId=%s listId=%q status=%v", reqID, listID, l.Status)
		return SNSCatalogDTO{}, ldom.ErrNotFound
	}

	out := SNSCatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// Inventory (prefer inventoryId; fallback to pb/tb query)
	// ------------------------------------------------------------
	var invDTO *SNSCatalogInventoryDTO

	var invID string
	var pbID string
	var tbID string

	if q.InventoryRepo == nil {
		out.InventoryError = "inventory repo is nil"
		log.Printf("[sns_catalog] inventory repo nil reqId=%s listId=%q", reqID, listID)
	} else {
		invID = strings.TrimSpace(out.List.InventoryID)
		pbID = strings.TrimSpace(out.List.ProductBlueprintID)
		tbID = strings.TrimSpace(out.List.TokenBlueprintID)

		log.Printf("[sns_catalog] linkage reqId=%s listId=%q invId=%q pbId=%q tbId=%q",
			reqID, listID, invID, pbID, tbID,
		)

		switch {
		case invID != "":
			v, e := q.InventoryRepo.GetByID(ctx, invID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[sns_catalog] inventory getById error reqId=%s invId=%q err=%v", reqID, invID, e)
			} else {
				normalizeInventoryStock(v)

				invDTO = v
				out.Inventory = v

				log.Printf("[sns_catalog] inventory getById ok reqId=%s invId=%q stockKeys=%d",
					reqID, invID, stockKeyCount(v.Stock),
				)
				logInventoryStockKeySample("sns_catalog", "inventory.dto.stockKeys", v.Stock, 5)
			}

		case pbID != "" && tbID != "":
			v, e := q.InventoryRepo.GetByProductAndTokenBlueprintID(ctx, pbID, tbID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[sns_catalog] inventory getByPbTb error reqId=%s pbId=%q tbId=%q err=%v", reqID, pbID, tbID, e)
			} else {
				normalizeInventoryStock(v)

				invDTO = v
				out.Inventory = v

				log.Printf("[sns_catalog] inventory getByPbTb ok reqId=%s pbId=%q tbId=%q stockKeys=%d",
					reqID, pbID, tbID, stockKeyCount(v.Stock),
				)
				logInventoryStockKeySample("sns_catalog", "inventory.dto.stockKeys", v.Stock, 5)
			}

		default:
			out.InventoryError = "inventory linkage is missing (inventoryId or productBlueprintId+tokenBlueprintId)"
			log.Printf("[sns_catalog] inventory linkage missing reqId=%s listId=%q", reqID, listID)
		}
	}

	// ✅ NEW: inventory.Stock が空なら、domain Mint.Stock から「modelId の集合(key)」を復元して埋める
	// - InventoryRepo が InventoryMintStockSource を実装している場合のみ有効
	if invDTO != nil {
		beforeKeys := stockKeyCount(invDTO.Stock)

		log.Printf("[sns_catalog] ensureStockKeysFromMint pre reqId=%s invId=%q stockKeys=%d",
			reqID, strings.TrimSpace(invDTO.ID), beforeKeys,
		)

		ensureInventoryStockKeysFromMintIfNeeded(ctx, q.InventoryRepo, invDTO, invID, pbID, tbID)

		afterKeys := stockKeyCount(invDTO.Stock)

		log.Printf("[sns_catalog] ensureStockKeysFromMint post reqId=%s invId=%q stockKeys=%d",
			reqID, strings.TrimSpace(invDTO.ID), afterKeys,
		)
		logInventoryStockKeySample("sns_catalog", "inventory.after.ensure.stockKeys", invDTO.Stock, 5)
	}

	// ------------------------------------------------------------
	// ProductBlueprint (inventory side wins)
	// ------------------------------------------------------------
	resolvedPBID := strings.TrimSpace(out.List.ProductBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.ProductBlueprintID); s != "" {
			resolvedPBID = s
		}
	}

	if q.ProductRepo == nil {
		out.ProductBlueprintError = "product repo is nil"
		log.Printf("[sns_catalog] product repo nil reqId=%s", reqID)
	} else if resolvedPBID == "" {
		out.ProductBlueprintError = "productBlueprintId is empty"
		log.Printf("[sns_catalog] productBlueprintId empty reqId=%s", reqID)
	} else {
		pb, e := q.ProductRepo.GetByID(ctx, resolvedPBID)
		if e != nil {
			out.ProductBlueprintError = e.Error()
			log.Printf("[sns_catalog] productBlueprint getById error reqId=%s pbId=%q err=%v", reqID, resolvedPBID, e)
		} else if pb != nil {
			dto := toCatalogProductBlueprintDTO(pb)
			out.ProductBlueprint = &dto
			log.Printf("[sns_catalog] productBlueprint getById ok reqId=%s pbId=%q", reqID, resolvedPBID)
		} else {
			out.ProductBlueprintError = "productBlueprint is nil"
			log.Printf("[sns_catalog] productBlueprint nil reqId=%s pbId=%q", reqID, resolvedPBID)
		}
	}

	// ------------------------------------------------------------
	// Models
	// - ListVariations(pbId=...) -> modelId list
	// - GetModelVariationByID -> dto
	// - dto.StockKeys に「inventory の stockKeys（key数）」を入れて返す
	// ------------------------------------------------------------
	if q.ModelRepo == nil {
		out.ModelVariationsError = "model repo is nil"
		log.Printf("[sns_catalog] model repo nil reqId=%s", reqID)
	} else if resolvedPBID == "" {
		out.ModelVariationsError = "productBlueprintId is empty (skip model fetch)"
		log.Printf("[sns_catalog] model skip pbId empty reqId=%s", reqID)
	} else {
		deletedFalse := false

		res, e := q.ModelRepo.ListVariations(
			ctx,
			modeldom.VariationFilter{
				ProductBlueprintID: resolvedPBID,
				Deleted:            &deletedFalse,
			},
			modeldom.Page{
				Number:  1,
				PerPage: 200,
			},
		)
		if e != nil {
			out.ModelVariationsError = e.Error()
			log.Printf("[sns_catalog] models listVariations error reqId=%s pbId=%q err=%v", reqID, resolvedPBID, e)
		} else {
			log.Printf("[sns_catalog] models listVariations ok reqId=%s pbId=%q items=%d", reqID, resolvedPBID, len(res.Items))

			items := make([]SNSCatalogModelVariationDTO, 0, len(res.Items))

			stockKeys := 0
			if invDTO != nil {
				stockKeys = stockKeyCount(invDTO.Stock)
			}
			log.Printf("[sns_catalog] models stockKeys resolved reqId=%s stockKeys=%d", reqID, stockKeys)

			for _, it := range res.Items {
				modelID := extractID(it)
				if modelID == "" {
					continue
				}

				mv, ge := q.ModelRepo.GetModelVariationByID(ctx, modelID)
				if ge != nil {
					if out.ModelVariationsError == "" {
						out.ModelVariationsError = ge.Error()
					}
					log.Printf("[sns_catalog] model getById error reqId=%s modelId=%q err=%v", reqID, modelID, ge)
					continue
				}

				dto, ok := toCatalogModelVariationDTOAny(mv)
				if !ok {
					dto = SNSCatalogModelVariationDTO{ID: strings.TrimSpace(modelID)}
				}

				// ✅ 画面へ渡すのは key数のみ（modelIdの種類数）
				dto.StockKeys = stockKeys

				items = append(items, dto)
			}

			out.ModelVariations = items
		}
	}

	invKeys := 0
	if invDTO != nil {
		invKeys = stockKeyCount(invDTO.Stock)
	}

	log.Printf("[sns_catalog] done reqId=%s listId=%q invStockKeys=%d modelVariations=%d dur=%s",
		reqID,
		listID,
		invKeys,
		len(out.ModelVariations),
		time.Since(start).String(),
	)

	return out, nil
}

// ============================================================
// inventory stock helpers (keys + products safety)
// ============================================================

// normalizeInventoryStock ensures:
// - Stock map is non-nil (if empty, keep nil OK)
// - each stock[modelId].Products is non-nil (so JSON includes {} not null when needed)
func normalizeInventoryStock(inv *SNSCatalogInventoryDTO) {
	if inv == nil {
		return
	}
	if inv.Stock == nil {
		// nil のままでもOK（keys count は 0）
		return
	}
	for k, v := range inv.Stock {
		m := strings.TrimSpace(k)
		if m == "" {
			continue
		}
		if v.Products == nil {
			v.Products = map[string]bool{}
			inv.Stock[k] = v
		}
	}
}

func ensureInventoryStockKeysFromMintIfNeeded(
	ctx context.Context,
	invRepo InventoryRepository,
	invDTO *SNSCatalogInventoryDTO,
	invID, pbID, tbID string,
) {
	if invRepo == nil || invDTO == nil {
		return
	}

	// 既に key が入っているなら何もしない
	if stockKeyCount(invDTO.Stock) > 0 {
		return
	}

	src, ok := invRepo.(InventoryMintStockSource)
	if !ok {
		log.Printf("[sns_catalog] ensureStockKeysFromMint skip: InventoryRepo does NOT implement InventoryMintStockSource")
		return
	}
	log.Printf("[sns_catalog] ensureStockKeysFromMint: InventoryRepo implements InventoryMintStockSource")

	// mint を取りに行けるなら取り、Stock の key(modelId) を復元
	var mint invdom.Mint
	var err error

	invID = strings.TrimSpace(invID)
	pbID = strings.TrimSpace(pbID)
	tbID = strings.TrimSpace(tbID)

	switch {
	case invID != "":
		log.Printf("[sns_catalog] ensureStockKeysFromMint mintGetById invId=%q", invID)
		mint, err = src.GetMintByID(ctx, invID)
	case pbID != "" && tbID != "":
		log.Printf("[sns_catalog] ensureStockKeysFromMint mintGetByPbTb pbId=%q tbId=%q", pbID, tbID)
		mint, err = src.GetMintByProductAndTokenBlueprintID(ctx, pbID, tbID)
	default:
		log.Printf("[sns_catalog] ensureStockKeysFromMint skip: linkage missing invId=%q pbId=%q tbId=%q", invID, pbID, tbID)
		return
	}
	if err != nil {
		log.Printf("[sns_catalog] ensureStockKeysFromMint mintGet error err=%v", err)
		return
	}

	keys := ExtractStockModelIDsFromMint(mint)
	log.Printf("[sns_catalog] ensureStockKeysFromMint extracted keys=%d", len(keys))
	logStringSample("sns_catalog", "mint.stock.modelIds", keys, 5)

	if len(keys) == 0 {
		return
	}
	if invDTO.Stock == nil {
		invDTO.Stock = map[string]SNSCatalogInventoryModelStockDTO{}
	}

	// value は products empty で良い（フロントは key数も products も扱える）
	for _, modelID := range keys {
		m := strings.TrimSpace(modelID)
		if m == "" {
			continue
		}
		if _, exists := invDTO.Stock[m]; !exists {
			invDTO.Stock[m] = SNSCatalogInventoryModelStockDTO{Products: map[string]bool{}}
		}
	}

	normalizeInventoryStock(invDTO)
}

// ExtractStockModelIDsFromMint extracts modelId keys by reading Mint.Stock map keys.
func ExtractStockModelIDsFromMint(m invdom.Mint) []string {
	rv := reflect.ValueOf(m)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	stock := rv.FieldByName("Stock")
	if !stock.IsValid() {
		return nil
	}
	if stock.Kind() == reflect.Pointer {
		if stock.IsNil() {
			return nil
		}
		stock = stock.Elem()
	}
	if stock.Kind() != reflect.Map {
		return nil
	}
	if stock.Type().Key().Kind() != reflect.String {
		return nil
	}

	keys := make([]string, 0, stock.Len())
	iter := stock.MapRange()
	for iter.Next() {
		k := strings.TrimSpace(iter.Key().String())
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

// ============================================================
// debug log helpers
// ============================================================

func stockKeyCount(stock map[string]SNSCatalogInventoryModelStockDTO) int {
	return len(stock)
}

func logInventoryStockKeySample(tag, label string, stock map[string]SNSCatalogInventoryModelStockDTO, max int) {
	if max <= 0 {
		return
	}
	if len(stock) == 0 {
		log.Printf("[%s] %s empty", tag, label)
		return
	}

	keys := make([]string, 0, len(stock))
	for k := range stock {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	if len(keys) > max {
		keys = keys[:max]
	}

	for _, k := range keys {
		log.Printf("[%s] %s sample modelId=%q", tag, label, k)
	}
}

func logStringSample(tag, label string, items []string, max int) {
	if max <= 0 {
		return
	}
	if len(items) == 0 {
		log.Printf("[%s] %s empty", tag, label)
		return
	}
	if len(items) > max {
		items = items[:max]
	}
	for _, s := range items {
		log.Printf("[%s] %s sample value=%q", tag, label, strings.TrimSpace(s))
	}
}

// ============================================================
// mappers
// ============================================================

func toCatalogListDTO(l ldom.List) SNSCatalogListDTO {
	return SNSCatalogListDTO{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,

		InventoryID: strings.TrimSpace(l.InventoryID),

		// ✅ list 側のフィールド名が ProductBlueprintID / ProductBlueprintId など揺れても拾う
		ProductBlueprintID: pickStringField(l, "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId"),
		TokenBlueprintID:   pickStringField(l, "TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId"),
	}
}

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) SNSCatalogProductBlueprintDTO {
	out := SNSCatalogProductBlueprintDTO{
		ID:          strings.TrimSpace(pb.ID),
		ProductName: strings.TrimSpace(pb.ProductName),
		BrandID:     strings.TrimSpace(pb.BrandID),
		CompanyID:   strings.TrimSpace(pb.CompanyID),

		ItemType: fmt.Sprint(pb.ItemType),
		Fit:      fmt.Sprint(pb.Fit),
		Material: fmt.Sprint(pb.Material),

		Weight:  pb.Weight,
		Printed: pb.Printed,

		QualityAssurance: append([]string{}, pb.QualityAssurance...),

		ProductIDTagType: pickProductIDTagType(pb),
	}
	return out
}

// toCatalogModelVariationDTOAny converts *any* ModelVariation-like struct into DTO by reflection.
func toCatalogModelVariationDTOAny(v any) (SNSCatalogModelVariationDTO, bool) {
	if v == nil {
		return SNSCatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return SNSCatalogModelVariationDTO{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return SNSCatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return SNSCatalogModelVariationDTO{}, false
	}

	// strings
	id := pickStringField(v, "ID", "Id", "ModelID", "ModelId", "modelId")
	if id == "" {
		id = pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	}
	if strings.TrimSpace(id) == "" {
		return SNSCatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	dto := SNSCatalogModelVariationDTO{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(pbID),
		ModelNumber:        strings.TrimSpace(modelNumber),
		Size:               strings.TrimSpace(size),
	}

	// color: Color.{Name,RGB}
	if c := rv.FieldByName("Color"); c.IsValid() {
		if c.Kind() == reflect.Pointer {
			if !c.IsNil() {
				c = c.Elem()
			}
		}
		if c.IsValid() && c.Kind() == reflect.Struct {
			name := ""
			rgb := 0

			nf := c.FieldByName("Name")
			if nf.IsValid() && nf.Kind() == reflect.String {
				name = strings.TrimSpace(nf.String())
			}
			rf := c.FieldByName("RGB")
			if rf.IsValid() {
				rgb = toInt(rf)
			}

			dto.Color = SNSCatalogColorDTO{Name: name, RGB: rgb}
		}
	}

	// measurements: map[string]int (or map[string]any/number)
	if m := rv.FieldByName("Measurements"); m.IsValid() {
		if m.Kind() == reflect.Map && m.Type().Key().Kind() == reflect.String {
			out := make(map[string]int)
			iter := m.MapRange()
			for iter.Next() {
				k := strings.TrimSpace(iter.Key().String())
				if k == "" {
					continue
				}
				out[k] = toInt(iter.Value())
			}
			if len(out) > 0 {
				dto.Measurements = out
			}
		}
	}

	return dto, true
}

// ============================================================
// reflection helpers (field-name tolerant)
// ============================================================

func pickStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
	}
	return ""
}

func pickProductIDTagType(pb *pbdom.ProductBlueprint) string {
	if pb == nil {
		return ""
	}

	// 1) 直下フィールド
	if s := pickStringField(*pb, "ProductIDTagType", "ProductIdTagType", "productIdTagType"); s != "" {
		return s
	}

	// 2) ネスト: ProductIDTag / ProductIdTag の中の Type
	rv := reflect.ValueOf(pb)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, parent := range []string{"ProductIDTag", "ProductIdTag", "productIdTag"} {
		f := rv.FieldByName(parent)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		if f.Kind() != reflect.Struct {
			continue
		}

		tf := f.FieldByName("Type")
		if tf.IsValid() && tf.Kind() == reflect.String {
			return strings.TrimSpace(tf.String())
		}
	}

	return ""
}

// extractID tries common field names (ID/Id/ModelID/ModelId) by reflection.
func extractID(v any) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range []string{"ID", "Id", "ModelID", "ModelId"} {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
	}

	return ""
}

// toInt converts common numeric kinds into int (best-effort).
func toInt(v reflect.Value) int {
	if !v.IsValid() {
		return 0
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	default:
		return 0
	}
}
