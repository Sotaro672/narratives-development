// backend/internal/platform/di/admin/container.go
package admin

import (
	"context"
	"errors"
	"os"
	"strings"

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

	adminFirebaseUID := strings.TrimSpace(os.Getenv(adminFirebaseUIDEnv))
	if adminFirebaseUID == "" {
		return nil, errors.New("di.admin: AMOL_ADMIN_FIREBASE_UID is empty")
	}

	adminEmail := strings.TrimSpace(os.Getenv(adminEmailEnv))
	if adminEmail == "" {
		return nil, errors.New("di.admin: AMOL_ADMIN_EMAIL is empty")
	}

	return &Container{
		Infra:            infra,
		adminFirebaseUID: adminFirebaseUID,
		adminEmail:       adminEmail,
	}, nil
}
