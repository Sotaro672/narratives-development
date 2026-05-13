// backend\internal\domain\common\product_category.go
package common

type ProductCategoryKind string

const (
	ProductCategoryKindApparel    ProductCategoryKind = "apparel"
	ProductCategoryKindFood       ProductCategoryKind = "food"
	ProductCategoryKindAlcohol    ProductCategoryKind = "alcohol"
	ProductCategoryKindCosmetics  ProductCategoryKind = "cosmetics"
	ProductCategoryKindGoods      ProductCategoryKind = "goods"
	ProductCategoryKindHealthcare ProductCategoryKind = "healthcare"
	ProductCategoryKindOther      ProductCategoryKind = "other"
)

func IsValidProductCategoryKind(v ProductCategoryKind) bool {
	switch v {
	case ProductCategoryKindApparel,
		ProductCategoryKindFood,
		ProductCategoryKindAlcohol,
		ProductCategoryKindCosmetics,
		ProductCategoryKindGoods,
		ProductCategoryKindHealthcare,
		ProductCategoryKindOther:
		return true
	default:
		return false
	}
}
