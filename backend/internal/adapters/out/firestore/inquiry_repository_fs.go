// backend/internal/adapters/out/firestore/inquiry_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	idom "narratives/internal/domain/inquiry"
)

// InquiryRepositoryFS implements inquiry.Repository using Firestore.
type InquiryRepositoryFS struct {
	Client *firestore.Client
}

func NewInquiryRepositoryFS(client *firestore.Client) *InquiryRepositoryFS {
	return &InquiryRepositoryFS{Client: client}
}

func (r *InquiryRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inquiries")
}

// Compile-time check
var _ idom.Repository = (*InquiryRepositoryFS)(nil)

// =======================
// Queries
// =======================

func (r *InquiryRepositoryFS) List(
	ctx context.Context,
	filter idom.Filter,
	sort idom.Sort,
	page idom.Page,
) (idom.PageResult[idom.Inquiry], error) {
	if r.Client == nil {
		return idom.PageResult[idom.Inquiry]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)

	q := r.col().Query
	q = applyInquirySort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []idom.Inquiry
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return idom.PageResult[idom.Inquiry]{}, err
		}
		in, err := docToInquiry(doc)
		if err != nil {
			return idom.PageResult[idom.Inquiry]{}, err
		}
		if matchInquiryFilter(in, filter) {
			all = append(all, in)
		}
	}

	total := len(all)
	if total == 0 {
		return idom.PageResult[idom.Inquiry]{
			Items:      []idom.Inquiry{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	// apply offset/limit in-memory
	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := fscommon.ComputeTotalPages(total, perPage)

	return idom.PageResult[idom.Inquiry]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *InquiryRepositoryFS) ListByCursor(
	ctx context.Context,
	filter idom.Filter,
	_ idom.Sort,
	cpage idom.CursorPage,
) (idom.CursorPageResult[idom.Inquiry], error) {
	if r.Client == nil {
		return idom.CursorPageResult[idom.Inquiry]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Cursor pagination by id ASC (string compare)
	q := r.col().OrderBy("id", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items  []idom.Inquiry
		lastID string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return idom.CursorPageResult[idom.Inquiry]{}, err
		}

		in, err := docToInquiry(doc)
		if err != nil {
			return idom.CursorPageResult[idom.Inquiry]{}, err
		}
		if !matchInquiryFilter(in, filter) {
			continue
		}

		if skipping {
			if in.ID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, in)
		lastID = in.ID
		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return idom.CursorPageResult[idom.Inquiry]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *InquiryRepositoryFS) GetByID(ctx context.Context, id string) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return idom.Inquiry{}, idom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return idom.Inquiry{}, idom.ErrNotFound
		}
		return idom.Inquiry{}, err
	}

	return docToInquiry(snap)
}

func (r *InquiryRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *InquiryRepositoryFS) Count(ctx context.Context, filter idom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.col().Query
	// To avoid complex composite indexes, fetch and filter in-memory
	it := q.Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		in, err := docToInquiry(doc)
		if err != nil {
			return 0, err
		}
		if matchInquiryFilter(in, filter) {
			total++
		}
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *InquiryRepositoryFS) Create(ctx context.Context, inq idom.Inquiry) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	// ID: if empty, use auto ID
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(inq.ID) == "" {
		docRef = r.col().NewDoc()
		inq.ID = docRef.ID
	} else {
		inq.ID = strings.TrimSpace(inq.ID)
		docRef = r.col().Doc(inq.ID)
	}

	if inq.CreatedAt.IsZero() {
		inq.CreatedAt = now
	}
	if inq.UpdatedAt.IsZero() {
		inq.UpdatedAt = now
	}

	data := inquiryToDocData(inq)
	data["id"] = inq.ID

	_, err := docRef.Create(ctx, data)
	if err != nil {
		// Firestore no unique constraint; treat AlreadyExists as conflict
		if status.Code(err) == codes.AlreadyExists {
			return idom.Inquiry{}, idom.ErrConflict
		}
		return idom.Inquiry{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return idom.Inquiry{}, err
	}
	return docToInquiry(snap)
}

