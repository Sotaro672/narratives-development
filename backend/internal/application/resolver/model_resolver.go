// backend\internal\application\resolver\model_resolver.go
package resolver

import (
	"context"
	"reflect"
	"strings"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ✅ ModelVariation (modelId → modelNumber/size/color/rgb)
// - Firestore の保存 label を正とする（名揺れ吸収はしない）
//   - modelNumber: mv.ModelNumber
//   - size:        mv.Size
//   - color:       mv.Color.name   (Firestore: color(map){ name, rgb })
//   - rgb:         mv.Color.rgb
// ------------------------------------------------------------

type ModelResolved struct {
	ModelNumber string
	Size        string
	Color       string
	RGB         *int
}

// ResolveModelResolved は modelId から modelNumber/size/color/rgb を解決する。
// 取得できなかった場合はゼロ値を返す。
func (r *NameResolver) ResolveModelResolved(ctx context.Context, variationID string) ModelResolved {
	if r == nil || r.modelNumberRepo == nil {
		return ModelResolved{}
	}
	id := strings.TrimSpace(variationID)
	if id == "" {
		return ModelResolved{}
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil || mv == nil {
		return ModelResolved{}
	}

	modelNumber := strings.TrimSpace(mv.ModelNumber)
	size := strings.TrimSpace(mv.Size)

	// Firestore: color(map){ name, rgb }
	colorName, rgb := extractColorNameAndRGBFromModelVariation(mv)

	return ModelResolved{
		ModelNumber: modelNumber,
		Size:        size,
		Color:       strings.TrimSpace(colorName),
		RGB:         rgb,
	}
}

// Firestore の保存 label を正として読む（名揺れ吸収しない）
// - color.name (string)
// - color.rgb  (number)
func extractColorNameAndRGBFromModelVariation(mv *modeldom.ModelVariation) (string, *int) {
	if mv == nil {
		return "", nil
	}

	// mv.Color を reflect で読む（Color の型が struct/map どちらでも対応）
	rv := reflect.ValueOf(mv)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return "", nil
	}

	f := rv.FieldByName("Color")
	if !f.IsValid() {
		return "", nil
	}
	f = deref(f)
	if !f.IsValid() {
		return "", nil
	}

	switch f.Kind() {
	case reflect.Map:
		// color(map)
		name := mapString(f, "name")
		rgb := mapIntPtr(f, "rgb")
		return strings.TrimSpace(name), rgb

	case reflect.Struct:
		// color(struct)
		// Firestore label を正: Name / RGB
		name := structString(f, "Name")
		rgb := structIntPtr(f, "RGB")
		return strings.TrimSpace(name), rgb

	default:
		return "", nil
	}
}

func deref(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func mapString(m reflect.Value, key string) string {
	if !m.IsValid() || m.Kind() != reflect.Map {
		return ""
	}
	kv := m.MapIndex(reflect.ValueOf(key))
	kv = deref(kv)
	if !kv.IsValid() || kv.Kind() != reflect.String {
		return ""
	}
	return kv.String()
}

func mapIntPtr(m reflect.Value, key string) *int {
	if !m.IsValid() || m.Kind() != reflect.Map {
		return nil
	}
	kv := m.MapIndex(reflect.ValueOf(key))
	kv = deref(kv)
	if !kv.IsValid() {
		return nil
	}
	if n, ok := asInt(kv); ok {
		x := n
		return &x
	}
	return nil
}

func structString(s reflect.Value, fieldName string) string {
	if !s.IsValid() || s.Kind() != reflect.Struct {
		return ""
	}
	f := s.FieldByName(fieldName)
	f = deref(f)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

func structIntPtr(s reflect.Value, fieldName string) *int {
	if !s.IsValid() || s.Kind() != reflect.Struct {
		return nil
	}
	f := s.FieldByName(fieldName)
	f = deref(f)
	if !f.IsValid() {
		return nil
	}
	if n, ok := asInt(f); ok {
		x := n
		return &x
	}
	return nil
}

func asInt(v reflect.Value) (int, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		// Firestore number が float で入ってくるケース
		return int(v.Float()), true
	default:
		return 0, false
	}
}
