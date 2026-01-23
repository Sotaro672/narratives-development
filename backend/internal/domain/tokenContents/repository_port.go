package tokenContents

import (
	"context"
	"io"
)

// Domain alias: use TokenContent as the public name for GCSTokenContent.
type TokenContent = GCSTokenContent

// ==============================
// Create/Update inputs (contract)
// ==============================

type CreateTokenContentInput struct {
	Name string      `json:"name"`
	Type ContentType `json:"type"`
	URL  string      `json:"url"`
	Size int64       `json:"size"`
}

type UpdateTokenContentInput struct {
	Name *string      `json:"name,omitempty"`
	Type *ContentType `json:"type,omitempty"`
	URL  *string      `json:"url,omitempty"`
	Size *int64       `json:"size,omitempty"`
}

// ==============================
// Repository Port (contracts only)
// ==============================
//
// ✅ Removed:
// - Stats
// - Filter/Sort/Page/Search
// - Transaction boundary
// - Maintenance reset
type RepositoryPort interface {
	// Read
	GetByID(ctx context.Context, id string) (*TokenContent, error)

	// Write
	Create(ctx context.Context, in CreateTokenContentInput) (*TokenContent, error)
	Update(ctx context.Context, id string, in UpdateTokenContentInput) (*TokenContent, error)
	Delete(ctx context.Context, id string) error

	// Upload abstraction (storage-agnostic)
	// Return: URL, actual size(bytes)
	UploadContent(ctx context.Context, fileName, contentType string, r io.Reader) (url string, size int64, err error)
}
