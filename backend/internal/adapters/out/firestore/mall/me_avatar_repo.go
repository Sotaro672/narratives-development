package mall

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var ErrAvatarNotFoundForUID = errors.New("avatar_not_found_for_uid")

// MeAvatarRepo resolves avatarId(docId) by Firebase UID (userId field).
type MeAvatarRepo struct {
	Client *firestore.Client
}

func NewMeAvatarRepo(client *firestore.Client) *MeAvatarRepo {
	return &MeAvatarRepo{Client: client}
}

func (r *MeAvatarRepo) ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("me_avatar_repo: firestore client is nil")
	}

	u := strings.TrimSpace(uid)
	if u == "" {
		return "", errors.New("me_avatar_repo: uid is empty")
	}

	// avatars where userId == uid
	iter := r.Client.Collection("avatars").
		Where("userId", "==", u).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return "", ErrAvatarNotFoundForUID
	}
	if err != nil {
		return "", err
	}

	avatarId := strings.TrimSpace(doc.Ref.ID) // ✅ docId が avatarId
	if avatarId == "" {
		return "", ErrAvatarNotFoundForUID
	}
	return avatarId, nil
}
