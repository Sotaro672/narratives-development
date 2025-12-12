// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

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

func (r *MintRepositoryFS) Create(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	col := r.Client.Collection("mints")

	// ID が空なら自動採番
	var docRef *firestore.DocumentRef
	if m.ID == "" {
		docRef = col.NewDoc()
		m.ID = docRef.ID
	} else {
		docRef = col.Doc(m.ID)
	}

	// CreatedAt がゼロならここで補完（通常は usecase 側で埋めている想定）
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	// ドメインの Validate
	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	// ★ products は「productId の配列」で保存する（"" を保存しない）
	//   - 旧/新スキーマ（slice/map）どちらが来ても、保存時は []string に正規化する
	productIDs := normalizeProductsToIDs(any(m.Products))

	// Firestore に保存するデータ
	// 🔸 ここでドメインのフィールドを落とさないように明示的にマッピングする
	data := map[string]interface{}{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         productIDs, // ← 常に []string
		"createdAt":        m.CreatedAt,
		"createdBy":        m.CreatedBy,
		"minted":           m.Minted,
	}

	// mintedAt（任意）
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}

	// ★ ScheduledBurnDate（任意）も保存
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	// ★ InspectionID（任意）も保存
	//    InspectionBatch に ID フィールドを追加し、Usecase 側で m.InspectionID に詰めた値がここに反映される想定
	if m.InspectionID != "" {
		data["inspectionId"] = m.InspectionID
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}
