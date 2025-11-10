// backend/internal/adapters/out/firestore/campaign_repository_fs.go
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

	camdom "narratives/internal/domain/campaign"
)

// CampaignRepositoryFS is a Firestore implementation of camdom.Repository.
type CampaignRepositoryFS struct {
	Client *firestore.Client
}

// Ensure interface implementation at compile time (if the interface exists as expected).
var _ camdom.Repository = (*CampaignRepositoryFS)(nil)

func NewCampaignRepositoryFS(client *firestore.Client) *CampaignRepositoryFS {
	return &CampaignRepositoryFS{Client: client}
}

func (r *CampaignRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("campaigns")
}

// ========== Queries ==========

func (r *CampaignRepositoryFS) List(
	ctx context.Context,
	filter camdom.Filter,
	sort camdom.Sort,
	page camdom.Page,
) (camdom.PageResult[camdom.Campaign], error) {

	// Firestore 制約を避けるため、まずクエリで sort のみ適用し、
	// Filter はアプリ側で絞り込む実装にしています。
	q := r.col().Query
	q = applyCampaignSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []camdom.Campaign
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return camdom.PageResult[camdom.Campaign]{}, err
		}

		c, err := docToCampaign(doc)
		if err != nil {
			return camdom.PageResult[camdom.Campaign]{}, err
		}
		if matchCampaignFilter(c, filter) {
			all = append(all, c)
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

	return camdom.PageResult[camdom.Campaign]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *CampaignRepositoryFS) ListByCursor(
	ctx context.Context,
	filter camdom.Filter,
	_ camdom.Sort,
	cpage camdom.CursorPage,
) (camdom.CursorPageResult[camdom.Campaign], error) {

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// PostgreSQL 実装と同様に id 昇順のカーソルベース。
	q := r.col().OrderBy("id", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	var (
		items  []camdom.Campaign
		lastID string
	)
	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return camdom.CursorPageResult[camdom.Campaign]{}, err
		}

		c, err := docToCampaign(doc)
		if err != nil {
			return camdom.CursorPageResult[camdom.Campaign]{}, err
		}
		if !matchCampaignFilter(c, filter) {
			continue
		}

		// カーソルより後のみ対象
		if skipping {
			if c.ID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, c)
		lastID = c.ID
		if len(items) >= limit+1 {
			break
		}
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

func (r *CampaignRepositoryFS) GetByID(ctx context.Context, id string) (camdom.Campaign, error) {
	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return camdom.Campaign{}, camdom.ErrNotFound
		}
		return camdom.Campaign{}, err
	}
	return docToCampaign(snap)
}

func (r *CampaignRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *CampaignRepositoryFS) Count(ctx context.Context, filter camdom.Filter) (int, error) {
	q := r.col().Query
	// Sort 不要なので applyCampaignSort は呼ばない
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
		c, err := docToCampaign(doc)
		if err != nil {
			return 0, err
		}
		if matchCampaignFilter(c, filter) {
			count++
		}
	}
	return count, nil
}

// ========== Mutations ==========

func (r *CampaignRepositoryFS) Create(ctx context.Context, c camdom.Campaign) (camdom.Campaign, error) {
	now := time.Now().UTC()

	// ID 未指定なら自動採番
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(c.ID) == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(c.ID))
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	// UpdatedAt は nil または指定値を尊重。未指定なら CreatedAt と同じでもよいが、
	// ここでは nil のままにしておき、必要なら上位で設定してもらう。
	data := campaignToDocData(c)
	data["id"] = c.ID

	_, err := docRef.Create(ctx, data)
	if err != nil {
		// Firestore は unique 制約がないため、既存時は AlreadyExists として扱う
		if status.Code(err) == codes.AlreadyExists {
			return camdom.Campaign{}, camdom.ErrConflict
		}
		return camdom.Campaign{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return camdom.Campaign{}, err
	}
	return docToCampaign(snap)
}

