package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	cperfdom "narratives/internal/domain/campaignPerformance"
)

// CampaignPerformanceRepositoryPG implements campaignPerformance.Repository using PostgreSQL.
type CampaignPerformanceRepositoryPG struct {
	DB *sql.DB
}

func NewCampaignPerformanceRepositoryPG(db *sql.DB) *CampaignPerformanceRepositoryPG {
	return &CampaignPerformanceRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

func (r *CampaignPerformanceRepositoryPG) List(ctx context.Context, filter cperfdom.Filter, sort cperfdom.Sort, page cperfdom.Page) (cperfdom.PageResult[cperfdom.CampaignPerformance], error) {
	where, args := buildCampaignPerformanceWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildCampaignPerformanceOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY last_updated_at DESC, id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM campaign_performances %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
FROM campaign_performances
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
	}
	defer rows.Close()

	var items []cperfdom.CampaignPerformance
	for rows.Next() {
		item, err := scanCampaignPerformance(rows)
		if err != nil {
			return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return cperfdom.PageResult[cperfdom.CampaignPerformance]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *CampaignPerformanceRepositoryPG) ListByCursor(ctx context.Context, filter cperfdom.Filter, _ cperfdom.Sort, cpage cperfdom.CursorPage) (cperfdom.CursorPageResult[cperfdom.CampaignPerformance], error) {
	where, args := buildCampaignPerformanceWhere(filter)

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
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
FROM campaign_performances
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{}, err
	}
	defer rows.Close()

	var items []cperfdom.CampaignPerformance
	var lastID string
	for rows.Next() {
		item, err := scanCampaignPerformance(rows)
		if err != nil {
			return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{}, err
		}
		items = append(items, item)
		lastID = item.ID
	}
	if err := rows.Err(); err != nil {
		return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *CampaignPerformanceRepositoryPG) GetByID(ctx context.Context, id string) (cperfdom.CampaignPerformance, error) {
	const q = `
SELECT
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
FROM campaign_performances
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	item, err := scanCampaignPerformance(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrNotFound
		}
		return cperfdom.CampaignPerformance{}, err
	}
	return item, nil
}

// ★ New: thin adapter for usecase.CampaignPerformanceLister
// Matches the interface:
//
//	ListByCampaignID(ctx context.Context, campaignID string) ([]cperfdom.CampaignPerformance, error)
//
// Internally we just reuse List() with a default sort/page and unwrap .Items.
func (r *CampaignPerformanceRepositoryPG) ListByCampaignID(
	ctx context.Context,
	campaignID string,
) ([]cperfdom.CampaignPerformance, error) {

	filter := cperfdom.Filter{
		CampaignID: &campaignID,
	}

	// We'll use a stable sort (latest first) and a "large enough" page size.
	sort := cperfdom.Sort{
		Column: "last_updated_at",
		Order:  "DESC",
	}
	page := cperfdom.Page{
		Number:  1,
		PerPage: 500, // arbitrary large page to "get them all" for the usecase
	}

	res, err := r.List(ctx, filter, sort, page)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (r *CampaignPerformanceRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM campaign_performances WHERE id = $1`
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

func (r *CampaignPerformanceRepositoryPG) Count(ctx context.Context, filter cperfdom.Filter) (int, error) {
	where, args := buildCampaignPerformanceWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaign_performances `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *CampaignPerformanceRepositoryPG) Create(ctx context.Context, cp cperfdom.CampaignPerformance) (cperfdom.CampaignPerformance, error) {
	const q = `
INSERT INTO campaign_performances (
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
RETURNING
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(cp.ID),
		strings.TrimSpace(cp.CampaignID),
		cp.Impressions,
		cp.Clicks,
		cp.Conversions,
		cp.Purchases,
		cp.LastUpdatedAt.UTC(),
	)
	out, err := scanCampaignPerformance(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrConflict
		}
		return cperfdom.CampaignPerformance{}, err
	}
	return out, nil
}

func (r *CampaignPerformanceRepositoryPG) Update(ctx context.Context, id string, patch cperfdom.CampaignPerformancePatch) (cperfdom.CampaignPerformance, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.Impressions != nil {
		sets = append(sets, fmt.Sprintf("impressions = $%d", i))
		args = append(args, *patch.Impressions)
		i++
	}
	if patch.Clicks != nil {
		sets = append(sets, fmt.Sprintf("clicks = $%d", i))
		args = append(args, *patch.Clicks)
		i++
	}
	if patch.Conversions != nil {
		sets = append(sets, fmt.Sprintf("conversions = $%d", i))
		args = append(args, *patch.Conversions)
		i++
	}
	if patch.Purchases != nil {
		sets = append(sets, fmt.Sprintf("purchases = $%d", i))
		args = append(args, *patch.Purchases)
		i++
	}
	// last_updated_at: explicit or NOW()
	if patch.LastUpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("last_updated_at = $%d", i))
		args = append(args, patch.LastUpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("last_updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
UPDATE campaign_performances
SET %s
WHERE id = $%d
RETURNING
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanCampaignPerformance(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrNotFound
		}
		return cperfdom.CampaignPerformance{}, err
	}
	return out, nil
}

func (r *CampaignPerformanceRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM campaign_performances WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return cperfdom.ErrNotFound
	}
	return nil
}

func (r *CampaignPerformanceRepositoryPG) Save(ctx context.Context, cp cperfdom.CampaignPerformance, _ *cperfdom.SaveOptions) (cperfdom.CampaignPerformance, error) {
	const q = `
INSERT INTO campaign_performances (
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
ON CONFLICT (id) DO UPDATE SET
  campaign_id     = EXCLUDED.campaign_id,
  impressions     = EXCLUDED.impressions,
  clicks          = EXCLUDED.clicks,
  conversions     = EXCLUDED.conversions,
  purchases       = EXCLUDED.purchases,
  last_updated_at = EXCLUDED.last_updated_at
RETURNING
  id, campaign_id, impressions, clicks, conversions, purchases, last_updated_at
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(cp.ID),
		strings.TrimSpace(cp.CampaignID),
		cp.Impressions,
		cp.Clicks,
		cp.Conversions,
		cp.Purchases,
		cp.LastUpdatedAt.UTC(),
	)
	out, err := scanCampaignPerformance(row)
	if err != nil {
		return cperfdom.CampaignPerformance{}, err
	}
	return out, nil
}

// =======================
// Helpers
// =======================

func scanCampaignPerformance(s dbcommon.RowScanner) (cperfdom.CampaignPerformance, error) {
	var (
		idNS, campaignIDNS                          sql.NullString
		impressions, clicks, conversions, purchases int
		lastUpdatedAt                               time.Time
	)
	if err := s.Scan(
		&idNS, &campaignIDNS, &impressions, &clicks, &conversions, &purchases, &lastUpdatedAt,
	); err != nil {
		return cperfdom.CampaignPerformance{}, err
	}
	return cperfdom.CampaignPerformance{
		ID:            strings.TrimSpace(idNS.String),
		CampaignID:    strings.TrimSpace(campaignIDNS.String),
		Impressions:   impressions,
		Clicks:        clicks,
		Conversions:   conversions,
		Purchases:     purchases,
		LastUpdatedAt: lastUpdatedAt.UTC(),
	}, nil
}

func buildCampaignPerformanceWhere(f cperfdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// CampaignID exact
	if f.CampaignID != nil && strings.TrimSpace(*f.CampaignID) != "" {
		where = append(where, fmt.Sprintf("campaign_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.CampaignID))
	}

	// CampaignIDs IN (...)
	if len(f.CampaignIDs) > 0 {
		ph := make([]string, 0, len(f.CampaignIDs))
		for _, v := range f.CampaignIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "campaign_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Number ranges
	if f.ImpressionsMin != nil {
		where = append(where, fmt.Sprintf("impressions >= $%d", len(args)+1))
		args = append(args, *f.ImpressionsMin)
	}
	if f.ImpressionsMax != nil {
		where = append(where, fmt.Sprintf("impressions <= $%d", len(args)+1))
		args = append(args, *f.ImpressionsMax)
	}
	if f.ClicksMin != nil {
		where = append(where, fmt.Sprintf("clicks >= $%d", len(args)+1))
		args = append(args, *f.ClicksMin)
	}
	if f.ClicksMax != nil {
		where = append(where, fmt.Sprintf("clicks <= $%d", len(args)+1))
		args = append(args, *f.ClicksMax)
	}
	if f.ConversionsMin != nil {
		where = append(where, fmt.Sprintf("conversions >= $%d", len(args)+1))
		args = append(args, *f.ConversionsMin)
	}
	if f.ConversionsMax != nil {
		where = append(where, fmt.Sprintf("conversions <= $%d", len(args)+1))
		args = append(args, *f.ConversionsMax)
	}
	if f.PurchasesMin != nil {
		where = append(where, fmt.Sprintf("purchases >= $%d", len(args)+1))
		args = append(args, *f.PurchasesMin)
	}
	if f.PurchasesMax != nil {
		where = append(where, fmt.Sprintf("purchases <= $%d", len(args)+1))
		args = append(args, *f.PurchasesMax)
	}

	// Date ranges (last_updated_at)
	if f.LastUpdatedFrom != nil {
		where = append(where, fmt.Sprintf("last_updated_at >= $%d", len(args)+1))
		args = append(args, f.LastUpdatedFrom.UTC())
	}
	if f.LastUpdatedTo != nil {
		where = append(where, fmt.Sprintf("last_updated_at < $%d", len(args)+1))
		args = append(args, f.LastUpdatedTo.UTC())
	}

	return where, args
}

func buildCampaignPerformanceOrderBy(sort cperfdom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "id"
	case "campaignid", "campaign_id":
		col = "campaign_id"
	case "impressions":
		col = "impressions"
	case "clicks":
		col = "clicks"
	case "conversions":
		col = "conversions"
	case "purchases":
		col = "purchases"
	case "lastupdatedat", "last_updated_at":
		col = "last_updated_at"
	default:
		return ""
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
