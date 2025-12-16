// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
	"sort"
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
// (inventory domain: inventories collection)
// ============================================================

const collectionNameInventories = "inventories"

type InventoryRepositoryFS struct {
	Client *firestore.Client
}

func NewInventoryRepositoryFS(client *firestore.Client) *InventoryRepositoryFS {
	return &InventoryRepositoryFS{Client: client}
}

func (r *InventoryRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(collectionNameInventories)
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

func (r *InventoryRepositoryFS) Create(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	start := time.Now()

	log.Printf(
		"[inventory_repo_fs] Create start repo_nil=%t client_nil=%t id=%q tokenBlueprintId=%q productBlueprintId=%q products=%d accumulation=%d createdAtZero=%t updatedAtZero=%t",
		r == nil,
		r == nil || r.Client == nil,
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.TokenBlueprintID),
		strings.TrimSpace(m.ProductBlueprintID),
		func() int {
			if m.Products == nil {
				return 0
			}
			return len(m.Products)
		}(),
		m.Accumulation,
		m.CreatedAt.IsZero(),
		m.UpdatedAt.IsZero(),
	)

	if r == nil || r.Client == nil {
		log.Printf("[inventory_repo_fs] Create abort reason=repo_or_client_nil elapsed=%s", time.Since(start))
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	now := time.Now().UTC()

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
		log.Printf("[inventory_repo_fs] Create allocated new doc docId=%q", doc.ID)
	} else {
		doc = r.col().Doc(strings.TrimSpace(m.ID))
		log.Printf("[inventory_repo_fs] Create using provided id docId=%q", doc.ID)
	}

	rec := toRecord(m)
	rec.ID = doc.ID
	m.ID = doc.ID

	if _, err := doc.Set(ctx, rec); err != nil {
		log.Printf(
			"[inventory_repo_fs] Create error docId=%q err=%v elapsed=%s",
			doc.ID, err, time.Since(start),
		)
		return invdom.Mint{}, err
	}

	log.Printf(
		"[inventory_repo_fs] Create done docId=%q elapsed=%s createdAt=%s updatedAt=%s accumulation=%d products=%d",
		doc.ID,
		time.Since(start),
		m.CreatedAt.UTC().Format(time.RFC3339),
		m.UpdatedAt.UTC().Format(time.RFC3339),
		m.Accumulation,
		func() int {
			if m.Products == nil {
				return 0
			}
			return len(m.Products)
		}(),
	)

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

func (r *InventoryRepositoryFS) Update(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	m.UpdatedAt = time.Now().UTC()

	if m.CreatedAt.IsZero() {
		existing, err := r.GetByID(ctx, id)
		if err != nil {
			return invdom.Mint{}, err
		}
		m.CreatedAt = existing.CreatedAt
		if m.CreatedAt.IsZero() {
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
// Accumulation operations (atomic)
// ============================================================

func (r *InventoryRepositoryFS) IncrementAccumulation(ctx context.Context, id string, delta int) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	if delta == 0 {
		return r.GetByID(ctx, id)
	}

	doc := r.col().Doc(id)
	now := time.Now().UTC()

	// ✅ firestore.Client.RunTransaction は error だけ返すため、代入は 1 つだけ
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return invdom.ErrNotFound
			}
			return err
		}

		var rec inventoryRecord
		if err := snap.DataTo(&rec); err != nil {
			return err
		}

		newAccum := rec.Accumulation + delta
		if newAccum < 0 {
			return invdom.ErrInvalidAccumulation
		}

		return tx.Update(doc, []firestore.Update{
			{Path: "accumulation", Value: firestore.Increment(int64(delta))},
			{Path: "updatedAt", Value: now},
		})
	})
	if err != nil {
		return invdom.Mint{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *InventoryRepositoryFS) IncrementAccumulationByMintProducts(ctx context.Context, id string) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	m, err := r.GetByID(ctx, id)
	if err != nil {
		return invdom.Mint{}, err
	}

	delta := 0
	if m.Products != nil {
		delta = len(m.Products)
	}
	return r.IncrementAccumulation(ctx, id, delta)
}

