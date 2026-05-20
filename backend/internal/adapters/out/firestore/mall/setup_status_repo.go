// backend/internal/adapters/out/firestore/mall/setup_status_repo.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"
)

// SetupStatusRepoFirestore implements mall.SetupStatusRepo backed by Firestore.
//
// Strategy:
// - Avatar is checked by owner field because avatar document id is avatarId.
type SetupStatusRepoFirestore struct {
	Client *firestore.Client

	// Collection names (top-level collections by default)
	AvatarCollection string

	// Field name used to identify avatar owner.
	AvatarOwnerField string
}

const (
	defaultAvatarCollection = "avatars"

	defaultAvatarOwnerField = "userId"
)

func NewSetupStatusRepoFirestore(client *firestore.Client) *SetupStatusRepoFirestore {
	return &SetupStatusRepoFirestore{
		Client:           client,
		AvatarCollection: defaultAvatarCollection,
		AvatarOwnerField: defaultAvatarOwnerField,
	}
}

func (r *SetupStatusRepoFirestore) HasAvatar(ctx context.Context, uid string) (bool, error) {
	return r.existsAvatarByOwner(ctx, uid)
}

// ------------------------------------------------------------
// Helpers

func (r *SetupStatusRepoFirestore) existsAvatarByOwner(
	ctx context.Context,
	uid string,
) (bool, error) {
	if r == nil || r.Client == nil {
		return false, nil
	}
	if r.AvatarCollection == "" || uid == "" {
		return false, nil
	}

	ownerField := r.AvatarOwnerField
	if ownerField == "" {
		ownerField = defaultAvatarOwnerField
	}

	iter := r.Client.Collection(r.AvatarCollection).
		Where(ownerField, "==", uid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	docs, err := iter.GetAll()
	if err != nil {
		return false, err
	}

	return len(docs) > 0, nil
}
