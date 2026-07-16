package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	announcement "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
)

// Firestore implementation of announcement.Repository.
type AnnouncementRepositoryFS struct {
	Client *firestore.Client
}

// Firestore implementation of announcement.AvatarRepository.
type AnnouncementAvatarRepositoryFS struct {
	Client *firestore.Client
}

type announcementAvatarDoc struct {
	AnnouncementID string     `firestore:"announcementId"`
	AvatarID       string     `firestore:"avatarId"`
	IsRead         bool       `firestore:"isRead"`
	ReadAt         *time.Time `firestore:"readAt,omitempty"`
	CreatedAt      time.Time  `firestore:"createdAt"`
	UpdatedAt      *time.Time `firestore:"updatedAt,omitempty"`
}

func NewAnnouncementRepositoryFS(client *firestore.Client) *AnnouncementRepositoryFS {
	return &AnnouncementRepositoryFS{Client: client}
}

func NewAnnouncementAvatarRepositoryFS(client *firestore.Client) *AnnouncementAvatarRepositoryFS {
	return &AnnouncementAvatarRepositoryFS{Client: client}
}

// Compile-time checks.
var _ announcement.Repository = (*AnnouncementRepositoryFS)(nil)
var _ announcement.AvatarRepository = (*AnnouncementAvatarRepositoryFS)(nil)

func announcementCollection(client *firestore.Client) *firestore.CollectionRef {
	return client.Collection("announcements")
}

func announcementDoc(client *firestore.Client, announcementID string) *firestore.DocumentRef {
	return announcementCollection(client).Doc(announcementID)
}

func avatarCollection(client *firestore.Client, announcementID string) *firestore.CollectionRef {
	return announcementDoc(client, announcementID).Collection("avatars")
}

func avatarDoc(client *firestore.Client, announcementID string, avatarID string) *firestore.DocumentRef {
	return avatarCollection(client, announcementID).Doc(avatarID)
}

// GetByID retrieves an announcement by ID from Firestore.
func (r *AnnouncementRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}

	doc, err := announcementDoc(r.Client, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	return announcementFromDoc(ctx, doc)
}

// Create inserts a new announcement.
//
// Parent document does not store:
// - ID
// - TargetAvatars
// - Attachments
//
// Target avatars are stored under:
// announcements/{announcementId}/avatars/{avatarId}
//
// Attachment metadata is stored under:
// announcements/{announcementId}/attachments/{attachmentId}
func (r *AnnouncementRepositoryFS) Create(
	ctx context.Context,
	a announcement.Announcement,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}

	ref := announcementDoc(r.Client, a.ID)
	if a.ID == "" {
		ref = announcementCollection(r.Client).NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = &now

	if _, err := ref.Set(ctx, announcementData(a)); err != nil {
		return announcement.Announcement{}, err
	}

	if err := syncAnnouncementAvatars(
		ctx,
		r.Client,
		a.ID,
		a.TargetAvatars,
		a.CreatedAt,
	); err != nil {
		return announcement.Announcement{}, err
	}

	return a, nil
}

// Update replaces/upserts the mutable fields of a persisted announcement.
func (r *AnnouncementRepositoryFS) Update(
	ctx context.Context,
	id string,
	a announcement.Announcement,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}

	updatedAt := time.Now().UTC()
	if a.UpdatedAt != nil {
		updatedAt = *a.UpdatedAt
	}

	updates := []firestore.Update{
		{Path: "Title", Value: a.Title},
		{Path: "Content", Value: a.Content},
		{Path: "TargetToken", Value: a.TargetToken},
		{Path: "Published", Value: a.Published},
		{Path: "PublishedAt", Value: a.PublishedAt},
		{Path: "UpdatedAt", Value: updatedAt},

		// Remove legacy duplicated/embedded fields.
		{Path: "ID", Value: firestore.Delete},
		{Path: "TargetAvatars", Value: firestore.Delete},
		{Path: "Attachments", Value: firestore.Delete},
	}

	if a.UpdatedBy != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "UpdatedBy",
				Value: *a.UpdatedBy,
			},
		)
	}

	if _, err := announcementDoc(r.Client, id).Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	createdAt := a.CreatedAt
	if createdAt.IsZero() {
		createdAt = updatedAt
	}

	if err := syncAnnouncementAvatars(
		ctx,
		r.Client,
		id,
		a.TargetAvatars,
		createdAt,
	); err != nil {
		return announcement.Announcement{}, err
	}

	return r.GetByID(ctx, id)
}

