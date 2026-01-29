// backend/internal/adapters/out/firestore/productBlueprint_history_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHistoryRepositoryFS implements ProductBlueprintHistoryRepo
// using Firestore の
// product_blueprints_history/{blueprintId}/versions/{1,2,3,...}
// サブコレクションを利用する実装。
type ProductBlueprintHistoryRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintHistoryRepositoryFS(client *firestore.Client) *ProductBlueprintHistoryRepositoryFS {
	return &ProductBlueprintHistoryRepositoryFS{Client: client}
}

// コンパイル時チェック: interface 満たしているか
var _ pbdom.ProductBlueprintHistoryRepo = (*ProductBlueprintHistoryRepositoryFS)(nil)

// historyCol: product_blueprints_history/{blueprintId}/versions
func (r *ProductBlueprintHistoryRepositoryFS) historyCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("versions")
}

// SaveSnapshot は、ライブの ProductBlueprint をそのままスナップショットとして保存する。
// - ドキュメントパス: product_blueprints_history/{pb.ID}/versions/{1,2,3,...}
// - UpdatedAt/UpdatedBy は ProductBlueprint 側の値をそのまま利用。
// - 連番 version 自体はドメインには持たず、このリポジトリ内でのみ index として管理する。
func (r *ProductBlueprintHistoryRepositoryFS) SaveSnapshot(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID := strings.TrimSpace(pb.ID)
	if blueprintID == "" {
		return pbdom.ErrInvalidID
	}

	// UpdatedAt / CreatedAt 補完
	if pb.UpdatedAt.IsZero() {
		pb.UpdatedAt = time.Now().UTC()
	}
	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = pb.UpdatedAt
	}

	hCol := r.historyCol(blueprintID)

	// -------------------------
	// ★ 直近の index を取得して nextIndex を決定
	//     - index フィールドで降順ソート → 先頭の index + 1
	//     - 1 件も無ければ 1 から開始
	// -------------------------
	var nextIndex int
	q := hCol.OrderBy("index", firestore.Desc).Limit(1)
	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("SaveSnapshot: get latest index failed: %w", err)
	}

	if len(snaps) == 0 {
		nextIndex = 1
	} else {
		data := snaps[0].Data()
		cur := 0
		if v, ok := data["index"]; ok {
			switch x := v.(type) {
			case int64:
				cur = int(x)
			case int:
				cur = x
			case float64:
				cur = int(x)
			}
		}
		if cur <= 0 {
			nextIndex = 1
		} else {
			nextIndex = cur + 1
		}
	}

	// docID を "1", "2", ... の文字列にする
	docID := fmt.Sprintf("%d", nextIndex)
	docRef := hCol.Doc(docID)

	// 既存の productBlueprintToDoc を流用してフィールド構成を揃える
	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return err
	}

	// history 用のメタ情報
	data["id"] = blueprintID
	data["index"] = nextIndex                     // ★ 連番
	data["historyUpdatedAt"] = pb.UpdatedAt.UTC() // 履歴としての時刻
	if pb.UpdatedBy != nil {
		if s := strings.TrimSpace(*pb.UpdatedBy); s != "" {
			data["historyUpdatedBy"] = s
		}
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

// ListByProductBlueprintID は、指定された productBlueprintID に紐づく
// 履歴 ProductBlueprint 一覧を、新しい順（index 降順）で返す。
// LogCard 側では ProductBlueprint.UpdatedAt / UpdatedBy を利用する想定。
func (r *ProductBlueprintHistoryRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, pbdom.ErrInvalidID
	}

	// index（1,2,3,...）の降順で取得
	q := r.historyCol(productBlueprintID).OrderBy("index", firestore.Desc)

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		// UpdatedAt / UpdatedBy は、historyUpdatedAt / historyUpdatedBy があればそちらを優先する。
		if t, ok := data["historyUpdatedAt"].(time.Time); ok && !t.IsZero() {
			pb.UpdatedAt = t.UTC()
		}
		if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			pb.UpdatedBy = &s
		}

		out = append(out, pb)
	}

	return out, nil
}

