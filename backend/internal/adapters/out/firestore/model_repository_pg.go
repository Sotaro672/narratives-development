package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	modeldom "narratives/internal/domain/model"
)

type ModelRepositoryPG struct {
	DB *sql.DB
}

func NewModelRepositoryPG(db *sql.DB) *ModelRepositoryPG {
	return &ModelRepositoryPG{DB: db}
}

// WithTx executes fn within a DB transaction and injects it into ctx.
func (r *ModelRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	ctxTx := dbcommon.CtxWithTx(ctx, tx)
	if err := fn(ctxTx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ==========================================================
// Implement usecase.ModelRepo interface (basic CRUD facade)
// ==========================================================

// GetByID returns a single modeldom.Model by its ID.
// NOTE: not implemented yet – stub to satisfy interface.
func (r *ModelRepositoryPG) GetByID(ctx context.Context, id string) (modeldom.Model, error) {
	_ = r
	_ = ctx
	_ = id
	return modeldom.Model{}, errors.New("GetByID not implemented")
}

// Exists returns true if a model with given ID exists.
// NOTE: not implemented yet – stub to satisfy interface.
func (r *ModelRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	_ = r
	_ = ctx
	_ = id
	return false, errors.New("Exists not implemented")
}

// Create persists a new modeldom.Model.
// NOTE: not implemented yet – stub to satisfy interface.
func (r *ModelRepositoryPG) Create(ctx context.Context, m modeldom.Model) (modeldom.Model, error) {
	_ = r
	_ = ctx
	_ = m
	return modeldom.Model{}, errors.New("Create not implemented")
}

// Save updates an existing modeldom.Model.
// NOTE: not implemented yet – stub to satisfy interface.
func (r *ModelRepositoryPG) Save(ctx context.Context, m modeldom.Model) (modeldom.Model, error) {
	_ = r
	_ = ctx
	_ = m
	return modeldom.Model{}, errors.New("Save not implemented")
}

// Delete removes a model by ID.
// NOTE: not implemented yet – stub to satisfy interface.
func (r *ModelRepositoryPG) Delete(ctx context.Context, id string) error {
	_ = r
	_ = ctx
	_ = id
	return errors.New("Delete not implemented")
}

// ==========================
// Product-scoped model data
// ==========================

func (r *ModelRepositoryPG) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const qSet = `
SELECT product_id, product_blueprint_id, updated_at
FROM model_sets
WHERE product_id = $1`
	var (
		pid, blueprintID string
		updatedAt        time.Time
	)
	if err := run.QueryRowContext(ctx, qSet, strings.TrimSpace(productID)).Scan(&pid, &blueprintID, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	vars, err := r.listVariationsByBlueprintID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          strings.TrimSpace(pid),
		ProductBlueprintID: strings.TrimSpace(blueprintID),
		Variations:         vars,
		UpdatedAt:          updatedAt.UTC(),
	}, nil
}

func (r *ModelRepositoryPG) GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const qSet = `
SELECT product_id, product_blueprint_id, updated_at
FROM model_sets
WHERE product_blueprint_id = $1`
	var (
		pid, blueprintID string
		updatedAt        time.Time
	)
	if err := run.QueryRowContext(ctx, qSet, strings.TrimSpace(productBlueprintID)).Scan(&pid, &blueprintID, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	vars, err := r.listVariationsByBlueprintID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          strings.TrimSpace(pid),
		ProductBlueprintID: strings.TrimSpace(blueprintID),
		Variations:         vars,
		UpdatedAt:          updatedAt.UTC(),
	}, nil
}

func (r *ModelRepositoryPG) UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	// Supported keys: productBlueprintID (or product_blueprint_id)
	if v, ok := updates["productBlueprintID"]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			sets = append(sets, fmt.Sprintf("product_blueprint_id = $%d", i))
			args = append(args, strings.TrimSpace(s))
			i++
		}
	}
	if v, ok := updates["product_blueprint_id"]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			sets = append(sets, fmt.Sprintf("product_blueprint_id = $%d", i))
			args = append(args, strings.TrimSpace(s))
			i++
		}
	}

	// Always bump updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	now := time.Now().UTC()
	args = append(args, now)
	i++

	if len(sets) == 0 {
		// nothing to update; still touch updated_at
	}

	args = append(args, strings.TrimSpace(productID))
	q := fmt.Sprintf(`
UPDATE model_sets
SET %s
WHERE product_id = $%d
RETURNING product_id, product_blueprint_id, updated_at
`, strings.Join(sets, ", "), i)

	var (
		pid, blueprintID string
		updatedAt        time.Time
	)
	if err := run.QueryRowContext(ctx, q, args...).Scan(&pid, &blueprintID, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	vars, err := r.listVariationsByBlueprintID(ctx, blueprintID)
	if err != nil {
		return nil, err
	}

	return &modeldom.ModelData{
		ProductID:          strings.TrimSpace(pid),
		ProductBlueprintID: strings.TrimSpace(blueprintID),
		Variations:         vars,
		UpdatedAt:          updatedAt.UTC(),
	}, nil
}

