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

// Compile-time check.
var _ idom.Repository = (*InquiryRepositoryFS)(nil)

// =======================
// Queries
// =======================

func (r *InquiryRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
	filter idom.Filter,
	sort idom.Sort,
	page idom.Page,
) (idom.PageResult[idom.Inquiry], error) {
	if r.Client == nil {
		return idom.PageResult[idom.Inquiry]{}, errors.New("firestore client is nil")
	}
	if companyID == "" {
		return idom.PageResult[idom.Inquiry]{}, idom.ErrNotFound
	}

	pageNum := page.Number
	perPage := page.PerPage
	if pageNum <= 0 {
		pageNum = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	q := r.col().Where("companyId", "==", companyID)
	q = applyInquirySort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]idom.Inquiry, 0)

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
			items = append(items, in)
		}
	}

	total := len(items)
	if total == 0 {
		return idom.PageResult[idom.Inquiry]{
			Items:      []idom.Inquiry{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}

	end := offset + perPage
	if end > total {
		end = total
	}

	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return idom.PageResult[idom.Inquiry]{
		Items:      items[offset:end],
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *InquiryRepositoryFS) CountUnreadByCompanyID(
	ctx context.Context,
	companyID string,
	filter idom.Filter,
) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}
	if companyID == "" {
		return 0, idom.ErrNotFound
	}

	q := r.col().Where("companyId", "==", companyID)

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

		in, err := docToInquiry(doc)
		if err != nil {
			return 0, err
		}

		if in.IsRead {
			continue
		}
		if !matchInquiryFilter(in, filter) {
			continue
		}

		count++
	}

	return count, nil
}

func (r *InquiryRepositoryFS) GetByID(ctx context.Context, id string) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}
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

// =======================
// Mutations
// =======================

