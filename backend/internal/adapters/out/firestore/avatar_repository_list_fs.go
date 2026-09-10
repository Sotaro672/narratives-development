// backend/internal/adapters/out/firestore/avatar_repository_list_fs.go
package firestore

import (
	"context"

	"cloud.google.com/go/firestore"

	avdom "narratives/internal/domain/avatar"
)

// ListAll returns all avatars ordered by createdAt descending.
func (r *AvatarRepositoryFS) ListAll(
	ctx context.Context,
) ([]avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return nil, errBadClient
	}

	snapshots, err := r.col().
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	avatars := make([]avdom.Avatar, 0, len(snapshots))

	for _, snapshot := range snapshots {
		avatar, err := r.docToDomain(snapshot)
		if err != nil {
			return nil, err
		}

		avatars = append(avatars, avatar)
	}

	return avatars, nil
}
