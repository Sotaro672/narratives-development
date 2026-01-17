// backend/internal/application/query/mall/catalog_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
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

type InventoryRepository interface {
	GetByID(ctx context.Context, id string) (invdom.Mint, error)
	GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (invdom.Mint, error)
}

type ProductBlueprintRepository interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type TokenBlueprintPatchRepository interface {
	GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
}

// ============================================================
// Query
// ============================================================

type CatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository
	TokenRepo     TokenBlueprintPatchRepository

	ModelRepo modeldom.RepositoryPort

	NameResolver *appresolver.NameResolver
}

func NewCatalogQuery(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     nil,
		ModelRepo:     modelRepo,
		NameResolver:  nil,
	}
}

func NewCatalogQueryWithTokenBlueprintPatch(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	tokenRepo TokenBlueprintPatchRepository,
	modelRepo modeldom.RepositoryPort,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     tokenRepo,
		ModelRepo:     modelRepo,
		NameResolver:  nil,
	}
}

func NewCatalogQueryWithNameResolver(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
	nameResolver *appresolver.NameResolver,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     nil,
		ModelRepo:     modelRepo,
		NameResolver:  nameResolver,
	}
}

func NewCatalogQueryWithTokenBlueprintPatchAndNameResolver(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	tokenRepo TokenBlueprintPatchRepository,
	modelRepo modeldom.RepositoryPort,
	nameResolver *appresolver.NameResolver,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		TokenRepo:     tokenRepo,
		ModelRepo:     modelRepo,
		NameResolver:  nameResolver,
	}
}

