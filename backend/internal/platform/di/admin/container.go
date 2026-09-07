// backend/internal/platform/di/admin/container.go
package admin

import (
	"context"
	"errors"
	"os"
	"strings"

	fsrepo "narratives/internal/adapters/out/firestore"
	adminquery "narratives/internal/application/query/admin"
	usecase "narratives/internal/application/usecase"
	solanainfra "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

const (
	adminFirebaseUIDEnv = "AMOL_ADMIN_FIREBASE_UID"
	adminEmailEnv       = "AMOL_ADMIN_EMAIL"
)

type Container struct {
	Infra *shared.Infra

	adminFirebaseUID               string
	adminEmail                     string
	contactUsecase                 *usecase.ContactUsecase
	reportUsecase                  *usecase.ReportUsecase
	companyRepo                    *fsrepo.CompanyRepositoryFS
	memberRepo                     *fsrepo.MemberRepositoryFS
	avatarRepo                     *fsrepo.AvatarRepositoryFS
	brandRepo                      *fsrepo.BrandRepositoryFS
	productBlueprintRepo           *fsrepo.ProductBlueprintRepositoryFS
	tokenBlueprintRepo             *fsrepo.TokenBlueprintRepositoryFS
	reportDecisionNotificationRepo *fsrepo.ReportDecisionNotificationRepositoryFS
	reportNameQuery                *adminquery.ReportNameQuery
	gasBalanceQuery                *adminquery.GasBalanceQuery
}

func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
	if infra == nil {
		var err error
		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}

	if infra == nil {
		return nil, errors.New("di.admin: shared infra is nil")
	}
	if infra.FirebaseAuth == nil {
		return nil, errors.New("di.admin: firebase auth is nil")
	}
	if infra.Firestore == nil {
		return nil, errors.New("di.admin: firestore is nil")
	}

	adminFirebaseUID := strings.TrimSpace(os.Getenv(adminFirebaseUIDEnv))
	if adminFirebaseUID == "" {
		return nil, errors.New("di.admin: AMOL_ADMIN_FIREBASE_UID is empty")
	}

	adminEmail := strings.TrimSpace(os.Getenv(adminEmailEnv))
	if adminEmail == "" {
		return nil, errors.New("di.admin: AMOL_ADMIN_EMAIL is empty")
	}

	contactRepo := fsrepo.NewContactRepositoryFS(infra.Firestore)
	contactUsecase := usecase.NewContactUsecase(contactRepo, nil, nil)

	companyRepo := fsrepo.NewCompanyRepositoryFS(infra.Firestore)
	memberRepo := fsrepo.NewMemberRepositoryFS(infra.Firestore)
	avatarRepo := fsrepo.NewAvatarRepositoryFS(infra.Firestore)
	brandRepo := fsrepo.NewBrandRepositoryFS(infra.Firestore)
	productBlueprintRepo := fsrepo.NewProductBlueprintRepositoryFS(infra.Firestore)
	tokenBlueprintRepo := fsrepo.NewTokenBlueprintRepositoryFS(infra.Firestore)

	reportNameQuery := adminquery.NewReportNameQuery(
		avatarRepo,
		brandRepo,
		companyRepo,
		memberRepo,
		productBlueprintRepo,
		tokenBlueprintRepo,
	)

	reportRepo := fsrepo.NewReportRepositoryFS(infra.Firestore)
	if reportRepo == nil {
		return nil, errors.New("di.admin: report repository is nil")
	}

	reportDecisionNotificationRepo := fsrepo.NewReportDecisionNotificationRepositoryFS(infra.Firestore)
	if reportDecisionNotificationRepo == nil {
		return nil, errors.New("di.admin: report decision notification repository is nil")
	}

	productBlueprintReviewRepo := fsrepo.NewProductBlueprintReviewRepositoryFS(infra.Firestore)
	if productBlueprintReviewRepo == nil {
		return nil, errors.New("di.admin: product blueprint review repository is nil")
	}

	productBlueprintReviewUsecase := usecase.NewProductBlueprintReviewUsecase(
		productBlueprintReviewRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if productBlueprintReviewUsecase == nil {
		return nil, errors.New("di.admin: product blueprint review usecase is nil")
	}

	tokenBlueprintReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(infra.Firestore)
	if tokenBlueprintReviewRepo == nil {
		return nil, errors.New("di.admin: token blueprint review repository is nil")
	}

	tokenBlueprintReviewUsecase := usecase.NewTokenBlueprintReviewUsecase(
		tokenBlueprintReviewRepo,
		nil,
		nil,
		nil,
	)
	if tokenBlueprintReviewUsecase == nil {
		return nil, errors.New("di.admin: token blueprint review usecase is nil")
	}

	// Admin側のTokenBlueprintUsecaseは通報裁定によるAMOL上の非表示専用。
	// TokenBlueprint本体、Firebase Storage、metadataUri、
	// オンチェーン上のトークン・メタデータは削除しない。
	tokenBlueprintUsecase := usecase.NewTokenBlueprintUsecase(
		tokenBlueprintRepo,
		nil,
		nil,
		nil,
	)
	if tokenBlueprintUsecase == nil {
		return nil, errors.New("di.admin: token blueprint usecase is nil")
	}

	resaleRepo := fsrepo.NewResaleRepositoryFS(infra.Firestore)
	if resaleRepo == nil {
		return nil, errors.New("di.admin: resale repository is nil")
	}

	cartRepo := fsrepo.NewCartRepositoryFS(infra.Firestore)
	if cartRepo == nil {
		return nil, errors.New("di.admin: cart repository is nil")
	}

	// Admin側のResaleUsecaseはアバター通報裁定による再販停止専用。
	// 出品作成・画像操作は行わないため、imageRepo / imageStorage /
	// product identity repositories は不要。
	resaleUsecase := usecase.NewResaleUsecase(
		resaleRepo,
		nil,
		nil,
	).WithCartItemCleanup(
		cartRepo,
	)
	if resaleUsecase == nil {
		return nil, errors.New("di.admin: resale usecase is nil")
	}

	reportUsecase := usecase.NewReportUsecase(
		usecase.ReportUsecaseDeps{
			ReportRepo:               reportRepo,
			DecisionNotificationRepo: reportDecisionNotificationRepo,
			ProductReviewModerator:   productBlueprintReviewUsecase,
			TokenBlueprintModerator:  tokenBlueprintUsecase,
			TokenCommentModerator:    tokenBlueprintReviewUsecase,
			AvatarRepo:               avatarRepo,
			AvatarResaleModerator:    resaleUsecase,
		},
	)
	if reportUsecase == nil {
		return nil, errors.New("di.admin: report usecase is nil")
	}

	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		return nil, err
	}

	gasBalanceQuery := adminquery.NewGasBalanceQuery(
		func(ctx context.Context) (*adminquery.GasBalanceResult, error) {
			result, err := solanaClient.GetReserveBalance(ctx)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, errors.New("di.admin: reserve balance result is nil")
			}

			return &adminquery.GasBalanceResult{
				Cluster:         result.Cluster,
				Address:         result.Address,
				BalanceLamports: result.BalanceLamports,
				BalanceSOL:      result.BalanceSOL,
			}, nil
		},
	)

	return &Container{
		Infra:                          infra,
		adminFirebaseUID:               adminFirebaseUID,
		adminEmail:                     adminEmail,
		contactUsecase:                 contactUsecase,
		reportUsecase:                  reportUsecase,
		companyRepo:                    companyRepo,
		memberRepo:                     memberRepo,
		avatarRepo:                     avatarRepo,
		brandRepo:                      brandRepo,
		productBlueprintRepo:           productBlueprintRepo,
		tokenBlueprintRepo:             tokenBlueprintRepo,
		reportDecisionNotificationRepo: reportDecisionNotificationRepo,
		reportNameQuery:                reportNameQuery,
		gasBalanceQuery:                gasBalanceQuery,
	}, nil
}