func (r *InquiryRepositoryFS) Update(ctx context.Context, id string, patch idom.InquiryPatch) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return idom.Inquiry{}, idom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	var updates []firestore.Update

	if patch.Subject != nil {
		updates = append(updates, firestore.Update{Path: "subject", Value: strings.TrimSpace(*patch.Subject)})
	}
	if patch.Content != nil {
		updates = append(updates, firestore.Update{Path: "content", Value: strings.TrimSpace(*patch.Content)})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: string(*patch.Status)})
	}
	if patch.InquiryType != nil {
		updates = append(updates, firestore.Update{Path: "inquiryType", Value: string(*patch.InquiryType)})
	}
	if patch.ProductBlueprintID != nil {
		if v := fscommon.TrimPtr(patch.ProductBlueprintID); v != nil {
			updates = append(updates, firestore.Update{Path: "productBlueprintId", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "productBlueprintId", Value: firestore.Delete})
		}
	}
	if patch.TokenBlueprintID != nil {
		if v := fscommon.TrimPtr(patch.TokenBlueprintID); v != nil {
			updates = append(updates, firestore.Update{Path: "tokenBlueprintId", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "tokenBlueprintId", Value: firestore.Delete})
		}
	}
	if patch.AssigneeID != nil {
		if v := fscommon.TrimPtr(patch.AssigneeID); v != nil {
			updates = append(updates, firestore.Update{Path: "assigneeId", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "assigneeId", Value: firestore.Delete})
		}
	}
	// Image(ID): domain patch is Image (string), mapped to imageId
	if patch.Image != nil {
		if v := fscommon.TrimPtr(patch.Image); v != nil {
			updates = append(updates, firestore.Update{Path: "imageId", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "imageId", Value: firestore.Delete})
		}
	}
	if patch.UpdatedBy != nil {
		if v := fscommon.TrimPtr(patch.UpdatedBy); v != nil {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: firestore.Delete})
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
		if v := fscommon.TrimPtr(patch.DeletedBy); v != nil {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: *v})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: firestore.Delete})
		}
	}

	// updatedAt: explicit or now if any other field updated
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
		// no-op
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return idom.Inquiry{}, idom.ErrNotFound
		}
		return idom.Inquiry{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *InquiryRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return idom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return idom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *InquiryRepositoryFS) Save(
	ctx context.Context,
	inq idom.Inquiry,
	_ *idom.SaveOptions,
) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	// ID: if empty, auto-generate
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(inq.ID) == "" {
		docRef = r.col().NewDoc()
		inq.ID = docRef.ID
	} else {
		inq.ID = strings.TrimSpace(inq.ID)
		docRef = r.col().Doc(inq.ID)
	}

	if inq.CreatedAt.IsZero() {
		inq.CreatedAt = now
	}
	if inq.UpdatedAt.IsZero() {
		inq.UpdatedAt = now
	}

	data := inquiryToDocData(inq)
	data["id"] = inq.ID

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return idom.Inquiry{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return idom.Inquiry{}, err
	}
	return docToInquiry(snap)
}

// =======================
// Mapping Helpers
// =======================

func inquiryToDocData(in idom.Inquiry) map[string]any {
	m := map[string]any{
		"id":          strings.TrimSpace(in.ID),
		"avatarId":    strings.TrimSpace(in.AvatarID),
		"subject":     strings.TrimSpace(in.Subject),
		"content":     strings.TrimSpace(in.Content),
		"status":      string(in.Status),
		"inquiryType": string(in.InquiryType),
		"createdAt":   in.CreatedAt.UTC(),
		"updatedAt":   in.UpdatedAt.UTC(),
	}

	if v := fscommon.TrimPtr(in.ProductBlueprintID); v != nil {
		m["productBlueprintId"] = *v
	}
	if v := fscommon.TrimPtr(in.TokenBlueprintID); v != nil {
		m["tokenBlueprintId"] = *v
	}
	if v := fscommon.TrimPtr(in.AssigneeID); v != nil {
		m["assigneeId"] = *v
	}
	if v := fscommon.TrimPtr(in.ImageID); v != nil {
		m["imageId"] = *v
	}
	if v := fscommon.TrimPtr(in.UpdatedBy); v != nil {
		m["updatedBy"] = *v
	}
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		m["deletedAt"] = in.DeletedAt.UTC()
	}
	if v := fscommon.TrimPtr(in.DeletedBy); v != nil {
		m["deletedBy"] = *v
	}

	return m
}

