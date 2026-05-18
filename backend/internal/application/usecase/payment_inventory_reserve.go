// backend/internal/application/usecase/payment_inventory_reserve.go
package usecase

/*
責任と機能:
- paid 後に inventory reserve を行うために、order から items を抽出し、
  inventoryId/modelId 単位で qty を集計して安定順に並べる。
*/

import (
	"sort"

	orderdom "narratives/internal/domain/order"
)

type reserveItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

func extractOrderItems(ord orderdom.Order) []reserveItem {
	if len(ord.Items) == 0 {
		return nil
	}

	out := make([]reserveItem, 0, len(ord.Items))
	for _, it := range ord.Items {
		if it.InventoryID == "" || it.ModelID == "" || it.Qty <= 0 {
			continue
		}

		out = append(out, reserveItem{
			InventoryID: it.InventoryID,
			ModelID:     it.ModelID,
			Qty:         it.Qty,
		})
	}

	if len(out) == 0 {
		return nil
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
		inv := it.InventoryID
		mdl := it.ModelID

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

		out = append(out, reserveItem{
			InventoryID: k.Inv,
			ModelID:     k.Mdl,
			Qty:         q,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].InventoryID == out[j].InventoryID {
			return out[i].ModelID < out[j].ModelID
		}
		return out[i].InventoryID < out[j].InventoryID
	})

	return out
}
