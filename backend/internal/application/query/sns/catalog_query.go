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

	snsdto "narratives/internal/application/query/sns/dto"
	appresolver "narratives/internal/application/resolver"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Ports (minimal contracts for this query)
// ============================================================

// InventoryRepository returns already-shaped buyer-facing inventory DTO.
// ✅ stock(products) を含んだ shape を返す前提
type InventoryRepository interface {
	GetByID(ctx context.Context, id string) (*snsdto.SNSCatalogInventoryDTO, error)
	GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*snsdto.SNSCatalogInventoryDTO, error)
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

// ✅ NEW: tokenBlueprint patch getter (buyer-facing minimal info)
type TokenBlueprintPatchRepository interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

// ============================================================
// Query
// ============================================================

type SNSCatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository
	TokenRepo     TokenBlueprintPatchRepository // ✅ NEW (optional)

	ModelRepo modeldom.RepositoryPort

	// ✅ OPTIONAL: name resolver (brand/company) for display fields
	NameResolver *appresolver.NameResolver
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
		TokenRepo:     nil, // keep backward compatible
		ModelRepo:     modelRepo,
		NameResolver:  nil, // keep backward compatible
	}
}

// ✅ NEW: ctor with tokenBlueprint patch getter
func NewSNSCatalogQueryWithTokenBlueprintPatch(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	tokenRepo TokenBlueprintPatchRepository,
	modelRepo modeldom.RepositoryPort,
) *SNSCatalogQuery {
	return &SNSCatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     tokenRepo,
		ModelRepo:     modelRepo,
		NameResolver:  nil, // keep backward compatible
	}
}

// ✅ NEW: ctor with name resolver (brand/company)
func NewSNSCatalogQueryWithNameResolver(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
	nameResolver *appresolver.NameResolver,
) *SNSCatalogQuery {
	return &SNSCatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     nil,
		ModelRepo:     modelRepo,
		NameResolver:  nameResolver,
	}
}

// ✅ NEW: ctor with tokenBlueprint patch + name resolver
func NewSNSCatalogQueryWithTokenBlueprintPatchAndNameResolver(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	tokenRepo TokenBlueprintPatchRepository,
	modelRepo modeldom.RepositoryPort,
	nameResolver *appresolver.NameResolver,
) *SNSCatalogQuery {
	return &SNSCatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     tokenRepo,
		ModelRepo:     modelRepo,
		NameResolver:  nameResolver,
	}
}

