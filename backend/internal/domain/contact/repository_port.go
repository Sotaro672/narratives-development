// backend/internal/domain/contact/repository_port.go
package contact

import (
	"context"

	common "narratives/internal/domain/common"
)

// CollectionName is the Firestore collection name for Contact documents.
const CollectionName = "contacts"

// Filter is a domain-specific filter for listing contacts.
// Embed common.FilterCommon and extend as needed (e.g., Status).
type Filter struct {
	common.FilterCommon
	Status *Status `json:"status"` // optional filter (e.g. "new")
}

// Patch is a partial update payload for Contact.
// Keep this minimal and add fields only when you implement Update.
type Patch struct {
	Name    *string `json:"name"`
	Email   *string `json:"email"`
	Company *string `json:"company"`
	Message *string `json:"message"`

	Status *Status `json:"status"`

	Source *string `json:"source"`
}

// Repository is the port (interface) for Contact persistence.
type Repository interface {
	common.Repository[Contact, Filter, Patch]
}

// Usecase-facing minimal ports (optional):
// - If you only need "Create" for now, you can depend on Creator instead of Repository.
type Creator interface {
	Create(ctx context.Context, entity Contact) (Contact, error)
}
