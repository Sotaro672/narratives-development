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

	// Inquiry no longer stores companyId.
	// Company-scoped listing should eventually move to a query service that resolves
	// companyId -> productIds and then lists inquiries by productId.
	q := r.col().Query
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

	// Inquiry no longer stores companyId.
	// Company-scoped counting should eventually move to a query service that resolves
	// companyId -> productIds and then counts inquiries by productId.
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

	if err := inq.Validate(); err != nil {
		return idom.Inquiry{}, err
	}

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

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return idom.Inquiry{}, err
	}

	if err := applyInquiryPatchToDomain(&current, patch); err != nil {
		return idom.Inquiry{}, err
	}

	if err := current.Validate(); err != nil {
		return idom.Inquiry{}, err
	}

	_, err = r.col().Doc(id).Set(ctx, inquiryToDocData(current))
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
// Reply Subcollection Repository
// =======================

// InquiryReplyRepositoryFS implements Inquiry reply repository using Firestore.
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
type InquiryReplyRepositoryFS struct {
	Client *firestore.Client
}

func NewInquiryReplyRepositoryFS(client *firestore.Client) *InquiryReplyRepositoryFS {
	return &InquiryReplyRepositoryFS{Client: client}
}

func (r *InquiryReplyRepositoryFS) col(inquiryID string) *firestore.CollectionRef {
	return r.Client.
		Collection("inquiries").
		Doc(inquiryID).
		Collection("replies")
}

func (r *InquiryReplyRepositoryFS) Create(
	ctx context.Context,
	reply idom.Reply,
) (idom.Reply, error) {
	if r.Client == nil {
		return idom.Reply{}, errors.New("firestore client is nil")
	}

	if err := reply.Validate(); err != nil {
		return idom.Reply{}, err
	}

	docRef := r.col(reply.InquiryID).Doc(reply.ID)

	_, err := docRef.Create(ctx, replyToDocData(reply))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return idom.Reply{}, idom.ErrConflict
		}
		return idom.Reply{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return idom.Reply{}, err
	}

	return docToReply(snap)
}

func (r *InquiryReplyRepositoryFS) ListByInquiryID(
	ctx context.Context,
	inquiryID string,
) ([]idom.Reply, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if inquiryID == "" {
		return nil, idom.ErrInvalidReplyInquiryID
	}

	it := r.col(inquiryID).
		OrderBy("createdAt", firestore.Asc).
		OrderBy("id", firestore.Asc).
		Documents(ctx)
	defer it.Stop()

	replies := make([]idom.Reply, 0)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		reply, err := docToReply(doc)
		if err != nil {
			return nil, err
		}

		replies = append(replies, reply)
	}

	return replies, nil
}

// =======================
// Mapping Helpers
// =======================

