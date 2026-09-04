// backend/internal/platform/di/admin/container.go
package admin

import (
	"context"
	"errors"
	"os"
	"strings"

	fsrepo "narratives/internal/adapters/out/firestore"
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
	contactUsecase   *usecase.ContactUsecase
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

	return &Container{
		Infra:            infra,
		adminFirebaseUID: adminFirebaseUID,
		adminEmail:       adminEmail,
		contactUsecase:   contactUsecase,
	}, nil
}
