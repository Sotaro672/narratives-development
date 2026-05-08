// backend\internal\platform\di\mall\catalog_composite.go
package mall

import (
	"net/http"
	"strings"
)

// ✅ catalog composite: catalog handler 1つだけ登録し、内部で productBlueprint reviews へ振り分ける
func newCatalogCompositeHandler(catalog http.Handler, pbReview http.Handler) http.Handler {
	if catalog == nil {
		catalog = http.NotFoundHandler()
	}
	if pbReview == nil {
		// review が無いなら catalog のみ
		return catalog
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path

		// review endpoints under catalog:
		//  - /mall/catalog/product-blueprints/{id}/reviews
		//  - /mall/me/catalog/product-blueprints/{id}/reviews
		//
		// NOTE: router は /mall/catalog と /mall/me/catalog をこの handler に登録しているため
		//       ここで me/public を両方ハンドリングする
		if (strings.HasPrefix(p, "/mall/catalog/") || strings.HasPrefix(p, "/mall/me/catalog/")) &&
			strings.Contains(p, "/product-blueprints/") &&
			strings.Contains(p, "/reviews") {
			pbReview.ServeHTTP(w, r)
			return
		}

		catalog.ServeHTTP(w, r)
	})
}