func (r *InquiryRepositoryFS) Create(ctx context.Context, inq idom.Inquiry) (idom.Inquiry, error) {
	if r.Client == nil {
		return idom.Inquiry{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	var docRef *firestore.DocumentRef
	if inq.ID == "" {
		docRef = r.col().NewDoc()
		inq.ID = docRef.ID
	} else {
		docRef = r.col().Doc(inq.ID)
	}

	if inq.CreatedAt.IsZero() {
		inq.CreatedAt = now
	}
	if inq.UpdatedAt.IsZero() {
		inq.UpdatedAt = now
	}

	normalizeInquiryImages(&inq, now)

	data := inquiryToDocData(inq)
	data["id"] = inq.ID

	_, err := docRef.Create(ctx, data)
	if err != nil {
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
	if id == "" {
		return idom.Inquiry{}, idom.ErrNotFound
	}

	updates := inquiryPatchToUpdates(patch)
	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := r.col().Doc(id).Update(ctx, updates)
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

// =======================
// Mapping Helpers
// =======================

func inquiryToDocData(in idom.Inquiry) map[string]any {
	m := map[string]any{
		"id":          in.ID,
		"avatarId":    in.AvatarID,
		"subject":     in.Subject,
		"content":     in.Content,
		"status":      string(in.Status),
		"inquiryType": string(in.InquiryType),
		"isRead":      in.IsRead,
		"images":      imagesToDocData(in.Images),
		"createdAt":   in.CreatedAt.UTC(),
		"updatedAt":   in.UpdatedAt.UTC(),
	}

	setOptionalString(m, "productBlueprintId", in.ProductBlueprintID)
	setOptionalString(m, "tokenBlueprintId", in.TokenBlueprintID)
	setOptionalString(m, "assigneeId", in.AssigneeID)
	setOptionalString(m, "updatedBy", in.UpdatedBy)
	setOptionalTime(m, "deletedAt", in.DeletedAt)
	setOptionalString(m, "deletedBy", in.DeletedBy)

	return m
}

func imagesToDocData(images []idom.ImageFile) []map[string]any {
	out := make([]map[string]any, 0, len(images))

	for _, img := range images {
		m := map[string]any{
			"inquiryId": img.InquiryID,
			"fileName":  img.FileName,
			"fileUrl":   img.FileURL,
			"fileSize":  img.FileSize,
			"mimeType":  img.MimeType,
			"createdAt": img.CreatedAt.UTC(),
			"createdBy": img.CreatedBy,
		}

		setOptionalString(m, "objectPath", img.ObjectPath)
		setOptionalTime(m, "updatedAt", img.UpdatedAt)
		setOptionalString(m, "updatedBy", img.UpdatedBy)
		setOptionalTime(m, "deletedAt", img.DeletedAt)
		setOptionalString(m, "deletedBy", img.DeletedBy)

		out = append(out, m)
	}

	return out
}

func docToInquiry(doc *firestore.DocumentSnapshot) (idom.Inquiry, error) {
	data := doc.Data()
	if data == nil {
		return idom.Inquiry{}, fmt.Errorf("empty inquiry document: %s", doc.Ref.ID)
	}

	in := idom.Inquiry{
		ID:                 asString(data["id"]),
		AvatarID:           asString(data["avatarId"]),
		Subject:            asString(data["subject"]),
		Content:            asString(data["content"]),
		Status:             idom.InquiryStatus(asString(data["status"])),
		InquiryType:        idom.InquiryType(asString(data["inquiryType"])),
		IsRead:             asBool(data["isRead"]),
		ProductBlueprintID: ptrStringFromMap(data, "productBlueprintId"),
		TokenBlueprintID:   ptrStringFromMap(data, "tokenBlueprintId"),
		AssigneeID:         ptrStringFromMap(data, "assigneeId"),
		UpdatedBy:          ptrStringFromMap(data, "updatedBy"),
		DeletedAt:          ptrTimeFromMap(data, "deletedAt"),
		DeletedBy:          ptrStringFromMap(data, "deletedBy"),
	}

	if in.ID == "" {
		in.ID = doc.Ref.ID
	}

	if t, ok := asTime(data["createdAt"]); ok {
		in.CreatedAt = t.UTC()
	}
	if t, ok := asTime(data["updatedAt"]); ok {
		in.UpdatedAt = t.UTC()
	}

	in.Images = docImagesToDomain(data["images"], in.ID)

	return in, nil
}

func docImagesToDomain(raw any, fallbackInquiryID string) []idom.ImageFile {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return []idom.ImageFile{}
	}

	images := make([]idom.ImageFile, 0, len(items))

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		img := idom.ImageFile{
			InquiryID:  asString(m["inquiryId"]),
			FileName:   asString(m["fileName"]),
			FileURL:    asString(m["fileUrl"]),
			ObjectPath: ptrStringFromMap(m, "objectPath"),
			FileSize:   int64(asInt(m["fileSize"])),
			MimeType:   asString(m["mimeType"]),
			CreatedAt:  timeFromMap(m, "createdAt"),
			CreatedBy:  asString(m["createdBy"]),
			UpdatedAt:  ptrTimeFromMap(m, "updatedAt"),
			UpdatedBy:  ptrStringFromMap(m, "updatedBy"),
			DeletedAt:  ptrTimeFromMap(m, "deletedAt"),
			DeletedBy:  ptrStringFromMap(m, "deletedBy"),
		}

		if img.InquiryID == "" {
			img.InquiryID = fallbackInquiryID
		}

		images = append(images, img)
	}

	return images
}

func inquiryPatchToUpdates(patch idom.InquiryPatch) []firestore.Update {
	updates := make([]firestore.Update, 0)

	appendStringUpdate := func(path string, value *string) {
		if value == nil {
			return
		}
		if *value == "" {
			updates = append(updates, firestore.Update{Path: path, Value: firestore.Delete})
			return
		}
		updates = append(updates, firestore.Update{Path: path, Value: *value})
	}

	if patch.Subject != nil {
		updates = append(updates, firestore.Update{Path: "subject", Value: *patch.Subject})
	}
	if patch.Content != nil {
		updates = append(updates, firestore.Update{Path: "content", Value: *patch.Content})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: string(*patch.Status)})
	}
	if patch.InquiryType != nil {
		updates = append(updates, firestore.Update{Path: "inquiryType", Value: string(*patch.InquiryType)})
	}
	if patch.IsRead != nil {
		updates = append(updates, firestore.Update{Path: "isRead", Value: *patch.IsRead})
	}

	appendStringUpdate("productBlueprintId", patch.ProductBlueprintID)
	appendStringUpdate("tokenBlueprintId", patch.TokenBlueprintID)
	appendStringUpdate("assigneeId", patch.AssigneeID)

	if patch.Images != nil {
		updates = append(updates, firestore.Update{
			Path:  "images",
			Value: imagesToDocData(*patch.Images),
		})
	}

	appendStringUpdate("updatedBy", patch.UpdatedBy)

	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: patch.DeletedAt.UTC()})
		}
	}

	appendStringUpdate("deletedBy", patch.DeletedBy)

	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
		}
	} else if len(updates) > 0 {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	return updates
}

// =======================
// Filter / Sort Helpers
// =======================

func matchInquiryFilter(in idom.Inquiry, f idom.Filter) bool {
	if f.SearchQuery != "" {
		query := strings.ToLower(f.SearchQuery)
		target := strings.ToLower(searchText(in))
		if !strings.Contains(target, query) {
			return false
		}
	}

	if len(f.IDs) > 0 && !containsString(f.IDs, in.ID) {
		return false
	}

	if f.AvatarID != nil && *f.AvatarID != "" && in.AvatarID != *f.AvatarID {
		return false
	}
	if f.AssigneeID != nil && *f.AssigneeID != "" && ptrOrEmpty(in.AssigneeID) != *f.AssigneeID {
		return false
	}
	if f.Status != nil && string(*f.Status) != "" && in.Status != *f.Status {
		return false
	}
	if f.InquiryType != nil && string(*f.InquiryType) != "" && in.InquiryType != *f.InquiryType {
		return false
	}
	if f.ProductBlueprintID != nil && *f.ProductBlueprintID != "" && ptrOrEmpty(in.ProductBlueprintID) != *f.ProductBlueprintID {
		return false
	}
	if f.TokenBlueprintID != nil && *f.TokenBlueprintID != "" && ptrOrEmpty(in.TokenBlueprintID) != *f.TokenBlueprintID {
		return false
	}
	if f.UpdatedBy != nil && *f.UpdatedBy != "" && ptrOrEmpty(in.UpdatedBy) != *f.UpdatedBy {
		return false
	}
	if f.DeletedBy != nil && *f.DeletedBy != "" && ptrOrEmpty(in.DeletedBy) != *f.DeletedBy {
		return false
	}
	if f.Deleted != nil {
		isDeleted := in.DeletedAt != nil
		if *f.Deleted != isDeleted {
			return false
		}
	}

	return matchImageFilters(in.Images, f)
}

