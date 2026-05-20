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

// Firestore implementation of avatarstate.Repository.
//
// Firestore schema:
//
// Collection:
//   - avatar_states/{avatarId}
//
// Parent document fields:
//   - followerCount  int64
//   - followingCount int64
//   - postCount      int64
//   - lastActiveAt   timestamp
//   - updatedAt      timestamp
//
// Parent document does not store avatarId as a field.
// The parent document ID is the avatarId.
//
// Subcollections:
//   - avatar_states/{avatarId}/followers/{followerAvatarId}
//   - avatar_states/{avatarId}/following/{followingAvatarId}
//
// Follow document fields:
//   - avatarId    string
//   - followedAt  timestamp
//
// followerCount/followingCount are cached counts.
// followers/following subcollections are the source of truth when synced.
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

// Upsert upserts avatar state.
// The avatar state document ID is avatarID.
func (r *AvatarStateRepositoryFS) Upsert(ctx context.Context, state avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: client is nil")
	}

	avatarID := state.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: avatarID is empty")
	}

	data := avatarStateParentData(state)

	if _, err := r.col().Doc(avatarID).Set(ctx, data, firestore.MergeAll); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}

	if shouldReplaceFollowers(state) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(state.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	if shouldReplaceFollowing(state) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(state.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	return r.GetByAvatarID(ctx, avatarID)
}

// ==============================
// Get
// ==============================

// GetByID gets avatar state by avatarID.
// This method exists for common repository compatibility.
func (r *AvatarStateRepositoryFS) GetByID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	return r.GetByAvatarID(ctx, avatarID)
}

// GetByAvatarID gets avatar state by avatarID.
// The avatarID is the parent document ID.
func (r *AvatarStateRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	if avatarID == "" {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}

	doc, err := r.col().Doc(avatarID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	return r.docToDomain(ctx, doc)
}

// ==============================
// Exists
// ==============================

func (r *AvatarStateRepositoryFS) Exists(ctx context.Context, avatarID string) (bool, error) {
	if avatarID == "" {
		return false, nil
	}

	_, err := r.col().Doc(avatarID).Get(ctx)
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

// Create creates avatar state.
// The avatar state document ID is avatarID.
func (r *AvatarStateRepositoryFS) Create(ctx context.Context, state avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: client is nil")
	}

	avatarID := state.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: avatarID is empty")
	}

	data := avatarStateParentData(state)

	_, err := r.col().Doc(avatarID).Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}

	if shouldReplaceFollowers(state) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(state.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	if shouldReplaceFollowing(state) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(state.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	return r.GetByAvatarID(ctx, avatarID)
}

// Update updates avatar state by avatarID.
// This method exists for common repository compatibility.
func (r *AvatarStateRepositoryFS) Update(ctx context.Context, avatarID string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.UpdateByAvatarID(ctx, avatarID, patch)
}

// UpdateByAvatarID updates avatar state by avatarID.
// The avatarID is the parent document ID.
func (r *AvatarStateRepositoryFS) UpdateByAvatarID(ctx context.Context, avatarID string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	if avatarID == "" {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}

	return r.updateByAvatarID(ctx, avatarID, patch)
}

func (r *AvatarStateRepositoryFS) updateByAvatarID(
	ctx context.Context,
	avatarID string,
	patch avatarstate.AvatarStatePatch,
) (avatarstate.AvatarState, error) {
	ref := r.col().Doc(avatarID)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	updates := make([]firestore.Update, 0)

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

	updatedAt := time.Now().UTC()
	if patch.UpdatedAt != nil {
		updatedAt = patch.UpdatedAt.UTC()
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: updatedAt,
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	return r.GetByAvatarID(ctx, avatarID)
}

// Delete deletes avatar state by avatarID.
// This method exists for common repository compatibility.
func (r *AvatarStateRepositoryFS) Delete(ctx context.Context, avatarID string) error {
	return r.DeleteByAvatarID(ctx, avatarID)
}

// DeleteByAvatarID deletes avatar state by avatarID.
// The avatarID is the parent document ID.
func (r *AvatarStateRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	if avatarID == "" {
		return avatarstate.ErrNotFound
	}

	ref := r.col().Doc(avatarID)

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

// Save overwrites the avatar state parent document.
// The avatar state document ID is avatarID.
func (r *AvatarStateRepositoryFS) Save(ctx context.Context, state avatarstate.AvatarState, _ *avatarstate.SaveOptions) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: client is nil")
	}

	avatarID := state.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatar_state_repository_fs: avatarID is empty")
	}

	data := avatarStateParentData(state)

	if _, err := r.col().Doc(avatarID).Set(ctx, data); err != nil {
		return avatarstate.AvatarState{}, err
	}

	if shouldReplaceFollowers(state) {
		if err := r.replaceFollowRefs(ctx, r.followersCol(avatarID), sliceOrEmpty(state.Followers)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	if shouldReplaceFollowing(state) {
		if err := r.replaceFollowRefs(ctx, r.followingCol(avatarID), sliceOrEmpty(state.Following)); err != nil {
			return avatarstate.AvatarState{}, err
		}
	}

	return r.GetByAvatarID(ctx, avatarID)
}

// ==============================
// Mapping
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

	followerCount := raw.FollowerCount
	if followers != nil {
		followerCount = int64Ptr(int64(len(followers)))
	}

	followingCount := raw.FollowingCount
	if following != nil {
		followingCount = int64Ptr(int64(len(following)))
	}

	return avatarstate.New(
		avatarID,
		followerCount,
		followingCount,
		raw.PostCount,
		sliceOrEmpty(followers),
		sliceOrEmpty(following),
		raw.LastActiveAt.UTC(),
		raw.UpdatedAt,
	)
}

func avatarStateParentData(state avatarstate.AvatarState) map[string]any {
	now := time.Now().UTC()

	lastActiveAt := state.LastActiveAt
	if lastActiveAt.IsZero() {
		lastActiveAt = now
	}

	updatedAt := now
	if state.UpdatedAt != nil {
		updatedAt = state.UpdatedAt.UTC()
	}

	followerCount := state.FollowerCount
	if state.Followers != nil {
		followerCount = int64Ptr(int64(len(state.Followers)))
	}

	followingCount := state.FollowingCount
	if state.Following != nil {
		followingCount = int64Ptr(int64(len(state.Following)))
	}

	data := map[string]any{
		"lastActiveAt": lastActiveAt.UTC(),
		"updatedAt":    updatedAt.UTC(),
	}

	if followerCount != nil {
		data["followerCount"] = *followerCount
	}

	if followingCount != nil {
		data["followingCount"] = *followingCount
	}

	if state.PostCount != nil {
		data["postCount"] = *state.PostCount
	}

	return data
}

// ==============================
// Follow refs
// ==============================

func (r *AvatarStateRepositoryFS) listFollowRefs(ctx context.Context, col *firestore.CollectionRef) ([]avatarstate.AvatarFollowRef, error) {
	if col == nil {
		return []avatarstate.AvatarFollowRef{}, nil
	}

	iter := col.Documents(ctx)
	defer iter.Stop()

	refs := make([]avatarstate.AvatarFollowRef, 0)

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

		followAvatarID := raw.AvatarID
		if followAvatarID == "" {
			followAvatarID = doc.Ref.ID
		}

		refs = append(refs, avatarstate.AvatarFollowRef{
			AvatarID:   followAvatarID,
			FollowedAt: raw.FollowedAt.UTC(),
		})
	}

	return refs, nil
}

