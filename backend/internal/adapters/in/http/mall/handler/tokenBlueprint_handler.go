// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ImageURLResolver is a minimal contract for resolving iconUrl from stored objectPath.
type ImageURLResolver interface {
	ResolveForResponse(storedObjectPath string, storedIconURL string) string
}

type MallTokenBlueprintHandler struct {
	Repo any

	BrandNameResolver   any
	CompanyNameResolver any // NOTE: tbdom.Patch には CompanyName が無いので実質未使用

	// icon url resolver (objectPath -> public URL)
	ImageResolver ImageURLResolver
}

func NewMallTokenBlueprintHandler(repo any) http.Handler {
	return &MallTokenBlueprintHandler{Repo: repo}
}

func NewMallTokenBlueprintHandlerWithNameResolver(repo any, nameResolver any) http.Handler {
	return &MallTokenBlueprintHandler{
		Repo:                repo,
		BrandNameResolver:   nameResolver,
		CompanyNameResolver: nameResolver,
	}
}

// 推奨: NameResolver + ImageURLResolver を注入
func NewMallTokenBlueprintHandlerWithNameAndImageResolver(
	repo any,
	nameResolver any,
	imageResolver ImageURLResolver,
) http.Handler {
	return &MallTokenBlueprintHandler{
		Repo:                repo,
		BrandNameResolver:   nameResolver,
		CompanyNameResolver: nameResolver,
		ImageResolver:       imageResolver,
	}
}

func (h *MallTokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "mall tokenBlueprint handler is nil")
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, ok := parseTokenBlueprintPatchPath(r.URL.Path)
	if !ok {
		notFound(w)
		return
	}

	patch, err := h.getPatchByID(r.Context(), id)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "failed to get tokenBlueprint patch"
		}

		if isNotFoundError(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": msg})
			return
		}

		internalError(w, msg)
		return
	}

	writeJSON(w, http.StatusOK, patch)
}

func (h *MallTokenBlueprintHandler) getPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return tbdom.Patch{}, errors.New("tokenBlueprint id is empty")
	}
	if h.Repo == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint repo is nil")
	}

	v, err := callAny(h.Repo, []string{"GetByID", "GetById"}, ctx, id)
	if err != nil {
		return tbdom.Patch{}, err
	}
	if v == nil {
		return tbdom.Patch{}, errors.New("tokenBlueprint is nil")
	}

	patch := toPatchBestEffort(v, id)

	// ============================================================
	// IconURL 補完（Patch は string なので "" 判定）
	// ============================================================
	if strings.TrimSpace(patch.IconURL) == "" && h.ImageResolver != nil {
		// 1) objectPath/iconId があればそれを使う
		obj := pickStringPtrFieldDeep(
			v,
			"IconID", "IconId", "iconId",
			"IconObjectPath", "IconPath", "IconStoragePath", "IconGcsPath", "IconGCSPath",
			"Icon", "TokenIcon",
		)

		objStr := strings.TrimSpace(derefStr(obj))
		if objStr != "" {
			u := strings.TrimSpace(h.ImageResolver.ResolveForResponse(objStr, ""))
			if u != "" {
				patch.IconURL = u
			}
		}

		// 2) まだ空なら固定パス "{id}/icon"
		if strings.TrimSpace(patch.IconURL) == "" {
			fixed := strings.Trim(strings.TrimSpace(id), "/") + "/icon"
			u := strings.TrimSpace(h.ImageResolver.ResolveForResponse(fixed, ""))
			if u != "" {
				patch.IconURL = u
			}
		}
	}

	// ============================================================
	// companyId / brandName の補完（Patch は値型）
	// ============================================================

	// 1) companyId が空なら、brandId から companyId を補完する（resolver 経由）
	if strings.TrimSpace(patch.CompanyID) == "" && strings.TrimSpace(patch.BrandID) != "" {
		brandID := strings.TrimSpace(patch.BrandID)

		res := h.BrandNameResolver
		if res == nil {
			res = h.CompanyNameResolver
		}

		if cid, ok := resolveCompanyIDFromBrandBestEffort(ctx, res, brandID); ok {
			patch.CompanyID = strings.TrimSpace(cid)
		}
	}

	// 2) fill BrandName (optional)
	if strings.TrimSpace(patch.BrandName) == "" && strings.TrimSpace(patch.BrandID) != "" {
		if name, ok := resolveBrandNameBestEffort(ctx, h.BrandNameResolver, strings.TrimSpace(patch.BrandID)); ok {
			patch.BrandName = strings.TrimSpace(name)
		}
	}

	return patch, nil
}