// ==========================
// Variations CRUD + listing
// ==========================

func (r *ModelRepositoryPG) ListVariations(ctx context.Context, filter modeldom.VariationFilter, sort modeldom.VariationSort, page modeldom.Page) (modeldom.VariationPageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildVariationWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildVariationOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY model_number ASC, size ASC, color ASC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_variations mv "+whereSQL, args...).Scan(&total); err != nil {
		return modeldom.VariationPageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  mv.id, mv.product_blueprint_id, mv.model_number, mv.size, mv.color, mv.measurements,
  mv.created_at, mv.created_by, mv.updated_at, mv.updated_by, mv.deleted_at, mv.deleted_by
FROM model_variations mv
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return modeldom.VariationPageResult{}, err
	}
	defer rows.Close()

	items := make([]modeldom.ModelVariation, 0, perPage)
	for rows.Next() {
		v, err := scanModelVariation(rows)
		if err != nil {
			return modeldom.VariationPageResult{}, err
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return modeldom.VariationPageResult{}, err
	}

	return modeldom.VariationPageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ModelRepositoryPG) CountVariations(ctx context.Context, filter modeldom.VariationFilter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildVariationWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_variations mv "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ModelRepositoryPG) GetModelVariations(ctx context.Context, productID string) ([]modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  mv.id, mv.product_blueprint_id, mv.model_number, mv.size, mv.color, mv.measurements,
  mv.created_at, mv.created_by, mv.updated_at, mv.updated_by, mv.deleted_at, mv.deleted_by
FROM model_variations mv
JOIN model_sets ms ON ms.product_blueprint_id = mv.product_blueprint_id
WHERE ms.product_id = $1
ORDER BY mv.model_number ASC, mv.size ASC, mv.color ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []modeldom.ModelVariation
	for rows.Next() {
		v, err := scanModelVariation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ModelRepositoryPG) GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM model_variations
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(variationID))
	v, err := scanModelVariation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

func (r *ModelRepositoryPG) CreateModelVariation(ctx context.Context, productID string, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// Resolve blueprint by product
	const qBlueprint = `SELECT product_blueprint_id FROM model_sets WHERE product_id = $1`
	var blueprintID string
	if err := run.QueryRowContext(ctx, qBlueprint, strings.TrimSpace(productID)).Scan(&blueprintID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	measureJSON, err := json.Marshal(variation.Measurements)
	if err != nil {
		return nil, err
	}

	// Generate id in DB (gen_random_uuid()), set created_at NOW()
	const qIns = `
INSERT INTO model_variations (
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, updated_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5::jsonb,
  NOW(), NOW()
)
RETURNING
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := run.QueryRowContext(ctx, qIns,
		strings.TrimSpace(blueprintID),
		strings.TrimSpace(variation.ModelNumber),
		strings.TrimSpace(variation.Size),
		strings.TrimSpace(variation.Color),
		string(measureJSON),
	)
	v, err := scanModelVariation(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, modeldom.ErrConflict
		}
		return nil, err
	}
	return &v, nil
}

func (r *ModelRepositoryPG) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	if updates.Size != nil {
		sets = append(sets, fmt.Sprintf("size = $%d", i))
		args = append(args, strings.TrimSpace(*updates.Size))
		i++
	}
	if updates.Color != nil {
		sets = append(sets, fmt.Sprintf("color = $%d", i))
		args = append(args, strings.TrimSpace(*updates.Color))
		i++
	}
	if updates.ModelNumber != nil {
		sets = append(sets, fmt.Sprintf("model_number = $%d", i))
		args = append(args, strings.TrimSpace(*updates.ModelNumber))
		i++
	}
	if updates.Measurements != nil {
		jb, err := json.Marshal(updates.Measurements)
		if err != nil {
			return nil, err
		}
		sets = append(sets, fmt.Sprintf("measurements = $%d::jsonb", i))
		args = append(args, string(jb))
		i++
	}

	// Touch updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now().UTC())
	i++

	if len(sets) == 0 {
		return r.GetModelVariationByID(ctx, variationID)
	}

	args = append(args, strings.TrimSpace(variationID))
	q := fmt.Sprintf(`
UPDATE model_variations
SET %s
WHERE id = $%d
RETURNING
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	v, err := scanModelVariation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, modeldom.ErrConflict
		}
		return nil, err
	}
	return &v, nil
}

