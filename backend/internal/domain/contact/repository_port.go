// backend/internal/domain/contact/repository_port.go
package contact

import (
	"context"

	common "narratives/internal/domain/common"
)

// CollectionName is the Firestore collection name for Contact documents.
const CollectionName = "contacts"

// Filter is a domain-specific filter for listing contacts.
type Filter struct {
	common.FilterCommon
	IsRead *bool `json:"isRead"`
}

// Patch is a partial update payload for Contact.
type Patch struct {
	Name    *string `json:"name"`
	Email   *string `json:"email"`
	Company *string `json:"company"`
	Message *string `json:"message"`
	IsRead  *bool   `json:"isRead"`
	Source  *string `json:"source"`
}

// Repository is the port interface for Contact persistence.
type Repository interface {
	common.Repository[Contact, Filter, Patch]
}

// Creator is the minimal persistence port for creating contacts.
type Creator interface {
	Create(ctx context.Context, entity Contact) (Contact, error)
}
