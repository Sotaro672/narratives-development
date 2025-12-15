// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// Firestore-based Inventory Repository
// (inventory domain: mints collection)
// ============================================================
//
// NOTE:
// - ドメインの実体は backend/internal/domain/inventory/entity.go の構造体を正とする
// - Firestore のコレクション名は既存運用を想定し "mints" を使用
//   （もし "inventories" にしたい場合は collectionName を変更してください）

const collectionNameMints = "mints"

type InventoryRepositoryFS struct {
	Client *firestore.Client
}

func NewInventoryRepositoryFS(client *firestore.Client) *InventoryRepositoryFS {
	return &InventoryRepositoryFS{Client: client}
}

func (r *InventoryRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(collectionNameMints)
}

// ------------------------------------------------------------
// Firestore record shape
// ------------------------------------------------------------

type inventoryRecord struct {
	ID                 string            `firestore:"id"`
	TokenBlueprintID   string            `firestore:"tokenBlueprintId"`
	ProductBlueprintID string            `firestore:"productBlueprintId"`
	Products           map[string]string `firestore:"products"`
	Accumulation       int               `firestore:"accumulation"`
	CreatedAt          time.Time         `firestore:"createdAt"`
	UpdatedAt          time.Time         `firestore:"updatedAt"`
}

func toRecord(m invdom.Mint) inventoryRecord {
	return inventoryRecord{
		ID:                 strings.TrimSpace(m.ID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		Products:           m.Products,
		Accumulation:       m.Accumulation,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func fromRecord(docID string, rec inventoryRecord) invdom.Mint {
	id := strings.TrimSpace(rec.ID)
	if id == "" {
		id = docID
	}
	return invdom.Mint{
		ID:                 id,
		TokenBlueprintID:   rec.TokenBlueprintID,
		ProductBlueprintID: rec.ProductBlueprintID,
		Products:           rec.Products,
		Accumulation:       rec.Accumulation,
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}
}

// ============================================================
// CRUD
// ============================================================

// Create creates a new Mint record in Firestore.
// If m.ID is empty, it will be auto-generated.
func (r *InventoryRepositoryFS) Create(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	now := time.Now().UTC()

	// created/updated を最低限補完（ドメイン側で既に入っていれば尊重）
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}

	var doc *firestore.DocumentRef
	if strings.TrimSpace(m.ID) == "" {
		doc = r.col().NewDoc()
		m.ID = doc.ID
	} else {
		doc = r.col().Doc(strings.TrimSpace(m.ID))
	}

	rec := toRecord(m)
	// docID とフィールド id の整合性
	rec.ID = doc.ID
	m.ID = doc.ID

	if _, err := doc.Set(ctx, rec); err != nil {
		return invdom.Mint{}, err
	}

	return m, nil
}

func (r *InventoryRepositoryFS) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err
	}

	var rec inventoryRecord
	if err := snap.DataTo(&rec); err != nil {
		return invdom.Mint{}, err
	}

	return fromRecord(snap.Ref.ID, rec), nil
}

// Update overwrites the document (merge) and bumps UpdatedAt.
func (r *InventoryRepositoryFS) Update(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	// UpdatedAt は常に更新
	m.UpdatedAt = time.Now().UTC()

	// CreatedAt がゼロなら DB の既存値を維持したいので先に取得して補完する
	if m.CreatedAt.IsZero() {
		existing, err := r.GetByID(ctx, id)
		if err != nil {
			return invdom.Mint{}, err
		}
		m.CreatedAt = existing.CreatedAt
		if m.CreatedAt.IsZero() {
			// 既存が壊れている/無い場合のフォールバック
			m.CreatedAt = m.UpdatedAt
		}
	}

	rec := toRecord(m)
	rec.ID = id

	_, err := r.col().Doc(id).Set(ctx, rec, firestore.MergeAll)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err
	}

	return m, nil
}

func (r *InventoryRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrInvalidMintID
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Queries
// ============================================================

func (r *InventoryRepositoryFS) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]invdom.Mint, error) {
	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	if tokenBlueprintID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}

	iter := r.col().Where("tokenBlueprintId", "==", tokenBlueprintID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

func (r *InventoryRepositoryFS) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, invdom.ErrInvalidProductBlueprintID
	}

	iter := r.col().Where("productBlueprintId", "==", productBlueprintID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

func (r *InventoryRepositoryFS) ListByTokenAndProductBlueprintID(ctx context.Context, tokenBlueprintID, productBlueprintID string) ([]invdom.Mint, error) {
	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	productBlueprintID = strings.TrimSpace(productBlueprintID)

	if tokenBlueprintID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}
	if productBlueprintID == "" {
		return nil, invdom.ErrInvalidProductBlueprintID
	}

	iter := r.col().
		Where("tokenBlueprintId", "==", tokenBlueprintID).
		Where("productBlueprintId", "==", productBlueprintID).
		Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

// ============================================================
// Internal helpers
// ============================================================

func readAllInventoryDocs(iter *firestore.DocumentIterator) ([]invdom.Mint, error) {
	out := make([]invdom.Mint, 0, 16)

	for {
		snap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}

		var rec inventoryRecord
		if err := snap.DataTo(&rec); err != nil {
			return nil, err
		}
		out = append(out, fromRecord(snap.Ref.ID, rec))
	}

	return out, nil
}