func matchImageFilters(images []idom.ImageFile, f idom.Filter) bool {
	if f.ImageFileName != nil && *f.ImageFileName != "" {
		if !anyImageMatches(images, func(img idom.ImageFile) bool {
			return img.FileName == *f.ImageFileName
		}) {
			return false
		}
	}

	return true
}

func applyInquirySort(q firestore.Query, sort idom.Sort) firestore.Query {
	col, dir := mapInquirySort(sort)
	if col == "" {
		return q.OrderBy("createdAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapInquirySort(sort idom.Sort) (string, firestore.Direction) {
	switch strings.ToLower(string(sort.Column)) {
	case "id":
		return "id", sortDirection(sort)
	case "avatarid", "avatar_id":
		return "avatarId", sortDirection(sort)
	case "subject":
		return "subject", sortDirection(sort)
	case "status":
		return "status", sortDirection(sort)
	case "inquirytype", "inquiry_type":
		return "inquiryType", sortDirection(sort)
	case "assigneeid", "assignee_id":
		return "assigneeId", sortDirection(sort)
	case "createdat", "created_at":
		return "createdAt", sortDirection(sort)
	case "updatedat", "updated_at":
		return "updatedAt", sortDirection(sort)
	case "deletedat", "deleted_at":
		return "deletedAt", sortDirection(sort)
	default:
		return "", firestore.Desc
	}
}

func sortDirection(sort idom.Sort) firestore.Direction {
	if strings.EqualFold(string(sort.Order), "desc") {
		return firestore.Desc
	}
	return firestore.Asc
}

// =======================
// Small Helpers
// =======================

func normalizeInquiryImages(inq *idom.Inquiry, now time.Time) {
	if inq.Images == nil {
		inq.Images = []idom.ImageFile{}
		return
	}

	for i := range inq.Images {
		if inq.Images[i].InquiryID == "" {
			inq.Images[i].InquiryID = inq.ID
		}
		if inq.Images[i].CreatedAt.IsZero() {
			inq.Images[i].CreatedAt = now
		}
	}
}

func setOptionalString(m map[string]any, key string, value *string) {
	if value != nil && *value != "" {
		m[key] = *value
	}
}

func setOptionalTime(m map[string]any, key string, value *time.Time) {
	if value != nil && !value.IsZero() {
		m[key] = value.UTC()
	}
}

func ptrStringFromMap(m map[string]any, key string) *string {
	s := asString(m[key])
	if s == "" {
		return nil
	}
	return &s
}

func timeFromMap(m map[string]any, key string) time.Time {
	t, _ := asTime(m[key])
	return t.UTC()
}

func ptrTimeFromMap(m map[string]any, key string) *time.Time {
	t, ok := asTime(m[key])
	if !ok || t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func ptrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func anyImageMatches(images []idom.ImageFile, fn func(idom.ImageFile) bool) bool {
	for _, img := range images {
		if fn(img) {
			return true
		}
	}
	return false
}

func containsInquiryType(xs []idom.InquiryType, v idom.InquiryType) bool {
	if string(v) == "" || len(xs) == 0 {
		return false
	}
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func searchText(in idom.Inquiry) string {
	var b strings.Builder

	b.WriteString(in.Subject)
	b.WriteString(" ")
	b.WriteString(in.Content)
	b.WriteString(" ")
	b.WriteString(ptrOrEmpty(in.UpdatedBy))
	b.WriteString(" ")
	b.WriteString(ptrOrEmpty(in.AssigneeID))

	for _, img := range in.Images {
		b.WriteString(" ")
		b.WriteString(img.FileName)
		b.WriteString(" ")
		b.WriteString(img.FileURL)
		b.WriteString(" ")
		b.WriteString(ptrOrEmpty(img.ObjectPath))
		b.WriteString(" ")
		b.WriteString(img.MimeType)
		b.WriteString(" ")
		b.WriteString(img.CreatedBy)
		b.WriteString(" ")
		b.WriteString(ptrOrEmpty(img.UpdatedBy))
		b.WriteString(" ")
		b.WriteString(ptrOrEmpty(img.DeletedBy))
	}

	return b.String()
}
