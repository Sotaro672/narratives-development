// backend/internal/domain/print/repository_port.go
package print

import (
	"context"
	"errors"
)

type RepositoryPort interface {
	Create(ctx context.Context, log PrintLog) (PrintLog, error)
	GetByProductionID(ctx context.Context, productionID string) (PrintLog, error)
	ExistsByProductionID(ctx context.Context, productionID string) (bool, error)
}

var (
	ErrNotFound = errors.New("print: not found")
	ErrConflict = errors.New("print: conflict")
)
