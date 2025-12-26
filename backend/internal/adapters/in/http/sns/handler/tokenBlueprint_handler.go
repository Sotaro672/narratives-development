// backend/internal/adapters/in/http/sns/handler/tokenBlueprint_handler.go
package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// SNSTokenBlueprintHandler serves buyer-facing TokenBlueprint endpoints.
// - GET /sns/token-blueprints/{id}/patch : returns minimal Patch for TokenBlueprintCard.
type SNSTokenBlueprintHandler struct {
	Repo any

	// Optional: brand name resolver (any type)
	// - if it exposes a compatible method, BrandName is filled.
	BrandNameResolver any
}

func NewSNSTokenBlueprintHandler(repo any) http.Handler {
	return &SNSTokenBlueprintHandler{Repo: repo}
}

func NewSNSTokenBlueprintHandlerWithBrandNameResolver(repo any, brandNameResolver any) http.Handler {
	return &SNSTokenBlueprintHandler{Repo: repo, BrandNameResolver: brandNameResolver}
}

func (h *SNSTokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "sns tokenBlueprint handler is nil")
		return
	}

	// ✅ request log（フロントが叩けているか確認）
	log.Printf("[sns_tokenBlueprint] request method=%s path=%s rawQuery=%q", r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		log.Printf("[sns_tokenBlueprint] method not allowed method=%s path=%s", r.Method, r.URL.Path)
		methodNotAllowed(w)
		return
	}

	// Expect: /sns/token-blueprints/{id}/patch
	id, ok := parseTokenBlueprintPatchPath(r.URL.Path)
	if !ok {
		log.Printf("[sns_tokenBlueprint] not found (path mismatch) path=%s", r.URL.Path)
		notFound(w)
		return
	}

	log.Printf("[sns_tokenBlueprint] parsed patch path ok tokenBlueprintId=%q", id)

	patch, err := h.getPatchByID(r.Context(), id)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "failed to get tokenBlueprint patch"
		}

		// ✅ error log
		log.Printf("[sns_tokenBlueprint] getPatch error tokenBlueprintId=%q err=%q", id, msg)

		if isNotFoundError(err) {
			log.Printf("[sns_tokenBlueprint] respond 404 tokenBlueprintId=%q", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"error": msg})
			return
		}
		log.Printf("[sns_tokenBlueprint] respond 500 tokenBlueprintId=%q", id)
		internalError(w, msg)
		return
	}

	// ✅ success log（返す中身が想定通りか確認）
	log.Printf(
		"[sns_tokenBlueprint] ok tokenBlueprintId=%q name=%q symbol=%q brandId=%q brandName=%q minted=%s hasIconUrl=%t",
		id,
		ptrStr(patch.Name),
		ptrStr(patch.Symbol),
		ptrStr(patch.BrandID),
		ptrStr(patch.BrandName),
		ptrBoolStr(patch.Minted),
		strings.TrimSpace(ptrStr(patch.IconURL)) != "",
	)

	// ✅ helper_handler.go の writeJSON(w, code, v) を利用
	// ✅ response log（フロントへ返却されたか確認）
	log.Printf("[sns_tokenBlueprint] respond 200 tokenBlueprintId=%q", id)
	writeJSON(w, http.StatusOK, patch)
}

// ------------------------------
// core
// ------------------------------

func (h *SNSTokenBlueprintHandler) getPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return tbdom.Patch{}, errors.New("tokenBlueprint id is empty")
	}
	if h.Repo == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint repo is nil")
	}

	// call repo.GetByID(ctx, id) by reflection
	v, err := callAny(h.Repo, []string{"GetByID", "GetById"}, ctx, id)
	if err != nil {
		return tbdom.Patch{}, err
	}
	if v == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint is nil")
	}

	patch := toPatchBestEffort(v)

	// fill BrandName (optional)
	if patch.BrandName == nil {
		// 1) try from tokenBlueprint itself
		if bn := pickStringPtrField(v, "BrandName", "brandName"); bn != nil && strings.TrimSpace(*bn) != "" {
			patch.BrandName = bn
		}
	}
	if patch.BrandName == nil && patch.BrandID != nil && strings.TrimSpace(*patch.BrandID) != "" {
		if name, ok := resolveBrandNameBestEffort(ctx, h.BrandNameResolver, strings.TrimSpace(*patch.BrandID)); ok {
			patch.BrandName = strPtr(name)
		}
	}

	return patch, nil
}

// ------------------------------
// path
// ------------------------------

