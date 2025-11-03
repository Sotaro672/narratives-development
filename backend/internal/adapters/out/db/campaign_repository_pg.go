package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    camdom "narratives/internal/domain/campaign"
)

type CampaignRepositoryPG struct {
    DB *sql.DB
}

func NewCampaignRepositoryPG(db *sql.DB) *CampaignRepositoryPG {
    return &CampaignRepositoryPG{DB: db}
}

// ========== Queries ==========

func (r *CampaignRepositoryPG) List(ctx context.Context, filter camdom.Filter, sort camdom.Sort, page camdom.Page) (camdom.PageResult[camdom.Campaign], error) {
    where, args := buildCampaignWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildCampaignOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY created_at DESC, id DESC"
    }

    perPage := page.PerPage
    if perPage <= 0 {
        perPage = 50
    }
    number := page.Number
    if number <= 0 {
        number = 1
    }
    offset := (number - 1) * perPage

    var total int
    countSQL := fmt.Sprintf("SELECT COUNT(*) FROM campaigns %s", whereSQL)
    if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
        return camdom.PageResult[camdom.Campaign]{}, err
    }

    q := fmt.Sprintf(`
SELECT
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM campaigns
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)

    rows, err := r.DB.QueryContext(ctx, q, args...)
    if err != nil {
        return camdom.PageResult[camdom.Campaign]{}, err
    }
    defer rows.Close()

    var items []camdom.Campaign
    for rows.Next() {
        c, err := scanCampaign(rows)
        if err != nil {
            return camdom.PageResult[camdom.Campaign]{}, err
        }
        items = append(items, c)
    }
    if err := rows.Err(); err != nil {
        return camdom.PageResult[camdom.Campaign]{}, err
    }

    totalPages := (total + perPage - 1) / perPage
    return camdom.PageResult[camdom.Campaign]{
        Items:      items,
        TotalCount: total,
        TotalPages: totalPages,
        Page:       number,
        PerPage:    perPage,
    }, nil
}

func (r *CampaignRepositoryPG) ListByCursor(ctx context.Context, filter camdom.Filter, _ camdom.Sort, cpage camdom.CursorPage) (camdom.CursorPageResult[camdom.Campaign], error) {
    where, args := buildCampaignWhere(filter)
    if after := strings.TrimSpace(cpage.After); after != "" {
        where = append(where, fmt.Sprintf("id > $%d", len(args)+1))
        args = append(args, after)
    }
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    limit := cpage.Limit
    if limit <= 0 || limit > 200 {
        limit = 50
    }

    q := fmt.Sprintf(`
SELECT
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM campaigns
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

    args = append(args, limit+1)

    rows, err := r.DB.QueryContext(ctx, q, args...)
    if err != nil {
        return camdom.CursorPageResult[camdom.Campaign]{}, err
    }
    defer rows.Close()

    var items []camdom.Campaign
    var lastID string
    for rows.Next() {
        c, err := scanCampaign(rows)
        if err != nil {
            return camdom.CursorPageResult[camdom.Campaign]{}, err
        }
        items = append(items, c)
        lastID = c.ID
    }
    if err := rows.Err(); err != nil {
        return camdom.CursorPageResult[camdom.Campaign]{}, err
    }

    var next *string
    if len(items) > limit {
        items = items[:limit]
        next = &lastID
    }

    return camdom.CursorPageResult[camdom.Campaign]{
        Items:      items,
        NextCursor: next,
        Limit:      limit,
    }, nil
}

func (r *CampaignRepositoryPG) GetByID(ctx context.Context, id string) (camdom.Campaign, error) {
    const q = `
SELECT
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM campaigns
WHERE id = $1
`
    row := r.DB.QueryRowContext(ctx, q, id)
    c, err := scanCampaign(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return camdom.Campaign{}, camdom.ErrNotFound
        }
        return camdom.Campaign{}, err
    }
    return c, nil
}

