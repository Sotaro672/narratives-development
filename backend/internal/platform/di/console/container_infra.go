// backend/internal/platform/di/console/container_infra.go
package console

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"

	shared "narratives/internal/platform/di/shared"
)

type clients struct {
	infra *shared.Infra

	fsClient *firestore.Client

	firestoreProjectID string
}

func ensureClients(ctx context.Context, infra *shared.Infra) (*clients, error) {
	// shared infra
	if infra == nil {
		var err error
		infra, err = shared.NewInfra(ctx)
		if err != nil {
			return nil, err
		}
	}
	if infra == nil {
		return nil, errors.New("shared infra is nil")
	}
	if infra.Config == nil {
		return nil, errors.New("shared infra config is nil")
	}

	fsClient := infra.Firestore

	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")
	if firestoreProjectID == "" {
		firestoreProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	if fsClient == nil {
		hasCredFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""

		log.Printf(
			"[di.console] ERROR: infra.Firestore is nil (projectID=%q, GOOGLE_APPLICATION_CREDENTIALS_set=%t)",
			firestoreProjectID,
			hasCredFile,
		)

		return nil, fmt.Errorf(
			"shared infra firestore client is nil (projectID=%q). shared.NewInfra likely failed to initialize Firestore client",
			firestoreProjectID,
		)
	}

	return &clients{
		infra:              infra,
		fsClient:           fsClient,
		firestoreProjectID: firestoreProjectID,
	}, nil
}
