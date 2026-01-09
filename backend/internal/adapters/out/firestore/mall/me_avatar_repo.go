// backend/internal/adapters/out/firestore/mall/me_avatar_repo.go
package mall

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var ErrAvatarNotFoundForUID = errors.New("avatar_not_found_for_uid")

// MeAvatarRepo resolves avatarId(docId) by Firebase UID.
// NOTE:
// 既存データ/実装差分により、UID が入っているフィールド名が揺れる可能性があるため、
// userId / userUid / userUID を順に探索する（OR クエリの代替）。
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

	// ✅ UID が格納されていそうなフィールド名を順に探索
	// - userId   : 旧実装/別概念で UID を入れていたケース
	// - userUid  : API(AvatarHandler) が送っている Firebase UID の名前
	// - userUID  : 名寄せ揺れ（大文字混在）
	tryFields := []string{"userId", "userUid", "userUID"}

	for _, field := range tryFields {
		avatarId, err := r.resolveByField(ctx, field, u)
		if err == nil && strings.TrimSpace(avatarId) != "" {
			return strings.TrimSpace(avatarId), nil
		}
		if errors.Is(err, ErrAvatarNotFoundForUID) {
			continue
		}
		if err != nil {
			return "", err
		}
	}

	return "", ErrAvatarNotFoundForUID
}

func (r *MeAvatarRepo) resolveByField(ctx context.Context, field string, uid string) (string, error) {
	iter := r.Client.Collection("avatars").
		Where(field, "==", uid).
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