func parseTokenBlueprintPatchPath(path string) (id string, ok bool) {
	p := strings.TrimSuffix(strings.TrimSpace(path), "/")
	const prefix = "/mall/token-blueprints/"
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

// toPatchBestEffort converts arbitrary TokenBlueprint-like struct into tbdom.Patch (value types).
// entity.go（TokenBlueprint）を正として、Patch のフィールド名・型に合わせて組み立てる。
func toPatchBestEffort(tb any, id string) tbdom.Patch {
	// TokenBlueprint 側のフィールド候補
	name := pickStringPtrField(tb, "Name", "name")
	symbol := pickStringPtrField(tb, "Symbol", "symbol")
	brandID := pickStringPtrField(tb, "BrandID", "BrandId", "brandId")
	companyID := pickStringPtrField(tb, "CompanyID", "CompanyId", "companyId")
	desc := pickStringPtrField(tb, "Description", "description")

	// iconUrl は “URL” が埋まっているケースもあるが、基本は objectPath(iconId) から resolver で生成する想定。
	// ここでは既存データ/構造体に iconUrl があれば拾う。
	iconURL := pickStringPtrFieldDeep(
		tb,
		"IconURL", "IconUrl", "iconUrl",
		"TokenIconURL", "TokenIconUrl",
		"PublicIconURL", "PublicIconUrl",
		"IconObjectPath", "IconPath", "IconStoragePath", "IconGcsPath", "IconGCSPath",
		"Icon", "TokenIcon",
	)

	metadataURI := pickStringPtrField(tb, "MetadataURI", "MetadataUri", "metadataUri")

	minted := pickBoolPtrField(tb, "Minted", "minted")

	// BrandName を直接持っている構造体もあるので拾う（なければ後段で resolver）
	brandName := pickStringPtrField(tb, "BrandName", "brandName")

	return tbdom.Patch{
		ID:          strings.TrimSpace(id),
		TokenName:   strings.TrimSpace(derefStr(name)),
		Symbol:      strings.TrimSpace(derefStr(symbol)),
		BrandID:     strings.TrimSpace(derefStr(brandID)),
		BrandName:   strings.TrimSpace(derefStr(brandName)),
		CompanyID:   strings.TrimSpace(derefStr(companyID)),
		Description: strings.TrimSpace(derefStr(desc)),
		Minted:      derefBool(minted),
		MetadataURI: strings.TrimSpace(derefStr(metadataURI)),
		IconURL:     strings.TrimSpace(derefStr(iconURL)),
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
		// *string
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			if f.Elem().Kind() == reflect.String {
				s := strings.TrimSpace(f.Elem().String())
				if s == "" {
					continue
				}
				x := s
				return &x
			}
			continue
		}
		// string
		if f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s == "" {
				continue
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
		// *bool
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			if f.Elem().Kind() == reflect.Bool {
				b := f.Elem().Bool()
				return &b
			}
			continue
		}
		// bool
		if f.Kind() == reflect.Bool {
			b := f.Bool()
			return &b
		}
	}
	return nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func derefBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
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

// brandId から companyId を解決する（resolver に ResolveBrandCompanyID 等がある前提）
func resolveCompanyIDFromBrandBestEffort(ctx context.Context, resolver any, brandID string) (string, bool) {
	if resolver == nil {
		return "", false
	}
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", false
	}

	for _, m := range []string{
		"ResolveBrandCompanyID",
		"ResolveBrandCompanyId",
		"ResolveCompanyIDByBrandID",
		"ResolveCompanyIdByBrandId",
		"CompanyIDByBrandID",
		"CompanyIdByBrandId",
	} {
		s, ok := callStringWithCtxAndID(resolver, m, ctx, brandID)
		if ok {
			return s, true
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
