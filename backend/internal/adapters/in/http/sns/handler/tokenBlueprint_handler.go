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

// ImageURLResolver is a minimal contract for resolving iconUrl from stored objectPath.
type ImageURLResolver interface {
	ResolveForResponse(storedObjectPath string, storedIconURL string) string
}

type SNSTokenBlueprintHandler struct {
	Repo any

	BrandNameResolver   any
	CompanyNameResolver any

	// ✅ icon url resolver (objectPath -> public URL)
	ImageResolver ImageURLResolver
}

func NewSNSTokenBlueprintHandler(repo any) http.Handler {
	return &SNSTokenBlueprintHandler{Repo: repo}
}

func NewSNSTokenBlueprintHandlerWithNameResolver(repo any, nameResolver any) http.Handler {
	return &SNSTokenBlueprintHandler{
		Repo:                repo,
		BrandNameResolver:   nameResolver,
		CompanyNameResolver: nameResolver,
	}
}

// ✅ 推奨: NameResolver + ImageURLResolver を注入
func NewSNSTokenBlueprintHandlerWithNameAndImageResolver(
	repo any,
	nameResolver any,
	imageResolver ImageURLResolver,
) http.Handler {
	return &SNSTokenBlueprintHandler{
		Repo:                repo,
		BrandNameResolver:   nameResolver,
		CompanyNameResolver: nameResolver,
		ImageResolver:       imageResolver,
	}
}

func (h *SNSTokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "sns tokenBlueprint handler is nil")
		return
	}

	log.Printf("[sns_tokenBlueprint] request method=%s path=%s rawQuery=%q", r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		log.Printf("[sns_tokenBlueprint] method not allowed method=%s path=%s", r.Method, r.URL.Path)
		methodNotAllowed(w)
		return
	}

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

	log.Printf(
		"[sns_tokenBlueprint] ok tokenBlueprintId=%q name=%q symbol=%q brandId=%q brandName=%q companyId=%q companyName=%q minted=%s hasIconUrl=%t iconUrl=%q",
		id,
		ptrStr(patch.Name),
		ptrStr(patch.Symbol),
		ptrStr(patch.BrandID),
		ptrStr(patch.BrandName),
		ptrStr(patch.CompanyID),
		ptrStr(patch.CompanyName),
		ptrBoolStr(patch.Minted),
		strings.TrimSpace(ptrStr(patch.IconURL)) != "",
		ptrStr(patch.IconURL),
	)

	log.Printf("[sns_tokenBlueprint] respond 200 tokenBlueprintId=%q", id)
	writeJSON(w, http.StatusOK, patch)
}

func (h *SNSTokenBlueprintHandler) getPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return tbdom.Patch{}, errors.New("tokenBlueprint id is empty")
	}
	if h.Repo == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint repo is nil")
	}

	// ✅ 注入確認ログ（ここが重要）
	log.Printf("[sns_tokenBlueprint] imageResolver injected? %t type=%T", h.ImageResolver != nil, h.ImageResolver)

	v, err := callAny(h.Repo, []string{"GetByID", "GetById"}, ctx, id)
	if err != nil {
		return tbdom.Patch{}, err
	}
	if v == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint is nil")
	}

	log.Printf("[sns_tokenBlueprint] repo returned type=%T", v)

	log.Printf(
		"[sns_tokenBlueprint] icon candidates(raw): IconID=%q IconUrl=%q iconUrl=%q TokenIconURL=%q TokenIconUrl=%q IconObjectPath=%q IconPath=%q IconStoragePath=%q",
		ptrStr(pickStringPtrField(v, "IconID", "IconId", "iconId")),
		ptrStr(pickStringPtrField(v, "IconUrl")),
		ptrStr(pickStringPtrField(v, "iconUrl")),
		ptrStr(pickStringPtrField(v, "TokenIconURL")),
		ptrStr(pickStringPtrField(v, "TokenIconUrl")),
		ptrStr(pickStringPtrField(v, "IconObjectPath")),
		ptrStr(pickStringPtrField(v, "IconPath")),
		ptrStr(pickStringPtrField(v, "IconStoragePath")),
	)

	patch := toPatchBestEffort(v)

	// ✅ IconURL 補完（必ず試す + “空だった理由” をログ）
	if strings.TrimSpace(ptrStr(patch.IconURL)) == "" {
		if h.ImageResolver == nil {
			log.Printf("[sns_tokenBlueprint] iconUrl resolve skipped: ImageResolver is nil")
		} else {
			// 1) objectPath/iconId があればそれを使う
			obj := pickStringPtrFieldDeep(
				v,
				"IconID", "IconId", "iconId",
				"IconObjectPath", "IconPath", "IconStoragePath", "IconGcsPath", "IconGCSPath",
				"Icon", "TokenIcon",
			)

			objStr := strings.TrimSpace(ptrStr(obj))
			if objStr != "" {
				u := strings.TrimSpace(h.ImageResolver.ResolveForResponse(objStr, ""))
				log.Printf("[sns_tokenBlueprint] iconUrl resolve try objectPath=%q -> %q", objStr, u)
				if u != "" {
					patch.IconURL = strPtr(u)
				}
			}

			// 2) まだ空なら固定パス "{id}/icon"
			if strings.TrimSpace(ptrStr(patch.IconURL)) == "" {
				fixed := strings.Trim(strings.TrimSpace(id), "/") + "/icon"
				u := strings.TrimSpace(h.ImageResolver.ResolveForResponse(fixed, ""))
				log.Printf("[sns_tokenBlueprint] iconUrl resolve try fixed objectPath=%q -> %q", fixed, u)
				if u != "" {
					patch.IconURL = strPtr(u)
				}
			}
		}
	}

	log.Printf(
		"[sns_tokenBlueprint] mapped patch iconUrl=%q has=%t",
		ptrStr(patch.IconURL),
		strings.TrimSpace(ptrStr(patch.IconURL)) != "",
	)

	// fill BrandName (optional)
	if patch.BrandName == nil {
		if bn := pickStringPtrField(v, "BrandName", "brandName"); bn != nil && strings.TrimSpace(*bn) != "" {
			patch.BrandName = trimPtr(bn)
		}
	}
	if patch.BrandName == nil && patch.BrandID != nil && strings.TrimSpace(*patch.BrandID) != "" {
		if name, ok := resolveBrandNameBestEffort(ctx, h.BrandNameResolver, strings.TrimSpace(*patch.BrandID)); ok {
			patch.BrandName = strPtr(name)
		}
	}

	// fill CompanyName (optional)
	if patch.CompanyName == nil {
		if cn := pickStringPtrField(v, "CompanyName", "companyName"); cn != nil && strings.TrimSpace(*cn) != "" {
			patch.CompanyName = trimPtr(cn)
		}
	}
	if patch.CompanyName == nil && patch.CompanyID != nil && strings.TrimSpace(*patch.CompanyID) != "" {
		if name, ok := resolveCompanyNameBestEffort(ctx, h.CompanyNameResolver, strings.TrimSpace(*patch.CompanyID)); ok {
			patch.CompanyName = strPtr(name)
		}
	}

	return patch, nil
}

