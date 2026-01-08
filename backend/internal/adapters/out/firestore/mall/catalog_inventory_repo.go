// backend/internal/adapters/out/firestore/mall/catalog_inventory_repo.go
package mall

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mallquerydto "narratives/internal/application/query/mall/dto"
)

// CatalogInventoryRepo is a Firestore-backed reader that returns
// mall query DTO (CatalogInventoryDTO) by inventory doc ID.
//
// Motivation:
//   - mall query layer expects DTO shape (DataTo into DTO) for read models.
//   - domain-oriented repositories may return domain entities, causing type mismatch.
//   - This repo "direct-reads" Firestore and hydrates DTO.
type CatalogInventoryRepo struct {
	Client *firestore.Client
}

const catalogInventoryCollection = "inventories"

func NewCatalogInventoryRepo(client *firestore.Client) *CatalogInventoryRepo {
	return &CatalogInventoryRepo{Client: client}
}

func (r *CatalogInventoryRepo) col() *firestore.CollectionRef {
	return r.Client.Collection(catalogInventoryCollection)
}

func (r *CatalogInventoryRepo) GetByID(
	ctx context.Context,
	id string,
) (*mallquerydto.CatalogInventoryDTO, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore.mall.CatalogInventoryRepo: client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("firestore.mall.CatalogInventoryRepo: id is empty")
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "catalog inventory not found")
		}
		return nil, err
	}

	var dto mallquerydto.CatalogInventoryDTO
	if err := snap.DataTo(&dto); err != nil {
		return nil, err
	}

	// best-effort: if dto has "id" field and it's empty, DataTo might not set it
	// depending on firestore tags in DTO. We avoid reflecting; leave as-is.

	return &dto, nil
}
