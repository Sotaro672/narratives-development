// backend/internal/platform/di/sns_container.go
package di

import (
	"net/http"

	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	snsquery "narratives/internal/application/query/sns"
	usecase "narratives/internal/application/usecase"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
// Keep it minimal and independent from console/admin routing.
type SNSDeps struct {
	// Handlers
	List http.Handler
}

// NewSNSDeps wires SNS handlers.
//
// ✅ SNS は companyId 境界が無い（公開一覧）ため、console 用 query は使わない。
// ✅ SNSListQuery を注入することで、status=listing のみを companyId 非依存で取得できる。
//
// Minimum:
// - q があれば /sns/lists と /sns/lists/{id} が動く
// Optional:
// - listUC は将来の互換/フォールバック用（基本は不要）
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	q *snsquery.SNSListQuery,
) SNSDeps {
	var listHandler http.Handler

	switch {
	case q != nil:
		// ✅ preferred: SNS query
		listHandler = snshandler.NewSNSListHandlerWithQueries(listUC, q)
	case listUC != nil:
		// fallback (company boundary を要求する実装だと SNS では期待通り動かない可能性あり)
		listHandler = snshandler.NewSNSListHandler(listUC)
	default:
		listHandler = nil
	}

	return SNSDeps{
		List: listHandler,
	}
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
//
// Routes:
// - GET /sns/lists
// - GET /sns/lists/{id}
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}
	snshttp.Register(mux, snshttp.Deps{
		List: deps.List,
	})
}
