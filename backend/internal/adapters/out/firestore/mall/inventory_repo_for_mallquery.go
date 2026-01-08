// backend/internal/adapters/out/firestore/mall/inventory_repo_for_mallquery.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fs "narratives/internal/adapters/out/firestore"
	mallquery "narratives/internal/application/query/mall"
	invdom "narratives/internal/domain/inventory"
)

// InventoryRepoForMallQuery is a Firestore-backed repository that satisfies
// application/query/mall.InventoryRepository.
//
// Motivation:
//   - mallquery.NewCatalogQuery expects InventoryRepository with
//     GetByProductAndTokenBlueprintID(...), returning inventory.Mint.
//   - The domain InventoryRepositoryFS is domain-oriented and does not necessarily
//     match mallquery's interface.
//   - This adapter performs "Firestore read" for query usecases, with best-effort fallback.
type InventoryRepoForMallQuery struct {
	Client *firestore.Client

	// Primary inventory repo (inventories collection) for consistent record<->domain mapping.
	invRepo *fs.InventoryRepositoryFS
}

var _ mallquery.InventoryRepository = (*InventoryRepoForMallQuery)(nil)

func NewInventoryRepoForMallQuery(client *firestore.Client) *InventoryRepoForMallQuery {
	return &InventoryRepoForMallQuery{
		Client:  client,
		invRepo: fs.NewInventoryRepositoryFS(client),
	}
}

func (r *InventoryRepoForMallQuery) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
	if r == nil || r.Client == nil || r.invRepo == nil {
		return invdom.Mint{}, errors.New("firestore.mall.InventoryRepoForMallQuery: client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	// 1) inventories/{id} (canonical)
	m, err := r.invRepo.GetByID(ctx, id)
	if err == nil {
		return m, nil
	}
	if !errors.Is(err, invdom.ErrNotFound) {
		return invdom.Mint{}, err
	}

	// 2) best-effort fallback: mints/{id}
	// NOTE: Only used if some environments stored inventory-like docs under "mints".
	//       If the record shape differs, DataTo may fail and we surface the error.
	if m2, err2 := r.getMintDocAsDomain(ctx, "mints", id); err2 == nil {
		return m2, nil
	} else {
		// If it's NotFound, keep NotFound. Otherwise return the error to reveal mismatch early.
		if isFirestoreNotFound(err2) {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err2
	}
}

func (r *InventoryRepoForMallQuery) GetByProductAndTokenBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) (invdom.Mint, error) {
	if r == nil || r.Client == nil || r.invRepo == nil {
		return invdom.Mint{}, errors.New("firestore.mall.InventoryRepoForMallQuery: client is nil")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)

	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}
	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}

	// 1) Try canonical docID convention: "{productBlueprintId}__{tokenBlueprintId}"
	compoundID := buildInventoryDocID(pbID, tbID)

	if m, err := r.invRepo.GetByID(ctx, compoundID); err == nil {
		return m, nil
	} else if !errors.Is(err, invdom.ErrNotFound) {
		return invdom.Mint{}, err
	}

	// 2) Fallback query in inventories by fields, then re-hydrate via invRepo.GetByID for proper mapping
	if docID, err := r.queryDocIDByFields(ctx, "inventories", pbID, tbID); err == nil {
		return r.invRepo.GetByID(ctx, docID)
	} else if !isFirestoreNotFound(err) {
		return invdom.Mint{}, err
	}

	// 3) Best-effort fallback: mints collection
	if docID, err := r.queryDocIDByFields(ctx, "mints", pbID, tbID); err == nil {
		// attempt to read as domain directly (since InventoryRepositoryFS is inventories-only)
		return r.getMintDocAsDomain(ctx, "mints", docID)
	} else if isFirestoreNotFound(err) {
		return invdom.Mint{}, invdom.ErrNotFound
	} else {
		return invdom.Mint{}, err
	}
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

// buildInventoryDocID builds "{productBlueprintId}__{tokenBlueprintId}" with minimal sanitization.
// Must match the convention used by inventories collection.
func buildInventoryDocID(productBlueprintID, tokenBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		// Firestore doc ID safety (align with existing repo behavior)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(productBlueprintID) + "__" + sanitize(tokenBlueprintID)
}

func (r *InventoryRepoForMallQuery) queryDocIDByFields(
	ctx context.Context,
	collection string,
	productBlueprintID string,
	tokenBlueprintID string,
) (string, error) {
	q := r.Client.Collection(collection).
		Where("productBlueprintId", "==", productBlueprintID).
		Where("tokenBlueprintId", "==", tokenBlueprintID).
		Limit(1)

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return "", err
	}
	if len(snaps) == 0 {
		return "", status.Error(
			codes.NotFound,
			fmt.Sprintf("inventory not found: col=%s productBlueprintId=%s tokenBlueprintId=%s", collection, productBlueprintID, tokenBlueprintID),
		)
	}
	return snaps[0].Ref.ID, nil
}

func (r *InventoryRepoForMallQuery) getMintDocAsDomain(ctx context.Context, collection string, id string) (invdom.Mint, error) {
	snap, err := r.Client.Collection(collection).Doc(id).Get(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err
	}

	var m invdom.Mint
	if err := snap.DataTo(&m); err != nil {
		return invdom.Mint{}, err
	}

	// best-effort: ensure ID populated
	if strings.TrimSpace(m.ID) == "" {
		m.ID = snap.Ref.ID
	}
	return m, nil
}

func isFirestoreNotFound(err error) bool {
	if err == nil {
		return false
	}
	return status.Code(err) == codes.NotFound
}
