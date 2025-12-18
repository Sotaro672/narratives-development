// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"
	"log"
	"reflect"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ------------------------------------------------------------
// Repository interfaces (最小限の読み取り専用ポート)
// ------------------------------------------------------------

// Brand 名の取得に必要な最小限のインターフェース
type BrandNameRepository interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// Company 名の取得に必要な最小限のインターフェース
type CompanyNameRepository interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

// ProductBlueprint → productName だけ取得できればよい
type ProductBlueprintNameRepository interface {
	GetProductNameByID(ctx context.Context, id string) (string, error)
}

// Member → 氏名取得だけできればよい
type MemberNameRepository interface {
	GetByID(ctx context.Context, id string) (memberdom.Member, error)
}

// Model → modelId から ModelVariation を 1 件取得できればよい
// ★ ModelNameRepository → ModelNumberRepository にリネーム
type ModelNumberRepository interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// TokenBlueprint → name / symbol を取得できればよい
type TokenBlueprintNameRepository interface {
	GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error)
}

// ------------------------------------------------------------
// NameResolver 本体
// ------------------------------------------------------------

type NameResolver struct {
	brandRepo            BrandNameRepository
	companyRepo          CompanyNameRepository
	productBlueprintRepo ProductBlueprintNameRepository
	memberRepo           MemberNameRepository
	modelNumberRepo      ModelNumberRepository
	tokenBlueprintRepo   TokenBlueprintNameRepository
}

// NewNameResolver は各種 Name/Number 用リポジトリをまとめて受け取り、
// 画面向けの「名前解決ヘルパ」を生成する。
func NewNameResolver(
	brandRepo BrandNameRepository,
	companyRepo CompanyNameRepository,
	productBlueprintRepo ProductBlueprintNameRepository,
	memberRepo MemberNameRepository,
	modelNumberRepo ModelNumberRepository,
	tokenBlueprintRepo TokenBlueprintNameRepository,
) *NameResolver {
	return &NameResolver{
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		memberRepo:           memberRepo,
		modelNumberRepo:      modelNumberRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
	}
}

// ------------------------------------------------------------
// Brand 関連
// ------------------------------------------------------------

// ResolveBrandName は brandId からブランド名（Name）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveBrandName(ctx context.Context, brandID string) string {
	if r == nil || r.brandRepo == nil {
		return ""
	}
	id := strings.TrimSpace(brandID)
	if id == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(b.Name)
}

// ------------------------------------------------------------
// Company 関連
// ------------------------------------------------------------

// ResolveCompanyName は companyId から会社名（Name）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveCompanyName(ctx context.Context, companyID string) string {
	if r == nil || r.companyRepo == nil {
		return ""
	}
	id := strings.TrimSpace(companyID)
	if id == "" {
		return ""
	}

	c, err := r.companyRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(c.Name)
}

// ------------------------------------------------------------
// ProductBlueprint 関連
// ------------------------------------------------------------

// ResolveProductName は productBlueprintId から productName を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveProductName(ctx context.Context, productBlueprintID string) string {
	if r == nil || r.productBlueprintRepo == nil {
		return ""
	}
	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		return ""
	}

	name, err := r.productBlueprintRepo.GetProductNameByID(ctx, id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

// ------------------------------------------------------------
// Member 関連
// ------------------------------------------------------------

// ResolveMemberName は memberId から表示用の名前（例: "姓 名"）を解決する。
// Member ドメインの定義に合わせて LastName / FirstName を利用する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveMemberName(ctx context.Context, memberID string) string {
	if r == nil || r.memberRepo == nil {
		return ""
	}
	id := strings.TrimSpace(memberID)
	if id == "" {
		return ""
	}

	m, err := r.memberRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	family := strings.TrimSpace(m.LastName) // 姓
	given := strings.TrimSpace(m.FirstName) // 名

	switch {
	case family == "" && given == "":
		return ""
	case family == "":
		return given
	case given == "":
		return family
	default:
		return family + " " + given
	}
}

// ---- memberId 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameFromPtr(ctx context.Context, memberID *string) string {
	if memberID == nil {
		return ""
	}
	return r.ResolveMemberName(ctx, *memberID)
}

func (r *NameResolver) ResolveAssigneeName(ctx context.Context, assigneeID string) string {
	return r.ResolveMemberName(ctx, assigneeID)
}

