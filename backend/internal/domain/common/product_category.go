// backend/internal/domain/common/product_category.go

package common

type ProductCategoryKind string

const (
	ProductCategoryKindApparel    ProductCategoryKind = "apparel"
	ProductCategoryKindAlcohol    ProductCategoryKind = "alcohol"
	ProductCategoryKindCosmetics  ProductCategoryKind = "cosmetics"
	ProductCategoryKindHealthcare ProductCategoryKind = "healthcare"
	ProductCategoryKindOther      ProductCategoryKind = "other"
)

func IsValidProductCategoryKind(v ProductCategoryKind) bool {
	switch v {
	case ProductCategoryKindApparel,
		ProductCategoryKindAlcohol,
		ProductCategoryKindCosmetics,
		ProductCategoryKindHealthcare,
		ProductCategoryKindOther:
		return true
	default:
		return false
	}
}