func docToInquiry(doc *firestore.DocumentSnapshot) (idom.Inquiry, error) {
	data := doc.Data()
	if data == nil {
		return idom.Inquiry{}, fmt.Errorf("empty inquiry document: %s", doc.Ref.ID)
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
	getTime := func(key string) (time.Time, bool) {
		if v, ok := data[key].(time.Time); ok {
			return v.UTC(), !v.IsZero()
		}
		return time.Time{}, false
	}
	getPtrTime := func(key string) *time.Time {
		if t, ok := getTime(key); ok && !t.IsZero() {
			return &t
		}
		return nil
	}

	var in idom.Inquiry

	in.ID = getStr("id")
	if in.ID == "" {
		in.ID = doc.Ref.ID
	}

	in.AvatarID = getStr("avatarId")
	in.Subject = getStr("subject")
	in.Content = getStr("content")
	in.Status = idom.InquiryStatus(getStr("status"))
	in.InquiryType = idom.InquiryType(getStr("inquiryType"))

	in.ProductBlueprintID = getPtrStr("productBlueprintId")
	in.TokenBlueprintID = getPtrStr("tokenBlueprintId")
	in.AssigneeID = getPtrStr("assigneeId")
	// stored as imageId
	in.ImageID = getPtrStr("imageId")

	if t, ok := getTime("createdAt"); ok {
		in.CreatedAt = t
	}
	if t, ok := getTime("updatedAt"); ok {
		in.UpdatedAt = t
	}

	in.UpdatedBy = getPtrStr("updatedBy")
	in.DeletedAt = getPtrTime("deletedAt")
	in.DeletedBy = getPtrStr("deletedBy")

	return in, nil
}

// =======================
// Filter / Sort Helpers
// =======================

func matchInquiryFilter(in idom.Inquiry, f idom.Filter) bool {
	// Free text search: subject, content, updated_by, assignee_id
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		haystack := strings.ToLower(
			in.Subject + " " +
				in.Content + " " +
				ptrOrEmpty(in.UpdatedBy) + " " +
				ptrOrEmpty(in.AssigneeID),
		)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// IDs
	if len(f.IDs) > 0 {
		found := false
		for _, v := range f.IDs {
			if strings.TrimSpace(v) == in.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Avatar
	if f.AvatarID != nil && strings.TrimSpace(*f.AvatarID) != "" {
		if in.AvatarID != strings.TrimSpace(*f.AvatarID) {
			return false
		}
	}

	// Assignee
	if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
		if ptrOrEmpty(in.AssigneeID) != strings.TrimSpace(*f.AssigneeID) {
			return false
		}
	}

	// Status
	if f.Status != nil && strings.TrimSpace(string(*f.Status)) != "" {
		if in.Status != *f.Status {
			return false
		}
	}
	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if in.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// InquiryType
	if f.InquiryType != nil && strings.TrimSpace(string(*f.InquiryType)) != "" {
		if in.InquiryType != *f.InquiryType {
			return false
		}
	}
	if len(f.InquiryTypes) > 0 {
		ok := false
		for _, it := range f.InquiryTypes {
			if in.InquiryType == it {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Blueprint IDs
	if f.ProductBlueprintID != nil && strings.TrimSpace(*f.ProductBlueprintID) != "" {
		if ptrOrEmpty(in.ProductBlueprintID) != strings.TrimSpace(*f.ProductBlueprintID) {
			return false
		}
	}
	if f.TokenBlueprintID != nil && strings.TrimSpace(*f.TokenBlueprintID) != "" {
		if ptrOrEmpty(in.TokenBlueprintID) != strings.TrimSpace(*f.TokenBlueprintID) {
			return false
		}
	}

	// HasImage tri-state
	if f.HasImage != nil {
		has := in.ImageID != nil && strings.TrimSpace(*in.ImageID) != ""
		if *f.HasImage != has {
			return false
		}
	}

	// UpdatedBy / DeletedBy
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		if ptrOrEmpty(in.UpdatedBy) != strings.TrimSpace(*f.UpdatedBy) {
			return false
		}
	}
	if f.DeletedBy != nil && strings.TrimSpace(*f.DeletedBy) != "" {
		if ptrOrEmpty(in.DeletedBy) != strings.TrimSpace(*f.DeletedBy) {
			return false
		}
	}

	// Date ranges
	if f.CreatedFrom != nil && in.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !in.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && in.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !in.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}
	if f.DeletedFrom != nil {
		if in.DeletedAt == nil || in.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if in.DeletedAt == nil || !in.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		isDeleted := in.DeletedAt != nil
		if *f.Deleted != isDeleted {
			return false
		}
	}

	return true
}

func applyInquirySort(q firestore.Query, sort idom.Sort) firestore.Query {
	col, dir := mapInquirySort(sort)
	if col == "" {
		// default: createdAt DESC, id DESC
		return q.OrderBy("createdAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapInquirySort(sort idom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	var field string

	switch col {
	case "id":
		field = "id"
	case "avatarid", "avatar_id":
		field = "avatarId"
	case "subject":
		field = "subject"
	case "status":
		field = "status"
	case "inquirytype", "inquiry_type":
		field = "inquiryType"
	case "assigneeid", "assignee_id":
		field = "assigneeId"
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

// small helper
func ptrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
