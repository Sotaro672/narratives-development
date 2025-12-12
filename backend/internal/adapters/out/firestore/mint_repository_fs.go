// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	mintdom "narratives/internal/domain/mint"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
type MintRepositoryFS struct {
	Client *firestore.Client
}

func NewMintRepositoryFS(client *firestore.Client) *MintRepositoryFS {
	return &MintRepositoryFS{Client: client}
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
			// unwrap interface
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
			// unwrap interface
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
		// value は互換のため空文字
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
	m.ID = strings.TrimSpace(doc.Ref.ID)

	// ✅ 正テーブル（lower camelCase）のみ
	m.BrandID = asString(data["brandId"])
	m.TokenBlueprintID = asString(data["tokenBlueprintId"])
	m.InspectionID = asString(data["inspectionId"])

	// products: Firestore は array を正とする
	ids := normalizeProductsToIDs(data["products"])
	m.Products = idsToProductsMap(ids)

	m.CreatedBy = asString(data["createdBy"])
	m.CreatedAt = asTime(data["createdAt"])

	m.Minted = asBool(data["minted"])
	m.MintedAt = asTimePtr(data["mintedAt"])

	m.ScheduledBurnDate = asTimePtr(data["scheduledBurnDate"])

	// ★ onChainTxSignature はテーブルには存在しても、現状ドメイン Mint にフィールドが無いので扱わない

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}

func (r *MintRepositoryFS) Create(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	col := r.Client.Collection("mints")

	// ID が空なら自動採番
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(m.ID) == "" {
		docRef = col.NewDoc()
		m.ID = docRef.ID
	} else {
		docRef = col.Doc(strings.TrimSpace(m.ID))
		m.ID = docRef.ID
	}

	// CreatedAt がゼロなら補完
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	// ドメイン Validate
	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	// Firestore には products を []string で保存する（正テーブル準拠）
	productIDs := normalizeProductsToIDs(any(m.Products))

	data := map[string]interface{}{
		"brandId":          strings.TrimSpace(m.BrandID),
		"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),
		"inspectionId":     strings.TrimSpace(m.InspectionID),
		"products":         productIDs,
		"createdAt":        m.CreatedAt.UTC(),
		"createdBy":        strings.TrimSpace(m.CreatedBy),
		"minted":           m.Minted,
	}

	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}

	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	// ★ onChainTxSignature は Mint ドメインにフィールドが無いのでここでは保存しない
	//    （保存したい場合は mintdom.Mint にフィールド追加 → ここで data に入れる）

	if _, err := docRef.Set(ctx, data); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}

// GetByInspectionID:
// - inspectionId（= productionId を格納する運用）から mints を 1 件取得する
func (r *MintRepositoryFS) GetByInspectionID(ctx context.Context, inspectionID string) (mintdom.Mint, error) {
	if r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	iid := strings.TrimSpace(inspectionID)
	if iid == "" {
		return mintdom.Mint{}, errors.New("inspectionID is empty")
	}

	iter := r.Client.Collection("mints").
		Where("inspectionId", "==", iid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return mintdom.Mint{}, mintdom.ErrNotFound
		}
		return mintdom.Mint{}, err
	}

	return decodeMintFromDoc(doc)
}

// ListByInspectionIDs:
// - inspectionId（= productionId）を複数受け取り、該当する mints を map で返す
// - map の key は inspectionId（= productionId）
// - Firestore の "in" は最大 10 件なのでチャンクして取得する
func (r *MintRepositoryFS) ListByInspectionIDs(ctx context.Context, inspectionIDs []string) (map[string]mintdom.Mint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// trim + 空除去 + 重複除去
	uniq := make([]string, 0, len(inspectionIDs))
	seen := make(map[string]struct{}, len(inspectionIDs))
	for _, id := range inspectionIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		uniq = append(uniq, s)
	}

	out := make(map[string]mintdom.Mint, len(uniq))
	if len(uniq) == 0 {
		return out, nil
	}

	const inLimit = 10

	for start := 0; start < len(uniq); start += inLimit {
		end := start + inLimit
		if end > len(uniq) {
			end = len(uniq)
		}
		chunk := uniq[start:end]

		iter := r.Client.Collection("mints").
			Where("inspectionId", "in", chunk).
			Documents(ctx)

		for {
			doc, err := iter.Next()
			if err != nil {
				if errors.Is(err, iterator.Done) {
					break
				}
				iter.Stop()
				return nil, err
			}

			m, err := decodeMintFromDoc(doc)
			if err != nil {
				iter.Stop()
				return nil, err
			}

			key := strings.TrimSpace(m.InspectionID)
			if key == "" {
				// 念のため: inspectionId が無いデータは map に入れない
				continue
			}

			// 同一 inspectionId が複数ある場合は「先勝ち」にする（必要なら上書きに変更可）
			if _, exists := out[key]; !exists {
				out[key] = m
			}
		}

		iter.Stop()
	}

	return out, nil
}
