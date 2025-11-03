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
// Query contracts
// ==============================

type Filter struct {
	IDs   []string
	Types []ContentType

	NameLike string // 部分一致

	SizeMin *int64
	SizeMax *int64
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortBySize SortColumn = "size"
	SortByName SortColumn = "name"
	SortByType SortColumn = "type"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []TokenContent
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ==============================
// Stats (optional contract)
// ==============================

type TokenContentStats struct {
	TotalCount         int
	TotalSize          int64
	TotalSizeFormatted string // human readable (e.g., "12.3 MB")
	CountByType        struct {
		Image    int
		Video    int
		PDF      int
		Document int
	}
}

// ==============================
// Repository Port (contracts only)
// ==============================

type RepositoryPort interface {
	// Read
	GetByID(ctx context.Context, id string) (*TokenContent, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Write
	Create(ctx context.Context, in CreateTokenContentInput) (*TokenContent, error)
	Update(ctx context.Context, id string, in UpdateTokenContentInput) (*TokenContent, error)
	Delete(ctx context.Context, id string) error

	// Upload abstraction (storage-agnostic)
	// Return: URL, actual size(bytes)
	UploadContent(ctx context.Context, fileName, contentType string, r io.Reader) (url string, size int64, err error)

	// Stats (optional)
	GetStats(ctx context.Context) (TokenContentStats, error)

	// Maintenance (optional)
	Reset(ctx context.Context) error

	// Transaction boundary (optional)
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
