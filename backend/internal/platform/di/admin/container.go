// backend/internal/platform/di/admin/container.go
package admin

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	firebaseadp "narratives/internal/adapters/out/firebase"
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

	adminFirebaseUID                    string
	adminEmail                          string
	contactUsecase                      *usecase.ContactUsecase
	newsUsecase                         *usecase.NewsUsecase
	reportUsecase                       *usecase.ReportUsecase
	companyRepo                         *fsrepo.CompanyRepositoryFS
	memberRepo                          *fsrepo.MemberRepositoryFS
	avatarRepo                          *fsrepo.AvatarRepositoryFS
	brandRepo                           *fsrepo.BrandRepositoryFS
	productBlueprintRepo                *fsrepo.ProductBlueprintRepositoryFS
	tokenBlueprintRepo                  *fsrepo.TokenBlueprintRepositoryFS
	newsRepo                            *fsrepo.NewsRepositoryFS
	newsReadRepo                        *fsrepo.NewsReadRepositoryFS
	reportDecisionNotificationRepo      *fsrepo.ReportDecisionNotificationRepositoryFS
	newsQuery                           *adminquery.NewsQuery
	reportNameQuery                     *adminquery.ReportNameQuery
	contractDetailQuery                 *adminquery.ContractDetailQuery
	contractListQuery                   *adminquery.ContractListQuery
	contractTokenBlueprintQuery         *adminquery.ContractTokenBlueprintQuery
	contractProductBlueprintQuery       *adminquery.ContractProductBlueprintQuery
	contractTokenBlueprintReviewQuery   *adminquery.ContractTokenBlueprintReviewQuery
	contractProductBlueprintReviewQuery *adminquery.ContractProductBlueprintReviewQuery
	gasBalanceQuery                     *adminquery.GasBalanceQuery
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
	listRepo := fsrepo.NewListRepositoryFS(infra.Firestore)
	listImageRepo := fsrepo.NewListImageRepositoryFS(infra.Firestore)
	inventoryRepo := fsrepo.NewInventoryRepositoryFS(infra.Firestore)
	modelRepo := fsrepo.NewModelRepositoryFS(infra.Firestore)
	orderConsoleLister := fsrepo.NewOrderConsoleListerFS(infra.Firestore)

	newsRepo := fsrepo.NewNewsRepositoryFS(infra.Firestore)
	if newsRepo == nil {
		return nil, errors.New("di.admin: news repository is nil")
	}

	newsReadRepo := fsrepo.NewNewsReadRepositoryFS(infra.Firestore)
	if newsReadRepo == nil {
		return nil, errors.New("di.admin: news read repository is nil")
	}

	newsImageStorage, err := firebaseadp.NewNewsImageStorageFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if newsImageStorage == nil {
		return nil, errors.New("di.admin: news image storage is nil")
	}

	newsUsecase := usecase.NewNewsUsecase(
		newsRepo,
		newsReadRepo,
	).WithImageStorage(newsImageStorage)
	if newsUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: news usecase is nil")
	}

	authUserReader := firebaseadp.NewAuthUserReader(infra.FirebaseAuth)
	if authUserReader == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: auth user reader is nil")
	}

	newsQuery := adminquery.NewNewsQuery(
		newsUsecase,
		authUserReader,
	)
	if newsQuery == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: news query is nil")
	}

	reportNameQuery := adminquery.NewReportNameQuery(
		avatarRepo,
		brandRepo,
		companyRepo,
		listRepo,
		memberRepo,
		productBlueprintRepo,
		tokenBlueprintRepo,
	)

	reportRepo := fsrepo.NewReportRepositoryFS(infra.Firestore)
	if reportRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: report repository is nil")
	}

	contractDetailQuery := adminquery.NewContractDetailQuery(
		companyRepo,
		brandRepo,
		memberRepo,
		productBlueprintRepo,
		tokenBlueprintRepo,
		inventoryRepo,
		listRepo,
		reportRepo,
	)

	contractListQuery := adminquery.NewContractListQuery(
		companyRepo,
		brandRepo,
		memberRepo,
		productBlueprintRepo,
		tokenBlueprintRepo,
		inventoryRepo,
		listRepo,
		listImageRepo,
		modelRepo,
		orderConsoleLister,
		reportRepo,
	)

	reportDecisionNotificationRepo := fsrepo.NewReportDecisionNotificationRepositoryFS(infra.Firestore)
	if reportDecisionNotificationRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: report decision notification repository is nil")
	}

	productBlueprintReviewRepo := fsrepo.NewProductBlueprintReviewRepositoryFS(infra.Firestore)
	if productBlueprintReviewRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: product blueprint review repository is nil")
	}

	contractProductBlueprintQuery := adminquery.NewContractProductBlueprintQuery(
		companyRepo,
		brandRepo,
		memberRepo,
		productBlueprintRepo,
		modelRepo,
		productBlueprintReviewRepo,
		reportRepo,
	)
	if contractProductBlueprintQuery == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: contract product blueprint query is nil")
	}

	productBlueprintReviewUsecase := usecase.NewProductBlueprintReviewUsecase(
		productBlueprintReviewRepo,
		productBlueprintRepo,
		brandRepo,
		memberRepo,
		nil,
		avatarRepo,
		nil,
	)
	if productBlueprintReviewUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: product blueprint review usecase is nil")
	}

	contractProductBlueprintReviewQuery := adminquery.NewContractProductBlueprintReviewQuery(
		companyRepo,
		productBlueprintRepo,
		productBlueprintReviewUsecase,
	)
	if contractProductBlueprintReviewQuery == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: contract product blueprint review query is nil")
	}

	tokenBlueprintReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(infra.Firestore)
	if tokenBlueprintReviewRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: token blueprint review repository is nil")
	}

	tokenBlueprintReviewUsecase := usecase.NewTokenBlueprintReviewUsecase(
		tokenBlueprintReviewRepo,
		avatarRepo,
		tokenBlueprintRepo,
		brandRepo,
	)
	if tokenBlueprintReviewUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: token blueprint review usecase is nil")
	}

	contractTokenBlueprintQuery := adminquery.NewContractTokenBlueprintQuery(
		companyRepo,
		brandRepo,
		memberRepo,
		tokenBlueprintRepo,
		tokenBlueprintReviewUsecase,
		reportRepo,
	)

	contractTokenBlueprintReviewQuery := adminquery.NewContractTokenBlueprintReviewQuery(
		companyRepo,
		tokenBlueprintRepo,
		tokenBlueprintReviewUsecase,
		reportRepo,
	)
	if contractTokenBlueprintReviewQuery == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: contract token blueprint review query is nil")
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
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: token blueprint usecase is nil")
	}

	resaleRepo := fsrepo.NewResaleRepositoryFS(infra.Firestore)
	if resaleRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: resale repository is nil")
	}

	cartRepo := fsrepo.NewCartRepositoryFS(infra.Firestore)
	if cartRepo == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: cart repository is nil")
	}

	// Admin側のListUsecaseは通報裁定によるMall上の出品停止専用。
	// List本体、ListImage、Firebase Storage上の画像は削除せず、
	// statusをsuspendedへ変更し、既存カートから対象Listを除去する。
	listUsecase := usecase.NewListUsecase(
		listRepo,
		nil,
		nil,
	).WithCartItemCleanup(cartRepo)
	if listUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: list usecase is nil")
	}

	// Admin側のResaleUsecaseはアバター通報および個別Resale通報の裁定専用。
	// 出品作成・画像操作は行わないため、imageRepo / imageStorage /
	// product identity repositories は不要。
	// 個別ResaleのREMOVEでは対象Resaleをsuspendedへ変更し、既存カートから除去する。
	resaleUsecase := usecase.NewResaleUsecase(
		resaleRepo,
		nil,
		nil,
		time.Now,
	).WithCartItemCleanup(cartRepo)
	if resaleUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: resale usecase is nil")
	}

	reportUsecase := usecase.NewReportUsecase(
		usecase.ReportUsecaseDeps{
			ReportRepo:               reportRepo,
			DecisionNotificationRepo: reportDecisionNotificationRepo,
			ProductBlueprintRepo:     productBlueprintRepo,
			ProductReviewModerator:   productBlueprintReviewUsecase,
			ListRepo:                 listRepo,
			InventoryRepo:            inventoryRepo,
			ListModerator:            listUsecase,
			TokenBlueprintRepo:       tokenBlueprintRepo,
			TokenBlueprintModerator:  tokenBlueprintUsecase,
			TokenCommentModerator:    tokenBlueprintReviewUsecase,
			AvatarRepo:               avatarRepo,
			AvatarResaleModerator:    resaleUsecase,
			ResaleRepo:               resaleRepo,
			ResaleModerator:          resaleUsecase,
		},
	)
	if reportUsecase == nil {
		_ = newsImageStorage.Close()
		return nil, errors.New("di.admin: report usecase is nil")
	}

	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		_ = newsImageStorage.Close()
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
		Infra:                               infra,
		adminFirebaseUID:                    adminFirebaseUID,
		adminEmail:                          adminEmail,
		contactUsecase:                      contactUsecase,
		newsUsecase:                         newsUsecase,
		reportUsecase:                       reportUsecase,
		companyRepo:                         companyRepo,
		memberRepo:                          memberRepo,
		avatarRepo:                          avatarRepo,
		brandRepo:                           brandRepo,
		productBlueprintRepo:                productBlueprintRepo,
		tokenBlueprintRepo:                  tokenBlueprintRepo,
		newsRepo:                            newsRepo,
		newsReadRepo:                        newsReadRepo,
		reportDecisionNotificationRepo:      reportDecisionNotificationRepo,
		newsQuery:                           newsQuery,
		reportNameQuery:                     reportNameQuery,
		contractDetailQuery:                 contractDetailQuery,
		contractListQuery:                   contractListQuery,
		contractTokenBlueprintQuery:         contractTokenBlueprintQuery,
		contractProductBlueprintQuery:       contractProductBlueprintQuery,
		contractTokenBlueprintReviewQuery:   contractTokenBlueprintReviewQuery,
		contractProductBlueprintReviewQuery: contractProductBlueprintReviewQuery,
		gasBalanceQuery:                     gasBalanceQuery,
	}, nil
}