func (r *CampaignRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
    const q = `SELECT 1 FROM campaigns WHERE id = $1`
    var one int
    err := r.DB.QueryRowContext(ctx, q, id).Scan(&one)
    if errors.Is(err, sql.ErrNoRows) {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}

func (r *CampaignRepositoryPG) Count(ctx context.Context, filter camdom.Filter) (int, error) {
    where, args := buildCampaignWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }
    var total int
    if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaigns `+whereSQL, args...).Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

// ========== Mutations ==========

func (r *CampaignRepositoryPG) Create(ctx context.Context, c camdom.Campaign) (camdom.Campaign, error) {
    const q = `
INSERT INTO campaigns (
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,
  $9,$10,$11,$12,$13,$14,
  $15,$16,$17,$18,$19,$20,$21,$22
)
RETURNING
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
`
    row := r.DB.QueryRowContext(ctx, q,
        strings.TrimSpace(c.ID),
        strings.TrimSpace(c.Name),
        strings.TrimSpace(c.BrandID),
        strings.TrimSpace(c.AssigneeID),
        strings.TrimSpace(c.ListID),
        string(c.Status),
        c.Budget,
        c.Spent,
        c.StartDate.UTC(),
        c.EndDate.UTC(),
        strings.TrimSpace(c.TargetAudience),
        string(c.AdType),
        strings.TrimSpace(c.Headline),
        strings.TrimSpace(c.Description),
        dbcommon.ToDBText(c.PerformanceID),
        dbcommon.ToDBText(c.ImageID),
        dbcommon.ToDBText(c.CreatedBy),
        c.CreatedAt.UTC(),
        dbcommon.ToDBText(c.UpdatedBy),
        dbcommon.ToDBTime(c.UpdatedAt),
        dbcommon.ToDBTime(c.DeletedAt),
        dbcommon.ToDBText(c.DeletedBy),
    )
    out, err := scanCampaign(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return camdom.Campaign{}, camdom.ErrConflict
        }
        return camdom.Campaign{}, err
    }
    return out, nil
}

func (r *CampaignRepositoryPG) Update(ctx context.Context, id string, patch camdom.CampaignPatch) (camdom.Campaign, error) {
    sets := []string{}
    args := []any{}
    i := 1

    if patch.Name != nil {
        sets = append(sets, fmt.Sprintf("name = $%d", i))
        args = append(args, strings.TrimSpace(*patch.Name))
        i++
    }
    if patch.BrandID != nil {
        sets = append(sets, fmt.Sprintf("brand_id = $%d", i))
        args = append(args, strings.TrimSpace(*patch.BrandID))
        i++
    }
    if patch.AssigneeID != nil {
        sets = append(sets, fmt.Sprintf("assignee_id = $%d", i))
        args = append(args, strings.TrimSpace(*patch.AssigneeID))
        i++
    }
    if patch.ListID != nil {
        sets = append(sets, fmt.Sprintf("list_id = $%d", i))
        args = append(args, strings.TrimSpace(*patch.ListID))
        i++
    }
    if patch.Status != nil {
        sets = append(sets, fmt.Sprintf("status = $%d", i))
        args = append(args, string(*patch.Status))
        i++
    }
    if patch.Budget != nil {
        sets = append(sets, fmt.Sprintf("budget = $%d", i))
        args = append(args, *patch.Budget)
        i++
    }
    if patch.Spent != nil {
        sets = append(sets, fmt.Sprintf("spent = $%d", i))
        args = append(args, *patch.Spent)
        i++
    }
    if patch.StartDate != nil {
        sets = append(sets, fmt.Sprintf("start_date = $%d", i))
        args = append(args, patch.StartDate.UTC())
        i++
    }
    if patch.EndDate != nil {
        sets = append(sets, fmt.Sprintf("end_date = $%d", i))
        args = append(args, patch.EndDate.UTC())
        i++
    }
    if patch.TargetAudience != nil {
        sets = append(sets, fmt.Sprintf("target_audience = $%d", i))
        args = append(args, strings.TrimSpace(*patch.TargetAudience))
        i++
    }
    if patch.AdType != nil {
        sets = append(sets, fmt.Sprintf("ad_type = $%d", i))
        args = append(args, string(*patch.AdType))
        i++
    }
    if patch.Headline != nil {
        sets = append(sets, fmt.Sprintf("headline = $%d", i))
        args = append(args, strings.TrimSpace(*patch.Headline))
        i++
    }
    if patch.Description != nil {
        sets = append(sets, fmt.Sprintf("description = $%d", i))
        args = append(args, strings.TrimSpace(*patch.Description))
        i++
    }
    if patch.PerformanceID != nil {
        sets = append(sets, fmt.Sprintf("performance_id = $%d", i))
        args = append(args, dbcommon.ToDBText(patch.PerformanceID))
        i++
    }
    if patch.ImageID != nil {
        sets = append(sets, fmt.Sprintf("image_id = $%d", i))
        args = append(args, dbcommon.ToDBText(patch.ImageID))
        i++
    }
    if patch.UpdatedBy != nil {
        sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
        args = append(args, dbcommon.ToDBText(patch.UpdatedBy))
        i++
    }
    if patch.DeletedAt != nil {
        sets = append(sets, fmt.Sprintf("deleted_at = $%d", i))
        args = append(args, dbcommon.ToDBTime(patch.DeletedAt))
        i++
    }
    if patch.DeletedBy != nil {
        sets = append(sets, fmt.Sprintf("deleted_by = $%d", i))
        args = append(args, dbcommon.ToDBText(patch.DeletedBy))
        i++
    }
    // updated_at: explicit or NOW()
    if patch.UpdatedAt != nil {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, dbcommon.ToDBTime(patch.UpdatedAt))
        i++
    } else {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, time.Now().UTC())
        i++
    }

    if len(sets) == 0 {
        return r.GetByID(ctx, id)
    }

    args = append(args, id)
    q := fmt.Sprintf(`
UPDATE campaigns
SET %s
WHERE id = $%d
RETURNING
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

    row := r.DB.QueryRowContext(ctx, q, args...)
    out, err := scanCampaign(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return camdom.Campaign{}, camdom.ErrNotFound
        }
        return camdom.Campaign{}, err
    }
    return out, nil
}

