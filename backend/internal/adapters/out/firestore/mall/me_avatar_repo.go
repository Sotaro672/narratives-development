// backend/internal/adapters/out/firestore/mall/me_avatar_repo.go
package mall

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var ErrAvatarNotFoundForUID = errors.New("avatar_not_found_for_uid")

// MeAvatarRepo resolves avatarId(docId) by Firebase UID.
// Firestore schema:
// avatars/{avatarId}.userId == Firebase Auth UID
type MeAvatarRepo struct {
	Client *firestore.Client
}

func NewMeAvatarRepo(client *firestore.Client) *MeAvatarRepo {
	return &MeAvatarRepo{Client: client}
}

// ResolveAvatarByUID resolves avatarId(docId) + walletAddress by Firebase UID.
// Used by /mall/me/avatar and AvatarContextMiddleware.
func (r *MeAvatarRepo) ResolveAvatarByUID(ctx context.Context, uid string) (string, string, error) {
	if r == nil || r.Client == nil {
		return "", "", errors.New("me_avatar_repo: firestore client is nil")
	}
	if uid == "" {
		return "", "", errors.New("me_avatar_repo: uid is empty")
	}

	doc, err := r.resolveDocByField(ctx, "userId", uid)
	if err != nil {
		return "", "", err
	}

	avatarId := doc.Ref.ID
	if avatarId == "" {
		return "", "", ErrAvatarNotFoundForUID
	}

	walletAddress := ""
	if v, err := doc.DataAt("walletAddress"); err == nil {
		if s, ok := v.(string); ok {
			walletAddress = s
		}
	}

	return avatarId, walletAddress, nil
}

func (r *MeAvatarRepo) resolveDocByField(ctx context.Context, field string, uid string) (*firestore.DocumentSnapshot, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("me_avatar_repo: firestore client is nil")
	}

	iter := r.Client.Collection("avatars").
		Where(field, "==", uid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, ErrAvatarNotFoundForUID
	}
	if err != nil {
		return nil, err
	}

	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return nil, ErrAvatarNotFoundForUID
	}
	return doc, nil
}
