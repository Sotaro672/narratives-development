// backend/internal/adapters/out/firestore/like_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	likedom "narratives/internal/domain/like"
)

const (
	likesCollection        = "likes"
	likeItemsSubCollection = "items"
	defaultLikePerPage     = 20
)

var errLikeRepositoryNotConfigured = errors.New(
	"like_repository_fs: firestore client is nil",
)

type LikeRepositoryFS struct {
	Client *gfs.Client
}

var _ likedom.Repository = (*LikeRepositoryFS)(nil)

func NewLikeRepositoryFS(client *gfs.Client) *LikeRepositoryFS {
	return &LikeRepositoryFS{
		Client: client,
	}
}

// Firestore structure:
//
// likes/{avatarId}/items/{documentId}
//
// documentId:
// - list_{listId}
// - resale_{resaleId}
func (r *LikeRepositoryFS) avatarDoc(
	avatarID string,
) *gfs.DocumentRef {
	return r.Client.
		Collection(likesCollection).
		Doc(avatarID)
}

func (r *LikeRepositoryFS) itemsCol(
	avatarID string,
) *gfs.CollectionRef {
	return r.avatarDoc(avatarID).
		Collection(likeItemsSubCollection)
}

func (r *LikeRepositoryFS) itemDoc(
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (*gfs.DocumentRef, error) {
	documentID, err := likeDocumentID(
		targetType,
		targetID,
	)
	if err != nil {
		return nil, err
	}

	return r.itemsCol(avatarID).Doc(documentID), nil
}

func (r *LikeRepositoryFS) validateConfigured() error {
	if r == nil || r.Client == nil {
		return errLikeRepositoryNotConfigured
	}

	return nil
}

// ============================================================
// Firestore document
// ============================================================

type likeDocument struct {
	AvatarID   string             `firestore:"avatarId"`
	TargetType likedom.TargetType `firestore:"targetType"`
	TargetID   string             `firestore:"targetId"`
	CreatedAt  time.Time          `firestore:"createdAt"`
}

func likeToDocument(
	entity likedom.Like,
) likeDocument {
	return likeDocument{
		AvatarID:   entity.AvatarID,
		TargetType: entity.TargetType,
		TargetID:   entity.TargetID,
		CreatedAt:  entity.CreatedAt.UTC(),
	}
}

func likeFromSnapshot(
	snapshot *gfs.DocumentSnapshot,
) (likedom.Like, error) {
	if snapshot == nil ||
		snapshot.Ref == nil ||
		!snapshot.Exists() {
		return likedom.Like{}, likedom.ErrNotFound
	}

	var document likeDocument

	if err := snapshot.DataTo(&document); err != nil {
		return likedom.Like{}, err
	}

	entity, err := likedom.RestoreLike(
		document.AvatarID,
		document.TargetType,
		document.TargetID,
		document.CreatedAt,
	)
	if err != nil {
		return likedom.Like{}, err
	}

	return *entity, nil
}

// ============================================================
// List
// ============================================================

func (r *LikeRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	filter likedom.Filter,
	sortSpec likedom.Sort,
	page likedom.Page,
) (likedom.PageResult[likedom.Like], error) {
	if err := r.validateConfigured(); err != nil {
		return likedom.PageResult[likedom.Like]{}, err
	}

	normalizedAvatarID := strings.TrimSpace(avatarID)
	if !isValidLikeReferenceID(normalizedAvatarID) {
		return likedom.PageResult[likedom.Like]{}, likedom.ErrInvalidAvatarID
	}

	if filter.TargetType != nil &&
		!likedom.IsValidTargetType(*filter.TargetType) {
		return likedom.PageResult[likedom.Like]{}, likedom.ErrInvalidTargetType
	}

	targetIDSet := make(map[string]struct{}, len(filter.TargetIDs))
	for _, rawTargetID := range filter.TargetIDs {
		targetID := strings.TrimSpace(rawTargetID)

		if !isValidLikeReferenceID(targetID) {
			return likedom.PageResult[likedom.Like]{}, likedom.ErrInvalidTargetID
		}

		targetIDSet[targetID] = struct{}{}
	}

	items := make([]likedom.Like, 0)

	iter := r.itemsCol(normalizedAvatarID).Documents(ctx)
	defer iter.Stop()

	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return likedom.PageResult[likedom.Like]{}, err
		}

		entity, err := likeFromSnapshot(snapshot)
		if err != nil {
			return likedom.PageResult[likedom.Like]{}, err
		}

		if filter.TargetType != nil &&
			entity.TargetType != *filter.TargetType {
			continue
		}

		if len(targetIDSet) > 0 {
			if _, ok := targetIDSet[entity.TargetID]; !ok {
				continue
			}
		}

		items = append(items, entity)
	}

	if err := sortLikes(items, sortSpec); err != nil {
		return likedom.PageResult[likedom.Like]{}, err
	}

	return paginateLikes(items, page), nil
}

// ============================================================
// Exists
// ============================================================

