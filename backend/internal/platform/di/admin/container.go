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

	adminFirebaseUID    string
	adminEmail          string
	contactUsecase      *usecase.ContactUsecase
	reviewReportUsecase *usecase.ReviewReportUsecase
	companyRepo         *fsrepo.CompanyRepositoryFS
	memberRepo          *fsrepo.MemberRepositoryFS
	gasBalanceQuery     *adminquery.GasBalanceQuery
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

	reviewReportRepo := fsrepo.NewReviewReportRepositoryFS(infra.Firestore)
	if reviewReportRepo == nil {
		return nil, errors.New("di.admin: review report repository is nil")
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

	reviewReportUsecase := usecase.NewReviewReportUsecase(
		usecase.ReviewReportUsecaseDeps{
			ReportRepo:             reviewReportRepo,
			ProductReviewModerator: productBlueprintReviewUsecase,
			TokenCommentModerator:  tokenBlueprintReviewUsecase,
		},
	)
	if reviewReportUsecase == nil {
		return nil, errors.New("di.admin: review report usecase is nil")
	}

	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		return nil, err
	}

	gasBalanceQuery := adminquery.NewGasBalanceQuery(func(ctx context.Context) (*adminquery.GasBalanceResult, error) {
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
	})

	return &Container{
		Infra:               infra,
		adminFirebaseUID:    adminFirebaseUID,
		adminEmail:          adminEmail,
		contactUsecase:      contactUsecase,
		reviewReportUsecase: reviewReportUsecase,
		companyRepo:         companyRepo,
		memberRepo:          memberRepo,
		gasBalanceQuery:     gasBalanceQuery,
	}, nil
}
