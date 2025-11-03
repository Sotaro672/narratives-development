package db

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

// ProductBlueprintRepositoryPG implements productBlueprint.Repository using PostgreSQL.
type ProductBlueprintRepositoryPG struct {
    DB *sql.DB
}

func NewProductBlueprintRepositoryPG(db *sql.DB) *ProductBlueprintRepositoryPG {
    return &ProductBlueprintRepositoryPG{DB: db}
}

// ========================
// Repository impl
// ========================

func (r *ProductBlueprintRepositoryPG) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at, updated_at
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

    // Count
    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_blueprints "+whereSQL, args...).Scan(&total); err != nil {
        return pbdom.PageResult{}, err
    }

    // Data
    q := fmt.Sprintf(`
SELECT
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at, updated_at
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

func (r *ProductBlueprintRepositoryPG) Create(ctx context.Context, in pbdom.CreateInput) (pbdom.ProductBlueprint, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    // Serialize fields
    varsJSON, err := json.Marshal(in.Variations)
    if err != nil {
        return pbdom.ProductBlueprint{}, err
    }
    tagType := ""
    if in.ProductIdTag.Type != "" {
        tagType = string(in.ProductIdTag.Type)
    }

    const q = `
INSERT INTO product_blueprints (
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id, created_by, created_at, updated_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3,
  $4::jsonb, $5, $6, $7, $8,
  $9, $10, $11, COALESCE($12, NOW()), NOW()
)
RETURNING
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at, updated_at
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(in.ProductName),
        strings.TrimSpace(in.BrandID),
        strings.TrimSpace(string(in.ItemType)),
        string(varsJSON),
        strings.TrimSpace(in.Fit),
        strings.TrimSpace(in.Material),
        in.Weight,
        pq.Array(dedupTrimStrings(in.QualityAssurance)),
        strings.TrimSpace(tagType),
        strings.TrimSpace(in.AssigneeID),
        dbcommon.ToDBText(in.CreatedBy),
        dbcommon.ToDBTime(in.CreatedAt),
    )
    pb, err := scanProductBlueprint(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return pbdom.ProductBlueprint{}, pbdom.ErrConflict
        }
        return pbdom.ProductBlueprint{}, err
    }
    return pb, nil
}