func (r *AvatarStateRepositoryFS) replaceFollowRefs(
	ctx context.Context,
	col *firestore.CollectionRef,
	refs []avatarstate.AvatarFollowRef,
) error {
	if col == nil {
		return errors.New("avatar_state_repository_fs: follow subcollection is nil")
	}

	if err := r.deleteAllDocs(ctx, col); err != nil {
		return err
	}

	if len(refs) == 0 {
		return nil
	}

	batch := r.Client.Batch()

	for _, ref := range refs {
		followAvatarID := ref.AvatarID
		if followAvatarID == "" {
			return errors.New("avatar_state_repository_fs: follow avatarID is empty")
		}

		followedAt := ref.FollowedAt
		if followedAt.IsZero() {
			followedAt = time.Now().UTC()
		}

		batch.Set(col.Doc(followAvatarID), map[string]any{
			"avatarId":   followAvatarID,
			"followedAt": followedAt.UTC(),
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

// ==============================
// Helpers
// ==============================

func shouldReplaceFollowers(state avatarstate.AvatarState) bool {
	if state.Followers != nil {
		return true
	}

	return state.FollowerCount != nil && *state.FollowerCount == 0
}

func shouldReplaceFollowing(state avatarstate.AvatarState) bool {
	if state.Following != nil {
		return true
	}

	return state.FollowingCount != nil && *state.FollowingCount == 0
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
		followedAt := item.FollowedAt
		if followedAt.IsZero() {
			followedAt = time.Now().UTC()
		}

		out = append(out, avatarstate.AvatarFollowRef{
			AvatarID:   item.AvatarID,
			FollowedAt: followedAt.UTC(),
		})
	}

	return out
}

func int64Ptr(v int64) *int64 {
	return &v
}
