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
//
// NOTE:
// 既存データ/実装差分により、UID が入っているフィールド名が揺れる可能性があるため、
// userId / userUid / userUID を順に探索する（OR クエリの代替）。
type MeAvatarRepo struct {
	Client *firestore.Client
}

func NewMeAvatarRepo(client *firestore.Client) *MeAvatarRepo {
	return &MeAvatarRepo{Client: client}
}

// ResolveAvatarByUID resolves avatarId(docId) + walletAddress by Firebase UID.
// This is the "extended" API required by /mall/me/avatar and AvatarContextMiddleware.
func (r *MeAvatarRepo) ResolveAvatarByUID(ctx context.Context, uid string) (string, string, error) {
	if r == nil || r.Client == nil {
		return "", "", errors.New("me_avatar_repo: firestore client is nil")
	}

	u := strings.TrimSpace(uid)
	if u == "" {
		return "", "", errors.New("me_avatar_repo: uid is empty")
	}

	// ✅ UID が格納されていそうなフィールド名を順に探索
	tryFields := []string{"userId", "userUid", "userUID"}

	var lastErr error
	for _, field := range tryFields {
		doc, err := r.resolveDocByField(ctx, field, u)
		if err == nil && doc != nil {
			avatarId := strings.TrimSpace(doc.Ref.ID) // ✅ docId が avatarId
			if avatarId == "" {
				return "", "", ErrAvatarNotFoundForUID
			}

			// walletAddress は実データ上 "walletAddress"
			walletAddress := ""
			if v, err := doc.DataAt("walletAddress"); err == nil {
				if s, ok := v.(string); ok {
					walletAddress = strings.TrimSpace(s)
				}
			}

			// walletAddress が空でも avatarId は返す（既存データ互換）
			return avatarId, walletAddress, nil
		}

		if errors.Is(err, ErrAvatarNotFoundForUID) {
			lastErr = err
			continue
		}
		if err != nil {
			return "", "", err
		}
	}

	if lastErr == nil {
		lastErr = ErrAvatarNotFoundForUID
	}
	return "", "", lastErr
}

// ResolveAvatarIDByUID resolves only avatarId(docId) by Firebase UID.
// Kept for backward compatibility.
func (r *MeAvatarRepo) ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("me_avatar_repo: firestore client is nil")
	}

	u := strings.TrimSpace(uid)
	if u == "" {
		return "", errors.New("me_avatar_repo: uid is empty")
	}

	// ✅ UID が格納されていそうなフィールド名を順に探索
	tryFields := []string{"userId", "userUid", "userUID"}

	var lastErr error
	for _, field := range tryFields {
		doc, err := r.resolveDocByField(ctx, field, u)
		if err == nil && doc != nil {
			avatarId := strings.TrimSpace(doc.Ref.ID) // ✅ docId が avatarId
			if avatarId == "" {
				return "", ErrAvatarNotFoundForUID
			}
			return avatarId, nil
		}

		if errors.Is(err, ErrAvatarNotFoundForUID) {
			lastErr = err
			continue
		}
		if err != nil {
			return "", err
		}
	}

	if lastErr == nil {
		lastErr = ErrAvatarNotFoundForUID
	}
	return "", lastErr
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

	if doc == nil || doc.Ref == nil || strings.TrimSpace(doc.Ref.ID) == "" {
		return nil, ErrAvatarNotFoundForUID
	}
	return doc, nil
}
