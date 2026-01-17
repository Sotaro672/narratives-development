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
//
// ✅ 修正方針:
//   - 既存 model の追加反映が反射経由の Get->merge->Update で失敗し得るため、
//     repo の atomic upsert（transaction + UNION）に委譲する。
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

	// docId をここで確定（repo 側の sanitize と揃える）
	inventoryID := buildInventoryID(pbID, tbID)

	log.Printf(
		"[inventory_uc] UpsertFromMintByModel start inventoryId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q products=%d",
		inventoryID, tbID, pbID, mID, len(ids),
	)

	// ✅ repo の atomic upsert に委譲（既存 model でも UNION で確実に追記される）
	updated, err := uc.repo.UpsertByProductBlueprintAndToken(ctx, tbID, pbID, mID, ids)
	if err != nil {
		log.Printf(
			"[inventory_uc] UpsertFromMintByModel upsert error inventoryId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q err=%v",
			inventoryID, tbID, pbID, mID, err,
		)
		return invdom.Mint{}, err
	}

	log.Printf("[inventory_uc] UpsertFromMintByModel upsert ok inventoryId=%q", inventoryID)
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
