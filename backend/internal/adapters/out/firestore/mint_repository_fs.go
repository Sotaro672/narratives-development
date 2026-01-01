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
	if rv.Kind() == reflect.Ptr {
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
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
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

func pickFirstNonEmptyStringField(v any, candidates []string) string {
	for _, name := range candidates {
		if s := strings.TrimSpace(getStringFieldIfExists(v, name)); s != "" {
			return s
		}
	}
	return ""
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
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return []string{}
		}
		v = v.Elem()
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

func asBool(v any) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

// s trims helper_repository_fs.go's asString(v any) result.
func s(v any) string {
	return strings.TrimSpace(asString(v))
}

// asTimeUTC adapts helper_repository_fs.go's asTime(v any) (time.Time, bool) to UTC time.Time.
func asTimeUTC(v any) time.Time {
	if tt, ok := asTime(v); ok {
		return tt.UTC()
	}
	return time.Time{}
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

// ============================================================
// minted/mintedAt normalization (self-heal)
// ============================================================

// normalizeMintedFields ensures Mint.Validate() won't fail due to inconsistency.
// Rules:
// - mintedAt != nil => minted must be true
// - minted == true but mintedAt == nil => mintedAt is auto-filled with now (UTC)
// - otherwise => minted=false, mintedAt=nil
func normalizeMintedFields(minted bool, mintedAt *time.Time) (bool, *time.Time) {
	if mintedAt != nil && !mintedAt.IsZero() {
		return true, mintedAt
	}
	if minted {
		now := time.Now().UTC()
		return true, &now
	}
	return false, nil
}

// If Firestore doc is inconsistent (minted=false but mintedAt exists), fix minted=true in Firestore.
func healDocMintedInconsistency(ctx context.Context, doc *firestore.DocumentSnapshot) {
	if doc == nil || !doc.Exists() {
		return
	}
	data := doc.Data()
	minted := asBool(data["minted"])
	mintedAt := asTimePtr(data["mintedAt"])
	if mintedAt != nil && !mintedAt.IsZero() && !minted {
		_, _ = doc.Ref.Update(ctx, []firestore.Update{
			{Path: "minted", Value: true},
		})
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
	m.BrandID = s(data["brandId"])
	m.TokenBlueprintID = s(data["tokenBlueprintId"])

	// products: Firestore は array を正とする（互換で map も吸える）
	ids := normalizeProductsToIDs(data["products"])
	sort.Strings(ids)
	m.Products = idsToProductsMap(ids)

	m.CreatedBy = s(data["createdBy"])
	m.CreatedAt = asTimeUTC(data["createdAt"])

	// minted/mintedAt を必ず整合させる
	rawMinted := asBool(data["minted"])
	rawMintedAt := asTimePtr(data["mintedAt"])
	m.Minted, m.MintedAt = normalizeMintedFields(rawMinted, rawMintedAt)

	m.ScheduledBurnDate = asTimePtr(data["scheduledBurnDate"])

	// ✅ onchain結果（任意）
	txSig := ""
	for _, k := range []string{"onChainTxSignature", "onchainTxSignature", "txSignature", "signature"} {
		if sv := s(data[k]); sv != "" {
			txSig = sv
			break
		}
	}
	if txSig != "" {
		setStringFieldIfExists(&m, "OnChainTxSignature", txSig)
		setStringFieldIfExists(&m, "OnchainTxSignature", txSig)
		setStringFieldIfExists(&m, "TxSignature", txSig)
		setStringFieldIfExists(&m, "Signature", txSig)
	}

	mintAddr := ""
	for _, k := range []string{"onChainMintAddress", "onchainMintAddress", "mintAddress"} {
		if sv := s(data[k]); sv != "" {
			mintAddr = sv
			break
		}
	}
	if mintAddr != "" {
		setStringFieldIfExists(&m, "MintAddress", mintAddr)
		setStringFieldIfExists(&m, "OnChainMintAddress", mintAddr)
		setStringFieldIfExists(&m, "OnchainMintAddress", mintAddr)
	}

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

	// Firestore には products を []string で保存する（正テーブル準拠）
	productIDs := normalizeProductsToIDs(any(m.Products))
	sort.Strings(productIDs)

	// まず存在チェックして「createdAt を上書きしない」かつ「minted を壊さない」ようにする
	existingSnap, getErr := docRef.Get(ctx)
	exists := getErr == nil
	if getErr != nil && status.Code(getErr) != codes.NotFound {
		return mintdom.Mint{}, getErr
	}

	// ---- exists の場合は「既存の minted/mintedAt/txSignature/mintAddress」を優先して保持する ----
	existingMinted := false
	var existingMintedAt *time.Time
	existingTxSig := ""
	existingMintAddr := ""

	if exists && existingSnap != nil && existingSnap.Exists() {
		edata := existingSnap.Data()

		rawMinted := asBool(edata["minted"])
		rawMintedAt := asTimePtr(edata["mintedAt"])
		existingMinted, existingMintedAt = normalizeMintedFields(rawMinted, rawMintedAt)

		for _, k := range []string{"onChainTxSignature", "onchainTxSignature", "txSignature", "signature"} {
			if sv := s(edata[k]); sv != "" {
				existingTxSig = sv
				break
			}
		}
		for _, k := range []string{"onChainMintAddress", "onchainMintAddress", "mintAddress"} {
			if sv := s(edata[k]); sv != "" {
				existingMintAddr = sv
				break
			}
		}
	}

	// minted/mintedAt を正規化して Validate
	m.Minted, m.MintedAt = normalizeMintedFields(m.Minted, m.MintedAt)
	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	data := map[string]any{
		"brandId":          strings.TrimSpace(m.BrandID),
		"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),
		"products":         productIDs,
		"createdBy":        strings.TrimSpace(m.CreatedBy),
	}

	// createdAt は「新規時のみ」入れる
	if !exists {
		data["createdAt"] = m.CreatedAt.UTC()
	}

	// minted/mintedAt は「exists の場合は既存を優先」
	if exists {
		data["minted"] = existingMinted
		if existingMintedAt != nil && !existingMintedAt.IsZero() {
			data["mintedAt"] = existingMintedAt.UTC()
		}
	} else {
		data["minted"] = m.Minted
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			data["mintedAt"] = m.MintedAt.UTC()
		}
	}

	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	// ✅ onchain結果（exists の場合は既存があれば既存優先）
	sig := strings.TrimSpace(existingTxSig)
	if sig == "" {
		sig = pickFirstNonEmptyStringField(m, []string{"OnChainTxSignature", "OnchainTxSignature", "TxSignature", "Signature"})
	}
	if sig != "" {
		data["txSignature"] = sig
	}

	addr := strings.TrimSpace(existingMintAddr)
	if addr == "" {
		addr = pickFirstNonEmptyStringField(m, []string{"MintAddress", "OnChainMintAddress", "OnchainMintAddress"})
	}
	if addr != "" {
		data["mintAddress"] = addr
	}

	// 新規は Create、既存は MergeAll
	if !exists {
		if _, err := docRef.Create(ctx, data); err != nil {
			if status.Code(err) != codes.AlreadyExists {
				return mintdom.Mint{}, err
			}
			if _, err2 := docRef.Set(ctx, data, firestore.MergeAll); err2 != nil {
				return mintdom.Mint{}, err2
			}
		}
	} else {
		if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
			return mintdom.Mint{}, err
		}
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

	// Update も minted/mintedAt を正規化して Validate
	m.Minted, m.MintedAt = normalizeMintedFields(m.Minted, m.MintedAt)

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	productIDs := normalizeProductsToIDs(any(m.Products))
	sort.Strings(productIDs)

	data := map[string]any{
		"brandId":          strings.TrimSpace(m.BrandID),
		"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),
		"products":         productIDs,
		"createdBy":        strings.TrimSpace(m.CreatedBy),
		"minted":           m.Minted,
	}

	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	if sig := pickFirstNonEmptyStringField(m, []string{"OnChainTxSignature", "OnchainTxSignature", "TxSignature", "Signature"}); sig != "" {
		data["txSignature"] = sig
	}
	if addr := pickFirstNonEmptyStringField(m, []string{"MintAddress", "OnChainMintAddress", "OnchainMintAddress"}); addr != "" {
		data["mintAddress"] = addr
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

	// 自己修復: mintedAt があるのに minted=false の doc は minted=true に戻す
	healDocMintedInconsistency(ctx, doc)

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
		sid := strings.TrimSpace(id)
		if sid == "" {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		seen[sid] = struct{}{}
		ids = append(ids, sid)
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
				continue
			}
			return nil, err
		}

		healDocMintedInconsistency(ctx, doc)

		m, err := decodeMintFromDoc(doc)
		if err != nil {
			return nil, err
		}

		key := strings.TrimSpace(doc.Ref.ID)
		if key == "" {
			continue
		}
		out[key] = m
	}

	return out, nil
}
