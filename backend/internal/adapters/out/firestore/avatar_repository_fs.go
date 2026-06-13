// backend/internal/adapters/out/firestore/avatar_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avdom "narratives/internal/domain/avatar"
)

// Firestore implementation of avatar.Repository.
type AvatarRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarRepositoryFS(client *firestore.Client) *AvatarRepositoryFS {
	return &AvatarRepositoryFS{Client: client}
}

func (r *AvatarRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatars")
}

// Compile-time check: ensure AvatarRepositoryFS satisfies avatar.Repository.
var _ avdom.Repository = (*AvatarRepositoryFS)(nil)

var (
	errNotFound             = errors.New("avatar: not found")
	errAvatarNotFoundForUID = errors.New("avatar_not_found_for_uid")
	errConflict             = errors.New("avatar: conflict")
	errBadClient            = errors.New("firestore client is nil")
	errInvalidWalletAddr    = errors.New("avatar: invalid walletAddress")
	errWalletAlreadyBound   = errors.New("avatar: walletAddress already set")
)

// ==============================
// GetByID
// ==============================

func (r *AvatarRepositoryFS) GetByID(ctx context.Context, id string) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}
	if id == "" {
		return avdom.Avatar{}, errNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// GetByUserID
// ==============================
//
// Avatar document id は avatarId であり userId ではないため、
// Firebase UID / userId から avatar document を取得する。
// uid -> avatarId 解決や mall/me/avatar 判定で使用する。
func (r *AvatarRepositoryFS) GetByUserID(ctx context.Context, userID string) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}
	if userID == "" {
		return avdom.Avatar{}, errNotFound
	}

	q := r.col().Where("userId", "==", userID).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return avdom.Avatar{}, errNotFound
	}

	return r.docToDomain(doc)
}

// ==============================
// ResolveAvatarByUID
// ==============================
//
// Avatar document id は avatarId であり userId ではないため、
// Firebase UID から現在の avatarId と walletAddress を解決する。
// AvatarContextMiddleware / /mall/me/avatars 系の uid -> avatar 解決で使用する。
func (r *AvatarRepositoryFS) ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error) {
	if r == nil || r.Client == nil {
		return "", "", errBadClient
	}
	if uid == "" {
		return "", "", errAvatarNotFoundForUID
	}

	q := r.col().Where("userId", "==", uid).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return "", "", errAvatarNotFoundForUID
	}
	if err != nil {
		return "", "", err
	}
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return "", "", errAvatarNotFoundForUID
	}

	avatarID = doc.Ref.ID

	if v, err := doc.DataAt("walletAddress"); err == nil {
		if s, ok := v.(string); ok {
			walletAddress = s
		}
	}

	return avatarID, walletAddress, nil
}

// ==============================
// ExistsByUserID
// ==============================
//
// Avatar document id は avatarId であり userId ではないため、
// setup status などの owner 判定では userId field を検索する。
func (r *AvatarRepositoryFS) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errBadClient
	}
	if userID == "" {
		return false, nil
	}

	q := r.col().Where("userId", "==", userID).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ==============================
// Create
// ==============================

