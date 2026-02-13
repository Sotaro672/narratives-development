// backend\internal\application\query\mall\catalog\catalog_query_stock.go
package catalogQuery

import (
	dto "narratives/internal/application/query/mall/dto"
)

func stockKeyCount(stock map[string]dto.CatalogInventoryModelStockDTO) int {
	return len(stock)
}

// attachStockToModelVariations sets StockKeys only.
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
