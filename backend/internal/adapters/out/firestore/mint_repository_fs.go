// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mintdom "narratives/internal/domain/mint"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
type MintRepositoryFS struct {
	Client *firestore.Client
}

func NewMintRepositoryFS(client *firestore.Client) *MintRepositoryFS {
	return &MintRepositoryFS{Client: client}
}

func (r *MintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("mints")
}

// ============================================================
// Policy A helpers
// - docId = productionId = inspectionId = mintId
// ============================================================

func getStringFieldIfExists(v any, fieldName string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(f.String())
}

func setStringFieldIfExists(ptr any, fieldName string, value string) {
	rv := reflect.ValueOf(ptr)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.String {
		return
	}
	f.SetString(strings.TrimSpace(value))
}

// normalizeProductsToIDs converts Mint.Products into []string (productId list) and removes empty strings.
// - If Products is a slice/array: keeps string elements only (trimmed, non-empty)
// - If Products is a map: uses map keys as productIds (trimmed, non-empty)
// - Otherwise: returns empty slice
func normalizeProductsToIDs(products any) []string {
	if products == nil {
		return []string{}
	}

	v := reflect.ValueOf(products)
	if !v.IsValid() {
		return []string{}
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Interface && !elem.IsNil() {
				elem = elem.Elem()
			}
			if elem.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(elem.String())
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out

	case reflect.Map:
		out := make([]string, 0, v.Len())
		for _, key := range v.MapKeys() {
			k := key
			if k.Kind() == reflect.Interface && !k.IsNil() {
				k = k.Elem()
			}
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out

	default:
		return []string{}
	}
}

// idsToProductsMap converts []string productIds to map[string]string.
// Domain Mint.Products is map[string]string, so we restore in that shape.
func idsToProductsMap(ids []string) map[string]string {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		out[s] = ""
	}
	return out
}

func asString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func asBool(v any) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

func asTime(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t.UTC()
	case *time.Time:
		if t == nil {
			return time.Time{}
		}
		return t.UTC()
	default:
		return time.Time{}
	}
}

func asTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		tt := t.UTC()
		return &tt
	case *time.Time:
		if t == nil || t.IsZero() {
			return nil
		}
		tt := t.UTC()
		return &tt
	default:
		return nil
	}
}

func decodeMintFromDoc(doc *firestore.DocumentSnapshot) (mintdom.Mint, error) {
	if doc == nil || !doc.Exists() {
		return mintdom.Mint{}, errors.New("doc is nil or not exists")
	}

	data := doc.Data()

	var m mintdom.Mint

	// ✅ docId (= productionId/inspectionId/mintId) を Mint.ID として扱う
	docID := strings.TrimSpace(doc.Ref.ID)
	m.ID = docID

	// （存在するなら）InspectionID も docID で揃える（Policy A）
	setStringFieldIfExists(&m, "InspectionID", docID)
	setStringFieldIfExists(&m, "InspectionId", docID)

	// ✅ 正テーブル（lower camelCase）
	m.BrandID = asString(data["brandId"])
	m.TokenBlueprintID = asString(data["tokenBlueprintId"])

	// products: Firestore は array を正とする（互換で map も吸える）
	ids := normalizeProductsToIDs(data["products"])
	m.Products = idsToProductsMap(ids)

	m.CreatedBy = asString(data["createdBy"])
	m.CreatedAt = asTime(data["createdAt"])

	m.Minted = asBool(data["minted"])
	m.MintedAt = asTimePtr(data["mintedAt"])

	m.ScheduledBurnDate = asTimePtr(data["scheduledBurnDate"])

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}
	return m, nil
}

// ============================================================
// CRUD
// ============================================================

