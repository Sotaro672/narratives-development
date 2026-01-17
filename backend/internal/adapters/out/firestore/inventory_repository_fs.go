// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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

// ============================================================
// Firestore record shape (entity.go 準拠・互換なし)
//
// inventories/{docId}
// - docId = productBlueprintId__tokenBlueprintId
//
// stock: {
//   "{modelId}": {
//     products: ["{productId}", ...],
//     accumulation: 123,
//     reservedByOrder: { "{orderId}": 2, ... },
//     reservedCount: 3
//   }
// }
// modelIds: ["{modelId}", ...]
//
// NOTE:
// - 旧式（products を map で持つ）との互換は削除。
// - docId と同値の冗長フィールド "id" は廃止（持たない）。
// ============================================================

type modelStockRecord struct {
	Products        []string       `firestore:"products"`
	Accumulation    int            `firestore:"accumulation"`
	ReservedByOrder map[string]int `firestore:"reservedByOrder"`
	ReservedCount   int            `firestore:"reservedCount"`
}

type inventoryRecord struct {
	TokenBlueprintID   string                      `firestore:"tokenBlueprintId"`
	ProductBlueprintID string                      `firestore:"productBlueprintId"`
	Stock              map[string]modelStockRecord `firestore:"stock"`
	ModelIDs           []string                    `firestore:"modelIds"`
	CreatedAt          time.Time                   `firestore:"createdAt"`
	UpdatedAt          time.Time                   `firestore:"updatedAt"`
}

func toRecord(m invdom.Mint) inventoryRecord {
	stock := normalizeStockRecord(stockRecordFromDomain(m.Stock))

	modelIDs := normalizeModelIDs(m.ModelIDs)
	if len(modelIDs) == 0 {
		modelIDs = modelIDsFromStockRecord(stock)
	}

	return inventoryRecord{
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		Stock:              stock,
		ModelIDs:           modelIDs,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func fromRecord(docID string, rec inventoryRecord) invdom.Mint {
	stock := normalizeStockRecord(rec.Stock)

	modelIDs := normalizeModelIDs(rec.ModelIDs)
	if len(modelIDs) == 0 {
		modelIDs = modelIDsFromStockRecord(stock)
	}

	out := invdom.Mint{
		ID:                 strings.TrimSpace(docID),
		TokenBlueprintID:   strings.TrimSpace(rec.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(rec.ProductBlueprintID),
		Stock:              stockDomainFromRecord(stock),
		ModelIDs:           modelIDs,
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}

	if out.CreatedAt.IsZero() {
		out.CreatedAt = out.UpdatedAt
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

	m.TokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)
	m.ProductBlueprintID = strings.TrimSpace(m.ProductBlueprintID)

	if m.TokenBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if m.ProductBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}

	// id が空なら docId = productBlueprintId__tokenBlueprintId を採用
	m.ID = strings.TrimSpace(m.ID)
	if m.ID == "" {
		m.ID = buildInventoryDocIDByProduct(m.TokenBlueprintID, m.ProductBlueprintID)
	}

	doc := r.col().Doc(m.ID)

	rec := toRecord(m)
	m.ID = doc.ID

	if _, err := doc.Set(ctx, rec); err != nil {
		return invdom.Mint{}, err
	}

	return r.GetByID(ctx, doc.ID)
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
	if r == nil || r.Client == nil {
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	id := strings.TrimSpace(m.ID)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}

	m.TokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)
	m.ProductBlueprintID = strings.TrimSpace(m.ProductBlueprintID)
	if m.TokenBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if m.ProductBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}

	if _, err := r.col().Doc(id).Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err
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

	if _, err := r.col().Doc(id).Set(ctx, rec); err != nil {
		return invdom.Mint{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *InventoryRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("inventory repo is nil")
	}

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
	if r == nil || r.Client == nil {
		return nil, errors.New("inventory repo is nil")
	}

	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	if tokenBlueprintID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}

	iter := r.col().Where("tokenBlueprintId", "==", tokenBlueprintID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

func (r *InventoryRepositoryFS) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("inventory repo is nil")
	}

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, invdom.ErrInvalidProductBlueprintID
	}

	iter := r.col().Where("productBlueprintId", "==", productBlueprintID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

func (r *InventoryRepositoryFS) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("inventory repo is nil")
	}

	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, invdom.ErrInvalidModelID
	}

	iter := r.col().Documents(ctx)
	defer iter.Stop()

	all, err := readAllInventoryDocs(iter)
	if err != nil {
		return nil, err
	}

	out := make([]invdom.Mint, 0, len(all))
	for _, m := range all {
		if hasModelStock(m.Stock, modelID) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *InventoryRepositoryFS) ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]invdom.Mint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("inventory repo is nil")
	}

	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	modelID = strings.TrimSpace(modelID)

	if tokenBlueprintID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}
	if modelID == "" {
		return nil, invdom.ErrInvalidModelID
	}

	iter := r.col().Where("tokenBlueprintId", "==", tokenBlueprintID).Documents(ctx)
	defer iter.Stop()

	all, err := readAllInventoryDocs(iter)
	if err != nil {
		return nil, err
	}

	out := make([]invdom.Mint, 0, len(all))
	for _, m := range all {
		if hasModelStock(m.Stock, modelID) {
			out = append(out, m)
		}
	}
	return out, nil
}

