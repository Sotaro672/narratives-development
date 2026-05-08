// backend/internal/platform/di/console/container_infra.go
package console

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"

	"cloud.google.com/go/firestore"

	shared "narratives/internal/platform/di/shared"
)

type clients struct {
	infra *shared.Infra

	fsClient *firestore.Client

	// shared.Config の型に依存しない（shared.Config が存在しないため）
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

	// FirestoreProjectID を reflect で取得（Config の具体型に依存しない）
	firestoreProjectID := getStringField(infra.Config, "FirestoreProjectID")

	if fsClient == nil {
		projectID := firestoreProjectID
		if projectID == "" {
			projectID = os.Getenv("FIRESTORE_PROJECT_ID")
		}
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}

		hasCredFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""
		log.Printf(
			"[di.console] ERROR: infra.Firestore is nil (projectID=%q, GOOGLE_APPLICATION_CREDENTIALS_set=%t)",
			projectID,
			hasCredFile,
		)

		return nil, fmt.Errorf(
			"shared infra firestore client is nil (projectID=%q). shared.NewInfra likely failed to initialize Firestore client",
			projectID,
		)
	}

	return &clients{
		infra:              infra,
		fsClient:           fsClient,
		firestoreProjectID: firestoreProjectID,
	}, nil
}

// getStringField tries to read a string field from an arbitrary struct pointer / struct.
// If it cannot, it returns "".
func getStringField(obj any, fieldName string) string {
	if obj == nil || fieldName == "" {
		return ""
	}

	rv := reflect.ValueOf(obj)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanInterface() {
		return ""
	}
	if f.Kind() == reflect.String {
		return f.String()
	}

	return ""
}
