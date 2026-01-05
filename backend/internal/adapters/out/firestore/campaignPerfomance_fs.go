// backend/internal/adapters/out/firestore/campaignPerformance_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cperfdom "narratives/internal/domain/campaignPerformance"
)

// CampaignPerformanceRepositoryFS implements campaignPerformance.Repository using Firestore.
type CampaignPerformanceRepositoryFS struct {
	Client *firestore.Client
}

func NewCampaignPerformanceRepositoryFS(client *firestore.Client) *CampaignPerformanceRepositoryFS {
	return &CampaignPerformanceRepositoryFS{Client: client}
}

func (r *CampaignPerformanceRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("campaign_performances")
}

// Compile-time check
var _ cperfdom.Repository = (*CampaignPerformanceRepositoryFS)(nil)

// =======================
// Queries
// =======================

func (r *CampaignPerformanceRepositoryFS) List(
	ctx context.Context,
	filter cperfdom.Filter,
	sort cperfdom.Sort,
	page cperfdom.Page,
) (cperfdom.PageResult[cperfdom.CampaignPerformance], error) {

	q := r.col().Query
	q = applyCampaignPerformanceSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []cperfdom.CampaignPerformance
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
		}

		cp, err := docToCampaignPerformance(doc)
		if err != nil {
			return cperfdom.PageResult[cperfdom.CampaignPerformance]{}, err
		}
		if matchCampaignPerformanceFilter(cp, filter) {
			all = append(all, cp)
		}
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
	total := len(all)

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	items := all[offset:end]

	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return cperfdom.PageResult[cperfdom.CampaignPerformance]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *CampaignPerformanceRepositoryFS) ListByCursor(
	ctx context.Context,
	filter cperfdom.Filter,
	sort cperfdom.Sort,
	cpage cperfdom.CursorPage,
) (cperfdom.CursorPageResult[cperfdom.CampaignPerformance], error) {

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Keyset pagination: default order by id ASC (plus sort if needed later)
	q := r.col().OrderBy("id", firestore.Asc)
	q = applyCampaignPerformanceSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items  []cperfdom.CampaignPerformance
		lastID string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{}, err
		}

		cp, err := docToCampaignPerformance(doc)
		if err != nil {
			return cperfdom.CursorPageResult[cperfdom.CampaignPerformance]{}, err
		}
		if !matchCampaignPerformanceFilter(cp, filter) {
			continue
		}

		if skipping {
			if cp.ID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, cp)
		lastID = cp.ID

		if len(items) >= limit+1 {
			break
		}
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

func (r *CampaignPerformanceRepositoryFS) GetByID(ctx context.Context, id string) (cperfdom.CampaignPerformance, error) {
	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrNotFound
		}
		return cperfdom.CampaignPerformance{}, err
	}
	return docToCampaignPerformance(snap)
}

// GetByCampaignID implements the interface method:
// GetByCampaignID(ctx, campaignID string, sort Sort, page Page) (PageResult[CampaignPerformance], error)
func (r *CampaignPerformanceRepositoryFS) GetByCampaignID(
	ctx context.Context,
	campaignID string,
	sort cperfdom.Sort,
	page cperfdom.Page,
) (cperfdom.PageResult[cperfdom.CampaignPerformance], error) {

	cid := strings.TrimSpace(campaignID)
	if cid == "" {
		// 空なら空ページを返す
		return cperfdom.PageResult[cperfdom.CampaignPerformance]{
			Items:      []cperfdom.CampaignPerformance{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       page.Number,
			PerPage:    page.PerPage,
		}, nil
	}

	// Filter に詰めて List を利用して挙動を統一
	filter := cperfdom.Filter{
		CampaignID: &cid,
	}
	return r.List(ctx, filter, sort, page)
}

func (r *CampaignPerformanceRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *CampaignPerformanceRepositoryFS) Count(ctx context.Context, filter cperfdom.Filter) (int, error) {
	q := r.col().Query

	it := q.Documents(ctx)
	defer it.Stop()

	count := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		cp, err := docToCampaignPerformance(doc)
		if err != nil {
			return 0, err
		}
		if matchCampaignPerformanceFilter(cp, filter) {
			count++
		}
	}
	return count, nil
}

// =======================
// Mutations
// =======================

func (r *CampaignPerformanceRepositoryFS) Create(
	ctx context.Context,
	cp cperfdom.CampaignPerformance,
) (cperfdom.CampaignPerformance, error) {

	now := time.Now().UTC()

	// Assign ID if empty.
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(cp.ID) == "" {
		docRef = r.col().NewDoc()
		cp.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(cp.ID))
	}

	if cp.LastUpdatedAt.IsZero() {
		cp.LastUpdatedAt = now
	}

	data := campaignPerformanceToDocData(cp)
	data["id"] = cp.ID

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrConflict
		}
		return cperfdom.CampaignPerformance{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return cperfdom.CampaignPerformance{}, err
	}
	return docToCampaignPerformance(snap)
}