func (r *CampaignRepositoryPG) Delete(ctx context.Context, id string) error {
    res, err := r.DB.ExecContext(ctx, `DELETE FROM campaigns WHERE id = $1`, id)
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return camdom.ErrNotFound
    }
    return nil
}

func (r *CampaignRepositoryPG) Save(ctx context.Context, c camdom.Campaign, _ *camdom.SaveOptions) (camdom.Campaign, error) {
    const q = `
INSERT INTO campaigns (
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,
  $9,$10,$11,$12,$13,$14,
  $15,$16,$17,$18,$19,$20,$21,$22
)
ON CONFLICT (id) DO UPDATE SET
  name            = EXCLUDED.name,
  brand_id        = EXCLUDED.brand_id,
  assignee_id     = EXCLUDED.assignee_id,
  list_id         = EXCLUDED.list_id,
  status          = EXCLUDED.status,
  budget          = EXCLUDED.budget,
  spent           = EXCLUDED.spent,
  start_date      = EXCLUDED.start_date,
  end_date        = EXCLUDED.end_date,
  target_audience = EXCLUDED.target_audience,
  ad_type         = EXCLUDED.ad_type,
  headline        = EXCLUDED.headline,
  description     = EXCLUDED.description,
  performance_id  = EXCLUDED.performance_id,
  image_id        = EXCLUDED.image_id,
  created_by      = EXCLUDED.created_by,
  created_at      = LEAST(campaigns.created_at, EXCLUDED.created_at),
  updated_by      = EXCLUDED.updated_by,
  updated_at      = COALESCE(EXCLUDED.updated_at, NOW()),
  deleted_at      = EXCLUDED.deleted_at,
  deleted_by      = EXCLUDED.deleted_by
RETURNING
  id, name, brand_id, assignee_id, list_id, status, budget, spent,
  start_date, end_date, target_audience, ad_type, headline, description,
  performance_id, image_id, created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
`
    row := r.DB.QueryRowContext(ctx, q,
        strings.TrimSpace(c.ID),
        strings.TrimSpace(c.Name),
        strings.TrimSpace(c.BrandID),
        strings.TrimSpace(c.AssigneeID),
        strings.TrimSpace(c.ListID),
        string(c.Status),
        c.Budget,
        c.Spent,
        c.StartDate.UTC(),
        c.EndDate.UTC(),
        strings.TrimSpace(c.TargetAudience),
        string(c.AdType),
        strings.TrimSpace(c.Headline),
        strings.TrimSpace(c.Description),
        dbcommon.ToDBText(c.PerformanceID),
        dbcommon.ToDBText(c.ImageID),
        dbcommon.ToDBText(c.CreatedBy),
        c.CreatedAt.UTC(),
        dbcommon.ToDBText(c.UpdatedBy),
        dbcommon.ToDBTime(c.UpdatedAt),
        dbcommon.ToDBTime(c.DeletedAt),
        dbcommon.ToDBText(c.DeletedBy),
    )
    out, err := scanCampaign(row)
    if err != nil {
        return camdom.Campaign{}, err
    }
    return out, nil
}

// ========== Helpers ==========

