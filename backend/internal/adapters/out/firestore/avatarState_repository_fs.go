// backend/internal/adapters/out/firestore/avatarState_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avatarstate "narratives/internal/domain/avatarState"
)

// Firestore implementation of avatarState.Repository
//
// Collection design:
// - collection: avatar_states
// - docId: avatarId
// - fields: followerCount, followingCount, postCount, lastActiveAt, updatedAt
// - subcollections:
//   - followers/{followerAvatarId}
//   - following/{followingAvatarId}
//
// - avatarId field is NOT stored in parent document (docId is source of truth).
type AvatarStateRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarStateRepositoryFS(client *firestore.Client) *AvatarStateRepositoryFS {
	return &AvatarStateRepositoryFS{Client: client}
}

func (r *AvatarStateRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatar_states")
}

func (r *AvatarStateRepositoryFS) followersCol(avatarID string) *firestore.CollectionRef {
	return r.col().Doc(avatarID).Collection("followers")
}

func (r *AvatarStateRepositoryFS) followingCol(avatarID string) *firestore.CollectionRef {
	return r.col().Doc(avatarID).Collection("following")
}

// ==============================
// Upsert
// ==============================

// Upsert upserts avatar_state for the given AvatarState (docId=avatarId).
func (r *AvatarStateRepositoryFS) Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: client is nil")
	}

	avatarID := s.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: id(avatarId) is empty")
	}

	now := time.Now().UTC()

	lastActiveAt := s.LastActiveAt
	if lastActiveAt.IsZero() {
		lastActiveAt = now
	}

	updatedAt := now
	if s.UpdatedAt != nil {
		updatedAt = s.UpdatedAt.UTC()
	}

	followerCountValue := s.FollowerCount
	if s.Followers != nil {
		followerCountValue = int64Ptr(int64(len(s.Followers)))
	}

	followingCountValue := s.FollowingCount
	if s.Following != nil {
		followingCountValue = int64Ptr(int64(len(s.Following)))
	}

	data := map[string]any{
		"lastActiveAt": lastActiveAt.UTC(),
		"updatedAt":    updatedAt.UTC(),
	}

	if followerCountValue != nil {
		data["followerCount"] = *followerCountValue
	}
	if followingCountValue != nil {
		data["followingCount"] = *followingCountValue
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	}

	if _, err := r.col().Doc(avatarID).Set(ctx, data, firestore.MergeAll); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}

	if shouldSyncFollowersOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(s.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}
	if shouldSyncFollowingOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(s.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	latest, err := r.GetByID(ctx, avatarID)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return latest, nil
}

// ==============================
// GetByID / GetByAvatarID
// ==============================

func (r *AvatarStateRepositoryFS) GetByID(ctx context.Context, id string) (avatarstate.AvatarState, error) {
	if id == "" {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(ctx, doc)
}

// GetByAvatarID is identical to GetByID (docId=avatarId).
func (r *AvatarStateRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	return r.GetByID(ctx, avatarID)
}

// ==============================
// Exists
// ==============================

func (r *AvatarStateRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// ==============================
// Create / Update / Delete / Save
// ==============================

// Create creates a document with docId=avatarId.
// If the document already exists, returns ErrConflict.
func (r *AvatarStateRepositoryFS) Create(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: client is nil")
	}

	avatarID := s.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: id(avatarId) is empty")
	}

	now := time.Now().UTC()

	lastActiveAt := s.LastActiveAt
	if lastActiveAt.IsZero() {
		lastActiveAt = now
	}

	updatedAt := now
	if s.UpdatedAt != nil {
		updatedAt = s.UpdatedAt.UTC()
	}

	followerCountValue := s.FollowerCount
	if s.Followers != nil {
		followerCountValue = int64Ptr(int64(len(s.Followers)))
	}

	followingCountValue := s.FollowingCount
	if s.Following != nil {
		followingCountValue = int64Ptr(int64(len(s.Following)))
	}

	data := map[string]any{
		"lastActiveAt": lastActiveAt.UTC(),
		"updatedAt":    updatedAt.UTC(),
	}
	if followerCountValue != nil {
		data["followerCount"] = *followerCountValue
	}
	if followingCountValue != nil {
		data["followingCount"] = *followingCountValue
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	}

	_, err := r.col().Doc(avatarID).Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}

	if shouldSyncFollowersOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(s.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}
	if shouldSyncFollowingOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(s.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	return r.GetByID(ctx, avatarID)
}

func (r *AvatarStateRepositoryFS) Update(ctx context.Context, id string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.updateBy(ctx, r.col().Doc(id), patch)
}

