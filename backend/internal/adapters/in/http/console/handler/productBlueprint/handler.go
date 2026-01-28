// backend\internal\adapters\in\http\console\handler\productBlueprint\handler.go
package productBlueprint

import (
	"net/http"

	pbuc "narratives/internal/application/productBlueprint/usecase"
	brand "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
)

// Handler は ProductBlueprint 用の HTTP ハンドラです。
type Handler struct {
	uc        *pbuc.ProductBlueprintUsecase
	brandSvc  *brand.Service
	memberSvc *memdom.Service
}

func NewProductBlueprintHandler(
	uc *pbuc.ProductBlueprintUsecase,
	brandSvc *brand.Service,
	memberSvc *memdom.Service,
) http.Handler {
	return &Handler{
		uc:        uc,
		brandSvc:  brandSvc,
		memberSvc: memberSvc,
	}
}
