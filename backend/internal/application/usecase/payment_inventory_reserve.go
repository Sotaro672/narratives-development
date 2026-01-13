// backend/internal/application/usecase/payment_inventory_reserve.go
package usecase

/*
責任と機能:
- paid 後に inventory reserve を行うために、order から items を抽出し、
  inventoryId/modelId 単位で qty を集計して安定順に並べる。
- inventoryId が肥大化しているケースに備え、docId(product__token) に正規化する。
*/

import (
	"reflect"
	"sort"
	"strings"
)

type reserveItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

func extractOrderItemsBestEffort(orderAny any) []reserveItem {
	if orderAny == nil {
		return nil
	}

	// order.Items (slice/array) を探す
	sv, ok := getSliceFieldBestEffort(orderAny, "Items", "items")
	if !ok {
		return nil
	}

	out := make([]reserveItem, 0, sv.Len())
	for i := 0; i < sv.Len(); i++ {
		el := sv.Index(i)
		if !el.IsValid() {
			continue
		}
		if el.Kind() == reflect.Pointer {
			if el.IsNil() {
				continue
			}
			el = el.Elem()
		}

		// struct / map の両対応
		var invID, modelID string
		var qty int

		switch el.Kind() {
		case reflect.Struct:
			invID = getStringFieldFromValueBestEffort(el, "InventoryID", "InventoryId", "inventoryId")
			modelID = getStringFieldFromValueBestEffort(el, "ModelID", "ModelId", "modelId")
			qty = getIntFieldFromValueBestEffort(el, "Qty", "qty", "Quantity", "quantity")
		case reflect.Map:
			// map[string]any
			invID = getStringMapValueBestEffort(el, "inventoryId", "inventoryID", "InventoryID", "InventoryId")
			modelID = getStringMapValueBestEffort(el, "modelId", "modelID", "ModelID", "ModelId")
			qty = getIntMapValueBestEffort(el, "qty", "Qty", "quantity", "Quantity")
		default:
			continue
		}

		invID = strings.TrimSpace(invID)
		modelID = strings.TrimSpace(modelID)
		if qty <= 0 || invID == "" || modelID == "" {
			continue
		}

		out = append(out, reserveItem{InventoryID: invID, ModelID: modelID, Qty: qty})
	}

	return out
}

func aggregateReserveItems(items []reserveItem) []reserveItem {
	if len(items) == 0 {
		return nil
	}

	type key struct {
		Inv string
		Mdl string
	}
	m := map[key]int{}
	for _, it := range items {
		inv := strings.TrimSpace(it.InventoryID)
		mdl := strings.TrimSpace(it.ModelID)
		if inv == "" || mdl == "" || it.Qty <= 0 {
			continue
		}
		m[key{Inv: inv, Mdl: mdl}] += it.Qty
	}

	out := make([]reserveItem, 0, len(m))
	for k, q := range m {
		if q <= 0 {
			continue
		}
		out = append(out, reserveItem{InventoryID: k.Inv, ModelID: k.Mdl, Qty: q})
	}

	// stable order for logs (optional)
	sort.Slice(out, func(i, j int) bool {
		if out[i].InventoryID == out[j].InventoryID {
			return out[i].ModelID < out[j].ModelID
		}
		return out[i].InventoryID < out[j].InventoryID
	})

	return out
}

// cart/order/items で使っている inventoryId が
//
//	product__token__list__model
//
// のように肥大化しているケースがあるため、docId(product__token) に正規化する。
func normalizeInventoryDocIDBestEffort(inventoryID string) string {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ""
	}

	parts := strings.Split(inventoryID, "__")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]) + "__" + strings.TrimSpace(parts[1])
	}
	return inventoryID
}