func (r *AvatarRepositoryFS) Create(ctx context.Context, a avdom.Avatar) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	var ref *firestore.DocumentRef
	if a.ID == "" {
		ref = r.col().NewDoc()
		a.ID = ref.ID
	} else {
		ref = r.col().Doc(a.ID)
	}

	// userId は Firebase UID を格納している前提（= /mall/me/avatar 解決キー）
	data := r.domainToDocData(a)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avdom.Avatar{}, errConflict
		}
		return avdom.Avatar{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// Update (patch)
// ==============================
//
// 重要: walletAddress は「avatar につき 1回だけ」設定可能。
// - すでに walletAddress が入っている場合は上書きしない（Conflict）。
// - 空文字/nil で walletAddress を消すことも許可しない。
// - 競合を避けるため walletAddress を含む更新は Transaction で行う。
func (r *AvatarRepositoryFS) Update(ctx context.Context, id string, patch avdom.AvatarPatch) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}
	if id == "" {
		return avdom.Avatar{}, errNotFound
	}

	ref := r.col().Doc(id)

	// walletAddress を含む場合は transaction で「未設定ならセット」を保証
	if patch.WalletAddress != nil {
		want := *patch.WalletAddress
		if want == "" {
			return avdom.Avatar{}, errInvalidWalletAddr
		}

		// sanitize optional strings (empty -> nil)
		sAvatarIcon := nilIfEmptyPtr(patch.AvatarIcon)
		sProfile := nilIfEmptyPtr(patch.Profile)
		sExternalLink := nilIfEmptyPtr(patch.ExternalLink)

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(ref)
			if status.Code(err) == codes.NotFound {
				return errNotFound
			}
			if err != nil {
				return err
			}

			// 既に walletAddress があるなら上書き禁止
			existing := getStringField(snap, "walletAddress")
			if existing != "" {
				// 既に同じ値が入っている場合も「もう開設済み」として Conflict 扱い
				return errWalletAlreadyBound
			}

			var updates []firestore.Update

			// walletAddress はこの transaction で一度だけセット可能
			updates = append(updates, firestore.Update{
				Path:  "walletAddress",
				Value: want,
			})

			if patch.AvatarName != nil {
				updates = append(updates, firestore.Update{
					Path:  "avatarName",
					Value: *patch.AvatarName,
				})
			}

			// entity.go 正: AvatarIconURL/Path -> AvatarIcon
			if patch.AvatarIcon != nil {
				var v any
				if sAvatarIcon == nil {
					v = nil
				} else {
					v = *sAvatarIcon
				}
				updates = append(updates, firestore.Update{
					Path:  "avatarIcon",
					Value: v,
				})
			}

			if patch.Profile != nil {
				var v any
				if sProfile == nil {
					v = nil
				} else {
					v = *sProfile
				}
				updates = append(updates, firestore.Update{
					Path:  "profile",
					Value: v,
				})
			}

			if patch.ExternalLink != nil {
				var v any
				if sExternalLink == nil {
					v = nil
				} else {
					v = *sExternalLink
				}
				updates = append(updates, firestore.Update{
					Path:  "externalLink",
					Value: v,
				})
			}

			// Always bump updatedAt
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: time.Now().UTC(),
			})

			if err := tx.Update(ref, updates); err != nil {
				if status.Code(err) == codes.NotFound {
					return errNotFound
				}
				return err
			}
			return nil
		})
		if err != nil {
			// wallet already set は conflict として返す
			if errors.Is(err, errWalletAlreadyBound) {
				return avdom.Avatar{}, errConflict
			}
			if errors.Is(err, errNotFound) {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}

		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}
		return r.docToDomain(snap)
	}

	// ------------------------------
	// walletAddress を含まない通常更新
	// ------------------------------

	// Ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
	} else if err != nil {
		return avdom.Avatar{}, err
	}

	var updates []firestore.Update

	if patch.AvatarName != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarName",
			Value: *patch.AvatarName,
		})
	}

	// entity.go 正: AvatarIconURL/Path -> AvatarIcon
	if patch.AvatarIcon != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarIcon",
			Value: optionalAvatarString(*patch.AvatarIcon),
		})
	}

	// walletAddress は通常 Update では扱わない（上書き防止のため）

	if patch.Profile != nil {
		updates = append(updates, firestore.Update{
			Path:  "profile",
			Value: optionalAvatarString(*patch.Profile),
		})
	}

	if patch.ExternalLink != nil {
		updates = append(updates, firestore.Update{
			Path:  "externalLink",
			Value: optionalAvatarString(*patch.ExternalLink),
		})
	}

	if len(updates) == 0 {
		// no-op: return current
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}
		return r.docToDomain(snap)
	}

	// Always bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avdom.Avatar{}, errNotFound
		}
		return avdom.Avatar{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// Delete
// ==============================

func (r *AvatarRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errBadClient
	}
	if id == "" {
		return errNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return errNotFound
	} else if err != nil {
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

// ==============================
// Mapping helpers
// ==============================

func (r *AvatarRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (avdom.Avatar, error) {
	var raw struct {
		// userId は Firebase UID を格納している前提
		UserID string `firestore:"userId"`

		AvatarName    string    `firestore:"avatarName"`
		AvatarIcon    *string   `firestore:"avatarIcon"`
		WalletAddress *string   `firestore:"walletAddress"`
		Profile       *string   `firestore:"profile"`
		ExternalLink  *string   `firestore:"externalLink"`
		CreatedAt     time.Time `firestore:"createdAt"`
		UpdatedAt     time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return avdom.Avatar{}, err
	}

	a := avdom.Avatar{
		ID:         doc.Ref.ID,
		UserID:     raw.UserID,
		AvatarName: raw.AvatarName,
		CreatedAt:  raw.CreatedAt.UTC(),
		UpdatedAt:  raw.UpdatedAt.UTC(),
	}

	if raw.AvatarIcon != nil && *raw.AvatarIcon != "" {
		v := *raw.AvatarIcon
		a.AvatarIcon = &v
	}
	if raw.WalletAddress != nil && *raw.WalletAddress != "" {
		v := *raw.WalletAddress
		a.WalletAddress = &v
	}
	if raw.Profile != nil && *raw.Profile != "" {
		v := *raw.Profile
		a.Profile = &v
	}
	if raw.ExternalLink != nil && *raw.ExternalLink != "" {
		v := *raw.ExternalLink
		a.ExternalLink = &v
	}

	return a, nil
}

func (r *AvatarRepositoryFS) domainToDocData(a avdom.Avatar) map[string]any {
	data := map[string]any{
		// userId は Firebase UID を格納している前提
		"userId":     a.UserID,
		"avatarName": a.AvatarName,
		"createdAt":  a.CreatedAt.UTC(),
		"updatedAt":  a.UpdatedAt.UTC(),
	}

	if a.AvatarIcon != nil && *a.AvatarIcon != "" {
		data["avatarIcon"] = *a.AvatarIcon
	}
	if a.WalletAddress != nil && *a.WalletAddress != "" {
		data["walletAddress"] = *a.WalletAddress
	}
	if a.Profile != nil && *a.Profile != "" {
		data["profile"] = *a.Profile
	}
	if a.ExternalLink != nil && *a.ExternalLink != "" {
		data["externalLink"] = *a.ExternalLink
	}

	return data
}

// ==============================
// small utils
// ==============================

func optionalAvatarString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nilIfEmptyPtr(p *string) *string {
	if p == nil {
		return nil
	}
	if *p == "" {
		return nil
	}
	return p
}
