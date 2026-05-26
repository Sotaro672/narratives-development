// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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
// Firestore record shape
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

func fromRecord(docID string, rec inventoryRecord) invdom.Mint {
	stock := normalizeStockRecord(rec.Stock)

	modelIDs := normalizeModelIDs(rec.ModelIDs)
	if len(modelIDs) == 0 {
		modelIDs = modelIDsFromStockRecord(stock)
	}

	out := invdom.Mint{
		ID:                 docID,
		TokenBlueprintID:   rec.TokenBlueprintID,
		ProductBlueprintID: rec.ProductBlueprintID,
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

// ResolveBlueprintIDsByInventoryID implements invdom.RepositoryPort.
// inventoryID から productBlueprintId と tokenBlueprintId を返す。
// - not found: invdom.ErrNotFound
// - empty id:  invdom.ErrInvalidMintID
func (r *InventoryRepositoryFS) ResolveBlueprintIDsByInventoryID(
	ctx context.Context,
	inventoryID string,
) (productBlueprintID string, tokenBlueprintID string, err error) {
	m, err := r.GetByID(ctx, inventoryID)
	if err != nil {
		return "", "", err
	}
	return m.ProductBlueprintID, m.TokenBlueprintID, nil
}

// ============================================================
// Read
// ============================================================

func (r *InventoryRepositoryFS) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
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

// ============================================================
// Queries
// ============================================================

func (r *InventoryRepositoryFS) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("inventory repo is nil")
	}

	if productBlueprintID == "" {
		return nil, invdom.ErrInvalidProductBlueprintID
	}

	iter := r.col().Where("productBlueprintId", "==", productBlueprintID).Documents(ctx)
	defer iter.Stop()

	return readAllInventoryDocs(iter)
}

// ============================================================
// Transfer後の予約解放
// - stock[modelId].products から productId を削除
// - reservedByOrder[orderId] を削除件数分だけ減算（<=0はキー削除）
// - accumulation/reservedCount を正規化
//
// Contract:
// - transaction-safe
// - idempotent（無ければ removed=0, nil）
// - removedCount は通常 1 だが、重複等に備えて int を返す
// ============================================================

func (r *InventoryRepositoryFS) ReleaseReservationAfterTransfer(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) (removedCount int, err error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("inventory repo is nil")
	}

	pid := productID
	oid := orderID
	if pid == "" {
		return 0, errors.New("inventory repo: productID is empty")
	}
	if oid == "" {
		return 0, errors.New("inventory repo: orderID is empty")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	docRef, modelID, findErr := r.findInventoryDocByProductID(ctx, pid)
	if findErr != nil {
		return 0, findErr
	}
	if docRef == nil || modelID == "" {
		return 0, nil
	}

	err = r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
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
			return nil
		}

		ms, ok := stock[modelID]
		if !ok {
			return nil
		}

		removed := 0
		if len(ms.Products) > 0 {
			newProducts := make([]string, 0, len(ms.Products))
			for _, x := range ms.Products {
				if x == "" {
					continue
				}
				if x == pid {
					removed++
					continue
				}
				newProducts = append(newProducts, x)
			}
			ms.Products = newProducts
		}

		if removed == 0 {
			return nil
		}

		if ms.ReservedByOrder == nil {
			ms.ReservedByOrder = map[string]int{}
		}

		cur := ms.ReservedByOrder[oid]
		cur = cur - removed
		if cur <= 0 {
			delete(ms.ReservedByOrder, oid)
		} else {
			ms.ReservedByOrder[oid] = cur
		}

		ms = normalizeModelStockRecord(ms)
		stock[modelID] = ms

		stock = normalizeStockRecord(stock)
		modelIDs := modelIDsFromStockRecord(stock)

		updates := []firestore.Update{
			{Path: "stock", Value: stock},
			{Path: "modelIds", Value: modelIDs},
			{Path: "updatedAt", Value: now},
		}

		removedCount = removed
		return tx.Update(docRef, updates)
	})

	if err != nil {
		return 0, err
	}

	return removedCount, nil
}

func (r *InventoryRepositoryFS) findInventoryDocByProductID(ctx context.Context, productID string) (*firestore.DocumentRef, string, error) {
	if r == nil || r.Client == nil {
		return nil, "", errors.New("inventory repo is nil")
	}

	pid := productID
	if pid == "" {
		return nil, "", errors.New("inventory repo: productID is empty")
	}

	iter := r.col().Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, "", err
		}

		var rec inventoryRecord
		if err := snap.DataTo(&rec); err != nil {
			return nil, "", err
		}
		if rec.Stock == nil {
			continue
		}

		for modelID, ms := range rec.Stock {
			if modelID == "" {
				continue
			}
			if containsString(ms.Products, pid) {
				return snap.Ref, modelID, nil
			}
		}
	}

	return nil, "", nil
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
	if r == nil || r.Client == nil {
		return errors.New("inventory repo is nil")
	}

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
		return err
	}

	return nil
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
	if r == nil || r.Client == nil {
		return invdom.Mint{}, errors.New("inventory repo is nil")
	}

	tbID := tokenBlueprintID
	pbID := productBlueprintID
	mID := modelID
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
		return invdom.Mint{}, err
	}

	out, err := r.GetByID(ctx, docID)
	if err != nil {
		return invdom.Mint{}, err
	}

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
		if k != "" {
			out = append(out, k)
		}
	}
	return normalizeModelIDs(out)
}

func stockDomainFromRecord(raw map[string]modelStockRecord) map[string]invdom.ModelStock {
	if raw == nil {
		return nil
	}
	out := map[string]invdom.ModelStock{}
	for modelID, msr := range raw {
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
		if s == "" {
			continue
		}
		set[s] = struct{}{}
	}
	for _, s := range b {
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
