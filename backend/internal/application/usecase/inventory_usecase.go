// backend/internal/application/usecase/inventory_usecase.go
package usecase

import (
	"context"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"

	invdom "narratives/internal/domain/inventory"
)

type InventoryUsecase struct {
	repo invdom.RepositoryPort
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// ============================================================
// Upsert entry from Mint by Model
// ============================================================
//
// - mint から在庫へ反映する唯一の入口
// - 在庫の蓄積は Stock（modelId -> {Products: ...}）で表現する前提
func (uc *InventoryUsecase) UpsertFromMintByModel(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	productIDs []string,
) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	mID := strings.TrimSpace(modelID)

	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}
	if mID == "" {
		return invdom.Mint{}, invdom.ErrInvalidModelID
	}

	ids := normalizeIDs(productIDs)
	if len(ids) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}

	// ✅ docId をここで確定（repo 側の sanitize と揃える）
	inventoryID := buildInventoryID(pbID, tbID)

	log.Printf(
		"[inventory_uc] UpsertFromMintByModel start inventoryId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q products=%d",
		inventoryID, tbID, pbID, mID, len(ids),
	)

	// 1) 既存取得
	current, err := uc.repo.GetByID(ctx, inventoryID)
	if err != nil {
		// ✅ NotFound の判定を堅牢化（inventory not found を取りこぼさない）
		if isNotFoundInventory(err) {
			// 2) 新規作成
			var m invdom.Mint
			setStringFieldIfExists(&m, "ID", inventoryID)
			setStringFieldIfExists(&m, "TokenBlueprintID", tbID)
			setStringFieldIfExists(&m, "ProductBlueprintID", pbID)

			// Stock[modelId] を構築してセット（※「蓄積」なので追加マージ）
			if err := upsertStockByModel(&m, mID, ids); err != nil {
				return invdom.Mint{}, err
			}

			created, cerr := uc.repo.Create(ctx, m)
			if cerr != nil {
				log.Printf("[inventory_uc] UpsertFromMintByModel create error inventoryId=%q err=%v", inventoryID, cerr)
				return invdom.Mint{}, cerr
			}

			log.Printf("[inventory_uc] UpsertFromMintByModel create ok inventoryId=%q", inventoryID)
			return created, nil
		}

		log.Printf("[inventory_uc] UpsertFromMintByModel GetByID error inventoryId=%q err=%v", inventoryID, err)
		return invdom.Mint{}, err
	}

	// 3) 既存更新
	// 念のためID/TB/PBを揃える
	setStringFieldIfExists(&current, "ID", inventoryID)
	setStringFieldIfExists(&current, "TokenBlueprintID", tbID)
	setStringFieldIfExists(&current, "ProductBlueprintID", pbID)

	if err := upsertStockByModel(&current, mID, ids); err != nil {
		return invdom.Mint{}, err
	}

	updated, uerr := uc.repo.Update(ctx, current)
	if uerr != nil {
		log.Printf("[inventory_uc] UpsertFromMintByModel update error inventoryId=%q err=%v", inventoryID, uerr)
		return invdom.Mint{}, uerr
	}

	log.Printf("[inventory_uc] UpsertFromMintByModel update ok inventoryId=%q", inventoryID)
	return updated, nil
}

// ============================================================
// ✅ NEW: Reserve by Order (payment success -> invoice.paid=true と同時に呼ぶ想定)
// ============================================================

type ReserveByOrderItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

// ReserveByOrder adds (orderID -> qty) into Stock[modelId].ReservedByOrder
// and updates ReservedCount accordingly.
//
// Intended timing:
// - payment 起票/確定（paid/succeeded）
// - invoice.paid=true に更新
// - then ReserveByOrder(orderID, items)
func (uc *InventoryUsecase) ReserveByOrder(ctx context.Context, orderID string, items []ReserveByOrderItem) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}

	oid := strings.TrimSpace(orderID)
	if oid == "" {
		return errors.New("inventory reserve: invalid orderId")
	}
	if len(items) == 0 {
		// 何もしない（呼び出し側が「対象なし」でも安全）
		return nil
	}

	for _, it := range items {
		invID := strings.TrimSpace(it.InventoryID)
		mid := strings.TrimSpace(it.ModelID)
		qty := it.Qty
		if invID == "" || mid == "" || qty <= 0 {
			return errors.New("inventory reserve: invalid item")
		}

		m, err := uc.repo.GetByID(ctx, invID)
		if err != nil {
			return err
		}

		if err := reserveStockByModelOrder(&m, mid, oid, qty); err != nil {
			return err
		}

		if _, err := uc.repo.Update(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// CRUD (raw persistence access; no legacy fields assumed here)
// ============================================================

func (uc *InventoryUsecase) Create(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.Create(ctx, m)
}

func (uc *InventoryUsecase) Update(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	if strings.TrimSpace(getStringFieldIfExists(m, "ID")) == "" && strings.TrimSpace(getStringFieldIfExists(m, "Id")) == "" {
		// 基本は ID フィールドを想定するが、念のため Id も見る
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.Update(ctx, m)
}

func (uc *InventoryUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrInvalidMintID
	}
	return uc.repo.Delete(ctx, id)
}

// ============================================================
// Queries
// ============================================================

func (uc *InventoryUsecase) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *InventoryUsecase) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByTokenBlueprintID(ctx, tokenBlueprintID)
}

