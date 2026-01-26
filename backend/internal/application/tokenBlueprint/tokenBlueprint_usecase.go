// backend/internal/application/tokenBlueprint/tokenBlueprint_usecase.go
package tokenBlueprint

import (
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/storage"

	resolver "narratives/internal/application/resolver"
	tbdom "narratives/internal/domain/tokenBlueprint"
	"narratives/internal/infra/arweave"
)

type TokenBlueprintUsecase struct {
	crud     *TokenBlueprintCRUDUsecase
	icon     *TokenBlueprintIconUsecase
	query    *TokenBlueprintQueryUsecase
	content  *TokenBlueprintContentUsecase
	command  *TokenBlueprintCommandUsecase
	metadata *TokenBlueprintMetadataUsecase

	// ★追加
	buckets *TokenBlueprintBucketUsecase
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	nameResolver *resolver.NameResolver, // ★変更: memberSvc ではなく resolver を受け取る
	gcsClient *storage.Client, // ★追加: .keep 作成に必要
) *TokenBlueprintUsecase {

	if tbRepo == nil {
		// ここで panic せず、起動時に原因が分かるよう明示
		// （DI 側のログは別途出る想定）
		panic(fmt.Errorf("NewTokenBlueprintUsecase: tbRepo is nil"))
	}
	if gcsClient == nil {
		panic(fmt.Errorf("NewTokenBlueprintUsecase: gcsClient is nil"))
	}

	crud := NewTokenBlueprintCRUDUsecase(tbRepo)
	icon := NewTokenBlueprintIconUsecase(tbRepo)

	// ★変更: Query は resolver のみで名前解決する
	query := NewTokenBlueprintQueryUsecase(tbRepo, nameResolver)

	content := NewTokenBlueprintContentUsecase(tbRepo)
	command := NewTokenBlueprintCommandUsecase(tbRepo)

	// ------------------------------------------------------------
	// ★追加: Arweave/Irys uploader (Cloud Run) を注入して metadataUri を生成
	// ------------------------------------------------------------
	baseURL := strings.TrimSpace(os.Getenv("ARWEAVE_BASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("IRYS_SERVICE_API_KEY"))
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	metadata := NewTokenBlueprintMetadataUsecase(tbRepo, uploader)

	// ★変更: GCS client を注入
	buckets := NewTokenBlueprintBucketUsecase(gcsClient)

	return &TokenBlueprintUsecase{
		crud:     crud,
		icon:     icon,
		query:    query,
		content:  content,
		command:  command,
		metadata: metadata,
		buckets:  buckets,
	}
}
