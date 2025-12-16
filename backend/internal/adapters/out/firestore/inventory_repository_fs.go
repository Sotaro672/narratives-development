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

// Firestore record shape
type inventoryRecord struct {
	ID                 string `firestore:"id"`
	TokenBlueprintID   string `firestore:"tokenBlueprintId"`
	ProductBlueprintID string `firestore:"productBlueprintId"` // 参照用（docId には使わない）
	ModelID            string `firestore:"modelId"`            // ★ NEW

	Products     []string  `firestore:"products"`
	Accumulation int       `firestore:"accumulation"`
	CreatedAt    time.Time `firestore:"createdAt"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}

func toRecord(m invdom.Mint) inventoryRecord {
	return inventoryRecord{
		ID:                 strings.TrimSpace(m.ID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		ModelID:            strings.TrimSpace(m.ModelID),
		Products:           normalizeIDs(m.Products),
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
	out := invdom.Mint{
		ID:                 id,
		TokenBlueprintID:   strings.TrimSpace(rec.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(rec.ProductBlueprintID),
		ModelID:            strings.TrimSpace(rec.ModelID),
		Products:           normalizeIDs(rec.Products),
		Accumulation:       rec.Accumulation,
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}
	// accumulation が壊れている/空の場合の保険
	if out.Accumulation <= 0 && len(out.Products) > 0 {
		out.Accumulation = len(out.Products)
	}
	return out
}

// ============================================================
// CRUD
// ============================================================

func (r *InventoryRepositoryFS) Create(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	if r == nil || r.Client == nil {
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}

	m.ID = strings.TrimSpace(m.ID)
	m.TokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)
	m.ProductBlueprintID = strings.TrimSpace(m.ProductBlueprintID)
	m.ModelID = strings.TrimSpace(m.ModelID)

	m.Products = normalizeIDs(m.Products)
	if m.Accumulation <= 0 {
		m.Accumulation = len(m.Products)
	}

	if m.ID == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	doc := r.col().Doc(m.ID)
	rec := toRecord(m)
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

func (r *InventoryRepositoryFS) Update(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	m.UpdatedAt = time.Now().UTC()
	m.Products = normalizeIDs(m.Products)
	if m.Accumulation <= 0 {
		m.Accumulation = len(m.Products)
	}

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

func (r *InventoryRepositoryFS) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, invdom.ErrInvalidModelID
	}

	iter := r.col().Where("modelId", "==", modelID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

func (r *InventoryRepositoryFS) ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]invdom.Mint, error) {
	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	modelID = strings.TrimSpace(modelID)

	if tokenBlueprintID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}
	if modelID == "" {
		return nil, invdom.ErrInvalidModelID
	}

	iter := r.col().
		Where("tokenBlueprintId", "==", tokenBlueprintID).
		Where("modelId", "==", modelID).
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

// ============================================================
// ★ NEW: Upsert (docId = modelId__tokenBlueprintId)
// ============================================================

func (r *InventoryRepositoryFS) UpsertByModelAndToken(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	productIDs []string,
) (invdom.Mint, error) {

	start := time.Now()

	if r == nil || r.Client == nil {
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	mID := strings.TrimSpace(modelID)
	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}
	if mID == "" {
		return invdom.Mint{}, invdom.ErrInvalidModelID
	}

	ids := normalizeIDs(productIDs)
	if len(ids) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}

	docID := buildInventoryDocIDByModel(tbID, mID) // modelId__tokenBlueprintId
	doc := r.col().Doc(docID)
	now := time.Now().UTC()

	log.Printf(
		"[inventory_repo_fs] UpsertByModelAndToken start docId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q productIds=%d",
		docID, tbID, pbID, mID, len(ids),
	)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				ent, err := invdom.NewMint(
					docID,
					tbID,
					pbID,
					mID,
					ids,
					len(ids),
					now,
				)
				if err != nil {
					return err
				}
				ent.ID = docID
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

		existing := normalizeIDs(rec.Products)
		seen := map[string]struct{}{}
		for _, p := range existing {
			seen[p] = struct{}{}
		}

		added := 0
		for _, pid := range ids {
			if pid == "" {
				continue
			}
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			existing = append(existing, pid)
			added++
		}
		sort.Strings(existing)

		updates := []firestore.Update{
			{Path: "products", Value: existing},
			{Path: "updatedAt", Value: now},
			// 参照用に保持（docId には使わないが値は更新しておく）
			{Path: "tokenBlueprintId", Value: tbID},
			{Path: "productBlueprintId", Value: pbID},
			{Path: "modelId", Value: mID},
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
		log.Printf("[inventory_repo_fs] UpsertByModelAndToken error docId=%q err=%v elapsed=%s", docID, err, time.Since(start))
		return invdom.Mint{}, err
	}

	out, err := r.GetByID(ctx, docID)
	if err != nil {
		log.Printf("[inventory_repo_fs] UpsertByModelAndToken GetByID error docId=%q err=%v elapsed=%s", docID, err, time.Since(start))
		return invdom.Mint{}, err
	}

	log.Printf(
		"[inventory_repo_fs] UpsertByModelAndToken done docId=%q accumulation=%d products=%d elapsed=%s",
		docID, out.Accumulation, len(out.Products), time.Since(start),
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

// ★ docId = modelId__tokenBlueprintId
func buildInventoryDocIDByModel(tokenBlueprintID, modelID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(modelID) + "__" + sanitize(tokenBlueprintID)
}