func (uc *InventoryUsecase) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

func (uc *InventoryUsecase) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByModelID(ctx, modelID)
}

func (uc *InventoryUsecase) ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByTokenAndModelID(ctx, tokenBlueprintID, modelID)
}

// ============================================================
// Helpers
// ============================================================

func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		// Firestore docId に "/" が入ると階層扱いになるので repo と揃えて潰す
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	pb := sanitize(productBlueprintID)
	tb := sanitize(tokenBlueprintID)
	return pb + "__" + tb
}

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// isNotFoundInventory: repo が返す NotFound を取りこぼさないためのヘルパ
func isNotFoundInventory(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, invdom.ErrNotFound) {
		return true
	}
	// 念のため文字列も見る（"inventory not found" など）
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "not found")
}

// reserveStockByModelOrder updates reservation fields on Stock[modelID]:
// - ReservedByOrder[orderID] += qty
// - ReservedCount = sum(ReservedByOrder)
//
// It does NOT change Products / Accumulation.
func reserveStockByModelOrder(m *invdom.Mint, modelID, orderID string, qty int) error {
	if m == nil {
		return errors.New("mint is nil")
	}
	modelID = strings.TrimSpace(modelID)
	orderID = strings.TrimSpace(orderID)
	if modelID == "" {
		return invdom.ErrInvalidModelID
	}
	if orderID == "" {
		return errors.New("inventory reserve: invalid orderId")
	}
	if qty <= 0 {
		return errors.New("inventory reserve: invalid qty")
	}

	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("mint must be a non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("mint must be a struct")
	}

	stock := rv.FieldByName("Stock")
	if !stock.IsValid() || !stock.CanSet() {
		return errors.New("inventory.Mint.Stock is missing or cannot be set")
	}
	if stock.Kind() != reflect.Map {
		return errors.New("inventory.Mint.Stock must be a map")
	}
	if stock.IsNil() {
		return errors.New("inventory.Mint.Stock is nil (no stock to reserve)")
	}

	key := reflect.ValueOf(modelID)
	if key.Type() != stock.Type().Key() {
		return errors.New("inventory.Mint.Stock key type is not string")
	}

	existing := stock.MapIndex(key)
	if !existing.IsValid() {
		// 在庫に該当 model が無い（注文と在庫の整合が崩れている）
		return errors.New("inventory reserve: model stock not found")
	}

	valType := stock.Type().Elem()

	// 変更対象を作る（map要素がstructならコピーして編集→SetMapIndex）
	var val reflect.Value
	switch valType.Kind() {
	case reflect.Struct:
		val = reflect.New(valType).Elem()
		if existing.Type() == valType {
			val.Set(existing)
		} else {
			// 型が合わない場合は触れない
			return errors.New("inventory reserve: invalid stock value type")
		}
		if err := applyReserveToModelStockValue(val, orderID, qty); err != nil {
			return err
		}
		stock.SetMapIndex(key, val)
		return nil

	case reflect.Pointer:
		// *Struct を想定
		if existing.Type() != valType {
			return errors.New("inventory reserve: invalid stock value type")
		}
		val = existing
		if val.IsNil() {
			// nil pointer は想定外だが安全に作る
			val = reflect.New(valType.Elem())
		}
		ev := val.Elem()
		if ev.Kind() != reflect.Struct {
			return errors.New("inventory reserve: stock value must be struct")
		}
		if err := applyReserveToModelStockValue(ev, orderID, qty); err != nil {
			return err
		}
		stock.SetMapIndex(key, val)
		return nil

	default:
		return errors.New("inventory reserve: stock value must be struct or *struct")
	}
}