// GetByListID builds catalog payload by listId.
// - list must be status=listing, otherwise ErrNotFound.
// - inventory/product/model/tokenBlueprint are best-effort; failures populate "*Error" fields.
func (q *SNSCatalogQuery) GetByListID(ctx context.Context, listID string) (snsdto.SNSCatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return snsdto.SNSCatalogDTO{}, errors.New("sns catalog query: list repo is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return snsdto.SNSCatalogDTO{}, ldom.ErrNotFound
	}

	log.Printf("[sns_catalog] GetByListID start listId=%q", listID)

	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		log.Printf("[sns_catalog] list getById error listId=%q err=%q", listID, err.Error())
		return snsdto.SNSCatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		log.Printf("[sns_catalog] list not listing listId=%q status=%q", listID, fmt.Sprint(l.Status))
		return snsdto.SNSCatalogDTO{}, ldom.ErrNotFound
	}

	out := snsdto.SNSCatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// Inventory (prefer inventoryId; fallback to pb/tb query)
	// ------------------------------------------------------------
	var invDTO *snsdto.SNSCatalogInventoryDTO

	var invID string
	var pbID string
	var tbID string

	if q.InventoryRepo == nil {
		out.InventoryError = "inventory repo is nil"
		log.Printf("[sns_catalog] inventory repo is nil listId=%q", listID)
	} else {
		invID = strings.TrimSpace(out.List.InventoryID)
		pbID = strings.TrimSpace(out.List.ProductBlueprintID)
		tbID = strings.TrimSpace(out.List.TokenBlueprintID)

		log.Printf(
			"[sns_catalog] inventory linkage listId=%q inventoryId=%q pbId=%q tbId=%q",
			listID, invID, pbID, tbID,
		)

		switch {
		case invID != "":
			v, e := q.InventoryRepo.GetByID(ctx, invID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[sns_catalog] inventory getById error listId=%q invId=%q err=%q", listID, invID, e.Error())
			} else {
				normalizeInventoryStock(v)
				invDTO = v
				out.Inventory = v
				log.Printf("[sns_catalog] inventory getById ok listId=%q invId=%q stockKeys=%d", listID, invID, stockKeyCount(v.Stock))
			}

		case pbID != "" && tbID != "":
			v, e := q.InventoryRepo.GetByProductAndTokenBlueprintID(ctx, pbID, tbID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[sns_catalog] inventory getByPbTb error listId=%q pbId=%q tbId=%q err=%q", listID, pbID, tbID, e.Error())
			} else {
				normalizeInventoryStock(v)
				invDTO = v
				out.Inventory = v
				log.Printf("[sns_catalog] inventory getByPbTb ok listId=%q pbId=%q tbId=%q stockKeys=%d", listID, pbID, tbID, stockKeyCount(v.Stock))
			}

		default:
			out.InventoryError = "inventory linkage is missing (inventoryId or productBlueprintId+tokenBlueprintId)"
			log.Printf("[sns_catalog] inventory linkage missing listId=%q", listID)
		}
	}

	// ✅ NEW: inventory.Stock が空なら、domain Mint.Stock から「modelId の集合(key)」を復元して埋める
	// - InventoryRepo が InventoryMintStockSource を実装している場合のみ有効
	if invDTO != nil {
		before := stockKeyCount(invDTO.Stock)
		ensureInventoryStockKeysFromMintIfNeeded(ctx, q.InventoryRepo, invDTO, invID, pbID, tbID)
		after := stockKeyCount(invDTO.Stock)
		if after != before {
			log.Printf("[sns_catalog] inventory stockKeys restored from mint listId=%q before=%d after=%d", listID, before, after)
		}
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
		log.Printf("[sns_catalog] product repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ProductBlueprintError = "productBlueprintId is empty"
		log.Printf("[sns_catalog] productBlueprintId is empty listId=%q", listID)
	} else {
		pb, e := q.ProductRepo.GetByID(ctx, resolvedPBID)
		if e != nil {
			out.ProductBlueprintError = e.Error()
			log.Printf("[sns_catalog] product getById error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else if pb != nil {
			dto := toCatalogProductBlueprintDTO(pb)

			// ✅ NEW: resolve brandName/companyName (best-effort)
			if q.NameResolver != nil {
				fillProductBlueprintNames(ctx, q.NameResolver, &dto)
			}

			out.ProductBlueprint = &dto
			log.Printf(
				"[sns_catalog] product getById ok listId=%q pbId=%q brandId=%q companyId=%q brandName=%q companyName=%q",
				listID,
				resolvedPBID,
				strings.TrimSpace(dto.BrandID),
				strings.TrimSpace(dto.CompanyID),
				getStringFieldBestEffort(dto, "BrandName"),
				getStringFieldBestEffort(dto, "CompanyName"),
			)
		} else {
			out.ProductBlueprintError = "productBlueprint is nil"
			log.Printf("[sns_catalog] product is nil listId=%q pbId=%q", listID, resolvedPBID)
		}
	}

	// ------------------------------------------------------------
	// ✅ NEW: TokenBlueprint patch (inventory side wins)
	// ------------------------------------------------------------
	resolvedTBID := strings.TrimSpace(out.List.TokenBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.TokenBlueprintID); s != "" {
			resolvedTBID = s
		}
	}

	log.Printf("[sns_catalog] tokenBlueprint resolve listId=%q resolvedTbId=%q (list.tbId=%q inv.tbId=%q)",
		listID,
		resolvedTBID,
		strings.TrimSpace(out.List.TokenBlueprintID),
		func() string {
			if invDTO == nil {
				return ""
			}
			return strings.TrimSpace(invDTO.TokenBlueprintID)
		}(),
	)

	if q.TokenRepo == nil {
		// linkage が無いケースと区別したいので、ID がある時だけエラーにする
		if resolvedTBID != "" {
			out.TokenBlueprintError = "tokenBlueprint repo is nil"
			log.Printf("[sns_catalog] tokenBlueprint repo is nil listId=%q tbId=%q", listID, resolvedTBID)
		} else {
			log.Printf("[sns_catalog] tokenBlueprint skip (tbId empty & repo nil) listId=%q", listID)
		}
	} else if resolvedTBID == "" {
		out.TokenBlueprintError = "tokenBlueprintId is empty"
		log.Printf("[sns_catalog] tokenBlueprintId is empty listId=%q", listID)
	} else {
		log.Printf("[sns_catalog] tokenBlueprint getPatchById start listId=%q tbId=%q", listID, resolvedTBID)

		patch, e := q.TokenRepo.GetPatchByID(ctx, resolvedTBID)
		if e != nil {
			out.TokenBlueprintError = e.Error()
			log.Printf("[sns_catalog] tokenBlueprint getPatchById error listId=%q tbId=%q err=%q", listID, resolvedTBID, e.Error())
		} else {
			p := patch
			out.TokenBlueprint = &p
			log.Printf(
				"[sns_catalog] tokenBlueprint getPatchById ok listId=%q tbId=%q name=%q symbol=%q brandId=%q brandName=%q minted=%s hasIconUrl=%t",
				listID,
				resolvedTBID,
				ptrStr(p.Name),
				ptrStr(p.Symbol),
				ptrStr(p.BrandID),
				ptrStr(p.BrandName),
				ptrBoolStr(p.Minted),
				strings.TrimSpace(ptrStr(p.IconURL)) != "",
			)
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
		log.Printf("[sns_catalog] model repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ModelVariationsError = "productBlueprintId is empty (skip model fetch)"
		log.Printf("[sns_catalog] model skip (pbId empty) listId=%q", listID)
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
			log.Printf("[sns_catalog] model listVariations error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else {
			items := make([]snsdto.SNSCatalogModelVariationDTO, 0, len(res.Items))

			stockKeys := 0
			if invDTO != nil {
				stockKeys = stockKeyCount(invDTO.Stock)
			}

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
					continue
				}

				dto, ok := toCatalogModelVariationDTOAny(mv)
				if !ok {
					dto = snsdto.SNSCatalogModelVariationDTO{ID: strings.TrimSpace(modelID)}
				}

				// ✅ 画面へ渡すのは key数のみ（modelIdの種類数）
				dto.StockKeys = stockKeys

				items = append(items, dto)
			}

			out.ModelVariations = items
			log.Printf("[sns_catalog] model variations ok listId=%q pbId=%q items=%d stockKeys=%d", listID, resolvedPBID, len(items), stockKeys)
		}
	}

	log.Printf("[sns_catalog] GetByListID done listId=%q invErr=%q pbErr=%q tbErr=%q modelErr=%q",
		listID,
		strings.TrimSpace(out.InventoryError),
		strings.TrimSpace(out.ProductBlueprintError),
		strings.TrimSpace(out.TokenBlueprintError),
		strings.TrimSpace(out.ModelVariationsError),
	)

	return out, nil
}