func (r *ModelRepositoryPG) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
DELETE FROM model_variations
WHERE id = $1
RETURNING
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(variationID))
	v, err := scanModelVariation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

func (r *ModelRepositoryPG) ReplaceModelVariations(ctx context.Context, productID string, variations []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	// Transactional replace: delete all for blueprint, then insert provided list
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	ctxTx := dbcommon.CtxWithTx(ctx, tx)
	run := dbcommon.GetRunner(ctxTx, r.DB)

	// Resolve blueprint
	const qBlueprint = `SELECT product_blueprint_id FROM model_sets WHERE product_id = $1`
	var blueprintID string
	if err := run.QueryRowContext(ctxTx, qBlueprint, strings.TrimSpace(productID)).Scan(&blueprintID); err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modeldom.ErrNotFound
		}
		return nil, err
	}

	// Delete existing
	if _, err := run.ExecContext(ctxTx, `DELETE FROM model_variations WHERE product_blueprint_id = $1`, blueprintID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	// Insert new
	const qIns = `
INSERT INTO model_variations (
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, updated_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5::jsonb, NOW(), NOW()
)`
	for _, nv := range variations {
		jb, err := json.Marshal(nv.Measurements)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := run.ExecContext(ctxTx, qIns,
			strings.TrimSpace(blueprintID),
			strings.TrimSpace(nv.ModelNumber),
			strings.TrimSpace(nv.Size),
			strings.TrimSpace(nv.Color),
			string(jb),
		); err != nil {
			_ = tx.Rollback()
			if dbcommon.IsUniqueViolation(err) {
				return nil, modeldom.ErrConflict
			}
			return nil, err
		}
	}

	// Load back
	out, err := r.listVariationsByBlueprintID(ctxTx, blueprintID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	return out, tx.Commit()
}

func (r *ModelRepositoryPG) GetSizeVariations(ctx context.Context, productID string) ([]modeldom.SizeVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT mv.id, mv.size, mv.measurements
FROM model_variations mv
JOIN model_sets ms ON ms.product_blueprint_id = mv.product_blueprint_id
WHERE ms.product_id = $1
ORDER BY mv.size ASC, mv.id ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []modeldom.SizeVariation{}
	for rows.Next() {
		var id, size string
		var measurementsRaw []byte
		if err := rows.Scan(&id, &size, &measurementsRaw); err != nil {
			return nil, err
		}
		var meas map[string]float64
		if len(measurementsRaw) > 0 {
			if err := json.Unmarshal(measurementsRaw, &meas); err != nil {
				return nil, err
			}
		}
		out = append(out, modeldom.SizeVariation{
			ID:           strings.TrimSpace(id),
			Size:         strings.TrimSpace(size),
			Measurements: meas,
		})
	}
	return out, rows.Err()
}

// Note: Without a production quantities source, return 0 quantities.
func (r *ModelRepositoryPG) GetProductionQuantities(ctx context.Context, productID string) ([]modeldom.ProductionQuantity, error) {
	_ = ctx
	_ = productID
	return []modeldom.ProductionQuantity{}, nil
}

func (r *ModelRepositoryPG) GetModelNumbers(ctx context.Context, productID string) ([]modeldom.ModelNumber, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT mv.size, mv.color, mv.model_number
FROM model_variations mv
JOIN model_sets ms ON ms.product_blueprint_id = mv.product_blueprint_id
WHERE ms.product_id = $1
ORDER BY mv.model_number ASC, mv.size ASC, mv.color ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []modeldom.ModelNumber
	for rows.Next() {
		var size, color, modelNumber string
		if err := rows.Scan(&size, &color, &modelNumber); err != nil {
			return nil, err
		}
		out = append(out, modeldom.ModelNumber{
			Size:        strings.TrimSpace(size),
			Color:       strings.TrimSpace(color),
			ModelNumber: strings.TrimSpace(modelNumber),
		})
	}
	return out, rows.Err()
}

func (r *ModelRepositoryPG) GetModelVariationsWithQuantity(ctx context.Context, productID string) ([]modeldom.ModelVariationWithQuantity, error) {
	vars, err := r.GetModelVariations(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]modeldom.ModelVariationWithQuantity, 0, len(vars))
	for _, v := range vars {
		out = append(out, modeldom.ModelVariationWithQuantity{
			ModelVariation: v,
			Quantity:       0,
		})
	}
	return out, nil
}

// ==========================
// Helpers
// ==========================