func (r *CampaignRepositoryFS) Update(ctx context.Context, id string, patch camdom.CampaignPatch) (camdom.Campaign, error) {
	docRef := r.col().Doc(id)

	var updates []firestore.Update

	if patch.Name != nil {
		updates = append(updates, firestore.Update{Path: "name", Value: strings.TrimSpace(*patch.Name)})
	}
	if patch.BrandID != nil {
		updates = append(updates, firestore.Update{Path: "brandId", Value: strings.TrimSpace(*patch.BrandID)})
	}
	if patch.AssigneeID != nil {
		updates = append(updates, firestore.Update{Path: "assigneeId", Value: strings.TrimSpace(*patch.AssigneeID)})
	}
	if patch.ListID != nil {
		updates = append(updates, firestore.Update{Path: "listId", Value: strings.TrimSpace(*patch.ListID)})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: string(*patch.Status)})
	}
	if patch.Budget != nil {
		updates = append(updates, firestore.Update{Path: "budget", Value: *patch.Budget})
	}
	if patch.Spent != nil {
		updates = append(updates, firestore.Update{Path: "spent", Value: *patch.Spent})
	}
	if patch.StartDate != nil {
		updates = append(updates, firestore.Update{Path: "startDate", Value: patch.StartDate.UTC()})
	}
	if patch.EndDate != nil {
		updates = append(updates, firestore.Update{Path: "endDate", Value: patch.EndDate.UTC()})
	}
	if patch.TargetAudience != nil {
		updates = append(updates, firestore.Update{Path: "targetAudience", Value: strings.TrimSpace(*patch.TargetAudience)})
	}
	if patch.AdType != nil {
		updates = append(updates, firestore.Update{Path: "adType", Value: string(*patch.AdType)})
	}
	if patch.Headline != nil {
		updates = append(updates, firestore.Update{Path: "headline", Value: strings.TrimSpace(*patch.Headline)})
	}
	if patch.Description != nil {
		updates = append(updates, firestore.Update{Path: "description", Value: strings.TrimSpace(*patch.Description)})
	}
	if patch.PerformanceID != nil {
		if strings.TrimSpace(*patch.PerformanceID) == "" {
			updates = append(updates, firestore.Update{Path: "performanceId", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "performanceId", Value: strings.TrimSpace(*patch.PerformanceID)})
		}
	}
	if patch.ImageID != nil {
		if strings.TrimSpace(*patch.ImageID) == "" {
			updates = append(updates, firestore.Update{Path: "imageId", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "imageId", Value: strings.TrimSpace(*patch.ImageID)})
		}
	}
	if patch.UpdatedBy != nil {
		if strings.TrimSpace(*patch.UpdatedBy) == "" {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: strings.TrimSpace(*patch.UpdatedBy)})
		}
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: patch.DeletedAt.UTC()})
		}
	}
	if patch.DeletedBy != nil {
		if strings.TrimSpace(*patch.DeletedBy) == "" {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: strings.TrimSpace(*patch.DeletedBy)})
		}
	}

	// updatedAt: 明示指定があればそれ、なければ現在時刻
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
		}
	} else if len(updates) > 0 {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	if len(updates) == 0 {
		// no-op: 現在値を返す
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return camdom.Campaign{}, camdom.ErrNotFound
		}
		return camdom.Campaign{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *CampaignRepositoryFS) Delete(ctx context.Context, id string) error {
	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return camdom.Campaign.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *CampaignRepositoryFS) Save(ctx context.Context, c camdom.Campaign, _ *camdom.SaveOptions) (camdom.Campaign, error) {
	now := time.Now().UTC()

	// ID 未指定なら自動採番
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(c.ID) == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(c.ID))
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt == nil {
		ut := now
		c.UpdatedAt = &ut
	}

	data := campaignToDocData(c)
	data["id"] = c.ID

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return camdom.Campaign{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return camdom.Campaign{}, err
	}
	return docToCampaign(snap)
}

// ========== Mapping Helpers ==========

