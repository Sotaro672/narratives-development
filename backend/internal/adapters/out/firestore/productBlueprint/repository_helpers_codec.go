// backend/internal/adapters/out/firestore/productBlueprint/repository_helpers_codec.go
// Responsibility: Firestore ドキュメントとドメイン ProductBlueprint の相互変換（docToProductBlueprint / productBlueprintToDoc）を担い、保存形式を一元化する。
package productBlueprint

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

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
			return strings.TrimSpace(v)
		}
		return ""
	}
	getStrPtr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
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
				if s, ok := x.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
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

				mid = strings.TrimSpace(mid)
				if mid == "" || order <= 0 {
					continue
				}
				tmp = append(tmp, pbdom.ModelRef{
					ModelID:      mid,
					DisplayOrder: order,
				})
			}
			if len(tmp) > 0 {
				// 安全のため displayOrder 順に並べ替え
				sort.SliceStable(tmp, func(i, j int) bool {
					return tmp[i].DisplayOrder < tmp[j].DisplayOrder
				})
				modelRefs = tmp
			}
		}
	}

	var deletedAtPtr *time.Time
	if t := getTimeVal("deletedAt"); !t.IsZero() {
		deletedAtPtr = &t
	}

	var expireAtPtr *time.Time
	if t := getTimeVal("expireAt"); !t.IsZero() {
		expireAtPtr = &t
	}

	id := ""
	if v, ok := data["id"].(string); ok && strings.TrimSpace(v) != "" {
		id = strings.TrimSpace(v)
	} else {
		id = doc.Ref.ID
	}

	pb := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: getStr("productName"),
		BrandID:     getStr("brandId"),
		ItemType:    pbdom.ItemType(getStr("itemType")),
		Fit:         getStr("fit"),
		Material:    getStr("material"),
		Weight:      getFloat64(data["weight"]),

		QualityAssurance: dedupTrimStrings(getStringSlice("qualityAssurance")),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(getStr("productIdTagType")),
		},
		CompanyID:  getStr("companyId"),
		AssigneeID: getStr("assigneeId"),

		ModelRefs: modelRefs,

		Printed: printed,

		CreatedBy: getStrPtr("createdBy"),
		CreatedAt: getTimeVal("createdAt"),
		UpdatedBy: getStrPtr("updatedBy"),
		UpdatedAt: getTimeVal("updatedAt"),
		DeletedBy: getStrPtr("deletedBy"),
		DeletedAt: deletedAtPtr,
		ExpireAt:  expireAtPtr,
	}

	return pb, nil
}

func productBlueprintToDoc(v pbdom.ProductBlueprint, createdAt, updatedAt time.Time) (map[string]any, error) {
	m := map[string]any{
		"productName": strings.TrimSpace(v.ProductName),
		"brandId":     strings.TrimSpace(v.BrandID),
		"itemType":    strings.TrimSpace(string(v.ItemType)),
		"fit":         strings.TrimSpace(v.Fit),
		"material":    strings.TrimSpace(v.Material),
		"weight":      v.Weight,
		"assigneeId":  strings.TrimSpace(v.AssigneeID),
		"companyId":   strings.TrimSpace(v.CompanyID),
		"createdAt":   createdAt.UTC(),
		"updatedAt":   updatedAt.UTC(),
		"printed":     v.Printed,
	}

	if len(v.QualityAssurance) > 0 {
		m["qualityAssurance"] = dedupTrimStrings(v.QualityAssurance)
	}

	if v.ProductIdTag.Type != "" {
		m["productIdTagType"] = strings.TrimSpace(string(v.ProductIdTag.Type))
	}

	// modelRefs（nil の場合は「未指定」として保存しない。空スライスで明示したい場合は empty を渡す）
	if v.ModelRefs != nil {
		arr := make([]map[string]any, 0, len(v.ModelRefs))
		for _, mr := range v.ModelRefs {
			mid := strings.TrimSpace(mr.ModelID)
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
		if s := strings.TrimSpace(*v.CreatedBy); s != "" {
			m["createdBy"] = s
		}
	}
	if v.UpdatedBy != nil {
		if s := strings.TrimSpace(*v.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}
	if v.DeletedAt != nil && !v.DeletedAt.IsZero() {
		m["deletedAt"] = v.DeletedAt.UTC()
	}
	if v.DeletedBy != nil {
		if s := strings.TrimSpace(*v.DeletedBy); s != "" {
			m["deletedBy"] = s
		}
	}
	if v.ExpireAt != nil && !v.ExpireAt.IsZero() {
		m["expireAt"] = v.ExpireAt.UTC()
	}

	return m, nil
}
