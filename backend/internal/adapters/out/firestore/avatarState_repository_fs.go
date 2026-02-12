// backend/internal/adapters/out/firestore/avatarState_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avatarstate "narratives/internal/domain/avatarState"
)

// Firestore implementation of avatarState.Repository
//
// ✅ Collection design (after change):
// - collection: avatar_states
// - docId: avatarId
// - fields: followerCount, followingCount, postCount, lastActiveAt, updatedAt
// - ❌ avatarId field is NOT stored (docId is the source of truth).
type AvatarStateRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarStateRepositoryFS(client *firestore.Client) *AvatarStateRepositoryFS {
	return &AvatarStateRepositoryFS{Client: client}
}

func (r *AvatarStateRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatar_states")
}

// ==============================
// Upsert (required by usecase.AvatarStateRepo)
// ==============================

// Upsert upserts avatar_state for the given AvatarState (docId=avatarId).
// - If not found: create with zero counts (unless provided)
// - If exists: update provided fields + touch updatedAt (and lastActiveAt if provided)
func (r *AvatarStateRepositoryFS) Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: client is nil")
	}

	avatarID := s.ID // ✅ docId = avatarId
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: id(avatarId) is empty")
	}

	now := time.Now().UTC()
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	ref := r.col().Doc(avatarID)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, gerr := tx.Get(ref)
		if gerr != nil {
			// not found -> create
			if status.Code(gerr) == codes.NotFound {
				zero := int64(0)
				data := r.domainToDocDataForCreate(s, &zero, now)
				return tx.Create(ref, data)
			}
			return gerr
		}

		// exists -> update (only provided)
		var updates []firestore.Update

		if s.FollowerCount != nil {
			updates = append(updates, firestore.Update{Path: "followerCount", Value: *s.FollowerCount})
		}
		if s.FollowingCount != nil {
			updates = append(updates, firestore.Update{Path: "followingCount", Value: *s.FollowingCount})
		}
		if s.PostCount != nil {
			updates = append(updates, firestore.Update{Path: "postCount", Value: *s.PostCount})
		}

		// lastActiveAt は「指定があるなら反映」。TouchLastActive はこれを使う想定。
		if !s.LastActiveAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "lastActiveAt", Value: s.LastActiveAt.UTC()})
		}

		// updatedAt は常に更新
		if s.UpdatedAt != nil {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: s.UpdatedAt.UTC()})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: now})
		}

		return tx.Update(ref, updates)
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}

	latest, rerr := r.GetByID(ctx, avatarID)
	if rerr != nil {
		return avatarstate.AvatarState{}, rerr
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
	return r.docToDomain(doc)
}

// GetByAvatarID is now identical to GetByID (docId=avatarId).
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
// (If the document already exists, returns ErrConflict)
func (r *AvatarStateRepositoryFS) Create(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	if r == nil || r.Client == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: client is nil")
	}

	avatarID := s.ID
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatarState_repository_fs: id(avatarId) is empty")
	}

	now := time.Now().UTC()
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	ref := r.col().Doc(avatarID)

	_, err := ref.Create(ctx, r.domainToDocData(s))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}
	return s, nil
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

	if len(updates) == 0 {
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avatarstate.AvatarState{}, avatarstate.ErrNotFound
			}
			return avatarstate.AvatarState{}, err
		}
		return r.docToDomain(snap)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(snap)
}

func (r *AvatarStateRepositoryFS) Delete(ctx context.Context, id string) error {
	if id == "" {
		return avatarstate.ErrNotFound
	}
	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return avatarstate.ErrNotFound
	}
	_, err := ref.Delete(ctx)
	return err
}

// DeleteByAvatarID is now identical to Delete (docId=avatarId).
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
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	ref := r.col().Doc(avatarID)

	_, err := ref.Set(ctx, r.domainToDocData(s))
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return s, nil
}

// ==============================
// Helpers
// ==============================

func (r *AvatarStateRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (avatarstate.AvatarState, error) {
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

	return avatarstate.New(
		avatarID, // id (=avatarId)
		raw.FollowerCount,
		raw.FollowingCount,
		raw.PostCount,
		raw.LastActiveAt.UTC(),
		raw.UpdatedAt,
	)
}

func (r *AvatarStateRepositoryFS) domainToDocData(s avatarstate.AvatarState) map[string]any {
	// ✅ avatarId は保存しない（docId が source of truth）
	data := map[string]any{
		"lastActiveAt": s.LastActiveAt,
	}
	if s.FollowerCount != nil {
		data["followerCount"] = *s.FollowerCount
	}
	if s.FollowingCount != nil {
		data["followingCount"] = *s.FollowingCount
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	}
	if s.UpdatedAt != nil {
		data["updatedAt"] = s.UpdatedAt.UTC()
	}
	return data
}

func (r *AvatarStateRepositoryFS) domainToDocDataForCreate(s avatarstate.AvatarState, zero *int64, now time.Time) map[string]any {
	data := map[string]any{
		"lastActiveAt": s.LastActiveAt.UTC(),
		"updatedAt":    now,
	}
	if s.UpdatedAt != nil {
		data["updatedAt"] = s.UpdatedAt.UTC()
	}
	if s.FollowerCount != nil {
		data["followerCount"] = *s.FollowerCount
	} else if zero != nil {
		data["followerCount"] = *zero
	}
	if s.FollowingCount != nil {
		data["followingCount"] = *s.FollowingCount
	} else if zero != nil {
		data["followingCount"] = *zero
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	} else if zero != nil {
		data["postCount"] = *zero
	}
	return data
}