func inquiryToDocData(in idom.Inquiry) map[string]any {
	m := map[string]any{
		"id":          in.ID,
		"productId":   in.ProductID,
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

	setOptionalTime(m, "resolvedAt", in.ResolvedAt)
	setOptionalString(m, "resolvedBy", in.ResolvedBy)
	setOptionalTime(m, "closedAt", in.ClosedAt)
	setOptionalString(m, "closedBy", in.ClosedBy)
	setOptionalString(m, "updatedBy", in.UpdatedBy)
	setOptionalTime(m, "deletedAt", in.DeletedAt)
	setOptionalString(m, "deletedBy", in.DeletedBy)

	return m
}

func replyToDocData(reply idom.Reply) map[string]any {
	m := map[string]any{
		"id":         reply.ID,
		"inquiryId":  reply.InquiryID,
		"senderType": string(reply.SenderType),
		"senderId":   reply.SenderID,
		"content":    reply.Content,
		"images":     imagesToDocData(reply.Images),
		"createdAt":  reply.CreatedAt.UTC(),
		"createdBy":  reply.CreatedBy,
	}

	setOptionalTime(m, "updatedAt", reply.UpdatedAt)
	setOptionalString(m, "updatedBy", reply.UpdatedBy)
	setOptionalTime(m, "deletedAt", reply.DeletedAt)
	setOptionalString(m, "deletedBy", reply.DeletedBy)

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
		ID:          asString(data["id"]),
		ProductID:   asString(data["productId"]),
		AvatarID:    asString(data["avatarId"]),
		Subject:     asString(data["subject"]),
		Content:     asString(data["content"]),
		Status:      idom.InquiryStatus(asString(data["status"])),
		InquiryType: idom.InquiryType(asString(data["inquiryType"])),
		IsRead:      asBool(data["isRead"]),
		ResolvedAt:  ptrTimeFromMap(data, "resolvedAt"),
		ResolvedBy:  ptrStringFromMap(data, "resolvedBy"),
		ClosedAt:    ptrTimeFromMap(data, "closedAt"),
		ClosedBy:    ptrStringFromMap(data, "closedBy"),
		UpdatedBy:   ptrStringFromMap(data, "updatedBy"),
		DeletedAt:   ptrTimeFromMap(data, "deletedAt"),
		DeletedBy:   ptrStringFromMap(data, "deletedBy"),
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

func docToReply(doc *firestore.DocumentSnapshot) (idom.Reply, error) {
	data := doc.Data()
	if data == nil {
		return idom.Reply{}, fmt.Errorf("empty inquiry reply document: %s", doc.Ref.ID)
	}

	reply := idom.Reply{
		ID:         asString(data["id"]),
		InquiryID:  asString(data["inquiryId"]),
		SenderType: idom.ReplySenderType(asString(data["senderType"])),
		SenderID:   asString(data["senderId"]),
		Content:    asString(data["content"]),
		CreatedBy:  asString(data["createdBy"]),
		UpdatedAt:  ptrTimeFromMap(data, "updatedAt"),
		UpdatedBy:  ptrStringFromMap(data, "updatedBy"),
		DeletedAt:  ptrTimeFromMap(data, "deletedAt"),
		DeletedBy:  ptrStringFromMap(data, "deletedBy"),
	}

	if reply.ID == "" {
		reply.ID = doc.Ref.ID
	}

	if t, ok := asTime(data["createdAt"]); ok {
		reply.CreatedAt = t.UTC()
	}

	reply.Images = docImagesToDomain(data["images"], reply.InquiryID)

	if err := reply.Validate(); err != nil {
		return idom.Reply{}, err
	}

	return reply, nil
}

func docImagesToDomain(raw any, fallbackInquiryID string) []idom.ImageFile {
	if raw == nil {
		return []idom.ImageFile{}
	}

	images := make([]idom.ImageFile, 0)

	switch items := raw.(type) {
	case []any:
		for _, item := range items {
			img, ok := docImageItemToDomain(item, fallbackInquiryID)
			if ok {
				images = append(images, img)
			}
		}
	case []map[string]any:
		for _, item := range items {
			img, ok := docImageMapToDomain(item, fallbackInquiryID)
			if ok {
				images = append(images, img)
			}
		}
	}

	if len(images) == 0 {
		return []idom.ImageFile{}
	}

	return images
}

func docImageItemToDomain(item any, fallbackInquiryID string) (idom.ImageFile, bool) {
	m, ok := item.(map[string]any)
	if !ok {
		return idom.ImageFile{}, false
	}

	return docImageMapToDomain(m, fallbackInquiryID)
}

func docImageMapToDomain(m map[string]any, fallbackInquiryID string) (idom.ImageFile, bool) {
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

	return img, true
}

func applyInquiryPatchToDomain(in *idom.Inquiry, patch idom.InquiryPatch) error {
	if patch.ProductID != nil {
		in.ProductID = *patch.ProductID
	}
	if patch.Subject != nil {
		in.Subject = *patch.Subject
	}
	if patch.Content != nil {
		in.Content = *patch.Content
	}
	if patch.Status != nil {
		in.Status = *patch.Status
	}
	if patch.InquiryType != nil {
		in.InquiryType = *patch.InquiryType
	}
	if patch.IsRead != nil {
		in.IsRead = *patch.IsRead
	}

	if patch.Images != nil {
		if err := in.ReplaceImages(*patch.Images); err != nil {
			return err
		}
	}

	if patch.ResolvedAt != nil {
		in.ResolvedAt = optionalTimeFromPatch(patch.ResolvedAt)
	}
	if patch.ResolvedBy != nil {
		in.ResolvedBy = optionalStringFromPatch(patch.ResolvedBy)
	}
	if patch.ClosedAt != nil {
		in.ClosedAt = optionalTimeFromPatch(patch.ClosedAt)
	}
	if patch.ClosedBy != nil {
		in.ClosedBy = optionalStringFromPatch(patch.ClosedBy)
	}

	if patch.UpdatedBy != nil {
		in.UpdatedBy = optionalStringFromPatch(patch.UpdatedBy)
	}

	if patch.DeletedAt != nil {
		in.DeletedAt = optionalTimeFromPatch(patch.DeletedAt)
	}
	if patch.DeletedBy != nil {
		in.DeletedBy = optionalStringFromPatch(patch.DeletedBy)
	}

	if patch.UpdatedAt != nil && !patch.UpdatedAt.IsZero() {
		in.UpdatedAt = patch.UpdatedAt.UTC()
	} else {
		in.UpdatedAt = time.Now().UTC()
	}

	return nil
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

	if f.ProductID != nil && *f.ProductID != "" && in.ProductID != *f.ProductID {
		return false
	}
	if f.AvatarID != nil && *f.AvatarID != "" && in.AvatarID != *f.AvatarID {
		return false
	}
	if f.Status != nil && string(*f.Status) != "" && in.Status != *f.Status {
		return false
	}
	if f.InquiryType != nil && string(*f.InquiryType) != "" && in.InquiryType != *f.InquiryType {
		return false
	}
	if f.UpdatedBy != nil && *f.UpdatedBy != "" && ptrOrEmpty(in.UpdatedBy) != *f.UpdatedBy {
		return false
	}
	if f.DeletedBy != nil && *f.DeletedBy != "" && ptrOrEmpty(in.DeletedBy) != *f.DeletedBy {
		return false
	}
	if f.ResolvedBy != nil && *f.ResolvedBy != "" && ptrOrEmpty(in.ResolvedBy) != *f.ResolvedBy {
		return false
	}
	if f.ClosedBy != nil && *f.ClosedBy != "" && ptrOrEmpty(in.ClosedBy) != *f.ClosedBy {
		return false
	}
	if f.Deleted != nil {
		isDeleted := in.DeletedAt != nil
		if *f.Deleted != isDeleted {
			return false
		}
	}
	if f.Resolved != nil {
		isResolved := in.ResolvedAt != nil
		if *f.Resolved != isResolved {
			return false
		}
	}
	if f.Closed != nil {
		isClosed := in.ClosedAt != nil
		if *f.Closed != isClosed {
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
	case "productid", "product_id":
		return "productId", sortDirection(sort)
	case "avatarid", "avatar_id":
		return "avatarId", sortDirection(sort)
	case "subject":
		return "subject", sortDirection(sort)
	case "status":
		return "status", sortDirection(sort)
	case "inquirytype", "inquiry_type":
		return "inquiryType", sortDirection(sort)
	case "resolvedat", "resolved_at":
		return "resolvedAt", sortDirection(sort)
	case "closedat", "closed_at":
		return "closedAt", sortDirection(sort)
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
			inq.Images[i].CreatedAt = now.UTC()
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

func optionalStringFromPatch(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	v := *value
	return &v
}

func optionalTimeFromPatch(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
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

func searchText(in idom.Inquiry) string {
	var b strings.Builder

	b.WriteString(in.ProductID)
	b.WriteString(" ")
	b.WriteString(in.AvatarID)
	b.WriteString(" ")
	b.WriteString(in.Subject)
	b.WriteString(" ")
	b.WriteString(in.Content)
	b.WriteString(" ")
	b.WriteString(string(in.Status))
	b.WriteString(" ")
	b.WriteString(string(in.InquiryType))
	b.WriteString(" ")
	b.WriteString(ptrOrEmpty(in.ResolvedBy))
	b.WriteString(" ")
	b.WriteString(ptrOrEmpty(in.ClosedBy))
	b.WriteString(" ")
	b.WriteString(ptrOrEmpty(in.UpdatedBy))

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