func (r *AvatarStateRepositoryFS) UpdateByAvatarID(ctx context.Context, avatarID string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.updateBy(ctx, r.col().Doc(avatarID), patch)
}

func (r *AvatarStateRepositoryFS) updateBy(
	ctx context.Context,
	ref *firestore.DocumentRef,
	patch avatarstate.AvatarStatePatch,
) (avatarstate.AvatarState, error) {
	if ref == nil || ref.ID == "" {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	var updates []firestore.Update

	if patch.FollowerCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "followerCount",
			Value: *patch.FollowerCount,
		})
	}
	if patch.FollowingCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "followingCount",
			Value: *patch.FollowingCount,
		})
	}
	if patch.PostCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "postCount",
			Value: *patch.PostCount,
		})
	}

	if patch.Followers != nil {
		followers := cloneFollowRefs(*patch.Followers)
		if err := r.replaceFollowRefs(ctx, ref.Collection("followers"), followers); err != nil {
			return avatarstate.AvatarState{}, err
		}
		updates = append(updates, firestore.Update{
			Path:  "followerCount",
			Value: int64(len(followers)),
		})
	}

	if patch.Following != nil {
		following := cloneFollowRefs(*patch.Following)
		if err := r.replaceFollowRefs(ctx, ref.Collection("following"), following); err != nil {
			return avatarstate.AvatarState{}, err
		}
		updates = append(updates, firestore.Update{
			Path:  "followingCount",
			Value: int64(len(following)),
		})
	}

	if patch.LastActiveAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "lastActiveAt",
			Value: patch.LastActiveAt.UTC(),
		})
	}

	now := time.Now().UTC()
	if patch.UpdatedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: patch.UpdatedAt.UTC(),
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: now,
		})
	}

	if len(updates) > 0 {
		if _, err := ref.Update(ctx, updates); err != nil {
			if status.Code(err) == codes.NotFound {
				return avatarstate.AvatarState{}, avatarstate.ErrNotFound
			}
			return avatarstate.AvatarState{}, err
		}
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(ctx, snap)
}

func (r *AvatarStateRepositoryFS) Delete(ctx context.Context, id string) error {
	if id == "" {
		return avatarstate.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.ErrNotFound
		}
		return err
	}

	if err := r.deleteAllDocs(ctx, ref.Collection("followers")); err != nil {
		return err
	}
	if err := r.deleteAllDocs(ctx, ref.Collection("following")); err != nil {
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

// DeleteByAvatarID is identical to Delete (docId=avatarId).
func (r *AvatarStateRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	return r.Delete(ctx, avatarID)
}

// Save overwrites full document with docId=avatarId.
func (r *AvatarStateRepositoryFS) Save(ctx context.Context, s avatarstate.AvatarState, _ *avatarstate.SaveOptions) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: client is nil")
	}

	avatarID := s.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: id(avatarId) is empty")
	}

	now := time.Now().UTC()

	lastActiveAt := s.LastActiveAt
	if lastActiveAt.IsZero() {
		lastActiveAt = now
	}

	updatedAt := now
	if s.UpdatedAt != nil {
		updatedAt = s.UpdatedAt.UTC()
	}

	followerCountValue := s.FollowerCount
	if s.Followers != nil {
		followerCountValue = int64Ptr(int64(len(s.Followers)))
	}

	followingCountValue := s.FollowingCount
	if s.Following != nil {
		followingCountValue = int64Ptr(int64(len(s.Following)))
	}

	data := map[string]any{
		"lastActiveAt": lastActiveAt.UTC(),
		"updatedAt":    updatedAt.UTC(),
	}
	if followerCountValue != nil {
		data["followerCount"] = *followerCountValue
	}
	if followingCountValue != nil {
		data["followingCount"] = *followingCountValue
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	}

	if _, err := r.col().Doc(avatarID).Set(ctx, data); err != nil {
		return avatarstate.AvatarState{}, err
	}

	if shouldSyncFollowersOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(s.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}
	if shouldSyncFollowingOnState(s) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(s.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	return r.GetByID(ctx, avatarID)
}

// ==============================
// Helpers
// ==============================