func applyReserveToModelStockValue(stockStruct reflect.Value, orderID string, qty int) error {
	if !stockStruct.IsValid() || stockStruct.Kind() != reflect.Struct {
		return errors.New("inventory reserve: invalid model stock struct")
	}

	changed := false

	// ReservedByOrder map[string]int (best-effort: int kinds)
	rf := stockStruct.FieldByName("ReservedByOrder")
	if rf.IsValid() && rf.CanSet() && rf.Kind() == reflect.Map && rf.Type().Key().Kind() == reflect.String {
		if rf.IsNil() {
			rf.Set(reflect.MakeMap(rf.Type()))
		}

		okey := reflect.ValueOf(orderID)
		if okey.Type() != rf.Type().Key() {
			okey = okey.Convert(rf.Type().Key())
		}

		cur := 0
		if v := rf.MapIndex(okey); v.IsValid() {
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				cur = int(v.Int())
			case reflect.Float32, reflect.Float64:
				cur = int(v.Float())
			}
		}
		next := cur + qty

		nv := reflect.New(rf.Type().Elem()).Elem()
		switch nv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			nv.SetInt(int64(next))
		case reflect.Float32, reflect.Float64:
			nv.SetFloat(float64(next))
		default:
			// elem 型が想定外なら予約は入れない
			return errors.New("inventory reserve: ReservedByOrder value type unsupported")
		}

		rf.SetMapIndex(okey, nv)
		changed = true

		// ReservedCount = sum(ReservedByOrder)
		sum := 0
		iter := rf.MapRange()
		for iter.Next() {
			v := iter.Value()
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sum += int(v.Int())
			case reflect.Float32, reflect.Float64:
				sum += int(v.Float())
			}
		}

		cf := stockStruct.FieldByName("ReservedCount")
		if cf.IsValid() && cf.CanSet() {
			switch cf.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				cf.SetInt(int64(sum))
				changed = true
			}
		}
	}

	if !changed {
		return errors.New("inventory reserve: reservation fields not found on model stock")
	}
	return nil
}