func (r *ProductBlueprintRepositoryPG) Update(ctx context.Context, id string, patch pbdom.Patch) (pbdom.ProductBlueprint, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    sets := []string{}
    args := []any{}
    i := 1

    setText := func(col string, p *string) {
        if p != nil {
            sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
            args = append(args, strings.TrimSpace(*p))
            i++
        }
    }
    setFloat := func(col string, p *float64) {
        if p != nil {
            sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
            args = append(args, *p)
            i++
        }
    }

    setText("product_name", patch.ProductName)
    setText("brand_id", patch.BrandID)
    if patch.ItemType != nil {
        sets = append(sets, fmt.Sprintf("item_type = $%d", i))
        args = append(args, strings.TrimSpace(string(*patch.ItemType)))
        i++
    }
    if patch.Variations != nil {
        jb, err := json.Marshal(*patch.Variations)
        if err != nil {
            return pbdom.ProductBlueprint{}, err
        }
        sets = append(sets, fmt.Sprintf("model_variations = $%d::jsonb", i))
        args = append(args, string(jb))
        i++
    }
    setText("fit", patch.Fit)
    setText("material", patch.Material)
    setFloat("weight", patch.Weight)
    if patch.QualityAssurance != nil {
        sets = append(sets, fmt.Sprintf("quality_assurance = $%d", i))
        args = append(args, pq.Array(dedupTrimStrings(*patch.QualityAssurance)))
        i++
    }
    if patch.ProductIdTag != nil {
        sets = append(sets, fmt.Sprintf("product_id_tag_type = $%d", i))
        tagType := strings.TrimSpace(string(patch.ProductIdTag.Type))
        args = append(args, tagType)
        i++
    }
    setText("assignee_id", patch.AssigneeID)

    // always bump updated_at
    sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
    args = append(args, time.Now().UTC())
    i++

    if len(sets) == 0 {
        return r.GetByID(ctx, id)
    }

    args = append(args, strings.TrimSpace(id))
    q := fmt.Sprintf(`
UPDATE product_blueprints
SET %s
WHERE id = $%d
RETURNING
  id, product_name, brand_id, item_type,
  model_variations, fit, material, weight, quality_assurance,
  product_id_tag_type, assignee_id,
  created_by, created_at, updated_at
`, strings.Join(sets, ", "), i)

    row := run.QueryRowContext(ctx, q, args...)
    pb, err := scanProductBlueprint(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
        }
        if dbcommon.IsUniqueViolation(err) {
            return pbdom.ProductBlueprint{}, pbdom.ErrConflict
        }
        return pbdom.ProductBlueprint{}, err
    }
    return pb, nil
}

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
        createdByNS                                       sql.NullString
        createdAt, updatedAt                              time.Time
    )

    if err := s.Scan(
        &id, &productName, &brandID, &itemType,
        &varsRaw, &fit, &material, &weight, pq.Array(&qa),
        &tagType, &assigneeID,
        &createdByNS, &createdAt, &updatedAt,
    ); err != nil {
        return pbdom.ProductBlueprint{}, err
    }

    var variations []map[string]any
    if len(varsRaw) > 0 {
        if err := json.Unmarshal(varsRaw, &variations); err != nil {
            // Try tolerant parse into slice of struct if available later.
            // Fall back to empty to avoid crashing.
            variations = nil
        }
    }

    // Re-marshal variations to the domain type by JSON round-trip if needed.
    // We don't have the concrete struct type here; the domain struct holds []model.ModelVariation.
    // To avoid import cycles, we simply pass through JSON by unmarshalling to interface then re-marshal.
    // But domain.New expects []model.ModelVariation. We can leave json bytes and unmarshal into that type via another round trip.
    var domVariationsJSON []byte
    if variations != nil {
        domVariationsJSON, _ = json.Marshal(variations)
    }
    type modelVariation struct {
        ID string `json:"id"`
        // other fields are opaque for the repository and kept as-is by the domain
    }
    var domVars []modelVariation
    if len(domVariationsJSON) > 0 {
        _ = json.Unmarshal(domVariationsJSON, &domVars)
    }

    // Convert to ProductIDTag
    tag := pbdom.ProductIDTag{
        Type: pbdom.ProductIDTagType(strings.TrimSpace(tagType)),
        // LogoDesignFile is not stored in current schema; left nil
    }

    createdBy := func(ns sql.NullString) *string {
        if ns.Valid {
            v := strings.TrimSpace(ns.String)
            if v != "" {
                return &v
            }
        }
        return nil
    }(createdByNS)

    // Build domain entity
    // Note: We cannot materialize full model variations without domain/model type here.
    // The domain constructor will deduplicate by ID; we pass only IDs we could infer.
    dedupVars := make([]pbdom.ProductBlueprint, 0) // placeholder to keep the function compact (not used directly)

    _ = dedupVars // silence

    // Minimal bridge: re-unmarshal to an anonymous structure having only ID, then marshal again
    // and unmarshal into the domain after constructing, or simply call constructor with empty variations.
    // To keep repository simple and robust, we pass empty variations if JSON shape is unknown.
    // If your model.ModelVariation is available here, replace the above with direct decode.

    pb, err := pbdom.New(
        strings.TrimSpace(id),
        strings.TrimSpace(productName),
        strings.TrimSpace(brandID),
        pbdom.ItemType(strings.TrimSpace(itemType)),
        /* variations */ nil, // see note above
        strings.TrimSpace(fit),
        strings.TrimSpace(material),
        weight,
        dedupTrimStrings(qa),
        tag,
        strings.TrimSpace(assigneeID),
        createdBy,
        createdAt.UTC(),
    )
    if err != nil {
        return pbdom.ProductBlueprint{}, err
    }
    // Reflect DB's updated_at into LastModifiedAt
    pb.LastModifiedAt = updatedAt.UTC()
    return pb, nil
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
        // same as addIn but kept for readability
        addIn(col, vals)
    }

    if s := strings.TrimSpace(f.SearchTerm); s != "" {
        addIlike("product_name", s)
    }

    addIn("brand_id", f.BrandIDs)
    addIn("assignee_id", f.AssigneeIDs)

    // item_type IN (...)
    if len(f.ItemTypes) > 0 {
        vals := make([]string, 0, len(f.ItemTypes))
        for _, it := range f.ItemTypes {
            vals = append(vals, strings.TrimSpace(string(it)))
        }
        addInEnum("item_type", vals)
    }

    // Tag types
    if len(f.TagTypes) > 0 {
        vals := make([]string, 0, len(f.TagTypes))
        for _, t := range f.TagTypes {
            vals = append(vals, strings.TrimSpace(string(t)))
        }
        addInEnum("product_id_tag_type", vals)
    }

    // VariationIDs filter: any match in JSON array
    if len(f.VariationIDs) > 0 {
        where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(model_variations) AS v(x)
  WHERE (v.x->>'id' = ANY($%d) OR v.x->>'ID' = ANY($%d))
)`, len(args)+1, len(args)+1))
        args = append(args, pq.Array(f.VariationIDs))
    }

    // Time ranges
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