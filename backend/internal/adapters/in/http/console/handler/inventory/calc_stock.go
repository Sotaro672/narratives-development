// backend/internal/adapters/in/http/console/handler/inventory/calc_stock.go
package inventory

import invdom "narratives/internal/domain/inventory"

// totalAccumulation: 物理在庫（= Products 数 / Accumulation）
func totalAccumulation(m invdom.Mint) int {
	total := 0
	for _, ms := range m.Stock {
		total += modelStockAccumulation(ms)
	}
	if total < 0 {
		return 0
	}
	return total
}

// totalReserved: 予約数（= reservedCount / reservedByOrder 合計）
func totalReserved(m invdom.Mint) int {
	total := 0
	for _, ms := range m.Stock {
		total += modelStockReserved(ms)
	}
	if total < 0 {
		return 0
	}
	return total
}

// totalAvailable: 表示用の “引当後在庫” = accumulation - reservedCount
func totalAvailable(m invdom.Mint) int {
	total := 0
	for _, ms := range m.Stock {
		acc := modelStockAccumulation(ms)
		res := modelStockReserved(ms)
		avail := acc - res
		if avail < 0 {
			avail = 0
		}
		total += avail
	}
	if total < 0 {
		return 0
	}
	return total
}

func modelStockAccumulation(ms invdom.ModelStock) int {
	// 正: Accumulation（正規化済み）
	if ms.Accumulation > 0 {
		return ms.Accumulation
	}
	// 保険: Products の数
	return len(ms.Products)
}

func modelStockReserved(ms invdom.ModelStock) int {
	// 正: ReservedCount（正規化済み）
	if ms.ReservedCount > 0 {
		return ms.ReservedCount
	}
	// 保険: ReservedByOrder 合計
	sum := 0
	for _, n := range ms.ReservedByOrder {
		if n > 0 {
			sum += n
		}
	}
	if sum < 0 {
		return 0
	}
	return sum
}