func (r *InventoryRepositoryFS) DecrementAccumulationByOrderItemsCount(ctx context.Context, id string, orderItemsCount int) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	if orderItemsCount < 0 {
		return invdom.Mint{}, invdom.ErrInvalidAccumulation
	}
	if orderItemsCount == 0 {
		return r.GetByID(ctx, id)
	}
	return r.IncrementAccumulation(ctx, id, -orderItemsCount)
}

// ============================================================
// Upsert helpers (for InventoryUsecase / mint flow)
// ============================================================

// UpsertByTokenAndProductBlueprintID:
// - docID は buildInventoryDocID(tokenBlueprintID, productBlueprintID) 固定
// - 既存があれば products をマージし、added 分だけ accumulation を増やします（idempotent）
func (r *InventoryRepositoryFS) UpsertByTokenAndProductBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
) (invdom.Mint, error) {

	start := time.Now()

	if r == nil || r.Client == nil {
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}

	ids := normalizeIDs(productIDs)
	docID := buildInventoryDocID(tbID, pbID)
	doc := r.col().Doc(docID)
	now := time.Now().UTC()

	log.Printf(
		"[inventory_repo_fs] UpsertByTokenAndProductBlueprintID start docId=%q tokenBlueprintId=%q productBlueprintId=%q productIds=%d",
		docID, tbID, pbID, len(ids),
	)

	// Transaction で: 既存取得 -> products マージ -> accumulation を増減なしで増やす（addedのみ）
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				// create
				products := map[string]string{}
				for _, pid := range ids {
					products[pid] = "" // mintAddress は後から埋める想定
				}

				ent, err := invdom.NewMint(
					docID,
					tbID,
					pbID,
					ids,
					len(ids),
					now,
				)
				if err != nil {
					return err
				}

				// NewMint が Products を map にしていない可能性があるため、ここで確実化
				ent.ID = docID
				ent.TokenBlueprintID = tbID
				ent.ProductBlueprintID = pbID
				ent.Products = products
				ent.CreatedAt = now
				ent.UpdatedAt = now

				rec := toRecord(ent)
				rec.ID = docID
				return tx.Set(doc, rec)
			}
			return err
		}

		var rec inventoryRecord
		if err := snap.DataTo(&rec); err != nil {
			return err
		}

		if rec.Products == nil {
			rec.Products = map[string]string{}
		}

		added := 0
		for _, pid := range ids {
			if pid == "" {
				continue
			}
			if _, ok := rec.Products[pid]; ok {
				continue
			}
			rec.Products[pid] = ""
			added++
		}

		updates := []firestore.Update{
			{Path: "products", Value: rec.Products},
			{Path: "updatedAt", Value: now},
		}
		if added > 0 {
			updates = append(updates, firestore.Update{
				Path:  "accumulation",
				Value: firestore.Increment(int64(added)),
			})
		}

		return tx.Update(doc, updates)
	})
	if err != nil {
		log.Printf("[inventory_repo_fs] UpsertByTokenAndProductBlueprintID error docId=%q err=%v elapsed=%s", docID, err, time.Since(start))
		return invdom.Mint{}, err
	}

	out, err := r.GetByID(ctx, docID)
	if err != nil {
		log.Printf("[inventory_repo_fs] UpsertByTokenAndProductBlueprintID GetByID error docId=%q err=%v elapsed=%s", docID, err, time.Since(start))
		return invdom.Mint{}, err
	}

	log.Printf(
		"[inventory_repo_fs] UpsertByTokenAndProductBlueprintID done docId=%q accumulation=%d products=%d elapsed=%s",
		docID,
		out.Accumulation,
		func() int {
			if out.Products == nil {
				return 0
			}
			return len(out.Products)
		}(),
		time.Since(start),
	)

	return out, nil
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

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func buildInventoryDocID(tokenBlueprintID, productBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(tokenBlueprintID) + "__" + sanitize(productBlueprintID)
}
