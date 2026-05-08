// backend/internal/platform/di/introduction/container.go
package introduction

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"

	introhttp "narratives/internal/adapters/in/http/introduction"
	fsrepo "narratives/internal/adapters/out/firestore"
	mail "narratives/internal/adapters/out/mail"
	usecase "narratives/internal/application/usecase"
)

// Container wires dependencies for the "introduction" HTTP module.
type Container struct {
	fsClient *firestore.Client

	contactRepo    *fsrepo.ContactRepositoryFS
	contactUsecase *usecase.ContactUsecase
	contactHandler *introhttp.ContactHandler
}

// NewContainer builds a container.
// projectID should be your GCP/Firebase project id used by Firestore.
func NewContainer(ctx context.Context, projectID string) (*Container, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	repo := fsrepo.NewContactRepositoryFS(client)
	contactMailer := mail.NewContactMailerWithResend()
	uc := usecase.NewContactUsecase(repo, contactMailer, contactMailer)
	handler := introhttp.NewContactHandler(uc)

	return &Container{
		fsClient: client,

		contactRepo:    repo,
		contactUsecase: uc,
		contactHandler: handler,
	}, nil
}

// Close releases underlying resources.
func (c *Container) Close() error {
	if c.fsClient == nil {
		return nil
	}
	return c.fsClient.Close()
}

// Register registers introduction routes to the given mux.
func (c *Container) Register(mux *http.ServeMux) {
	if c.contactHandler != nil {
		c.contactHandler.Register(mux)
	}
}

// Expose dependencies (optional)
func (c *Container) ContactUsecase() *usecase.ContactUsecase {
	return c.contactUsecase
}
