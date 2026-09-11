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
	shared "narratives/internal/platform/di/shared"
)

const (
	adminFirebaseUIDEnv = "AMOL_ADMIN_FIREBASE_UID"
	adminEmailEnv       = "AMOL_ADMIN_EMAIL"
)

type Container struct {
	Infra *shared.Infra

	adminFirebaseUID string
	adminEmail       string

	contactUsecase      *usecase.ContactUsecase
	newsUsecase         *usecase.NewsUsecase
	reportUsecase       *usecase.ReportUsecase
	resaleReviewUsecase *usecase.ResaleReviewUsecase

	companyRepo                    *fsrepo.CompanyRepositoryFS
	memberRepo                     *fsrepo.MemberRepositoryFS
	userRepo                       *fsrepo.UserRepositoryFS
	avatarRepo                     *fsrepo.AvatarRepositoryFS
	brandRepo                      *fsrepo.BrandRepositoryFS
	productBlueprintRepo           *fsrepo.ProductBlueprintRepositoryFS
	tokenBlueprintRepo             *fsrepo.TokenBlueprintRepositoryFS
	newsRepo                       *fsrepo.NewsRepositoryFS
	newsReadRepo                   *fsrepo.NewsReadRepositoryFS
	reportRepo                     *fsrepo.ReportRepositoryFS
	resaleRepo                     *fsrepo.ResaleRepositoryFS
	resaleImageRepo                *fsrepo.ResaleImageRepositoryFS
	resaleReviewRepo               *fsrepo.ResaleReviewRepositoryFS
	resaleTradeReader              *fsrepo.ResaleTradeReaderFS
	tradeMessageRepo               *fsrepo.TradeMessageRepositoryFS
	tradeMessageStatsReader        *fsrepo.TradeMessageStatsReaderFS
	reportDecisionNotificationRepo *fsrepo.ReportDecisionNotificationRepositoryFS

	newsQuery                           *adminquery.NewsQuery
	reportNameQuery                     *adminquery.ReportNameQuery
	resaleTradeQuery                    *adminquery.ResaleTradeQuery
	tradeMessageQuery                   *adminquery.TradeMessageQuery
	contractDetailQuery                 *adminquery.ContractDetailQuery
	contractListQuery                   *adminquery.ContractListQuery
	contractTokenBlueprintQuery         *adminquery.ContractTokenBlueprintQuery
	contractProductBlueprintQuery       *adminquery.ContractProductBlueprintQuery
	contractTokenBlueprintReviewQuery   *adminquery.ContractTokenBlueprintReviewQuery
	contractProductBlueprintReviewQuery *adminquery.ContractProductBlueprintReviewQuery
	gasBalanceQuery                     *adminquery.GasBalanceQuery
	mintListQuery                       *adminquery.MintListQuery
	mintDetailQuery                     *adminquery.MintDetailQuery
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

	c := &Container{
		Infra:            infra,
		adminFirebaseUID: adminFirebaseUID,
		adminEmail:       adminEmail,
	}

	repos := buildRepos(infra.Firestore)
	if repos == nil {
		return nil, errors.New("di.admin: repositories are nil")
	}
	repos.applyToContainer(c)

	res := buildResolvers(repos)
	if res == nil || res.nameResolver == nil {
		return nil, errors.New("di.admin: resolvers are nil")
	}

	u, err := buildUsecases(ctx, repos)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("di.admin: usecases are nil")
	}
	u.applyToContainer(c)

	q, err := buildQueries(ctx, infra, repos, res, u)
	if err != nil {
		u.closeOnBuildError()
		return nil, err
	}
	if q == nil {
		u.closeOnBuildError()
		return nil, errors.New("di.admin: queries are nil")
	}
	q.applyToContainer(c)

	return c, nil
}