func (r *AvatarStateRepositoryFS) docToDomain(ctx context.Context, doc *firestore.DocumentSnapshot) (avatarstate.AvatarState, error) {
	var raw struct {
		FollowerCount  *int64     `firestore:"followerCount"`
		FollowingCount *int64     `firestore:"followingCount"`
		PostCount      *int64     `firestore:"postCount"`
		LastActiveAt   time.Time  `firestore:"lastActiveAt"`
		UpdatedAt      *time.Time `firestore:"updatedAt"`
	}
	if err := doc.DataTo(&raw); err != nil {
		return avatarstate.AvatarState{}, err
	}

	avatarID := doc.Ref.ID

	followers, err := r.listFollowRefs(ctx, doc.Ref.Collection("followers"))
	if err != nil {
		return avatarstate.AvatarState{}, err
	}

	following, err := r.listFollowRefs(ctx, doc.Ref.Collection("following"))
	if err != nil {
		return avatarstate.AvatarState{}, err
	}

	effectiveFollowerCount := raw.FollowerCount
	if followers != nil {
		effectiveFollowerCount = int64Ptr(int64(len(followers)))
	}

	effectiveFollowingCount := raw.FollowingCount
	if following != nil {
		effectiveFollowingCount = int64Ptr(int64(len(following)))
	}

	if effectiveFollowerCount == nil && raw.FollowerCount != nil {
		effectiveFollowerCount = raw.FollowerCount
	}
	if effectiveFollowingCount == nil && raw.FollowingCount != nil {
		effectiveFollowingCount = raw.FollowingCount
	}

	return avatarstate.New(
		avatarID,
		effectiveFollowerCount,
		effectiveFollowingCount,
		raw.PostCount,
		sliceOrEmpty(followers),
		sliceOrEmpty(following),
		raw.LastActiveAt.UTC(),
		raw.UpdatedAt,
	)
}

func (r *AvatarStateRepositoryFS) listFollowRefs(ctx context.Context, col *firestore.CollectionRef) ([]avatarstate.AvatarFollowRef, error) {
	if col == nil {
		return []avatarstate.AvatarFollowRef{}, nil
	}

	iter := col.Documents(ctx)
	defer iter.Stop()

	out := make([]avatarstate.AvatarFollowRef, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var raw struct {
			AvatarID   string    `firestore:"avatarId"`
			FollowedAt time.Time `firestore:"followedAt"`
		}
		if err := doc.DataTo(&raw); err != nil {
			return nil, err
		}

		avatarID := raw.AvatarID
		if avatarID == "" {
			avatarID = doc.Ref.ID
		}

		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   avatarID,
			FollowedAt: raw.FollowedAt.UTC(),
		})
	}

	return out, nil
}

func (r *AvatarStateRepositoryFS) replaceFollowRefs(
	ctx context.Context,
	col *firestore.CollectionRef,
	refs []avatarstate.AvatarFollowRef,
) error {
	if col == nil {
		return errors.New("avatarState_repository_fs: subcollection is nil")
	}

	if err := r.deleteAllDocs(ctx, col); err != nil {
		return err
	}

	if len(refs) == 0 {
		return nil
	}

	batch := r.Client.Batch()
	for _, ref := range refs {
		docID := ref.AvatarID
		if docID == "" {
			return avatarstate.ErrInvalidFollowingAvatarID
		}
		batch.Set(col.Doc(docID), map[string]any{
			"avatarId":   ref.AvatarID,
			"followedAt": ref.FollowedAt.UTC(),
		})
	}

	_, err := batch.Commit(ctx)
	return err
}

func (r *AvatarStateRepositoryFS) deleteAllDocs(ctx context.Context, col *firestore.CollectionRef) error {
	if col == nil {
		return nil
	}

	iter := col.Documents(ctx)
	defer iter.Stop()

	batch := r.Client.Batch()
	count := 0

	commit := func() error {
		if count == 0 {
			return nil
		}
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
		batch = r.Client.Batch()
		count = 0
		return nil
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		batch.Delete(doc.Ref)
		count++

		if count >= 400 {
			if err := commit(); err != nil {
				return err
			}
		}
	}

	return commit()
}

func shouldSyncFollowersOnState(s avatarstate.AvatarState) bool {
	if s.Followers != nil {
		return true
	}
	return s.FollowerCount != nil && *s.FollowerCount == 0
}

func shouldSyncFollowingOnState(s avatarstate.AvatarState) bool {
	if s.Following != nil {
		return true
	}
	return s.FollowingCount != nil && *s.FollowingCount == 0
}

func sliceOrEmpty(in []avatarstate.AvatarFollowRef) []avatarstate.AvatarFollowRef {
	if len(in) == 0 {
		return []avatarstate.AvatarFollowRef{}
	}
	return cloneFollowRefs(in)
}

func cloneFollowRefs(in []avatarstate.AvatarFollowRef) []avatarstate.AvatarFollowRef {
	if len(in) == 0 {
		return []avatarstate.AvatarFollowRef{}
	}

	out := make([]avatarstate.AvatarFollowRef, 0, len(in))
	for _, item := range in {
		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   item.AvatarID,
			FollowedAt: item.FollowedAt.UTC(),
		})
	}
	return out
}

func int64Ptr(v int64) *int64 {
	return &v
}
