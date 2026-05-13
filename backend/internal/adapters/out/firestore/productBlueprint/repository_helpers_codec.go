// backend/internal/adapters/out/firestore/productBlueprint/repository_helpers_codec.go
// Responsibility: Firestore ドキュメントとドメイン ProductBlueprint の相互変換（docToProductBlueprint / productBlueprintToDoc）を担い、保存形式を一元化する。
package productBlueprint

import (
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"

	"narratives/internal/domain/common"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ========================
// Helpers: Codec (doc <-> domain)
// ========================

func docToProductBlueprint(doc *firestore.DocumentSnapshot) (pbdom.ProductBlueprint, error) {
	data := doc.Data()
	if data == nil {
		return pbdom.ProductBlueprint{}, fmt.Errorf("empty product_blueprints document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	getStrPtr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := v
			if s != "" {
				return &s
			}
		}
		return nil
	}

	getTimeVal := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		return time.Time{}
	}

	getStringSlice := func(key string) []string {
		raw, ok := data[key]
		if !ok || raw == nil {
			return nil
		}

		switch vv := raw.(type) {
		case []interface{}:
			out := make([]string, 0, len(vv))
			for _, x := range vv {
				if s, ok := x.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			return dedupTrimStrings(out)

		case []string:
			return dedupTrimStrings(vv)

		default:
			return nil
		}
	}

	// printed は bool のみ
	printed := false
	if v, ok := data["printed"].(bool); ok {
		printed = v
	}

	// productBlueprintCategory denormalized snapshot
	category := pbdom.ProductBlueprintCategorySnapshot{
		ID:     getStr("productBlueprintCategoryId"),
		Code:   getStr("productBlueprintCategoryCode"),
		NameJa: getStr("productBlueprintCategoryNameJa"),
		NameEn: getStr("productBlueprintCategoryNameEn"),
		Kind:   common.ProductCategoryKind(getStr("productBlueprintCategoryKind")),
		Path:   getStringSlice("productBlueprintCategoryPath"),
	}

	// categoryFields
	categoryFields := docValueToCategoryFields(data["categoryFields"])

	// modelRefs
	var modelRefs []pbdom.ModelRef
	if raw, ok := data["modelRefs"]; ok && raw != nil {
		switch xs := raw.(type) {
		case []interface{}:
			tmp := make([]pbdom.ModelRef, 0, len(xs))
			for _, it := range xs {
				m, ok := it.(map[string]interface{})
				if !ok || m == nil {
					continue
				}

				mid, _ := m["modelId"].(string)

				order := 0
				switch v := m["displayOrder"].(type) {
				case int:
					order = v
				case int32:
					order = int(v)
				case int64:
					order = int(v)
				case float64:
					order = int(v)
				}

				if mid == "" || order <= 0 {
					continue
				}

				tmp = append(tmp, pbdom.ModelRef{
					ModelID:      mid,
					DisplayOrder: order,
				})
			}

			if len(tmp) > 0 {
				sort.SliceStable(tmp, func(i, j int) bool {
					return tmp[i].DisplayOrder < tmp[j].DisplayOrder
				})
				modelRefs = tmp
			}
		}
	}

	id := ""
	if v, ok := data["id"].(string); ok && v != "" {
		id = v
	} else {
		id = doc.Ref.ID
	}

	pb := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: getStr("productName"),
		Description: getStr("description"),

		BrandID:   getStr("brandId"),
		CompanyID: getStr("companyId"),

		ProductBlueprintCategory: category,
		CategoryFields:           categoryFields,

		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(getStr("productIdTagType")),
		},

		AssigneeID: getStr("assigneeId"),

		ModelRefs: modelRefs,

		Printed: printed,

		CreatedBy: getStrPtr("createdBy"),
		CreatedAt: getTimeVal("createdAt"),
		UpdatedBy: getStrPtr("updatedBy"),
		UpdatedAt: getTimeVal("updatedAt"),
	}

	return pb, nil
}

func productBlueprintToDoc(v pbdom.ProductBlueprint, createdAt, updatedAt time.Time) (map[string]any, error) {
	category := v.ProductBlueprintCategory

	m := map[string]any{
		"productName": v.ProductName,
		"description": v.Description,

		"brandId":   v.BrandID,
		"companyId": v.CompanyID,

		"productBlueprintCategoryId":     category.ID,
		"productBlueprintCategoryCode":   category.Code,
		"productBlueprintCategoryNameJa": category.NameJa,
		"productBlueprintCategoryNameEn": category.NameEn,
		"productBlueprintCategoryKind":   string(category.Kind),
		"productBlueprintCategoryPath":   append([]string(nil), category.Path...),

		"assigneeId": v.AssigneeID,

		"createdAt": createdAt.UTC(),
		"updatedAt": updatedAt.UTC(),
		"printed":   v.Printed,
	}

	if len(v.CategoryFields) > 0 {
		m["categoryFields"] = categoryFieldsToDoc(v.CategoryFields)
	}

	if v.ProductIdTag.Type != "" {
		m["productIdTagType"] = string(v.ProductIdTag.Type)
	}

	// modelRefs（nil の場合は「未指定」として保存しない。空スライスで明示したい場合は empty を渡す）
	if v.ModelRefs != nil {
		arr := make([]map[string]any, 0, len(v.ModelRefs))
		for _, mr := range v.ModelRefs {
			mid := mr.ModelID
			if mid == "" || mr.DisplayOrder <= 0 {
				continue
			}

			arr = append(arr, map[string]any{
				"modelId":      mid,
				"displayOrder": mr.DisplayOrder,
			})
		}
		m["modelRefs"] = arr
	}

	if v.CreatedBy != nil {
		if s := *v.CreatedBy; s != "" {
			m["createdBy"] = s
		}
	}

	if v.UpdatedBy != nil {
		if s := *v.UpdatedBy; s != "" {
			m["updatedBy"] = s
		}
	}

	return m, nil
}

func docValueToCategoryFields(raw any) pbdom.CategoryFields {
	if raw == nil {
		return nil
	}

	out := pbdom.CategoryFields{}

	switch m := raw.(type) {
	case map[string]any:
		for key, value := range m {
			if key == "" {
				continue
			}
			out[key] = value
		}

	default:
		return nil
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func categoryFieldsToDoc(in pbdom.CategoryFields) map[string]any {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