// MarkPublished marks an announcement as published.
func (r *AnnouncementRepositoryFS) MarkPublished(
	ctx context.Context,
	id string,
	publishedAt time.Time,
	updatedBy *string,
) (announcement.Announcement, error) {
	if r.Client == nil {
		return announcement.Announcement{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.Announcement{}, announcement.ErrInvalidID
	}
	if publishedAt.IsZero() {
		return announcement.Announcement{}, announcement.ErrInvalidPublishedAt
	}

	updates := []firestore.Update{
		{Path: "Published", Value: true},
		{Path: "PublishedAt", Value: publishedAt},
		{Path: "UpdatedAt", Value: publishedAt},

		// Remove legacy duplicated/embedded fields when touched.
		{Path: "ID", Value: firestore.Delete},
		{Path: "TargetAvatars", Value: firestore.Delete},
		{Path: "Attachments", Value: firestore.Delete},
	}

	if updatedBy != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "UpdatedBy",
				Value: *updatedBy,
			},
		)
	}

	if _, err := announcementDoc(r.Client, id).Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}

	return r.GetByID(ctx, id)
}

// Delete removes an announcement by ID.
func (r *AnnouncementRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return announcement.ErrInvalidID
	}

	ref := announcementDoc(r.Client, id)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.ErrNotFound
		}
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

// ListByTargetToken returns announcements whose TargetToken equals tokenBlueprintID.
func (r *AnnouncementRepositoryFS) ListByTargetToken(
	ctx context.Context,
	tokenBlueprintID string,
	page common.Page,
) (common.PageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.PageResult[announcement.Announcement]{},
			errors.New("firestore client is nil")
	}
	if tokenBlueprintID == "" {
		return common.PageResult[announcement.Announcement]{},
			announcement.ErrInvalidID
	}

	iter := announcementCollection(r.Client).
		Where("TargetToken", "==", tokenBlueprintID).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	items, err := announcementsFromIter(ctx, iter)
	if err != nil {
		return common.PageResult[announcement.Announcement]{}, err
	}

	return paginateAnnouncements(items, page), nil
}

// ListByTargetAvatar returns published announcements whose avatars subcollection
// contains avatarID.
func (r *AnnouncementRepositoryFS) ListByTargetAvatar(
	ctx context.Context,
	avatarID string,
	page common.Page,
) (common.PageResult[announcement.Announcement], error) {
	if r.Client == nil {
		return common.PageResult[announcement.Announcement]{},
			errors.New("firestore client is nil")
	}
	if avatarID == "" {
		return common.PageResult[announcement.Announcement]{},
			announcement.ErrInvalidAvatarID
	}

	iter := r.Client.
		CollectionGroup("avatars").
		Where("avatarId", "==", avatarID).
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	items := []announcement.Announcement{}
	seen := map[string]struct{}{}

	for {
		avatarDocSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		if avatarDocSnap.Ref == nil ||
			avatarDocSnap.Ref.Parent == nil ||
			avatarDocSnap.Ref.Parent.Parent == nil {
			continue
		}

		parentRef := avatarDocSnap.Ref.Parent.Parent
		if _, ok := seen[parentRef.ID]; ok {
			continue
		}
		seen[parentRef.ID] = struct{}{}

		parentDoc, err := parentRef.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return common.PageResult[announcement.Announcement]{}, err
		}

		a, err := announcementFromDoc(ctx, parentDoc)
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}

		if !a.Published {
			continue
		}

		items = append(items, a)
	}

	return paginateAnnouncements(items, page), nil
}

// ListByAnnouncementID returns avatar subcollection documents.
func (r *AnnouncementAvatarRepositoryFS) ListByAnnouncementID(
	ctx context.Context,
	announcementID string,
	filter announcement.AnnouncementAvatarFilter,
) ([]announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return nil, announcement.ErrInvalidAnnouncementID
	}

	iter := avatarCollection(
		r.Client,
		announcementID,
	).Documents(ctx)
	defer iter.Stop()

	results := []announcement.AnnouncementAvatar{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		avatar, err := announcementAvatarFromDoc(
			doc,
			announcementID,
		)
		if err != nil {
			return nil, err
		}

		if !avatarMatchesFilter(avatar, filter) {
			continue
		}

		results = append(results, avatar)
	}

	return results, nil
}