func (r *MintRepositoryFS) Create(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	// ✅ Policy A: docId = productionId (= inspectionId)
	// 互換: m.ID が空なら InspectionID から拾う
	docID := strings.TrimSpace(m.ID)
	if docID == "" {
		docID = getStringFieldIfExists(m, "InspectionID")
		if docID == "" {
			docID = getStringFieldIfExists(m, "InspectionId")
		}
	}
	if docID == "" {
		return mintdom.Mint{}, errors.New("mint.ID is empty (docId must be productionId/inspectionId)")
	}

	docRef := r.col().Doc(docID)
	m.ID = docRef.ID

	// （存在するなら）InspectionID も docID で揃える（Policy A）
	setStringFieldIfExists(&m, "InspectionID", docID)
	setStringFieldIfExists(&m, "InspectionId", docID)

	// CreatedAt がゼロなら補完
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	// Firestore には products を []string で保存する（正テーブル準拠）
	productIDs := normalizeProductsToIDs(any(m.Products))
	sort.Strings(productIDs)

	// まず存在チェックして「createdAt を上書きしない」ようにする
	_, getErr := docRef.Get(ctx)
	exists := getErr == nil
	if getErr != nil && status.Code(getErr) != codes.NotFound {
		return mintdom.Mint{}, getErr
	}

	data := map[string]interface{}{
		"brandId":          strings.TrimSpace(m.BrandID),
		"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),
		"products":         productIDs,
		"createdBy":        strings.TrimSpace(m.CreatedBy),
		"minted":           m.Minted,
	}
	// createdAt は「新規時のみ」入れる（既存がある場合は上書きしない）
	if !exists {
		data["createdAt"] = m.CreatedAt.UTC()
	}

	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	// 既存があっても idempotent に更新できるよう Merge で保存
	if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}

// Update updates a Mint (docId is fixed to m.ID under Policy A).
func (r *MintRepositoryFS) Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	docID := strings.TrimSpace(m.ID)
	if docID == "" {
		// 互換: InspectionID があればそこから docId を確定
		docID = getStringFieldIfExists(m, "InspectionID")
		if docID == "" {
			docID = getStringFieldIfExists(m, "InspectionId")
		}
	}
	if docID == "" {
		return mintdom.Mint{}, errors.New("mint.ID is empty")
	}

	docRef := r.col().Doc(docID)
	m.ID = docRef.ID
	setStringFieldIfExists(&m, "InspectionID", docID)
	setStringFieldIfExists(&m, "InspectionId", docID)

	// createdAt がゼロなら既存から補完（Validate を通すため）
	if m.CreatedAt.IsZero() {
		existing, err := r.GetByID(ctx, docID)
		if err != nil {
			return mintdom.Mint{}, err
		}
		m.CreatedAt = existing.CreatedAt
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	productIDs := normalizeProductsToIDs(any(m.Products))
	sort.Strings(productIDs)

	data := map[string]interface{}{
		"brandId":          strings.TrimSpace(m.BrandID),
		"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),
		"products":         productIDs,
		"createdBy":        strings.TrimSpace(m.CreatedBy),
		"minted":           m.Minted,
	}

	// mintedAt / scheduledBurnDate は「値がある時だけ」上書き。
	// 値を消したい場合は、別途 FieldValueDelete を使うAPIを用意してください。
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.Mint{}, mintdom.ErrNotFound
		}
		return mintdom.Mint{}, err
	}

	return m, nil
}

func (r *MintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	docID := strings.TrimSpace(id)
	if docID == "" {
		return errors.New("id is empty")
	}

	_, err := r.col().Doc(docID).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Queries
// ============================================================

// GetByID returns a Mint by docId.
// docId is expected to be productionId/inspectionId/mintId (same value).
func (r *MintRepositoryFS) GetByID(ctx context.Context, id string) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	docID := strings.TrimSpace(id)
	if docID == "" {
		return mintdom.Mint{}, errors.New("id is empty")
	}

	doc, err := r.col().Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.Mint{}, mintdom.ErrNotFound
		}
		return mintdom.Mint{}, err
	}

	return decodeMintFromDoc(doc)
}

// ListByProductionID lists mints by production docIds.
// Expectation: production docId == mint docId, so we Get() by docId for each id.
// Missing docs are treated as "mint not created yet" and skipped.
func (r *MintRepositoryFS) ListByProductionID(ctx context.Context, productionIDs []string) (map[string]mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))
	for _, id := range productionIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}

	out := make(map[string]mintdom.Mint, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	sort.Strings(ids)

	for _, id := range ids {
		doc, err := r.col().Doc(id).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue // mint 未作成
			}
			return nil, err
		}

		m, err := decodeMintFromDoc(doc)
		if err != nil {
			return nil, err
		}

		key := strings.TrimSpace(doc.Ref.ID) // = productionId (= docId)
		if key == "" {
			continue
		}
		out[key] = m
	}

	return out, nil
}