// ============================================================
// Reservation operations
// ============================================================

func (r *InventoryRepositoryFS) ReserveByOrder(
	ctx context.Context,
	inventoryID string,
	modelID string,
	orderID string,
	qty int,
) error {
	start := time.Now()

	if r == nil || r.Client == nil {
		return errors.New("inventory repo is nil")
	}

	inventoryID = strings.TrimSpace(inventoryID)
	modelID = strings.TrimSpace(modelID)
	orderID = strings.TrimSpace(orderID)

	if inventoryID == "" {
		return invdom.ErrInvalidMintID
	}
	if modelID == "" {
		return invdom.ErrInvalidModelID
	}
	if orderID == "" {
		return errors.New("inventory repo: orderID is empty")
	}
	if qty <= 0 {
		return errors.New("inventory repo: qty must be > 0")
	}

	doc := r.col().Doc(inventoryID)
	now := time.Now().UTC()

	log.Printf(
		"[inventory_repo_fs] ReserveByOrder start inventoryId=%q modelId=%q orderId=%q qty=%d",
		inventoryID, modelID, orderID, qty,
	)

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

		stock := rec.Stock
		if stock == nil {
			stock = map[string]modelStockRecord{}
		}

		ms, ok := stock[modelID]
		if !ok {
			return fmt.Errorf("inventory repo: model stock not found modelId=%s", modelID)
		}

		if ms.ReservedByOrder == nil {
			ms.ReservedByOrder = map[string]int{}
		}

		if existing, ok := ms.ReservedByOrder[orderID]; ok && existing == qty {
			return nil
		}

		ms.ReservedByOrder[orderID] = qty
		ms = normalizeModelStockRecord(ms)

		if ms.ReservedCount > ms.Accumulation {
			return fmt.Errorf(
				"inventory repo: insufficient stock (modelId=%s accumulation=%d reservedCount=%d orderId=%s qty=%d)",
				modelID, ms.Accumulation, ms.ReservedCount, orderID, qty,
			)
		}

		stock[modelID] = ms
		stock = normalizeStockRecord(stock)

		modelIDs := normalizeModelIDs(rec.ModelIDs)
		if !containsString(modelIDs, modelID) {
			modelIDs = append(modelIDs, modelID)
			modelIDs = normalizeModelIDs(modelIDs)
		}

		updates := []firestore.Update{
			{Path: "stock", Value: stock},
			{Path: "modelIds", Value: modelIDs},
			{Path: "updatedAt", Value: now},
		}

		return tx.Update(doc, updates)
	})
	if err != nil {
		log.Printf(
			"[inventory_repo_fs] ReserveByOrder error inventoryId=%q modelId=%q orderId=%q qty=%d err=%v elapsed=%s",
			inventoryID, modelID, orderID, qty, err, time.Since(start),
		)
		return err
	}

	log.Printf(
		"[inventory_repo_fs] ReserveByOrder done inventoryId=%q modelId=%q orderId=%q qty=%d elapsed=%s",
		inventoryID, modelID, orderID, qty, time.Since(start),
	)

	return nil
}