func parseTokenBlueprintPatchPath(path string) (id string, ok bool) {
	p := strings.TrimSuffix(strings.TrimSpace(path), "/")
	// /sns/token-blueprints/{id}/patch
	const prefix = "/sns/token-blueprints/"
	const suffix = "/patch"
	if !strings.HasPrefix(p, prefix) || !strings.HasSuffix(p, suffix) {
		return "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(p, prefix), suffix)
	inner = strings.Trim(inner, "/")
	if inner == "" || strings.Contains(inner, "/") {
		return "", false
	}
	return inner, true
}

// ------------------------------
// mapping (best-effort)
// ------------------------------

func toPatchBestEffort(tb any) tbdom.Patch {
	name := pickStringPtrField(tb, "Name", "name")
	symbol := pickStringPtrField(tb, "Symbol", "symbol")
	brandID := pickStringPtrField(tb, "BrandID", "BrandId", "brandId")
	desc := pickStringPtrField(tb, "Description", "description")

	// IconURL/Url are commonly present on patch-like view models or entities
	iconURL := pickStringPtrField(tb, "IconURL", "IconUrl", "iconUrl", "Icon", "icon")

	minted := pickBoolPtrField(tb, "Minted", "minted")

	return tbdom.Patch{
		Name:        trimPtr(name),
		Symbol:      trimPtr(symbol),
		BrandID:     trimPtr(brandID),
		Description: trimPtr(desc),
		IconURL:     trimPtr(iconURL),
		Minted:      minted,
	}
}

func pickStringPtrField(v any, names ...string) *string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	for _, n := range names {
		f := rv.FieldByName(n)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Pointer && !f.IsNil() {
			f = f.Elem()
		}
		if f.Kind() == reflect.String {
			s := f.String()
			return &s
		}
	}
	return nil
}

func pickBoolPtrField(v any, names ...string) *bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	for _, n := range names {
		f := rv.FieldByName(n)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		if f.Kind() == reflect.Bool {
			b := f.Bool()
			return &b
		}
	}
	return nil
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func strPtr(s string) *string {
	x := s
	return &x
}

// ✅ log helper: avoid nil pointer noise
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// ✅ log helper: bool pointer to string
func ptrBoolStr(b *bool) string {
	if b == nil {
		return "(nil)"
	}
	if *b {
		return "true"
	}
	return "false"
}

// ------------------------------
// optional brand name resolution (best-effort)
// ------------------------------

func resolveBrandNameBestEffort(ctx context.Context, resolver any, brandID string) (string, bool) {
	if resolver == nil {
		return "", false
	}
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", false
	}

	for _, m := range []string{
		"GetBrandNameByID",
		"GetBrandNameById",
		"BrandNameByID",
		"BrandNameById",
		"ResolveBrandName",
	} {
		s, ok := callStringWithCtxAndID(resolver, m, ctx, brandID)
		if ok {
			return s, true
		}
	}

	if v, err := callAny(resolver, []string{"GetByID", "GetById"}, ctx, brandID); err == nil && v != nil {
		if s := pickStringPtrField(v, "Name", "name", "BrandName", "brandName"); s != nil {
			t := strings.TrimSpace(*s)
			if t != "" {
				return t, true
			}
		}
	}

	return "", false
}

func callStringWithCtxAndID(target any, method string, ctx context.Context, id string) (string, bool) {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return "", false
	}
	m := rv.MethodByName(method)
	if !m.IsValid() {
		return "", false
	}
	out := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(id)})
	if len(out) == 0 {
		return "", false
	}
	if len(out) >= 2 {
		if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
			return "", false
		}
	}
	if s, ok := out[0].Interface().(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", false
		}
		return s, true
	}
	return "", false
}

// ------------------------------
// generic reflection call
// ------------------------------

func callAny(repo any, methodNames []string, args ...any) (any, error) {
	rv := reflect.ValueOf(repo)
	if !rv.IsValid() {
		return nil, errors.New("repo is invalid")
	}

	for _, name := range methodNames {
		m := rv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		in := make([]reflect.Value, 0, len(args))
		for _, a := range args {
			in = append(in, reflect.ValueOf(a))
		}

		out := m.Call(in)
		if len(out) == 0 {
			return nil, nil
		}

		if len(out) >= 2 {
			if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
				return nil, e
			}
		}

		return out[0].Interface(), nil
	}

	return nil, errors.New("method not found")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "errnotfound") || strings.Contains(msg, "404")
}
