// backend/internal/application/tokenBlueprint/tokenBlueprint_usecase.go
package tokenBlueprint

import (
	"fmt"
	"os"

	resolver "narratives/internal/application/resolver"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
	"narratives/internal/infra/arweave"
)

type TokenBlueprintUsecase struct {
	crud     *TokenBlueprintCRUDUsecase
	icon     *TokenBlueprintIconUsecase
	query    *TokenBlueprintQueryUsecase
	content  *TokenBlueprintContentUsecase
	command  *TokenBlueprintCommandUsecase
	metadata *TokenBlueprintMetadataUsecase
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tbReviewRepo tbReview.RepositoryPort,
	nameResolver *resolver.NameResolver,
) *TokenBlueprintUsecase {
	if tbRepo == nil {
		panic(fmt.Errorf("NewTokenBlueprintUsecase: tbRepo is nil"))
	}

	crud := NewTokenBlueprintCRUDUsecase(tbRepo, tbReviewRepo)
	icon := NewTokenBlueprintIconUsecase(tbRepo)
	query := NewTokenBlueprintQueryUsecase(tbRepo, nameResolver)
	content := NewTokenBlueprintContentUsecase(tbRepo)
	command := NewTokenBlueprintCommandUsecase(tbRepo)

	// ------------------------------------------------------------
	// Arweave/Irys uploader
	// ------------------------------------------------------------
	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	metadata := NewTokenBlueprintMetadataUsecase(tbRepo, uploader)

	return &TokenBlueprintUsecase{
		crud:     crud,
		icon:     icon,
		query:    query,
		content:  content,
		command:  command,
		metadata: metadata,
	}
}