// ============================================================
// Accumulation operations (deprecated)
// ============================================================

func (r *InventoryRepositoryFS) IncrementAccumulation(ctx context.Context, id string, delta int) (invdom.Mint, error) {
	_ = ctx
	_ = id
	_ = delta
	return invdom.Mint{}, errors.New("IncrementAccumulation is deprecated (use Stock accumulation per modelId/productId)")
}

// ============================================================
// Upsert
// - Stock[modelId].Products に productId を追記（UNION）
// - ReservedByOrder / ReservedCount は保持
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

	docID := buildInventoryDocIDByProduct(tbID, pbID)
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
				ms := modelStockRecord{
					Products:        ids,
					ReservedByOrder: map[string]int{},
				}
				ms = normalizeModelStockRecord(ms)

				rec := inventoryRecord{
					TokenBlueprintID:   tbID,
					ProductBlueprintID: pbID,
					Stock:              map[string]modelStockRecord{mID: ms},
					ModelIDs:           []string{mID},
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				rec.Stock = normalizeStockRecord(rec.Stock)
				rec.ModelIDs = normalizeModelIDs(rec.ModelIDs)
				return tx.Set(doc, rec)
			}
			return err
		}

		var rec inventoryRecord
		if err := snap.DataTo(&rec); err != nil {
			return err
		}

		stock := rec.Stock
		if stock == nil {
			stock = map[string]modelStockRecord{}
		}

		ms := stock[mID]
		if ms.Products == nil {
			ms.Products = []string{}
		}

		merged := unionStrings(ms.Products, ids)
		ms.Products = merged

		ms = normalizeModelStockRecord(ms)
		stock[mID] = ms
		stock = normalizeStockRecord(stock)

		modelIDs := normalizeModelIDs(rec.ModelIDs)
		if !containsString(modelIDs, mID) {
			modelIDs = append(modelIDs, mID)
			modelIDs = normalizeModelIDs(modelIDs)
		}

		updates := []firestore.Update{
			{Path: "stock", Value: stock},
			{Path: "modelIds", Value: modelIDs},
			{Path: "updatedAt", Value: now},
			{Path: "tokenBlueprintId", Value: tbID},
			{Path: "productBlueprintId", Value: pbID},
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
		"[inventory_repo_fs] UpsertByModelAndToken done docId=%q models=%d elapsed=%s",
		docID, len(out.Stock), time.Since(start),
	)

	return out, nil
}

