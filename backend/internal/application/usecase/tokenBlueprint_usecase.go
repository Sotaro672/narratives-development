// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"fmt"

	"cloud.google.com/go/storage"

	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
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
	memberSvc *memdom.Service,
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
	query := NewTokenBlueprintQueryUsecase(tbRepo, memberSvc)
	content := NewTokenBlueprintContentUsecase(tbRepo)
	command := NewTokenBlueprintCommandUsecase(tbRepo)
	metadata := NewTokenBlueprintMetadataUsecase(tbRepo)

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