func (r *LikeRepositoryFS) Exists(
	ctx context.Context,
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) (bool, error) {
	if err := r.validateConfigured(); err != nil {
		return false, err
	}

	normalizedAvatarID := strings.TrimSpace(avatarID)
	normalizedTargetID := strings.TrimSpace(targetID)

	if !isValidLikeReferenceID(normalizedAvatarID) {
		return false, likedom.ErrInvalidAvatarID
	}

	if !likedom.IsValidTargetType(targetType) {
		return false, likedom.ErrInvalidTargetType
	}

	if !isValidLikeReferenceID(normalizedTargetID) {
		return false, likedom.ErrInvalidTargetID
	}

	doc, err := r.itemDoc(
		normalizedAvatarID,
		targetType,
		normalizedTargetID,
	)
	if err != nil {
		return false, err
	}

	_, err = doc.Get(ctx)
	if err == nil {
		return true, nil
	}

	if status.Code(err) == codes.NotFound {
		return false, nil
	}

	return false, err
}

// ============================================================
// Create
// ============================================================

func (r *LikeRepositoryFS) Create(
	ctx context.Context,
	entity likedom.Like,
) (likedom.Like, error) {
	if err := r.validateConfigured(); err != nil {
		return likedom.Like{}, err
	}

	if err := entity.Validate(); err != nil {
		return likedom.Like{}, err
	}

	entity.AvatarID = strings.TrimSpace(entity.AvatarID)
	entity.TargetID = strings.TrimSpace(entity.TargetID)
	entity.CreatedAt = entity.CreatedAt.UTC()

	doc, err := r.itemDoc(
		entity.AvatarID,
		entity.TargetType,
		entity.TargetID,
	)
	if err != nil {
		return likedom.Like{}, err
	}

	_, err = doc.Create(
		ctx,
		likeToDocument(entity),
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return likedom.Like{}, likedom.ErrConflict
		}

		return likedom.Like{}, err
	}

	return entity, nil
}

// ============================================================
// Delete
// ============================================================

func (r *LikeRepositoryFS) Delete(
	ctx context.Context,
	avatarID string,
	targetType likedom.TargetType,
	targetID string,
) error {
	if err := r.validateConfigured(); err != nil {
		return err
	}

	normalizedAvatarID := strings.TrimSpace(avatarID)
	normalizedTargetID := strings.TrimSpace(targetID)

	if !isValidLikeReferenceID(normalizedAvatarID) {
		return likedom.ErrInvalidAvatarID
	}

	if !likedom.IsValidTargetType(targetType) {
		return likedom.ErrInvalidTargetType
	}

	if !isValidLikeReferenceID(normalizedTargetID) {
		return likedom.ErrInvalidTargetID
	}

	doc, err := r.itemDoc(
		normalizedAvatarID,
		targetType,
		normalizedTargetID,
	)
	if err != nil {
		return err
	}

	// Firestore Delete is effectively idempotent for this use case.
	// If the document is already absent, the favorite remains absent.
	_, err = doc.Delete(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	return nil
}

// ============================================================
// Sort / pagination
// ============================================================

func sortLikes(
	items []likedom.Like,
	sortSpec likedom.Sort,
) error {
	column := strings.TrimSpace(sortSpec.Column)
	if column == "" {
		column = "createdAt"
	}

	if column != "createdAt" {
		return likedom.ErrInvalid
	}

	order := sortSpec.Order
	if order == "" {
		order = likedom.SortDesc
	}

	if order != likedom.SortAsc &&
		order != likedom.SortDesc {
		return likedom.ErrInvalid
	}

	sort.SliceStable(
		items,
		func(i int, j int) bool {
			if order == likedom.SortAsc {
				return items[i].CreatedAt.Before(
					items[j].CreatedAt,
				)
			}

			return items[i].CreatedAt.After(
				items[j].CreatedAt,
			)
		},
	)

	return nil
}

func paginateLikes(
	items []likedom.Like,
	page likedom.Page,
) likedom.PageResult[likedom.Like] {
	pageNumber := page.Number
	perPage := page.PerPage

	if pageNumber <= 0 {
		pageNumber = 1
	}

	if perPage <= 0 {
		perPage = defaultLikePerPage
	}

	totalCount := len(items)

	totalPages := int(
		math.Ceil(
			float64(totalCount) /
				float64(perPage),
		),
	)

	if totalPages == 0 {
		totalPages = 1
	}

	start := (pageNumber - 1) * perPage

	if start >= totalCount {
		return likedom.PageResult[likedom.Like]{
			Items:      []likedom.Like{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNumber,
			PerPage:    perPage,
		}
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	pageItems := make(
		[]likedom.Like,
		end-start,
	)
	copy(pageItems, items[start:end])

	return likedom.PageResult[likedom.Like]{
		Items:      pageItems,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}
}

// ============================================================
// Helpers
// ============================================================

func likeDocumentID(
	targetType likedom.TargetType,
	targetID string,
) (string, error) {
	normalizedTargetID := strings.TrimSpace(targetID)

	if !likedom.IsValidTargetType(targetType) {
		return "", likedom.ErrInvalidTargetType
	}

	if !isValidLikeReferenceID(normalizedTargetID) {
		return "", likedom.ErrInvalidTargetID
	}

	return string(targetType) +
		"_" +
		normalizedTargetID, nil
}

func isValidLikeReferenceID(
	value string,
) bool {
	if value == "" {
		return false
	}

	if !utf8.ValidString(value) {
		return false
	}

	if len(value) > likedom.MaxReferenceIDLength {
		return false
	}

	if strings.Contains(value, "/") {
		return false
	}

	firstRune, _ := utf8.DecodeRuneInString(value)
	lastRune, _ := utf8.DecodeLastRuneInString(value)

	if unicode.IsSpace(firstRune) ||
		unicode.IsSpace(lastRune) {
		return false
	}

	return true
}