// Upsert creates or updates the avatar read state.
func (r *AnnouncementAvatarRepositoryFS) Upsert(
	ctx context.Context,
	announcementID string,
	avatarID string,
	patch announcement.AnnouncementAvatarPatch,
) (announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return announcement.AnnouncementAvatar{},
			errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.AnnouncementAvatar{},
			announcement.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return announcement.AnnouncementAvatar{},
			announcement.ErrInvalidAvatarID
	}

	now := time.Now().UTC()
	ref := avatarDoc(
		r.Client,
		announcementID,
		avatarID,
	)

	data := map[string]any{
		"announcementId": announcementID,
		"avatarId":       avatarID,
		"updatedAt":      now,
	}

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) != codes.NotFound {
			return announcement.AnnouncementAvatar{}, err
		}
		data["createdAt"] = now
	}

	if patch.IsRead != nil {
		data["isRead"] = *patch.IsRead
		if !*patch.IsRead {
			data["readAt"] = nil
		}
	}
	if patch.ReadAt != nil {
		data["readAt"] = *patch.ReadAt
	}
	if patch.UpdatedAt != nil {
		data["updatedAt"] = *patch.UpdatedAt
	}

	if _, err := ref.Set(
		ctx,
		data,
		firestore.MergeAll,
	); err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	return getAnnouncementAvatar(
		ctx,
		r.Client,
		announcementID,
		avatarID,
	)
}

// MarkRead marks announcements/{announcementId}/avatars/{avatarId} as read.
func (r *AnnouncementAvatarRepositoryFS) MarkRead(
	ctx context.Context,
	announcementID string,
	avatarID string,
	readAt time.Time,
) (announcement.AnnouncementAvatar, error) {
	if r.Client == nil {
		return announcement.AnnouncementAvatar{},
			errors.New("firestore client is nil")
	}
	if announcementID == "" {
		return announcement.AnnouncementAvatar{},
			announcement.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return announcement.AnnouncementAvatar{},
			announcement.ErrInvalidAvatarID
	}
	if readAt.IsZero() {
		return announcement.AnnouncementAvatar{},
			announcement.ErrInvalidReadAt
	}

	ref := avatarDoc(
		r.Client,
		announcementID,
		avatarID,
	)

	data := map[string]any{
		"announcementId": announcementID,
		"avatarId":       avatarID,
		"isRead":         true,
		"readAt":         readAt,
		"updatedAt":      readAt,
	}

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) != codes.NotFound {
			return announcement.AnnouncementAvatar{}, err
		}
		data["createdAt"] = readAt
	}

	if _, err := ref.Set(
		ctx,
		data,
		firestore.MergeAll,
	); err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	return getAnnouncementAvatar(
		ctx,
		r.Client,
		announcementID,
		avatarID,
	)
}

func announcementData(
	a announcement.Announcement,
) map[string]any {
	return map[string]any{
		"Title":       a.Title,
		"Content":     a.Content,
		"TargetToken": a.TargetToken,
		"Published":   a.Published,
		"PublishedAt": a.PublishedAt,
		"CreatedAt":   a.CreatedAt,
		"CreatedBy":   a.CreatedBy,
		"UpdatedAt":   a.UpdatedAt,
		"UpdatedBy":   a.UpdatedBy,
	}
}

func syncAnnouncementAvatars(
	ctx context.Context,
	client *firestore.Client,
	announcementID string,
	targetAvatarIDs []string,
	createdAt time.Time,
) error {
	targets := uniqueStringSet(targetAvatarIDs)
	col := avatarCollection(client, announcementID)
	batch := client.Batch()
	writes := 0

	iter := col.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		if _, ok := targets[doc.Ref.ID]; !ok {
			batch.Delete(doc.Ref)
			writes++
		}
	}

	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	for avatarID := range targets {
		ref := col.Doc(avatarID)

		doc, err := ref.Get(ctx)
		if err == nil && doc.Exists() {
			continue
		}
		if err != nil &&
			status.Code(err) != codes.NotFound {
			return err
		}

		batch.Set(ref, map[string]any{
			"announcementId": announcementID,
			"avatarId":       avatarID,
			"isRead":         false,
			"readAt":         nil,
			"createdAt":      createdAt,
			"updatedAt":      nil,
		})
		writes++
	}

	if writes == 0 {
		return nil
	}

	_, err := batch.Commit(ctx)
	return err
}

