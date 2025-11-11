// backend/internal/adapters/out/firestore/productBlueprint_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	dbcommon "narratives/internal/adapters/out/db/common"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepositoryPG implements usecase.ProductBlueprintRepo using PostgreSQL.
type ProductBlueprintRepositoryPG struct {
	DB *sql.DB
}

func NewProductBlueprintRepositoryPG(db *sql.DB) *ProductBlueprintRepositoryPG {
	return &ProductBlueprintRepositoryPG{DB: db}
}

// ========================
// usecase.ProductBlueprintRepo impl
// ========================

// GetByID returns a single ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryPG) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at,
  updated_by, updated_at
FROM product_blueprints
WHERE id = $1
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	pb, err := scanProductBlueprint(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}
	return pb, nil
}

// Exists checks if a ProductBlueprint with the given ID exists.
func (r *ProductBlueprintRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `SELECT 1 FROM product_blueprints WHERE id = $1 LIMIT 1`
	var dummy int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create inserts a new ProductBlueprint row.
func (r *ProductBlueprintRepositoryPG) Create(ctx context.Context, v pbdom.ProductBlueprint) (pbdom.ProductBlueprint, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// Serialize variations to JSON for DB.
	varsJSON, err := json.Marshal(v.Variations)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	tagType := strings.TrimSpace(string(v.ProductIdTag.Type))

	// Ensure timestamps
	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	// UpdatedBy が未指定の場合は CreatedBy を引き継ぐ（あれば）
	createdByPtr := v.CreatedBy
	updatedByPtr := v.UpdatedBy
	if updatedByPtr == nil && createdByPtr != nil {
		updatedByPtr = createdByPtr
	}

	createdByNS := dbcommon.ToDBText(createdByPtr)
	updatedByNS := dbcommon.ToDBText(updatedByPtr)

	const q = `
INSERT INTO product_blueprints (
  id,
  product_name,
  brand_id,
  item_type,
  model_variations,
  fit,
  material,
  weight,
  quality_assurance,
  product_id_tag_type,
  assignee_id,
  created_by,
  created_at,
  updated_by,
  updated_at
) VALUES (
  $1,              -- id
  $2,              -- product_name
  $3,              -- brand_id
  $4,              -- item_type
  $5::jsonb,       -- model_variations
  $6,              -- fit
  $7,              -- material
  $8,              -- weight
  $9,              -- quality_assurance
  $10,             -- product_id_tag_type
  $11,             -- assignee_id
  $12,             -- created_by
  $13,             -- created_at
  $14,             -- updated_by
  $15              -- updated_at
)
RETURNING
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at,
  updated_by, updated_at
`

	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.ProductName),
		strings.TrimSpace(v.BrandID),
		strings.TrimSpace(string(v.ItemType)),
		string(varsJSON),
		strings.TrimSpace(v.Fit),
		strings.TrimSpace(v.Material),
		v.Weight,
		pq.Array(dedupTrimStrings(v.QualityAssurance)),
		tagType,
		strings.TrimSpace(v.AssigneeID),
		createdByNS,
		createdAt,
		updatedByNS,
		updatedAt,
	)

	out, err := scanProductBlueprint(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}
	return out, nil
}

// Save updates an existing ProductBlueprint row.
func (r *ProductBlueprintRepositoryPG) Save(ctx context.Context, v pbdom.ProductBlueprint) (pbdom.ProductBlueprint, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	varsJSON, err := json.Marshal(v.Variations)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	tagType := strings.TrimSpace(string(v.ProductIdTag.Type))

	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	updatedByNS := dbcommon.ToDBText(v.UpdatedBy)

	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	const q = `
UPDATE product_blueprints
SET
  product_name         = $2,
  brand_id             = $3,
  item_type            = $4,
  model_variations     = $5::jsonb,
  fit                  = $6,
  material             = $7,
  weight               = $8,
  quality_assurance    = $9,
  product_id_tag_type  = $10,
  assignee_id          = $11,
  created_by           = $12,
  created_at           = $13,
  updated_by           = $14,
  updated_at           = $15
WHERE id = $1
RETURNING
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at,
  updated_by, updated_at
`

	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.ProductName),
		strings.TrimSpace(v.BrandID),
		strings.TrimSpace(string(v.ItemType)),
		string(varsJSON),
		strings.TrimSpace(v.Fit),
		strings.TrimSpace(v.Material),
		v.Weight,
		pq.Array(dedupTrimStrings(v.QualityAssurance)),
		tagType,
		strings.TrimSpace(v.AssigneeID),
		dbcommon.ToDBText(v.CreatedBy),
		createdAt.UTC(),
		updatedByNS,
		updatedAt,
	)

	out, err := scanProductBlueprint(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}
	return out, nil
}

// Delete removes a ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx, `DELETE FROM product_blueprints WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return pbdom.ErrNotFound
	}
	return nil
}

// ------------------------
// Extra helper methods
// ------------------------

