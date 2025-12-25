// backend/internal/application/query/sns/catalog_query.go
package sns

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs (for catalog.dart)
// ============================================================

type SNSCatalogDTO struct {
	List SNSCatalogListDTO `json:"list"`

	Inventory      *SNSCatalogInventoryDTO `json:"inventory,omitempty"`
	InventoryError string                  `json:"inventoryError,omitempty"`

	ProductBlueprint      *SNSCatalogProductBlueprintDTO `json:"productBlueprint,omitempty"`
	ProductBlueprintError string                         `json:"productBlueprintError,omitempty"`

	ModelVariations      []SNSCatalogModelVariationDTO `json:"modelVariations,omitempty"`
	ModelVariationsError string                        `json:"modelVariationsError,omitempty"`
}

type SNSCatalogListDTO struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // URL
	Prices      []ldom.ListPriceRow `json:"prices"`

	// linkage (catalog.dart uses these)
	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
}

type SNSCatalogInventoryDTO struct {
	ID                 string                        `json:"id"`
	ProductBlueprintID string                        `json:"productBlueprintId"`
	TokenBlueprintID   string                        `json:"tokenBlueprintId"`
	Stock              map[string]SNSCatalogStockDTO `json:"stock"` // key=modelId
}

type SNSCatalogStockDTO struct {
	Accumulation int `json:"accumulation"`
}

type SNSCatalogProductBlueprintDTO struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`

	ItemType string `json:"itemType"`
	Fit      string `json:"fit"`
	Material string `json:"material"`

	// domain 側が float64 のため、0 の場合は omitempty で落ちる
	Weight float64 `json:"weight,omitempty"`

	Printed bool `json:"printed"`

	QualityAssurance []string `json:"qualityAssurance"`
	ProductIDTagType string   `json:"productIdTagType"`
}

type SNSCatalogModelVariationDTO struct {
	ID                 string             `json:"id"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	ModelNumber        string             `json:"modelNumber"`
	Size               string             `json:"size"`
	Color              SNSCatalogColorDTO `json:"color"`
	Measurements       map[string]int     `json:"measurements,omitempty"`
}

type SNSCatalogColorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"`
}

// ============================================================
// Ports (minimal contracts for this query)
// ============================================================

// InventoryRepository returns already-shaped buyer-facing inventory DTO.
// （domain の型名揺れを避けるため、ここでは DTO で受ける）
type InventoryRepository interface {
	GetByID(ctx context.Context, id string) (*SNSCatalogInventoryDTO, error)
	GetByProductAndTokenBlueprintID(ctx context.Context, productBlueprintID, tokenBlueprintID string) (*SNSCatalogInventoryDTO, error)
}

type ProductBlueprintRepository interface {
	GetByID(ctx context.Context, id string) (*pbdom.ProductBlueprint, error)
}

// ============================================================
// Query
// ============================================================

type SNSCatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository

	ModelRepo modeldom.RepositoryPort
}

func NewSNSCatalogQuery(
	listRepo ldom.Repository,
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
) *SNSCatalogQuery {
	return &SNSCatalogQuery{
		ListRepo:      listRepo,
		InventoryRepo: invRepo,
		ProductRepo:   productRepo,
		ModelRepo:     modelRepo,
	}
}

// GetByListID builds catalog payload by listId.
// - list must be status=listing, otherwise ErrNotFound.
// - inventory/product/model are best-effort; failures populate "*Error" fields.
func (q *SNSCatalogQuery) GetByListID(ctx context.Context, listID string) (SNSCatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return SNSCatalogDTO{}, errors.New("sns catalog query: list repo is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return SNSCatalogDTO{}, ldom.ErrNotFound
	}

	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		return SNSCatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		return SNSCatalogDTO{}, ldom.ErrNotFound
	}

	out := SNSCatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// Inventory (prefer inventoryId; fallback to pb/tb query)
	// ------------------------------------------------------------
	var invDTO *SNSCatalogInventoryDTO

	if q.InventoryRepo == nil {
		out.InventoryError = "inventory repo is nil"
	} else {
		invID := strings.TrimSpace(out.List.InventoryID)
		pbID := strings.TrimSpace(out.List.ProductBlueprintID)
		tbID := strings.TrimSpace(out.List.TokenBlueprintID)

		switch {
		case invID != "":
			v, e := q.InventoryRepo.GetByID(ctx, invID)
			if e != nil {
				out.InventoryError = e.Error()
			} else {
				invDTO = v
				out.Inventory = v
			}

		case pbID != "" && tbID != "":
			v, e := q.InventoryRepo.GetByProductAndTokenBlueprintID(ctx, pbID, tbID)
			if e != nil {
				out.InventoryError = e.Error()
			} else {
				invDTO = v
				out.Inventory = v
			}

		default:
			out.InventoryError = "inventory linkage is missing (inventoryId or productBlueprintId+tokenBlueprintId)"
		}
	}

	// ------------------------------------------------------------
	// ProductBlueprint (inventory side wins)
	// ------------------------------------------------------------
	resolvedPBID := strings.TrimSpace(out.List.ProductBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.ProductBlueprintID); s != "" {
			resolvedPBID = s
		}
	}

	if q.ProductRepo == nil {
		out.ProductBlueprintError = "product repo is nil"
	} else if resolvedPBID == "" {
		out.ProductBlueprintError = "productBlueprintId is empty"
	} else {
		pb, e := q.ProductRepo.GetByID(ctx, resolvedPBID)
		if e != nil {
			out.ProductBlueprintError = e.Error()
		} else if pb != nil {
			dto := toCatalogProductBlueprintDTO(pb)
			out.ProductBlueprint = &dto
		}
	}

	// ------------------------------------------------------------
	// Models (by productBlueprintId)
	// ------------------------------------------------------------
	if q.ModelRepo == nil {
		out.ModelVariationsError = "model repo is nil"
	} else if resolvedPBID == "" {
		out.ModelVariationsError = "productBlueprintId is empty (skip model fetch)"
	} else {
		md, e := q.ModelRepo.GetModelDataByBlueprintID(ctx, resolvedPBID)
		if e != nil {
			out.ModelVariationsError = e.Error()
		} else if md != nil {
			out.ModelVariations = toCatalogModelVariationsDTO(md.Variations)
		}
	}

	return out, nil
}