// ============================================================
// Local helpers (moved/split対応): productBlueprintToDoc / docToProductBlueprint
// ============================================================

// productBlueprintToDoc converts domain ProductBlueprint to Firestore document map.
// It respects `firestore:"..."` struct tags on the domain model.
func productBlueprintToDoc(pb pbdom.ProductBlueprint, createdAt, updatedAt time.Time) (map[string]any, error) {
	// keep the same behavior as callers expect: ensure timestamps are reflected
	if !createdAt.IsZero() {
		pb.CreatedAt = createdAt
	}
	if !updatedAt.IsZero() {
		pb.UpdatedAt = updatedAt
	}
	return structToFirestoreMap(pb)
}

// docToProductBlueprint converts Firestore document snapshot to domain ProductBlueprint.
// Unknown fields in the document (e.g., history metadata) are ignored by DataTo.
func docToProductBlueprint(snap *firestore.DocumentSnapshot) (pbdom.ProductBlueprint, error) {
	if snap == nil {
		return pbdom.ProductBlueprint{}, errors.New("docToProductBlueprint: snapshot is nil")
	}
	var pb pbdom.ProductBlueprint
	if err := snap.DataTo(&pb); err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return pb, nil
}

// structToFirestoreMap converts a struct (or pointer to struct) into a map using `firestore` tags.
// Supports omitempty, "-" and embedded structs. This is intentionally minimal but sufficient
// to keep existing Firestore field layout stable after repository refactors.
func structToFirestoreMap(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return map[string]any{}, nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("structToFirestoreMap: expected struct, got %s", rv.Kind())
	}

	out := make(map[string]any)
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			// unexported
			continue
		}

		tag := sf.Tag.Get("firestore")
		name, omitEmpty, skip := parseFirestoreTag(tag)

		fv := rv.Field(i)

		// embedded struct: flatten if no explicit tag name and not skipped
		if sf.Anonymous && !skip && (name == "" || name == sf.Name) {
			sub, err := valueToFirestoreAny(fv)
			if err != nil {
				return nil, err
			}
			if m, ok := sub.(map[string]any); ok {
				for k, v := range m {
					out[k] = v
				}
			} else if sub != nil {
				// embedded non-struct; fall back to field name
				out[sf.Name] = sub
			}
			continue
		}

		if skip {
			continue
		}
		if name == "" {
			name = sf.Name
		}

		valAny, err := valueToFirestoreAny(fv)
		if err != nil {
			return nil, err
		}
		if omitEmpty && isEmptyValue(fv) {
			continue
		}

		out[name] = valAny
	}

	return out, nil
}

func valueToFirestoreAny(v reflect.Value) (any, error) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		// Special-case time.Time
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t := v.Interface().(time.Time)
			return t, nil
		}
		return structToFirestoreMap(v.Interface())
	case reflect.Slice, reflect.Array:
		// []byte should remain []byte
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			return v.Bytes(), nil
		}
		n := v.Len()
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			item, err := valueToFirestoreAny(v.Index(i))
			if err != nil {
				return nil, err
			}
			arr = append(arr, item)
		}
		return arr, nil
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("valueToFirestoreAny: map key must be string, got %s", v.Type().Key().Kind())
		}
		m := make(map[string]any)
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			item, err := valueToFirestoreAny(iter.Value())
			if err != nil {
				return nil, err
			}
			m[k] = item
		}
		return m, nil
	default:
		return v.Interface(), nil
	}
}

func parseFirestoreTag(tag string) (name string, omitempty bool, skip bool) {
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	if name == "" {
		name = ""
	}
	for _, p := range parts[1:] {
		if strings.TrimSpace(p) == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).IsZero()
		}
		// non-time structs are considered non-empty
		return false
	default:
		return false
	}
}