func campaignToDocData(c camdom.Campaign) map[string]any {
	m := map[string]any{
		"id":             strings.TrimSpace(c.ID),
		"name":           strings.TrimSpace(c.Name),
		"brandId":        strings.TrimSpace(c.BrandID),
		"assigneeId":     strings.TrimSpace(c.AssigneeID),
		"listId":         strings.TrimSpace(c.ListID),
		"status":         string(c.Status),
		"budget":         c.Budget,
		"spent":          c.Spent,
		"startDate":      c.StartDate.UTC(),
		"endDate":        c.EndDate.UTC(),
		"targetAudience": strings.TrimSpace(c.TargetAudience),
		"adType":         string(c.AdType),
		"headline":       strings.TrimSpace(c.Headline),
		"description":    strings.TrimSpace(c.Description),
		"createdAt":      c.CreatedAt.UTC(),
	}

	if v := strings.TrimSpace(ptrOrEmpty(c.PerformanceID)); v != "" {
		m["performanceId"] = v
	}
	if v := strings.TrimSpace(ptrOrEmpty(c.ImageID)); v != "" {
		m["imageId"] = v
	}
	if v := strings.TrimSpace(ptrOrEmpty(c.CreatedBy)); v != "" {
		m["createdBy"] = v
	}
	if v := strings.TrimSpace(ptrOrEmpty(c.UpdatedBy)); v != "" {
		m["updatedBy"] = v
	}
	if c.UpdatedAt != nil && !c.UpdatedAt.IsZero() {
		m["updatedAt"] = c.UpdatedAt.UTC()
	}
	if c.DeletedAt != nil && !c.DeletedAt.IsZero() {
		m["deletedAt"] = c.DeletedAt.UTC()
	}
	if v := strings.TrimSpace(ptrOrEmpty(c.DeletedBy)); v != "" {
		m["deletedBy"] = v
	}

	return m
}

func docToCampaign(doc *firestore.DocumentSnapshot) (camdom.Campaign, error) {
	data := doc.Data()
	if data == nil {
		return camdom.Campaign{}, fmt.Errorf("empty campaign document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getPtrStr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				return &s
			}
		}
		return nil
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
	getPtrTime := func(primary, legacy string) *time.Time {
		if t, ok := getTime(primary, legacy); ok && !t.IsZero() {
			return &t
		}
		return nil
	}
	getFloat := func(key string) float64 {
		if v, ok := data[key].(float64); ok {
			return v
		}
		return 0
	}

	var c camdom.Campaign

	// ID: ドキュメントID優先、フィールドにあればそれも許容
	c.ID = getStr("id")
	if c.ID == "" {
		c.ID = doc.Ref.ID
	}

	c.Name = getStr("name")
	c.BrandID = getStr("brandId")
	c.AssigneeID = getStr("assigneeId")
	c.ListID = getStr("listId")
	c.Status = camdom.CampaignStatus(getStr("status"))
	c.Budget = getFloat("budget")
	c.Spent = getFloat("spent")

	if t, ok := getTime("startDate", "start_date"); ok {
		c.StartDate = t
	}
	if t, ok := getTime("endDate", "end_date"); ok {
		c.EndDate = t
	}

	c.TargetAudience = getStr("targetAudience")
	c.AdType = camdom.AdType(getStr("adType"))
	c.Headline = getStr("headline")
	c.Description = getStr("description")

	c.PerformanceID = getPtrStr("performanceId")
	c.ImageID = getPtrStr("imageId")
	c.CreatedBy = getPtrStr("createdBy")

	if t, ok := getTime("createdAt", "created_at"); ok {
		c.CreatedAt = t
	}
	c.UpdatedBy = getPtrStr("updatedBy")
	c.UpdatedAt = getPtrTime("updatedAt", "updated_at")
	c.DeletedAt = getPtrTime("deletedAt", "deleted_at")
	c.DeletedBy = getPtrStr("deletedBy")

	return c, nil
}

// ========== Filter / Sort Helpers ==========

