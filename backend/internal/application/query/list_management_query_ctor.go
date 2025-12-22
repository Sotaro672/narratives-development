// backend/internal/application/query/list_management_query_ctor.go
package query

import (
	resolver "narratives/internal/application/resolver"
)

// ✅ Lists 一覧（listManagement.tsx 用）Query の ctor
// - company boundary は invRows(ListByCurrentCompany) で作る
// - brand 解決は pbGetter/tbGetter + nameResolver(brandName) を使う
func NewListManagementQueryWithBrandInventoryAndInventoryRows(
	lister ListLister,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invRows InventoryRowsLister,
) *ListManagementQuery {
	return &ListManagementQuery{
		lister:       lister,
		nameResolver: nameResolver,
		pbGetter:     pbGetter,
		tbGetter:     tbGetter,
		invRows:      invRows,
	}
}