// upsertStockByModel merges Stock[modelId].Products with productIDs,
// sets Accumulation, and keeps/initializes reservation fields (ReservedByOrder/ReservedCount) if they exist.
//
// - Stock が map である前提
// - Products が []string / map[string]bool / map[string]struct{} などでも対応
// - entity.go の拡張（ReservedByOrder/ReservedCount）に追随
func upsertStockByModel(m *invdom.Mint, modelID string, productIDs []string) error {
	if m == nil {
		return errors.New("mint is nil")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return invdom.ErrInvalidModelID
	}

	add := normalizeIDs(productIDs)
	if len(add) == 0 {
		return invdom.ErrInvalidProducts
	}

	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("mint must be a non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("mint must be a struct")
	}

	stock := rv.FieldByName("Stock")
	if !stock.IsValid() || !stock.CanSet() {
		return errors.New("inventory.Mint.Stock is missing or cannot be set")
	}
	if stock.Kind() != reflect.Map {
		return errors.New("inventory.Mint.Stock must be a map")
	}

	// Stock map を初期化
	if stock.IsNil() {
		stock.Set(reflect.MakeMap(stock.Type()))
	}

	key := reflect.ValueOf(modelID)
	if key.Type() != stock.Type().Key() {
		return errors.New("inventory.Mint.Stock key type is not string")
	}

	// 既存の Stock[modelID] があれば、それをベースに「追加マージ」する
	existing := stock.MapIndex(key)

	// ---- merge product IDs ----
	mergedSet := map[string]struct{}{}

	// 既存分
	if existing.IsValid() && existing.Kind() != reflect.Invalid && !(existing.Kind() == reflect.Ptr && existing.IsNil()) {
		for _, id := range extractProductIDsFromModelStockValue(existing) {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			mergedSet[id] = struct{}{}
		}
	}
	// 追加分
	for _, id := range add {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		mergedSet[id] = struct{}{}
	}

	merged := make([]string, 0, len(mergedSet))
	for id := range mergedSet {
		merged = append(merged, id)
	}
	sort.Strings(merged)

	valType := stock.Type().Elem()

	// val を作る（struct の場合は既存を可能な限り維持）
	var val reflect.Value
	if existing.IsValid() && existing.Type() == valType {
		val = existing
	} else {
		val = reflect.New(valType).Elem()
	}

	if val.Kind() != reflect.Struct {
		return errors.New("inventory.Mint.Stock value must be a struct (ModelStock)")
	}

	// (optional) val.ModelID = modelID があればセット
	if f := val.FieldByName("ModelID"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(modelID)
	}

	// ---- set Products ----
	if pf := val.FieldByName("Products"); pf.IsValid() && pf.CanSet() {
		switch pf.Kind() {
		case reflect.Slice:
			// []string 想定
			if pf.Type().Elem().Kind() == reflect.String {
				s := reflect.MakeSlice(pf.Type(), 0, len(merged))
				for _, id := range merged {
					s = reflect.Append(s, reflect.ValueOf(id))
				}
				pf.Set(s)
			}
		case reflect.Map:
			// map[string]bool / map[string]struct{} 想定
			if pf.Type().Key().Kind() == reflect.String {
				mm := reflect.MakeMapWithSize(pf.Type(), len(merged))
				for _, id := range merged {
					var mv reflect.Value
					switch pf.Type().Elem().Kind() {
					case reflect.Bool:
						mv = reflect.ValueOf(true).Convert(pf.Type().Elem())
					case reflect.Struct:
						mv = reflect.New(pf.Type().Elem()).Elem() // struct{} など
					default:
						mv = reflect.Zero(pf.Type().Elem())
					}
					mm.SetMapIndex(reflect.ValueOf(id), mv)
				}
				pf.Set(mm)
			}
		}
	}

	// ---- set Accumulation = len(products) ----
	if af := val.FieldByName("Accumulation"); af.IsValid() && af.CanSet() {
		switch af.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			af.SetInt(int64(len(merged)))
		}
	}

	// ---- reservation fields (keep existing; init for new) ----
	// ReservedByOrder: map[string]int
	var reservedSum int64 = -1
	if rf := val.FieldByName("ReservedByOrder"); rf.IsValid() && rf.CanSet() && rf.Kind() == reflect.Map {
		if rf.IsNil() {
			rf.Set(reflect.MakeMap(rf.Type()))
		}
		// best-effort sum
		if rf.Type().Key().Kind() == reflect.String {
			var sum int64
			iter := rf.MapRange()
			for iter.Next() {
				v := iter.Value()
				switch v.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					sum += v.Int()
				case reflect.Float32, reflect.Float64:
					sum += int64(v.Float())
				}
			}
			reservedSum = sum
		}
	}

	// ReservedCount: int
	if cf := val.FieldByName("ReservedCount"); cf.IsValid() && cf.CanSet() {
		switch cf.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if reservedSum >= 0 {
				cf.SetInt(reservedSum)
			}
		}
	}

	// Stock[modelID] = val
	stock.SetMapIndex(key, val)

	// ModelIDs に modelID を入れておく（検索補助）
	_ = upsertModelIDOnMint(m, modelID)

	return nil
}

// ensure mint.ModelIDs contains modelID (best-effort; keeps reflection style)
func upsertModelIDOnMint(m *invdom.Mint, modelID string) error {
	if m == nil {
		return nil
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}

	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return nil
	}

	f := rv.FieldByName("ModelIDs")
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.Slice || f.Type().Elem().Kind() != reflect.String {
		return nil
	}

	seen := map[string]struct{}{}
	for i := 0; i < f.Len(); i++ {
		s := strings.TrimSpace(f.Index(i).String())
		if s != "" {
			seen[s] = struct{}{}
		}
	}
	if _, ok := seen[modelID]; ok {
		return nil
	}

	f.Set(reflect.Append(f, reflect.ValueOf(modelID)))
	return nil
}

// extractProductIDsFromModelStockValue reads Products from a model-stock value (struct) into []string.
func extractProductIDsFromModelStockValue(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	pf := v.FieldByName("Products")
	if !pf.IsValid() {
		return nil
	}

	switch pf.Kind() {
	case reflect.Slice:
		if pf.Type().Elem().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, pf.Len())
		for i := 0; i < pf.Len(); i++ {
			s := strings.TrimSpace(pf.Index(i).String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	case reflect.Map:
		if pf.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, pf.Len())
		iter := pf.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}

func setStringFieldIfExists(target any, fieldName string, value string) {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.String {
		return
	}
	f.SetString(strings.TrimSpace(value))
}

func getStringFieldIfExists(target any, fieldName string) string {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Ptr {
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
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	return ""
}