func applyCampaignSort(q firestore.Query, sort camdom.Sort) firestore.Query {
	col, dir := mapCampaignSort(sort)
	if col == "" {
		// デフォルト: createdAt DESC, id DESC
		return q.OrderBy("createdAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapCampaignSort(sort camdom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	var field string

	switch col {
	case "id":
		field = "id"
	case "name":
		field = "name"
	case "brandid", "brand_id":
		field = "brandId"
	case "assigneeid", "assignee_id":
		field = "assigneeId"
	case "listid", "list_id":
		field = "listId"
	case "status":
		field = "status"
	case "budget":
		field = "budget"
	case "spent":
		field = "spent"
	case "startdate", "start_date":
		field = "startDate"
	case "enddate", "end_date":
		field = "endDate"
	case "targetaudience", "target_audience":
		field = "targetAudience"
	case "adtype", "ad_type":
		field = "adType"
	case "headline":
		field = "headline"
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "deletedat", "deleted_at":
		field = "deletedAt"
	default:
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}

// matchCampaignFilter は PostgreSQL 実装の buildCampaignWhere 相当の条件を
// Firestore 取得後に適用する（簡易だが挙動を揃えるための実装）。
func matchCampaignFilter(c camdom.Campaign, f camdom.Filter) bool {
	// SearchQuery: name, description, target_audience, headline
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		haystack := strings.ToLower(
			c.Name + " " + c.Description + " " + c.TargetAudience + " " + c.Headline,
		)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// Brand filters
	if f.BrandID != nil && strings.TrimSpace(*f.BrandID) != "" {
		if c.BrandID != strings.TrimSpace(*f.BrandID) {
			return false
		}
	}
	if len(f.BrandIDs) > 0 {
		if !containsString(f.BrandIDs, c.BrandID) {
			return false
		}
	}

	// Assignee filters
	if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
		if c.AssigneeID != strings.TrimSpace(*f.AssigneeID) {
			return false
		}
	}
	if len(f.AssigneeIDs) > 0 {
		if !containsString(f.AssigneeIDs, c.AssigneeID) {
			return false
		}
	}

	// List filters
	if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
		if c.ListID != strings.TrimSpace(*f.ListID) {
			return false
		}
	}
	if len(f.ListIDs) > 0 {
		if !containsString(f.ListIDs, c.ListID) {
			return false
		}
	}

	// Statuses
	if len(f.Statuses) > 0 {
		ok := false
		for _, s := range f.Statuses {
			if c.Status == s {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// AdTypes
	if len(f.AdTypes) > 0 {
		ok := false
		for _, t := range f.AdTypes {
			if c.AdType == t {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Numeric ranges
	if f.BudgetMin != nil && c.Budget < *f.BudgetMin {
		return false
	}
	if f.BudgetMax != nil && c.Budget > *f.BudgetMax {
		return false
	}
	if f.SpentMin != nil && c.Spent < *f.SpentMin {
		return false
	}
	if f.SpentMax != nil && c.Spent > *f.SpentMax {
		return false
	}

	// Date ranges
	if f.StartFrom != nil && c.StartDate.Before(f.StartFrom.UTC()) {
		return false
	}
	if f.StartTo != nil && !c.StartDate.Before(f.StartTo.UTC()) {
		return false
	}
	if f.EndFrom != nil && c.EndDate.Before(f.EndFrom.UTC()) {
		return false
	}
	if f.EndTo != nil && !c.EndDate.Before(f.EndTo.UTC()) {
		return false
	}
	if f.CreatedFrom != nil && c.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !c.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && (c.UpdatedAt == nil || c.UpdatedAt.Before(f.UpdatedFrom.UTC())) {
		return false
	}
	if f.UpdatedTo != nil && (c.UpdatedAt == nil || !c.UpdatedAt.Before(f.UpdatedTo.UTC())) {
		return false
	}
	if f.DeletedFrom != nil {
		if c.DeletedAt == nil || c.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if c.DeletedAt == nil || !c.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Nullable existence filters
	if f.HasPerformanceID != nil {
		has := c.PerformanceID != nil && strings.TrimSpace(*c.PerformanceID) != ""
		if *f.HasPerformanceID != has {
			return false
		}
	}
	if f.HasImageID != nil {
		has := c.ImageID != nil && strings.TrimSpace(*c.ImageID) != ""
		if *f.HasImageID != has {
			return false
		}
	}

	// CreatedBy
	if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
		if c.CreatedBy == nil || strings.TrimSpace(*c.CreatedBy) != strings.TrimSpace(*f.CreatedBy) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		isDeleted := c.DeletedAt != nil
		if *f.Deleted != isDeleted {
			return false
		}
	}

	return true
}

// ========== Small Utilities ==========

func ptrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func containsString(list []string, v string) bool {
	v = strings.TrimSpace(v)
	for _, s := range list {
		if strings.TrimSpace(s) == v {
			return true
		}
	}
	return false
}