func parseTokenBlueprintPatchPath(path string) (id string, ok bool) {
	p := strings.TrimSuffix(strings.TrimSpace(path), "/")
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

func toPatchBestEffort(tb any) tbdom.Patch {
	name := pickStringPtrField(tb, "Name", "name")
	symbol := pickStringPtrField(tb, "Symbol", "symbol")
	brandID := pickStringPtrField(tb, "BrandID", "BrandId", "brandId")
	companyID := pickStringPtrField(tb, "CompanyID", "CompanyId", "companyId")
	desc := pickStringPtrField(tb, "Description", "description")

	iconURL := pickStringPtrFieldDeep(
		tb,
		"IconURL", "IconUrl", "iconUrl",
		"TokenIconURL", "TokenIconUrl",
		"PublicIconURL", "PublicIconUrl",
		"IconObjectPath", "IconPath", "IconStoragePath", "IconGcsPath", "IconGCSPath",
		"Icon", "TokenIcon",
	)

	minted := pickBoolPtrField(tb, "Minted", "minted")

	return tbdom.Patch{
		Name:        trimPtr(name),
		Symbol:      trimPtr(symbol),
		BrandID:     trimPtr(brandID),
		CompanyID:   trimPtr(companyID),
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
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		if f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s == "" {
				return nil
			}
			x := s
			return &x
		}
	}
	return nil
}

func pickStringPtrFieldDeep(v any, names ...string) *string {
	if s := pickStringPtrField(v, names...); s != nil {
		return s
	}

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
		if f.Kind() != reflect.Struct {
			continue
		}

		inner := f.Interface()
		if s := pickStringPtrField(inner, "URL", "Url", "PublicURL", "PublicUrl", "IconURL", "IconUrl", "iconUrl"); s != nil {
			return s
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
	x := t
	return &x
}

func strPtr(s string) *string {
	x := s
	return &x
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func ptrBoolStr(b *bool) string {
	if b == nil {
		return "(nil)"
	}
	if *b {
		return "true"
	}
	return "false"
}

func resolveBrandNameBestEffort(ctx context.Context, resolver any, brandID string) (string, bool) {
	if resolver == nil {
		return "", false
	}
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", false
	}

	for _, m := range []string{
		"ResolveBrandName",
		"GetBrandNameByID",
		"GetBrandNameById",
		"BrandNameByID",
		"BrandNameById",
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

func resolveCompanyNameBestEffort(ctx context.Context, resolver any, companyID string) (string, bool) {
	if resolver == nil {
		return "", false
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return "", false
	}

	for _, m := range []string{
		"ResolveCompanyName",
		"GetCompanyNameByID",
		"GetCompanyNameById",
		"CompanyNameByID",
		"CompanyNameById",
	} {
		s, ok := callStringWithCtxAndID(resolver, m, ctx, companyID)
		if ok {
			return s, true
		}
	}

	if v, err := callAny(resolver, []string{"GetByID", "GetById"}, ctx, companyID); err == nil && v != nil {
		if s := pickStringPtrField(v, "Name", "name", "CompanyName", "companyName"); s != nil {
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

	defer func() { _ = recover() }()

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

func callAny(repo any, methodNames []string, args ...any) (ret any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("callAny: panic (signature mismatch?)")
			ret = nil
		}
	}()

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
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "errnotfound") {
		return true
	}
	if strings.Contains(msg, "statuscode=404") || strings.Contains(msg, "code=404") {
		return true
	}
	return false
}
