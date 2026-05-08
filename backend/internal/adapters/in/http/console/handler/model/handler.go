// backend/internal/adapters/in/http/console/handler/model/handler.go
package model

import (
	"net/http"

	usecase "narratives/internal/application/usecase"
)

// ModelHandler は /models 関連のエンドポイントを担当します。
type ModelHandler struct {
	uc *usecase.ModelUsecase
}

// NewModelHandler はHTTPハンドラを初期化します。
func NewModelHandler(uc *usecase.ModelUsecase) http.Handler {
	return &ModelHandler{uc: uc}
}