func (q *CatalogQuery) GetByListID(ctx context.Context, listID string) (dto.CatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return dto.CatalogDTO{}, errors.New("catalog query: list repo is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	log.Printf("[catalog] GetByListID start listId=%q", listID)

	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		log.Printf("[catalog] list getById error listId=%q err=%q", listID, err.Error())
		return dto.CatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		log.Printf("[catalog] list not listing listId=%q status=%q", listID, fmt.Sprint(l.Status))
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	out := dto.CatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// Inventory (prefer inventoryId; fallback to pb/tb query)
	// ------------------------------------------------------------
	var invDTO *dto.CatalogInventoryDTO

	var invID string
	var pbID string
	var tbID string

	if q.InventoryRepo == nil {
		out.InventoryError = "inventory repo is nil"
		log.Printf("[catalog] inventory repo is nil listId=%q", listID)
	} else {
		invID = strings.TrimSpace(out.List.InventoryID)
		pbID = strings.TrimSpace(out.List.ProductBlueprintID)
		tbID = strings.TrimSpace(out.List.TokenBlueprintID)

		// list 側が空でも inventoryId が pb__tb なら復元（ログで pbId/tbId が空だった対策）
		if (pbID == "" || tbID == "") && invID != "" {
			if p, t, ok := splitInventoryID(invID); ok {
				if pbID == "" {
					pbID = p
				}
				if tbID == "" {
					tbID = t
				}
			}
		}

		log.Printf(
			"[catalog] inventory linkage listId=%q inventoryId=%q pbId=%q tbId=%q",
			listID, invID, pbID, tbID,
		)

		switch {
		case invID != "":
			m, e := q.InventoryRepo.GetByID(ctx, invID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[catalog] inventory getById error listId=%q invId=%q err=%q", listID, invID, e.Error())
			} else {
				v := toCatalogInventoryDTOFromMint(m)
				normalizeInventoryStock(v)
				invDTO = v
				out.Inventory = v
				log.Printf("[catalog] inventory getById ok listId=%q invId=%q stockKeys=%d", listID, invID, stockKeyCount(v.Stock))
			}

		case pbID != "" && tbID != "":
			m, e := q.InventoryRepo.GetByProductAndTokenBlueprintID(ctx, pbID, tbID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[catalog] inventory getByPbTb error listId=%q pbId=%q tbId=%q err=%q", listID, pbID, tbID, e.Error())
			} else {
				v := toCatalogInventoryDTOFromMint(m)
				normalizeInventoryStock(v)
				invDTO = v
				out.Inventory = v
				log.Printf("[catalog] inventory getByPbTb ok listId=%q pbId=%q tbId=%q stockKeys=%d", listID, pbID, tbID, stockKeyCount(v.Stock))
			}

		default:
			out.InventoryError = "inventory linkage is missing (inventoryId or productBlueprintId+tokenBlueprintId)"
			log.Printf("[catalog] inventory linkage missing listId=%q", listID)
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
		log.Printf("[catalog] product repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ProductBlueprintError = "productBlueprintId is empty"
		log.Printf("[catalog] productBlueprintId is empty listId=%q", listID)
	} else {
		pb, e := q.ProductRepo.GetByID(ctx, resolvedPBID)
		if e != nil {
			out.ProductBlueprintError = e.Error()
			log.Printf("[catalog] product getById error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else {
			pbDTO := toCatalogProductBlueprintDTO(&pb)

			if q.NameResolver != nil {
				fillProductBlueprintNames(ctx, q.NameResolver, &pbDTO)
			}

			out.ProductBlueprint = &pbDTO
			log.Printf(
				"[catalog] product getById ok listId=%q pbId=%q brandId=%q companyId=%q brandName=%q companyName=%q",
				listID,
				resolvedPBID,
				strings.TrimSpace(pbDTO.BrandID),
				strings.TrimSpace(pbDTO.CompanyID),
				getStringFieldBestEffort(pbDTO, "BrandName"),
				getStringFieldBestEffort(pbDTO, "CompanyName"),
			)
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint patch (inventory side wins)
	// ------------------------------------------------------------
	resolvedTBID := strings.TrimSpace(out.List.TokenBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.TokenBlueprintID); s != "" {
			resolvedTBID = s
		}
	}

	log.Printf("[catalog] tokenBlueprint resolve listId=%q resolvedTbId=%q (list.tbId=%q inv.tbId=%q)",
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

	// best-effort: TokenRepo が nil なら “エラーを立てない”
	if q.TokenRepo == nil {
		if resolvedTBID != "" {
			log.Printf("[catalog] tokenBlueprint repo is nil (best-effort) listId=%q tbId=%q", listID, resolvedTBID)
		} else {
			log.Printf("[catalog] tokenBlueprint skip (tbId empty & repo nil) listId=%q", listID)
		}
	} else if resolvedTBID == "" {
		out.TokenBlueprintError = "tokenBlueprintId is empty"
		log.Printf("[catalog] tokenBlueprintId is empty listId=%q", listID)
	} else {
		log.Printf("[catalog] tokenBlueprint getPatchById start listId=%q tbId=%q", listID, resolvedTBID)

		patch, e := q.TokenRepo.GetPatchByID(ctx, resolvedTBID)
		if e != nil {
			out.TokenBlueprintError = e.Error()
			log.Printf("[catalog] tokenBlueprint getPatchById error listId=%q tbId=%q err=%q", listID, resolvedTBID, e.Error())
		} else {
			p := patch

			if q.NameResolver != nil {
				fillTokenBlueprintPatchNames(ctx, q.NameResolver, &p)
			}

			out.TokenBlueprint = &p
			log.Printf(
				"[catalog] tokenBlueprint getPatchById ok listId=%q tbId=%q name=%q symbol=%q brandId=%q brandName=%q companyId=%q companyName=%q minted=%s hasIconUrl=%t",
				listID,
				resolvedTBID,
				ptrStr(p.Name),
				ptrStr(p.Symbol),
				ptrStr(p.BrandID),
				ptrStr(p.BrandName),
				ptrStr(p.CompanyID),
				ptrStr(p.CompanyName),
				ptrBoolStr(p.Minted),
				strings.TrimSpace(ptrStr(p.IconURL)) != "",
			)
		}
	}

	// ------------------------------------------------------------
	// Models (UNIFIED)
	// ------------------------------------------------------------
	if q.ModelRepo == nil {
		out.ModelVariationsError = "model repo is nil"
		log.Printf("[catalog] model repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ModelVariationsError = "productBlueprintId is empty (skip model fetch)"
		log.Printf("[catalog] model skip (pbId empty) listId=%q", listID)
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
			log.Printf("[catalog] model listVariations error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else {
			items := make([]dto.CatalogModelVariationDTO, 0, len(res.Items))

			for _, it := range res.Items {
				modelID := extractID(it)
				if modelID == "" {
					continue
				}

				mv, ge := q.ModelRepo.GetModelVariationByID(ctx, modelID)
				if ge != nil {
					if strings.TrimSpace(out.ModelVariationsError) == "" {
						out.ModelVariationsError = ge.Error()
					}
					continue
				}

				mvDTO, ok := toCatalogModelVariationDTOAny(mv)
				if !ok {
					mvDTO = dto.CatalogModelVariationDTO{
						ID:           strings.TrimSpace(modelID),
						Measurements: map[string]int{},
					}
				}
				if mvDTO.Measurements == nil {
					mvDTO.Measurements = map[string]int{}
				}

				items = append(items, mvDTO)
			}

			attachStockToModelVariations(&items, invDTO)

			out.ModelVariations = items
			log.Printf(
				"[catalog] model variations ok(list unified) listId=%q pbId=%q items=%d stockKeys=%d",
				listID,
				resolvedPBID,
				len(items),
				func() int {
					if invDTO == nil {
						return 0
					}
					return stockKeyCount(invDTO.Stock)
				}(),
			)
		}
	}

	log.Printf("[catalog] GetByListID done listId=%q invErr=%q pbErr=%q tbErr=%q modelErr=%q",
		listID,
		strings.TrimSpace(out.InventoryError),
		strings.TrimSpace(out.ProductBlueprintError),
		strings.TrimSpace(out.TokenBlueprintError),
		strings.TrimSpace(out.ModelVariationsError),
	)

	return out, nil
}

// ============================================================
// name resolving
// ============================================================

func fillProductBlueprintNames(ctx context.Context, r *appresolver.NameResolver, dtoPB *dto.CatalogProductBlueprintDTO) {
	if r == nil || dtoPB == nil {
		return
	}

	brandID := strings.TrimSpace(dtoPB.BrandID)
	companyID := strings.TrimSpace(dtoPB.CompanyID)

	if brandID != "" {
		bn := strings.TrimSpace(r.ResolveBrandName(ctx, brandID))
		if bn != "" {
			setStringFieldBestEffort(dtoPB, "BrandName", bn)
		}
	}

	if companyID != "" {
		cn := strings.TrimSpace(r.ResolveCompanyName(ctx, companyID))
		if cn != "" {
			setStringFieldBestEffort(dtoPB, "CompanyName", cn)
		}
	}
}

func fillTokenBlueprintPatchNames(ctx context.Context, r *appresolver.NameResolver, p *tbdom.Patch) {
	if r == nil || p == nil {
		return
	}

	brandID := strings.TrimSpace(ptrStr(p.BrandID))
	companyID := strings.TrimSpace(ptrStr(p.CompanyID))

	if brandID != "" && strings.TrimSpace(ptrStr(p.BrandName)) == "" {
		if bn := strings.TrimSpace(r.ResolveBrandName(ctx, brandID)); bn != "" {
			s := bn
			p.BrandName = &s
		}
	}

	if companyID != "" && strings.TrimSpace(ptrStr(p.CompanyName)) == "" {
		if cn := strings.TrimSpace(r.ResolveCompanyName(ctx, companyID)); cn != "" {
			s := cn
			p.CompanyName = &s
		}
	}
}

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
// inventory stock helpers
// ============================================================

func normalizeInventoryStock(inv *dto.CatalogInventoryDTO) {
	if inv == nil || inv.Stock == nil {
		return
	}

	// Stock map のキー正規化と、nil map 回避のみ
	// ※ Products は廃止（Accumulation/ReservedCount を使う）
	norm := make(map[string]dto.CatalogInventoryModelStockDTO, len(inv.Stock))
	for k, v := range inv.Stock {
		m := strings.TrimSpace(k)
		if m == "" {
			continue
		}
		norm[m] = v
	}
	inv.Stock = norm
}

func stockKeyCount(stock map[string]dto.CatalogInventoryModelStockDTO) int {
	return len(stock)
}

// attachStockToModelVariations sets StockKeys only.
// NOTE: ModelVariationDTO no longer carries Products.
func attachStockToModelVariations(items *[]dto.CatalogModelVariationDTO, inv *dto.CatalogInventoryDTO) {
	if items == nil || len(*items) == 0 {
		return
	}

	stockKeys := 0
	if inv != nil {
		stockKeys = stockKeyCount(inv.Stock)
	}

	for i := range *items {
		(*items)[i].StockKeys = stockKeys
	}
}

// ============================================================
// mappers
// ============================================================

func toCatalogListDTO(l ldom.List) dto.CatalogListDTO {
	return dto.CatalogListDTO{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,

		InventoryID: strings.TrimSpace(l.InventoryID),

		ProductBlueprintID: pickStringField(l, "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId"),
		TokenBlueprintID:   pickStringField(l, "TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId"),
	}
}

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) dto.CatalogProductBlueprintDTO {
	out := dto.CatalogProductBlueprintDTO{
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

// Mint -> CatalogInventoryDTO（domain を正とする。旧互換は削除）
func toCatalogInventoryDTOFromMint(m invdom.Mint) *dto.CatalogInventoryDTO {
	out := &dto.CatalogInventoryDTO{
		ID:                 strings.TrimSpace(m.ID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ModelIDs:           append([]string{}, m.ModelIDs...),
		Stock:              map[string]dto.CatalogInventoryModelStockDTO{},
	}

	if m.Stock == nil {
		return out
	}

	// 期待する domain 側の表現:
	// m.Stock[modelId] has fields: Accumulation, ReservedCount (and/or legacy Products)
	// ここでは "availableStock = accumulation - reservedCount" をフロントで計算できるよう、
	// Accumulation / ReservedCount を DTO に詰める。
	for modelID, ms := range m.Stock {
		mid := strings.TrimSpace(modelID)
		if mid == "" {
			continue
		}

		a := pickIntField(ms, "Accumulation", "accumulation", "Count", "count")
		r := pickIntField(ms, "ReservedCount", "reservedCount", "Reserved", "reserved")

		out.Stock[mid] = dto.CatalogInventoryModelStockDTO{
			Accumulation:  a,
			ReservedCount: r,
		}
	}

	return out
}

// ============================================================
// model variation mapper (restore: toCatalogModelVariationDTOAny)
// ============================================================

// toCatalogModelVariationDTOAny converts *any* ModelVariation-like struct into DTO by reflection.
// It is tolerant to field-name drift and guarantees non-nil Measurements on success.
func toCatalogModelVariationDTOAny(v any) (dto.CatalogModelVariationDTO, bool) {
	if v == nil {
		return dto.CatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return dto.CatalogModelVariationDTO{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return dto.CatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return dto.CatalogModelVariationDTO{}, false
	}

	// strings
	id := pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	if strings.TrimSpace(id) == "" {
		return dto.CatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	out := dto.CatalogModelVariationDTO{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(pbID),
		ModelNumber:        strings.TrimSpace(modelNumber),
		Size:               strings.TrimSpace(size),

		ColorName: "",
		ColorRGB:  0,

		Measurements: map[string]int{},

		StockKeys: 0,
	}

	// color: Color.{Name,RGB} or flattened ColorName/ColorRGB
	if s := pickStringField(rv.Interface(), "ColorName", "colorName"); s != "" {
		out.ColorName = strings.TrimSpace(s)
	}

	// flattened rgb variants
	if f := rv.FieldByName("ColorRGB"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("ColorRgb"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("RGB"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else if f := rv.FieldByName("Rgb"); f.IsValid() {
		out.ColorRGB = toInt(f)
	} else {
		// nested Color struct
		if c := rv.FieldByName("Color"); c.IsValid() {
			if c.Kind() == reflect.Pointer {
				if !c.IsNil() {
					c = c.Elem()
				}
			}
			if c.IsValid() && c.Kind() == reflect.Struct {
				nf := c.FieldByName("Name")
				if nf.IsValid() && nf.Kind() == reflect.String {
					out.ColorName = strings.TrimSpace(nf.String())
				}
				rf := c.FieldByName("RGB")
				if rf.IsValid() {
					out.ColorRGB = toInt(rf)
				}
			}
		}
	}

	// measurements: map[string]int
	if m := rv.FieldByName("Measurements"); m.IsValid() {
		if m.Kind() == reflect.Pointer {
			if !m.IsNil() {
				m = m.Elem()
			}
		}
		if m.IsValid() && m.Kind() == reflect.Map && m.Type().Key().Kind() == reflect.String {
			mp := make(map[string]int)
			iter := m.MapRange()
			for iter.Next() {
				k := strings.TrimSpace(iter.Key().String())
				if k == "" {
					continue
				}
				mp[k] = toInt(iter.Value())
			}
			out.Measurements = mp
		}
	}

	if out.Measurements == nil {
		out.Measurements = map[string]int{}
	}

	return out, true
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

func pickIntField(v any, fieldNames ...string) int {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return 0
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		return toInt(f)
	}
	return 0
}

func pickProductIDTagType(pb *pbdom.ProductBlueprint) string {
	if pb == nil {
		return ""
	}

	if s := pickStringField(*pb, "ProductIDTagType", "ProductIdTagType", "productIdTagType"); s != "" {
		return s
	}

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
// util
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

// invID = "{pb}__{tb}" を前提に split
func splitInventoryID(invID string) (pbID string, tbID string, ok bool) {
	invID = strings.TrimSpace(invID)
	if invID == "" {
		return "", "", false
	}
	parts := strings.SplitN(invID, "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}