func scanCampaign(s dbcommon.RowScanner) (camdom.Campaign, error) {
    var (
        idNS, nameNS, brandIDNS, assigneeIDNS, listIDNS        sql.NullString
        statusNS, targetAudienceNS, adTypeNS, headlineNS        sql.NullString
        descNS, perfIDNS, imageIDNS, createdByNS, updatedByNS   sql.NullString
        budgetF, spentF                                         float64
        startDate, endDate, createdAt                           time.Time
        updatedAtNT, deletedAtNT                                sql.NullTime
        deletedByNS                                             sql.NullString
    )

    if err := s.Scan(
        &idNS, &nameNS, &brandIDNS, &assigneeIDNS, &listIDNS, &statusNS, &budgetF, &spentF,
        &startDate, &endDate, &targetAudienceNS, &adTypeNS, &headlineNS, &descNS,
        &perfIDNS, &imageIDNS, &createdByNS, &createdAt, &updatedByNS, &updatedAtNT, &deletedAtNT, &deletedByNS,
    ); err != nil {
        return camdom.Campaign{}, err
    }

    toPtrStr := func(ns sql.NullString) *string {
        if ns.Valid {
            v := strings.TrimSpace(ns.String)
            if v != "" {
                return &v
            }
        }
        return nil
    }
    toPtrTime := func(nt sql.NullTime) *time.Time {
        if nt.Valid {
            t := nt.Time.UTC()
            return &t
        }
        return nil
    }

    return camdom.Campaign{
        ID:             strings.TrimSpace(idNS.String),
        Name:           strings.TrimSpace(nameNS.String),
        BrandID:        strings.TrimSpace(brandIDNS.String),
        AssigneeID:     strings.TrimSpace(assigneeIDNS.String),
        ListID:         strings.TrimSpace(listIDNS.String),
        Status:         camdom.CampaignStatus(strings.TrimSpace(statusNS.String)),
        Budget:         budgetF,
        Spent:          spentF,
        StartDate:      startDate.UTC(),
        EndDate:        endDate.UTC(),
        TargetAudience: strings.TrimSpace(targetAudienceNS.String),
        AdType:         camdom.AdType(strings.TrimSpace(adTypeNS.String)),
        Headline:       strings.TrimSpace(headlineNS.String),
        Description:    strings.TrimSpace(descNS.String),
        PerformanceID:  toPtrStr(perfIDNS),
        ImageID:        toPtrStr(imageIDNS),
        CreatedBy:      toPtrStr(createdByNS),
        CreatedAt:      createdAt.UTC(),
        UpdatedBy:      toPtrStr(updatedByNS),
        UpdatedAt:      toPtrTime(updatedAtNT),
        DeletedAt:      toPtrTime(deletedAtNT),
        DeletedBy:      toPtrStr(deletedByNS),
    }, nil
}