// ============================================================
// name resolving (productBlueprint -> brandName/companyName)
// ============================================================

func fillProductBlueprintNames(ctx context.Context, r *appresolver.NameResolver, dto *snsdto.SNSCatalogProductBlueprintDTO) {
	if r == nil || dto == nil {
		return
	}

	brandID := strings.TrimSpace(dto.BrandID)
	companyID := strings.TrimSpace(dto.CompanyID)

	// BrandName
	if brandID != "" {
		bn := strings.TrimSpace(r.ResolveBrandName(ctx, brandID))
		if bn != "" {
			// DTO に BrandName フィールドがある場合だけセット（無い場合でもコンパイルは通る）
			setStringFieldBestEffort(dto, "BrandName", bn)
		}
	}

	// CompanyName
	if companyID != "" {
		cn := strings.TrimSpace(r.ResolveCompanyName(ctx, companyID))
		if cn != "" {
			setStringFieldBestEffort(dto, "CompanyName", cn)
		}
	}
}

// setStringFieldBestEffort sets either:
// - field string
// - field *string
// if the exported field exists.
// (DTO に該当フィールドが無くても安全に no-op)
func setStringFieldBestEffort(target any, fieldName string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(value)
	case reflect.Pointer:
		if f.Type().Elem().Kind() == reflect.String {
			s := value
			f.Set(reflect.ValueOf(&s))
		}
	}
}