func (r *InventoryRepositoryFS) UpsertByProductBlueprintAndToken(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	productIDs []string,
) (invdom.Mint, error) {
	return r.UpsertByModelAndToken(ctx, tokenBlueprintID, productBlueprintID, modelID, productIDs)
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

func normalizeModelIDs(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
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
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildInventoryDocIDByProduct(tokenBlueprintID, productBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(productBlueprintID) + "__" + sanitize(tokenBlueprintID)
}

// ------------------------------------------------------------
// record normalizers
// ------------------------------------------------------------

func normalizeModelStockRecord(ms modelStockRecord) modelStockRecord {
	ms.Products = normalizeIDs(ms.Products)
	if len(ms.Products) == 0 {
		ms.Products = nil
	}

	ms.Accumulation = len(ms.Products)

	rbo := map[string]int{}
	var sum int
	for oid, n := range ms.ReservedByOrder {
		oid = strings.TrimSpace(oid)
		if oid == "" {
			continue
		}
		if n <= 0 {
			continue
		}
		rbo[oid] = n
		sum += n
	}
	if len(rbo) == 0 {
		rbo = nil
		sum = 0
	}
	ms.ReservedByOrder = rbo
	ms.ReservedCount = sum

	return ms
}

func normalizeStockRecord(raw map[string]modelStockRecord) map[string]modelStockRecord {
	if raw == nil {
		return nil
	}
	out := map[string]modelStockRecord{}
	for modelID, ms := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		nms := normalizeModelStockRecord(ms)

		hasProducts := len(nms.Products) > 0
		hasReserved := len(nms.ReservedByOrder) > 0
		if !hasProducts && !hasReserved {
			continue
		}

		out[modelID] = nms
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func modelIDsFromStockRecord(stock map[string]modelStockRecord) []string {
	if stock == nil {
		return nil
	}
	out := make([]string, 0, len(stock))
	for k := range stock {
		k = strings.TrimSpace(k)
		if k != "" {
			out = append(out, k)
		}
	}
	return normalizeModelIDs(out)
}

// ------------------------------------------------------------
// domain <-> record stock conversion
// ------------------------------------------------------------

func stockRecordFromDomain(raw map[string]invdom.ModelStock) map[string]modelStockRecord {
	if raw == nil {
		return nil
	}
	out := map[string]modelStockRecord{}
	for modelID, ms := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}

		rec := modelStockRecord{
			Products:        nil,
			Accumulation:    0,
			ReservedByOrder: nil,
			ReservedCount:   0,
		}

		if len(ms.Products) > 0 {
			rec.Products = normalizeIDs(ms.Products)
		}

		if ms.ReservedByOrder != nil {
			rec.ReservedByOrder = map[string]int{}
			for oid, n := range ms.ReservedByOrder {
				oid = strings.TrimSpace(oid)
				if oid == "" || n <= 0 {
					continue
				}
				rec.ReservedByOrder[oid] = n
			}
		}

		rec = normalizeModelStockRecord(rec)

		hasProducts := len(rec.Products) > 0
		hasReserved := len(rec.ReservedByOrder) > 0
		if !hasProducts && !hasReserved {
			continue
		}

		out[modelID] = rec
	}

	return normalizeStockRecord(out)
}

func stockDomainFromRecord(raw map[string]modelStockRecord) map[string]invdom.ModelStock {
	if raw == nil {
		return nil
	}
	out := map[string]invdom.ModelStock{}
	for modelID, msr := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		msr = normalizeModelStockRecord(msr)

		var ms invdom.ModelStock

		ms.Products = normalizeIDs(msr.Products)
		ms.Accumulation = msr.Accumulation

		if msr.ReservedByOrder != nil {
			ms.ReservedByOrder = map[string]int{}
			for oid, n := range msr.ReservedByOrder {
				oid = strings.TrimSpace(oid)
				if oid == "" || n <= 0 {
					continue
				}
				ms.ReservedByOrder[oid] = n
			}
		}
		ms.ReservedCount = msr.ReservedCount

		hasProducts := len(ms.Products) > 0
		hasReserved := len(ms.ReservedByOrder) > 0
		if !hasProducts && !hasReserved {
			continue
		}

		out[modelID] = ms
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func hasModelStock(stock map[string]invdom.ModelStock, modelID string) bool {
	if stock == nil {
		return false
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	ms, ok := stock[modelID]
	if !ok {
		return false
	}
	if len(ms.Products) > 0 {
		return true
	}
	if len(ms.ReservedByOrder) > 0 {
		return true
	}
	return false
}

func unionStrings(a []string, b []string) []string {
	set := map[string]struct{}{}
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		set[s] = struct{}{}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		set[s] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