func (r *ModelRepositoryPG) listVariationsByBlueprintID(ctx context.Context, blueprintID string) ([]modeldom.ModelVariation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id, product_blueprint_id, model_number, size, color, measurements,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM model_variations
WHERE product_blueprint_id = $1
ORDER BY model_number ASC, size ASC, color ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(blueprintID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []modeldom.ModelVariation
	for rows.Next() {
		v, err := scanModelVariation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanModelVariation(s dbcommon.RowScanner) (modeldom.ModelVariation, error) {
	var (
		id, blueprintID, modelNumber, size, color string
		measurementsRaw                           []byte
		createdAtNS, updatedAtNS, deletedAtNS     sql.NullTime
		createdByNS, updatedByNS, deletedByNS     sql.NullString
	)
	if err := s.Scan(
		&id, &blueprintID, &modelNumber, &size, &color, &measurementsRaw,
		&createdAtNS, &createdByNS, &updatedAtNS, &updatedByNS, &deletedAtNS, &deletedByNS,
	); err != nil {
		return modeldom.ModelVariation{}, err
	}

	var meas map[string]float64
	if len(measurementsRaw) > 0 {
		if err := json.Unmarshal(measurementsRaw, &meas); err != nil {
			return modeldom.ModelVariation{}, err
		}
	}

	toTimePtr := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}
	toStrPtr := func(ns sql.NullString) *string {
		if ns.Valid {
			v := strings.TrimSpace(ns.String)
			if v == "" {
				return nil
			}
			return &v
		}
		return nil
	}

	return modeldom.ModelVariation{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(blueprintID),
		ModelNumber:        strings.TrimSpace(modelNumber),
		Size:               strings.TrimSpace(size),
		Color:              strings.TrimSpace(color),
		Measurements:       meas,
		CreatedAt:          toTimePtr(createdAtNS),
		CreatedBy:          toStrPtr(createdByNS),
		UpdatedAt:          toTimePtr(updatedAtNS),
		UpdatedBy:          toStrPtr(updatedByNS),
		DeletedAt:          toTimePtr(deletedAtNS),
		DeletedBy:          toStrPtr(deletedByNS),
	}, nil
}

func buildVariationWhere(f modeldom.VariationFilter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// product filter via join alias "mv" with optional join conditions added by callers
	if v := strings.TrimSpace(f.ProductID); v != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM model_sets ms WHERE ms.product_blueprint_id = mv.product_blueprint_id AND ms.product_id = $%d)", len(args)+1))
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.ProductBlueprintID); v != "" {
		where = append(where, fmt.Sprintf("mv.product_blueprint_id = $%d", len(args)+1))
		args = append(args, v)
	}

	// arrays
	if len(f.Sizes) > 0 {
		ph := make([]string, len(f.Sizes))
		for i := range f.Sizes {
			ph[i] = fmt.Sprintf("$%d", len(args)+i+1)
		}
		for _, s := range f.Sizes {
			args = append(args, strings.TrimSpace(s))
		}
		where = append(where, fmt.Sprintf("mv.size IN (%s)", strings.Join(ph, ",")))
	}
	if len(f.Colors) > 0 {
		ph := make([]string, len(f.Colors))
		for i := range f.Colors {
			ph[i] = fmt.Sprintf("$%d", len(args)+i+1)
		}
		for _, c := range f.Colors {
			args = append(args, strings.TrimSpace(c))
		}
		where = append(where, fmt.Sprintf("mv.color IN (%s)", strings.Join(ph, ",")))
	}
	if len(f.ModelNumbers) > 0 {
		ph := make([]string, len(f.ModelNumbers))
		for i := range f.ModelNumbers {
			ph[i] = fmt.Sprintf("$%d", len(args)+i+1)
		}
		for _, mn := range f.ModelNumbers {
			args = append(args, strings.TrimSpace(mn))
		}
		where = append(where, fmt.Sprintf("mv.model_number IN (%s)", strings.Join(ph, ",")))
	}

	// free text search
	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		where = append(where, fmt.Sprintf("(mv.model_number ILIKE $%d OR mv.size ILIKE $%d OR mv.color ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1))
		args = append(args, "%"+q+"%")
	}

	// time ranges
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("(mv.updated_at IS NOT NULL AND mv.updated_at >= $%d)", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("(mv.updated_at IS NOT NULL AND mv.updated_at < $%d)", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("(mv.created_at IS NOT NULL AND mv.created_at >= $%d)", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("(mv.created_at IS NOT NULL AND mv.created_at < $%d)", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}

	// deletion filter
	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "mv.deleted_at IS NOT NULL")
		} else {
			where = append(where, "mv.deleted_at IS NULL")
		}
	}

	return where, args
}

func buildVariationOrderBy(sort modeldom.VariationSort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "modelnumber", "model_number":
		col = "mv.model_number"
	case "size":
		col = "mv.size"
	case "color":
		col = "mv.color"
	case "createdat", "created_at":
		col = "mv.created_at"
	case "updatedat", "updated_at":
		col = "mv.updated_at"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s, mv.id %s", col, dir, dir)
}
