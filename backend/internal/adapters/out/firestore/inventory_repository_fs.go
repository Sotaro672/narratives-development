// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
	"reflect"
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
// - stock は Firestore では map[string][]string（modelId -> []productId）で保持する
type inventoryRecord struct {
	ID                 string              `firestore:"id"`
	TokenBlueprintID   string              `firestore:"tokenBlueprintId"`
	ProductBlueprintID string              `firestore:"productBlueprintId"`
	Stock              map[string][]string `firestore:"stock"`
	CreatedAt          time.Time           `firestore:"createdAt"`
	UpdatedAt          time.Time           `firestore:"updatedAt"`
}

func toRecord(m invdom.Mint) inventoryRecord {
	return inventoryRecord{
		ID:                 strings.TrimSpace(m.ID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		Stock:              normalizeStockRecord(stockRecordFromDomain(m.Stock)),
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
		Stock:              stockDomainFromRecord(normalizeStockRecord(rec.Stock)),
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}

	// CreatedAt が壊れているケースの保険
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

	m.TokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)
	m.ProductBlueprintID = strings.TrimSpace(m.ProductBlueprintID)
	if m.TokenBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if m.ProductBlueprintID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
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

	// ✅ Firestore Go SDK: MergeAll は map データでのみ使用可能。
	// ここでは record 全体を Set で上書きする（在庫は常に完全形で保持する前提）。
	rec := toRecord(m)
	rec.ID = id

	// NotFound を返したい場合は存在確認（Set は存在しなくても作れてしまうため）
	if _, err := r.col().Doc(id).Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Mint{}, invdom.ErrNotFound
		}
		return invdom.Mint{}, err
	}

	if _, err := r.col().Doc(id).Set(ctx, rec); err != nil {
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

// ListByModelID は Firestore のクエリで stock のキー存在判定ができないため、全件走査でフィルタ
func (r *InventoryRepositoryFS) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
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
// Accumulation operations (deprecated)
// ============================================================

// Accumulation は廃止方針のため、互換のためだけに残す（呼ばれたらエラー）
func (r *InventoryRepositoryFS) IncrementAccumulation(ctx context.Context, id string, delta int) (invdom.Mint, error) {
	_ = ctx
	_ = id
	_ = delta
	return invdom.Mint{}, errors.New("IncrementAccumulation is deprecated (use Stock accumulation per modelId/productId)")
}

// ============================================================
// Upsert (docId = productBlueprintId__tokenBlueprintId)
// - Stock[modelId] に productId を追記（UNION）
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

	docID := buildInventoryDocIDByProduct(tbID, pbID) // productBlueprintId__tokenBlueprintId
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
				// new record
				rec := inventoryRecord{
					ID:                 docID,
					TokenBlueprintID:   tbID,
					ProductBlueprintID: pbID,
					Stock:              map[string][]string{},
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				rec.Stock[mID] = ids
				rec.Stock = normalizeStockRecord(rec.Stock)
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
			stock = map[string][]string{}
		}

		// merge (UNION)
		existing := stock[mID]
		merged := normalizeIDs(append(existing, ids...))
		stock[mID] = merged
		stock = normalizeStockRecord(stock)

		updates := []firestore.Update{
			{Path: "stock", Value: stock},
			{Path: "updatedAt", Value: now},
			{Path: "tokenBlueprintId", Value: tbID},
			{Path: "productBlueprintId", Value: pbID},
			{Path: "id", Value: docID},
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

// ============================================================
// Compatibility method required by inventory.RepositoryPort
// ============================================================

// UpsertByProductBlueprintAndToken is kept for interface compatibility.
// It delegates to UpsertByModelAndToken with the canonical docId rule.
func (r *InventoryRepositoryFS) UpsertByProductBlueprintAndToken(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
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

// stock record normalizer (trim + remove empty + dedupe + sort)
func normalizeStockRecord(raw map[string][]string) map[string][]string {
	if raw == nil {
		return nil
	}
	out := map[string][]string{}
	for modelID, ids := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		nids := normalizeIDs(ids)
		if len(nids) == 0 {
			continue
		}
		out[modelID] = nids
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// docId = productBlueprintId__tokenBlueprintId（期待値どおりの順序）
// NOTE: 引数順は (tokenBlueprintID, productBlueprintID) だが、出力は product__token
func buildInventoryDocIDByProduct(tokenBlueprintID, productBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(productBlueprintID) + "__" + sanitize(tokenBlueprintID)
}

// ------------------------------------------------------------
// domain <-> record stock conversion
// ------------------------------------------------------------

// domain: map[string]invdom.ModelStock -> record: map[string][]string
func stockRecordFromDomain(raw map[string]invdom.ModelStock) map[string][]string {
	if raw == nil {
		return nil
	}
	out := map[string][]string{}
	for modelID, ms := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		ids := modelStockToIDs(ms)
		if len(ids) == 0 {
			continue
		}
		out[modelID] = ids
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// record: map[string][]string -> domain: map[string]invdom.ModelStock
func stockDomainFromRecord(raw map[string][]string) map[string]invdom.ModelStock {
	if raw == nil {
		return nil
	}
	out := map[string]invdom.ModelStock{}
	for modelID, ids := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		nids := normalizeIDs(ids)
		if len(nids) == 0 {
			continue
		}
		out[modelID] = modelStockFromIDs(nids)
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
	ms, ok := stock[modelID]
	if !ok {
		return false
	}
	return len(modelStockToIDs(ms)) > 0
}

// ------------------------------------------------------------
// ModelStock reflection helpers
// - ModelStock が []string の alias でも、struct{Products []string} でも、map 系でも吸収
// ------------------------------------------------------------

// ModelStock -> []string（コピーして返す）
func modelStockToIDs(ms invdom.ModelStock) []string {
	rv := reflect.ValueOf(ms)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			it := rv.Index(i)
			if it.Kind() == reflect.String {
				s := strings.TrimSpace(it.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeIDs(out)

	case reflect.Map:
		// map[string]T のキーを productId として扱う
		if rv.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			s := strings.TrimSpace(iter.Key().String())
			if s != "" {
				out = append(out, s)
			}
		}
		return normalizeIDs(out)

	case reflect.Struct:
		// struct{ Products ... } を探す
		pf := rv.FieldByName("Products")
		if pf.IsValid() {
			switch pf.Kind() {
			case reflect.Slice, reflect.Array:
				out := make([]string, 0, pf.Len())
				for i := 0; i < pf.Len(); i++ {
					it := pf.Index(i)
					if it.Kind() == reflect.String {
						s := strings.TrimSpace(it.String())
						if s != "" {
							out = append(out, s)
						}
					}
				}
				return normalizeIDs(out)

			case reflect.Map:
				if pf.Type().Key().Kind() != reflect.String {
					return nil
				}
				out := make([]string, 0, pf.Len())
				iter := pf.MapRange()
				for iter.Next() {
					s := strings.TrimSpace(iter.Key().String())
					if s != "" {
						out = append(out, s)
					}
				}
				return normalizeIDs(out)
			}
		}
	}

	return nil
}

// []string -> ModelStock（struct でも slice alias でも詰められるだけ詰める）
func modelStockFromIDs(ids []string) invdom.ModelStock {
	var ms invdom.ModelStock
	ids = normalizeIDs(ids)
	if len(ids) == 0 {
		return ms
	}

	rv := reflect.ValueOf(&ms).Elem()
	if !rv.IsValid() {
		return ms
	}

	// 1) ModelStock 自体が slice/array の alias
	if rv.Kind() == reflect.Slice {
		if rv.Type().Elem().Kind() == reflect.String {
			s := reflect.MakeSlice(rv.Type(), 0, len(ids))
			for _, id := range ids {
				s = reflect.Append(s, reflect.ValueOf(id))
			}
			rv.Set(s)
			return ms
		}
	}

	// 2) ModelStock 自体が map の alias（キーに productId を入れる）
	if rv.Kind() == reflect.Map {
		if rv.Type().Key().Kind() == reflect.String {
			rv.Set(reflect.MakeMapWithSize(rv.Type(), len(ids)))
			for _, id := range ids {
				var v reflect.Value
				switch rv.Type().Elem().Kind() {
				case reflect.Bool:
					v = reflect.ValueOf(true).Convert(rv.Type().Elem())
				case reflect.Struct:
					v = reflect.New(rv.Type().Elem()).Elem()
				default:
					v = reflect.Zero(rv.Type().Elem())
				}
				rv.SetMapIndex(reflect.ValueOf(id), v)
			}
			return ms
		}
	}

	// 3) struct の Products フィールドへ
	if rv.Kind() == reflect.Struct {
		pf := rv.FieldByName("Products")
		if pf.IsValid() && pf.CanSet() {
			switch pf.Kind() {
			case reflect.Slice:
				if pf.Type().Elem().Kind() == reflect.String {
					s := reflect.MakeSlice(pf.Type(), 0, len(ids))
					for _, id := range ids {
						s = reflect.Append(s, reflect.ValueOf(id))
					}
					pf.Set(s)
					return ms
				}
			case reflect.Map:
				if pf.Type().Key().Kind() == reflect.String {
					mm := reflect.MakeMapWithSize(pf.Type(), len(ids))
					for _, id := range ids {
						var v reflect.Value
						switch pf.Type().Elem().Kind() {
						case reflect.Bool:
							v = reflect.ValueOf(true).Convert(pf.Type().Elem())
						case reflect.Struct:
							v = reflect.New(pf.Type().Elem()).Elem()
						default:
							v = reflect.Zero(pf.Type().Elem())
						}
						mm.SetMapIndex(reflect.ValueOf(id), v)
					}
					pf.Set(mm)
					return ms
				}
			}
		}
	}

	return ms
}
