package sns

import (
	"net/http"

	usecase "narratives/internal/application/usecase"

	snshandler "narratives/internal/adapters/in/http/sns/handler"
)

// Deps: SNS 側で公開する handler を束ねる
type Deps struct {
	List http.Handler
}

// Register registers SNS routes onto mux.
// NOTE: ServeMux は "/sns/lists" と "/sns/lists/" の両方を登録しておくと安全。
func Register(mux *http.ServeMux, deps Deps) {
	if mux == nil {
		return
	}
	if deps.List != nil {
		mux.Handle("/sns/lists", deps.List)
		mux.Handle("/sns/lists/", deps.List)
	}
}

// NewDeps builds SNS handlers from application layer deps.
// - SNS は購入客向けの read-only 想定なので、List の参照(usecase/query)だけ注入すれば十分。
func NewDeps(listUC *usecase.ListUsecase) Deps {
	if listUC == nil {
		return Deps{}
	}

	return Deps{
		List: snshandler.NewSNSListHandler(listUC),
	}
}