func buildCampaignWhere(f camdom.Filter) ([]string, []any) {
    where := []string{}
    args := []any{}

    // Full-text-ish search
    if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
        // name, description, target_audience, headline
        where = append(where, fmt.Sprintf(
            "(name ILIKE $%d OR description ILIKE $%d OR target_audience ILIKE $%d OR headline ILIKE $%d)",
            len(args)+1, len(args)+1, len(args)+1, len(args)+1,
        ))
        args = append(args, "%"+sq+"%")
    }

    // Brand filters
    if f.BrandID != nil && strings.TrimSpace(*f.BrandID) != "" {
        where = append(where, fmt.Sprintf("brand_id = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.BrandID))
    }
    if len(f.BrandIDs) > 0 {
        ph := make([]string, 0, len(f.BrandIDs))
        for _, v := range f.BrandIDs {
            v = strings.TrimSpace(v)
            if v == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, v)
        }
        if len(ph) > 0 {
            where = append(where, "brand_id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // Assignee filters
    if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
        where = append(where, fmt.Sprintf("assignee_id = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.AssigneeID))
    }
    if len(f.AssigneeIDs) > 0 {
        ph := make([]string, 0, len(f.AssigneeIDs))
        for _, v := range f.AssigneeIDs {
            v = strings.TrimSpace(v)
            if v == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, v)
        }
        if len(ph) > 0 {
            where = append(where, "assignee_id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // List filters
    if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
        where = append(where, fmt.Sprintf("list_id = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.ListID))
    }
    if len(f.ListIDs) > 0 {
        ph := make([]string, 0, len(f.ListIDs))
        for _, v := range f.ListIDs {
            v = strings.TrimSpace(v)
            if v == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, v)
        }
        if len(ph) > 0 {
            where = append(where, "list_id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // Statuses
    if len(f.Statuses) > 0 {
        ph := make([]string, 0, len(f.Statuses))
        for _, v := range f.Statuses {
            s := strings.TrimSpace(string(v))
            if s == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, s)
        }
        if len(ph) > 0 {
            where = append(where, "status IN ("+strings.Join(ph, ",")+")")
        }
    }

    // AdTypes
    if len(f.AdTypes) > 0 {
        ph := make([]string, 0, len(f.AdTypes))
        for _, v := range f.AdTypes {
            s := strings.TrimSpace(string(v))
            if s == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, s)
        }
        if len(ph) > 0 {
            where = append(where, "ad_type IN ("+strings.Join(ph, ",")+")")
        }
    }

    // Numeric ranges
    if f.BudgetMin != nil {
        where = append(where, fmt.Sprintf("budget >= $%d", len(args)+1))
        args = append(args, *f.BudgetMin)
    }
    if f.BudgetMax != nil {
        where = append(where, fmt.Sprintf("budget <= $%d", len(args)+1))
        args = append(args, *f.BudgetMax)
    }
    if f.SpentMin != nil {
        where = append(where, fmt.Sprintf("spent >= $%d", len(args)+1))
        args = append(args, *f.SpentMin)
    }
    if f.SpentMax != nil {
        where = append(where, fmt.Sprintf("spent <= $%d", len(args)+1))
        args = append(args, *f.SpentMax)
    }

    // Date ranges
    if f.StartFrom != nil {
        where = append(where, fmt.Sprintf("start_date >= $%d", len(args)+1))
        args = append(args, f.StartFrom.UTC())
    }
    if f.StartTo != nil {
        where = append(where, fmt.Sprintf("start_date < $%d", len(args)+1))
        args = append(args, f.StartTo.UTC())
    }
    if f.EndFrom != nil {
        where = append(where, fmt.Sprintf("end_date >= $%d", len(args)+1))
        args = append(args, f.EndFrom.UTC())
    }
    if f.EndTo != nil {
        where = append(where, fmt.Sprintf("end_date < $%d", len(args)+1))
        args = append(args, f.EndTo.UTC())
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
        where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at >= $%d)", len(args)+1))
        args = append(args, f.UpdatedFrom.UTC())
    }
    if f.UpdatedTo != nil {
        where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at < $%d)", len(args)+1))
        args = append(args, f.UpdatedTo.UTC())
    }
    if f.DeletedFrom != nil {
        where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at >= $%d)", len(args)+1))
        args = append(args, f.DeletedFrom.UTC())
    }
    if f.DeletedTo != nil {
        where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at < $%d)", len(args)+1))
        args = append(args, f.DeletedTo.UTC())
    }

    // Nullable existence filters
    if f.HasPerformanceID != nil {
        if *f.HasPerformanceID {
            where = append(where, "performance_id IS NOT NULL")
        } else {
            where = append(where, "performance_id IS NULL")
        }
    }
    if f.HasImageID != nil {
        if *f.HasImageID {
            where = append(where, "image_id IS NOT NULL")
        } else {
            where = append(where, "image_id IS NULL")
        }
    }

    // CreatedBy
    if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
        where = append(where, fmt.Sprintf("created_by = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.CreatedBy))
    }

    // Deleted tri-state
    if f.Deleted != nil {
        if *f.Deleted {
            where = append(where, "deleted_at IS NOT NULL")
        } else {
            where = append(where, "deleted_at IS NULL")
        }
    }

    return where, args
}

func buildCampaignOrderBy(sort camdom.Sort) string {
    col := strings.ToLower(string(sort.Column))
    switch col {
    case "id":
        col = "id"
    case "name":
        col = "name"
    case "brandid", "brand_id":
        col = "brand_id"
    case "assigneeid", "assignee_id":
        col = "assignee_id"
    case "listid", "list_id":
        col = "list_id"
    case "status":
        col = "status"
    case "budget":
        col = "budget"
    case "spent":
        col = "spent"
    case "startdate", "start_date":
        col = "start_date"
    case "enddate", "end_date":
        col = "end_date"
    case "targetaudience", "target_audience":
        col = "target_audience"
    case "adtype", "ad_type":
        col = "ad_type"
    case "headline":
        col = "headline"
    case "createdat", "created_at":
        col = "created_at"
    case "updatedat", "updated_at":
        col = "updated_at"
    case "deletedat", "deleted_at":
        col = "deleted_at"
    default:
        return ""
    }

    dir := strings.ToUpper(string(sort.Order))
    if dir != "ASC" && dir != "DESC" {
        dir = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s", col, dir)
}