func (r *CampaignPerformanceRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch cperfdom.CampaignPerformancePatch,
) (cperfdom.CampaignPerformance, error) {

	docRef := r.col().Doc(id)

	var updates []firestore.Update

	if patch.Impressions != nil {
		updates = append(updates, firestore.Update{Path: "impressions", Value: *patch.Impressions})
	}
	if patch.Clicks != nil {
		updates = append(updates, firestore.Update{Path: "clicks", Value: *patch.Clicks})
	}
	if patch.Conversions != nil {
		updates = append(updates, firestore.Update{Path: "conversions", Value: *patch.Conversions})
	}
	if patch.Purchases != nil {
		updates = append(updates, firestore.Update{Path: "purchases", Value: *patch.Purchases})
	}

	// lastUpdatedAt: explicit or NOW() if any other field updated
	if patch.LastUpdatedAt != nil {
		if patch.LastUpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "lastUpdatedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "lastUpdatedAt", Value: patch.LastUpdatedAt.UTC()})
		}
	} else if len(updates) > 0 {
		updates = append(updates, firestore.Update{Path: "lastUpdatedAt", Value: time.Now().UTC()})
	}

	if len(updates) == 0 {
		// no-op update, return current
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return cperfdom.CampaignPerformance{}, cperfdom.ErrNotFound
		}
		return cperfdom.CampaignPerformance{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *CampaignPerformanceRepositoryFS) Delete(ctx context.Context, id string) error {
	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return cperfdom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *CampaignPerformanceRepositoryFS) Save(
	ctx context.Context,
	cp cperfdom.CampaignPerformance,
	_ *cperfdom.SaveOptions,
) (cperfdom.CampaignPerformance, error) {

	now := time.Now().UTC()

	// Assign ID if empty.
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(cp.ID) == "" {
		docRef = r.col().NewDoc()
		cp.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(cp.ID))
	}

	if cp.LastUpdatedAt.IsZero() {
		cp.LastUpdatedAt = now
	}

	data := campaignPerformanceToDocData(cp)
	data["id"] = cp.ID

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return cperfdom.CampaignPerformance{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return cperfdom.CampaignPerformance{}, err
	}
	return docToCampaignPerformance(snap)
}

// =======================
// Helpers
// =======================

func campaignPerformanceToDocData(cp cperfdom.CampaignPerformance) map[string]any {
	return map[string]any{
		"id":            strings.TrimSpace(cp.ID),
		"campaignId":    strings.TrimSpace(cp.CampaignID),
		"impressions":   cp.Impressions,
		"clicks":        cp.Clicks,
		"conversions":   cp.Conversions,
		"purchases":     cp.Purchases,
		"lastUpdatedAt": cp.LastUpdatedAt.UTC(),
	}
}

func docToCampaignPerformance(doc *firestore.DocumentSnapshot) (cperfdom.CampaignPerformance, error) {
	data := doc.Data()
	if data == nil {
		return cperfdom.CampaignPerformance{}, fmt.Errorf("empty campaign_performance document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getInt := func(key string) int {
		if v, ok := data[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
		return 0
	}
	getTime := func(primary, legacy string) (time.Time, bool) {
		if v, ok := data[primary].(time.Time); ok {
			return v.UTC(), true
		}
		if legacy != "" {
			if v, ok := data[legacy].(time.Time); ok {
				return v.UTC(), true
			}
		}
		return time.Time{}, false
	}

	var cp cperfdom.CampaignPerformance

	cp.ID = getStr("id")
	if cp.ID == "" {
		cp.ID = doc.Ref.ID
	}
	cp.CampaignID = getStr("campaignId")
	cp.Impressions = getInt("impressions")
	cp.Clicks = getInt("clicks")
	cp.Conversions = getInt("conversions")
	cp.Purchases = getInt("purchases")

	if t, ok := getTime("lastUpdatedAt", "last_updated_at"); ok {
		cp.LastUpdatedAt = t
	}

	return cp, nil
}

// matchCampaignPerformanceFilter applies the Filter conditions in-memory.
func matchCampaignPerformanceFilter(cp cperfdom.CampaignPerformance, f cperfdom.Filter) bool {
	// CampaignID exact
	if f.CampaignID != nil && strings.TrimSpace(*f.CampaignID) != "" {
		if cp.CampaignID != strings.TrimSpace(*f.CampaignID) {
			return false
		}
	}

	// CampaignIDs IN (...)
	if len(f.CampaignIDs) > 0 {
		if !containsString(f.CampaignIDs, cp.CampaignID) {
			return false
		}
	}

	// Number ranges
	if f.ImpressionsMin != nil && cp.Impressions < *f.ImpressionsMin {
		return false
	}
	if f.ImpressionsMax != nil && cp.Impressions > *f.ImpressionsMax {
		return false
	}
	if f.ClicksMin != nil && cp.Clicks < *f.ClicksMin {
		return false
	}
	if f.ClicksMax != nil && cp.Clicks > *f.ClicksMax {
		return false
	}
	if f.ConversionsMin != nil && cp.Conversions < *f.ConversionsMin {
		return false
	}
	if f.ConversionsMax != nil && cp.Conversions > *f.ConversionsMax {
		return false
	}
	if f.PurchasesMin != nil && cp.Purchases < *f.PurchasesMin {
		return false
	}
	if f.PurchasesMax != nil && cp.Purchases > *f.PurchasesMax {
		return false
	}

	// Date ranges (lastUpdatedAt)
	if f.LastUpdatedFrom != nil && cp.LastUpdatedAt.Before(f.LastUpdatedFrom.UTC()) {
		return false
	}
	if f.LastUpdatedTo != nil && !cp.LastUpdatedAt.Before(f.LastUpdatedTo.UTC()) {
		return false
	}

	return true
}

func applyCampaignPerformanceSort(q firestore.Query, sort cperfdom.Sort) firestore.Query {
	col, dir := mapCampaignPerformanceSort(sort)
	if col == "" {
		// default: lastUpdatedAt DESC, id DESC
		return q.OrderBy("lastUpdatedAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapCampaignPerformanceSort(sort cperfdom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	var field string

	switch col {
	case "id":
		field = "id"
	case "campaignid", "campaign_id":
		field = "campaignId"
	case "impressions":
		field = "impressions"
	case "clicks":
		field = "clicks"
	case "conversions":
		field = "conversions"
	case "purchases":
		field = "purchases"
	case "lastupdatedat", "last_updated_at":
		field = "lastUpdatedAt"
	default:
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}