func announcementFromDoc(
	ctx context.Context,
	doc *firestore.DocumentSnapshot,
) (announcement.Announcement, error) {
	if doc == nil {
		return announcement.Announcement{},
			announcement.ErrNotFound
	}

	var a announcement.Announcement
	if err := doc.DataTo(&a); err != nil {
		return announcement.Announcement{}, err
	}

	a.ID = doc.Ref.ID

	targetAvatarIDs, err := childDocIDs(
		ctx,
		doc.Ref.Collection("avatars"),
	)
	if err != nil {
		return announcement.Announcement{}, err
	}

	attachmentIDs, err := childDocIDs(
		ctx,
		doc.Ref.Collection("attachments"),
	)
	if err != nil {
		return announcement.Announcement{}, err
	}

	a.TargetAvatars = targetAvatarIDs
	a.Attachments = attachmentIDs

	return a, nil
}

func announcementsFromIter(
	ctx context.Context,
	iter *firestore.DocumentIterator,
) ([]announcement.Announcement, error) {
	items := []announcement.Announcement{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		a, err := announcementFromDoc(ctx, doc)
		if err != nil {
			return nil, err
		}

		items = append(items, a)
	}

	return items, nil
}

func childDocIDs(
	ctx context.Context,
	col *firestore.CollectionRef,
) ([]string, error) {
	iter := col.Documents(ctx)
	defer iter.Stop()

	ids := []string{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if doc.Ref != nil &&
			doc.Ref.ID != "" {
			ids = append(ids, doc.Ref.ID)
		}
	}

	return ids, nil
}

func getAnnouncementAvatar(
	ctx context.Context,
	client *firestore.Client,
	announcementID string,
	avatarID string,
) (announcement.AnnouncementAvatar, error) {
	doc, err := avatarDoc(
		client,
		announcementID,
		avatarID,
	).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return announcement.AnnouncementAvatar{},
				announcement.ErrNotFound
		}
		return announcement.AnnouncementAvatar{}, err
	}

	return announcementAvatarFromDoc(
		doc,
		announcementID,
	)
}

func announcementAvatarFromDoc(
	doc *firestore.DocumentSnapshot,
	announcementID string,
) (announcement.AnnouncementAvatar, error) {
	if doc == nil {
		return announcement.AnnouncementAvatar{},
			announcement.ErrNotFound
	}

	var data announcementAvatarDoc
	if err := doc.DataTo(&data); err != nil {
		return announcement.AnnouncementAvatar{}, err
	}

	if data.AvatarID == "" {
		data.AvatarID = doc.Ref.ID
	}
	if data.AnnouncementID == "" {
		data.AnnouncementID = announcementID
	}

	return announcement.NewAnnouncementAvatarWithState(
		data.AnnouncementID,
		data.AvatarID,
		data.IsRead,
		data.ReadAt,
		data.CreatedAt,
		data.UpdatedAt,
	)
}

func avatarMatchesFilter(
	avatar announcement.AnnouncementAvatar,
	filter announcement.AnnouncementAvatarFilter,
) bool {
	if len(filter.AvatarIDs) > 0 {
		if _, ok := uniqueStringSet(
			filter.AvatarIDs,
		)[avatar.AvatarID]; !ok {
			return false
		}
	}

	if filter.IsRead != nil &&
		avatar.IsRead != *filter.IsRead {
		return false
	}

	return true
}

func uniqueStringSet(
	values []string,
) map[string]struct{} {
	out := map[string]struct{}{}

	for _, value := range values {
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}

	return out
}

func paginateAnnouncements(
	items []announcement.Announcement,
	page common.Page,
) common.PageResult[announcement.Announcement] {
	total := len(items)
	if total == 0 {
		return common.PageResult[announcement.Announcement]{
			Items:      []announcement.Announcement{},
			TotalCount: 0,
			Page:       1,
			PerPage:    0,
		}
	}

	pageNum, perPage, _ := fscommon.NormalizePage(
		page.Number,
		page.PerPage,
		50,
		0,
	)

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}

	end := offset + perPage
	if end > total {
		end = total
	}

	return common.PageResult[announcement.Announcement]{
		Items:      items[offset:end],
		TotalCount: total,
		Page:       pageNum,
		PerPage:    perPage,
	}
}