// ============================================================
// mappers
// ============================================================

func toCatalogListDTO(l ldom.List) SNSCatalogListDTO {
	return SNSCatalogListDTO{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,

		InventoryID: strings.TrimSpace(l.InventoryID),

		// ✅ list 側のフィールド名が ProductBlueprintID / ProductBlueprintId など揺れても拾う
		ProductBlueprintID: pickStringField(l, "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId"),
		TokenBlueprintID:   pickStringField(l, "TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId"),
	}
}

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) SNSCatalogProductBlueprintDTO {
	out := SNSCatalogProductBlueprintDTO{
		ID:          strings.TrimSpace(pb.ID),
		ProductName: strings.TrimSpace(pb.ProductName),
		BrandID:     strings.TrimSpace(pb.BrandID),
		CompanyID:   strings.TrimSpace(pb.CompanyID),

		// named type でも安全に string 化する
		ItemType: fmt.Sprint(pb.ItemType),
		Fit:      fmt.Sprint(pb.Fit),
		Material: fmt.Sprint(pb.Material),

		Weight:  pb.Weight,
		Printed: pb.Printed,

		QualityAssurance: append([]string{}, pb.QualityAssurance...),

		// ✅ フィールド名揺れ / ネスト（ProductIDTag.Type 等）を吸収
		ProductIDTagType: pickProductIDTagType(pb),
	}
	return out
}

func toCatalogModelVariationsDTO(vars []modeldom.ModelVariation) []SNSCatalogModelVariationDTO {
	out := make([]SNSCatalogModelVariationDTO, 0, len(vars))
	for _, v := range vars {
		dto := SNSCatalogModelVariationDTO{
			ID:                 strings.TrimSpace(v.ID),
			ProductBlueprintID: strings.TrimSpace(v.ProductBlueprintID),
			ModelNumber:        strings.TrimSpace(v.ModelNumber),
			Size:               strings.TrimSpace(v.Size),
			Color: SNSCatalogColorDTO{
				Name: strings.TrimSpace(v.Color.Name),
				RGB:  v.Color.RGB,
			},
		}

		// ✅ S1009: len(nilMap) は 0 なので nil チェック不要
		if len(v.Measurements) > 0 {
			m := make(map[string]int, len(v.Measurements))
			for k, val := range v.Measurements {
				m[strings.TrimSpace(k)] = val
			}
			dto.Measurements = m
		}

		out = append(out, dto)
	}
	return out
}

// ============================================================
// reflection helpers (field-name tolerant)
// ============================================================

func pickStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
	}
	return ""
}

func pickProductIDTagType(pb *pbdom.ProductBlueprint) string {
	if pb == nil {
		return ""
	}

	// 1) 直下フィールド
	if s := pickStringField(*pb, "ProductIDTagType", "ProductIdTagType", "productIdTagType"); s != "" {
		return s
	}

	// 2) ネスト: ProductIDTag / ProductIdTag の中の Type
	rv := reflect.ValueOf(pb)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, parent := range []string{"ProductIDTag", "ProductIdTag", "productIdTag"} {
		f := rv.FieldByName(parent)
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

		tf := f.FieldByName("Type")
		if tf.IsValid() && tf.Kind() == reflect.String {
			return strings.TrimSpace(tf.String())
		}
	}

	return ""
}