func (r *ProductBlueprintRepositoryPG) List(ctx context.Context, filter pbdom.Filter, sort pbdom.Sort, page pbdom.Page) (pbdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildPBWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	orderBy := buildPBOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_blueprints "+whereSQL, args...).Scan(&total); err != nil {
		return pbdom.PageResult{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at,
  updated_by, updated_at
FROM product_blueprints
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return pbdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]pbdom.ProductBlueprint, 0, perPage)
	for rows.Next() {
		pb, err := scanProductBlueprint(rows)
		if err != nil {
			return pbdom.PageResult{}, err
		}
		items = append(items, pb)
	}
	if err := rows.Err(); err != nil {
		return pbdom.PageResult{}, err
	}

	return pbdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ProductBlueprintRepositoryPG) Count(ctx context.Context, filter pbdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildPBWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_blueprints "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Reset is mainly for tests.
func (r *ProductBlueprintRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM product_blueprints`)
	return err
}

// ========================
// Helpers
// ========================

func scanProductBlueprint(s dbcommon.RowScanner) (pbdom.ProductBlueprint, error) {
	var (
		id, productName, brandID, itemType, fit, material string
		varsRaw                                           []byte
		weight                                            float64
		qa                                                []string
		tagType                                           string
		assigneeID                                        string
		createdByNS, updatedByNS                          sql.NullString
		createdAt, updatedAt                              time.Time
	)

	if err := s.Scan(
		&id, &productName, &brandID, &itemType,
		&varsRaw, &fit, &material, &weight, pq.Array(&qa),
		&tagType, &assigneeID,
		&createdByNS, &createdAt,
		&updatedByNS, &updatedAt,
	); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	var createdByPtr *string
	if createdByNS.Valid {
		v := strings.TrimSpace(createdByNS.String)
		if v != "" {
			createdByPtr = &v
		}
	}

	var updatedByPtr *string
	if updatedByNS.Valid {
		v := strings.TrimSpace(updatedByNS.String)
		if v != "" {
			updatedByPtr = &v
		}
	}

	return pbdom.ProductBlueprint{
		ID:               strings.TrimSpace(id),
		ProductName:      strings.TrimSpace(productName),
		BrandID:          strings.TrimSpace(brandID),
		ItemType:         pbdom.ItemType(strings.TrimSpace(itemType)),
		Variations:       nil, // 必要なら varsRaw を Unmarshal
		Fit:              strings.TrimSpace(fit),
		Material:         strings.TrimSpace(material),
		Weight:           weight,
		QualityAssurance: dedupTrimStrings(qa),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(strings.TrimSpace(tagType)),
		},
		AssigneeID: strings.TrimSpace(assigneeID),
		CreatedBy:  createdByPtr,
		CreatedAt:  createdAt.UTC(),
		UpdatedBy:  updatedByPtr,
		UpdatedAt:  updatedAt.UTC(),
	}, nil
}

func buildPBWhere(f pbdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addIlike := func(col, term string) {
		term = strings.TrimSpace(term)
		if term != "" {
			where = append(where, fmt.Sprintf("LOWER(%s) LIKE LOWER($%d)", col, len(args)+1))
			args = append(args, "%"+term+"%")
		}
	}
	addIn := func(col string, vals []string) {
		if len(vals) == 0 {
			return
		}
		base := len(args)
		ph := make([]string, 0, len(vals))
		for i, v := range vals {
			ph = append(ph, fmt.Sprintf("$%d", base+i+1))
			args = append(args, strings.TrimSpace(v))
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
	}
	addInEnum := func(col string, vals []string) {
		addIn(col, vals)
	}

	if s := strings.TrimSpace(f.SearchTerm); s != "" {
		addIlike("product_name", s)
	}

	addIn("brand_id", f.BrandIDs)
	addIn("assignee_id", f.AssigneeIDs)

	if len(f.ItemTypes) > 0 {
		vals := make([]string, 0, len(f.ItemTypes))
		for _, it := range f.ItemTypes {
			vals = append(vals, strings.TrimSpace(string(it)))
		}
		addInEnum("item_type", vals)
	}

	if len(f.TagTypes) > 0 {
		vals := make([]string, 0, len(f.TagTypes))
		for _, t := range f.TagTypes {
			vals = append(vals, strings.TrimSpace(string(t)))
		}
		addInEnum("product_id_tag_type", vals)
	}

	if len(f.VariationIDs) > 0 {
		where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(model_variations) AS v(x)
  WHERE (v.x->>'id' = ANY($%d) OR v.x->>'ID' = ANY($%d))
)`, len(args)+1, len(args)+1))
		args = append(args, pq.Array(f.VariationIDs))
	}

	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("updated_at >= $%d", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("updated_at < $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}

	return where, args
}

func buildPBOrderBy(s pbdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "productname", "product_name":
		col = "product_name"
	case "brandid", "brand_id":
		col = "brand_id"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

func dedupTrimStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}