func (r *NameResolver) ResolveCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, updatedBy)
}

func (r *NameResolver) ResolveRequestedByName(ctx context.Context, requestedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, requestedBy)
}

func (r *NameResolver) ResolveInspectedByName(ctx context.Context, inspectedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, inspectedBy)
}

func (r *NameResolver) ResolvePrintedByName(ctx context.Context, printedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, printedBy)
}

// ------------------------------------------------------------
// ModelVariation (modelId → modelNumber) 関連
// ------------------------------------------------------------

// ResolveModelNumber は modelVariationId から modelNumber を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveModelNumber(ctx context.Context, variationID string) string {
	if r == nil || r.modelNumberRepo == nil {
		return ""
	}
	id := strings.TrimSpace(variationID)
	if id == "" {
		return ""
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil || mv == nil {
		return ""
	}

	return strings.TrimSpace(mv.ModelNumber)
}

// ------------------------------------------------------------
// ✅ ModelVariation (modelId → modelNumber/size/colorLabel/rgb)
// - Firestore の保存形式を正とする（名揺れ吸収はしない）
//   - modelNumber: mv.ModelNumber
//   - size:        mv.Size
//   - colorLabel:  mv.Color.label（無ければ mv.Color.name を読む）
//   - rgb:         mv.Color.rgb
// ------------------------------------------------------------

type ModelResolved struct {
	ModelNumber string
	Size        string
	Color       string
	RGB         *int
}

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
		log.Printf("[name_resolver][ResolveModelResolved] GetModelVariationByID failed modelId=%q err=%v mvNil=%v",
			id, err, mv == nil,
		)
		return ModelResolved{}
	}

	modelNumber := strings.TrimSpace(mv.ModelNumber)
	size := strings.TrimSpace(mv.Size)

	// Firestore: color(map){ label or name, rgb }
	colorLabel, rgb, dbg := extractColorLabelAndRGBFromModelVariation(mv)

	log.Printf("[name_resolver][ResolveModelResolved] fromModelVariation modelId=%q modelNumber=%q size=%q color=%q rgb=%v rgbType=%T dbg=%s",
		id, modelNumber, size, colorLabel, rgb, rgb, dbg,
	)

	return ModelResolved{
		ModelNumber: modelNumber,
		Size:        size,
		Color:       strings.TrimSpace(colorLabel),
		RGB:         rgb,
	}
}

// Firestore の保存形式を正として読む（名揺れ吸収しない）
func extractColorLabelAndRGBFromModelVariation(mv *modeldom.ModelVariation) (string, *int, string) {
	if mv == nil {
		return "", nil, "mv=nil"
	}

	// mv.Color を reflect で読む（Color の型が struct/map どちらでも対応）
	rv := reflect.ValueOf(mv)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return "", nil, "mv!=struct"
	}

	f := rv.FieldByName("Color")
	if !f.IsValid() {
		return "", nil, "field Color not found"
	}
	f = deref(f)
	if !f.IsValid() {
		return "", nil, "Color invalid"
	}

	switch f.Kind() {
	case reflect.Map:
		// color(map)
		label := mapString(f, "label")
		if strings.TrimSpace(label) == "" {
			// 互換：既存データが name の場合（同一フィールド内のキー差分のみ）
			label = mapString(f, "name")
		}
		rgb := mapIntPtr(f, "rgb")
		return strings.TrimSpace(label), rgb, "Color.kind=map"

	case reflect.Struct:
		// color(struct)
		// label が正。無い場合は name（同一 struct 内のフィールド差分のみ）
		label := structString(f, "Label")
		if strings.TrimSpace(label) == "" {
			label = structString(f, "Name")
		}
		rgb := structIntPtr(f, "RGB")
		if rgb == nil {
			// struct 側が rgb の場合
			rgb = structIntPtr(f, "Rgb")
		}
		return strings.TrimSpace(label), rgb, "Color.kind=struct"

	default:
		return "", nil, "Color unsupported kind=" + f.Kind().String()
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

// ------------------------------------------------------------
// TokenBlueprint 関連
// ------------------------------------------------------------

// ResolveTokenName は tokenBlueprintId からトークン名（例: name or symbol）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if r == nil || r.tokenBlueprintRepo == nil {
		return ""
	}
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return ""
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	name := strings.TrimSpace(tb.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(tb.Symbol)
}