// getStringFieldBestEffort reads either string or *string field (exported), else "".
func getStringFieldBestEffort(target any, fieldName string) string {
	rv := reflect.ValueOf(target)
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

	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		return ""
	}

	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return ""
		}
		f = f.Elem()
	}
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	return ""
}

// ============================================================
// inventory stock helpers (keys + products safety)
// ============================================================

// normalizeInventoryStock ensures:
// - Stock map is non-nil (if empty, keep nil OK)
// - each stock[modelId].Products is non-nil (so JSON includes {} not null when needed)
func normalizeInventoryStock(inv *snsdto.SNSCatalogInventoryDTO) {
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
	invDTO *snsdto.SNSCatalogInventoryDTO,
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
		return
	}

	// mint を取りに行けるなら取り、Stock の key(modelId) を復元
	var mint invdom.Mint
	var err error

	invID = strings.TrimSpace(invID)
	pbID = strings.TrimSpace(pbID)
	tbID = strings.TrimSpace(tbID)

	switch {
	case invID != "":
		mint, err = src.GetMintByID(ctx, invID)
	case pbID != "" && tbID != "":
		mint, err = src.GetMintByProductAndTokenBlueprintID(ctx, pbID, tbID)
	default:
		return
	}
	if err != nil {
		return
	}

	keys := ExtractStockModelIDsFromMint(mint)
	if len(keys) == 0 {
		return
	}
	if invDTO.Stock == nil {
		invDTO.Stock = map[string]snsdto.SNSCatalogInventoryModelStockDTO{}
	}

	// value は products empty で良い（フロントは key数も products も扱える）
	for _, modelID := range keys {
		m := strings.TrimSpace(modelID)
		if m == "" {
			continue
		}
		if _, exists := invDTO.Stock[m]; !exists {
			invDTO.Stock[m] = snsdto.SNSCatalogInventoryModelStockDTO{Products: map[string]bool{}}
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

func stockKeyCount(stock map[string]snsdto.SNSCatalogInventoryModelStockDTO) int {
	return len(stock)
}

// ============================================================
// mappers
// ============================================================

func toCatalogListDTO(l ldom.List) snsdto.SNSCatalogListDTO {
	return snsdto.SNSCatalogListDTO{
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

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) snsdto.SNSCatalogProductBlueprintDTO {
	out := snsdto.SNSCatalogProductBlueprintDTO{
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
func toCatalogModelVariationDTOAny(v any) (snsdto.SNSCatalogModelVariationDTO, bool) {
	if v == nil {
		return snsdto.SNSCatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return snsdto.SNSCatalogModelVariationDTO{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return snsdto.SNSCatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return snsdto.SNSCatalogModelVariationDTO{}, false
	}

	// strings
	id := pickStringField(v, "ID", "Id", "ModelID", "ModelId", "modelId")
	if id == "" {
		id = pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	}
	if strings.TrimSpace(id) == "" {
		return snsdto.SNSCatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	dto := snsdto.SNSCatalogModelVariationDTO{
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

			dto.Color = snsdto.SNSCatalogColorDTO{Name: name, RGB: rgb}
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

// ============================================================
// log helpers (avoid nil pointer noise)
// ============================================================

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func ptrBoolStr(b *bool) string {
	if b == nil {
		return "(nil)"
	}
	if *b {
		return "true"
	}
	return "false"
}
