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

// AvatarRepositoryFS はavatar.RepositoryのFirestore実装です。
type AvatarRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarRepositoryFS(
	client *firestore.Client,
) *AvatarRepositoryFS {
	return &AvatarRepositoryFS{
		Client: client,
	}
}

func (r *AvatarRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatars")
}

// Compile-time interface check.
var _ avdom.Repository = (*AvatarRepositoryFS)(nil)

var (
	errNotFound = errors.New(
		"avatar: not found",
	)
	errAvatarNotFoundForUID = errors.New(
		"avatar_not_found_for_uid",
	)
	errConflict = errors.New(
		"avatar: conflict",
	)
	errBadClient = errors.New(
		"firestore client is nil",
	)
	errInvalidWalletAddr = errors.New(
		"avatar: invalid walletAddress",
	)
	errWalletAlreadyBound = errors.New(
		"avatar: walletAddress already set",
	)
)

func (r *AvatarRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (avdom.Avatar, error) {
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

func (r *AvatarRepositoryFS) GetByUserID(
	ctx context.Context,
	userID string,
) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}

	if userID == "" {
		return avdom.Avatar{}, errNotFound
	}

	iter := r.col().
		Where("userId", "==", userID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}

	if doc == nil ||
		doc.Ref == nil ||
		doc.Ref.ID == "" {
		return avdom.Avatar{}, errNotFound
	}

	return r.docToDomain(doc)
}

func (r *AvatarRepositoryFS) ResolveAvatarByUID(
	ctx context.Context,
	uid string,
) (
	avatarID string,
	walletAddress string,
	err error,
) {
	if r == nil || r.Client == nil {
		return "", "", errBadClient
	}

	if uid == "" {
		return "", "", errAvatarNotFoundForUID
	}

	iter := r.col().
		Where("userId", "==", uid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return "", "", errAvatarNotFoundForUID
	}
	if err != nil {
		return "", "", err
	}

	if doc == nil ||
		doc.Ref == nil ||
		doc.Ref.ID == "" {
		return "", "", errAvatarNotFoundForUID
	}

	avatarID = doc.Ref.ID

	value, dataErr := doc.DataAt("walletAddress")
	if dataErr == nil {
		if s, ok := value.(string); ok {
			walletAddress = s
		}
	}

	return avatarID, walletAddress, nil
}

func (r *AvatarRepositoryFS) ExistsByUserID(
	ctx context.Context,
	userID string,
) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errBadClient
	}

	if userID == "" {
		return false, nil
	}

	iter := r.col().
		Where("userId", "==", userID).
		Limit(1).
		Documents(ctx)
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

func (r *AvatarRepositoryFS) Create(
	ctx context.Context,
	a avdom.Avatar,
) (avdom.Avatar, error) {
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

	// ID採番後、Firestoreへ保存する直前に完全検証する。
	if err := a.Validate(); err != nil {
		return avdom.Avatar{}, err
	}

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

func (r *AvatarRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch avdom.AvatarPatch,
) (avdom.Avatar, error) {
	if r == nil || r.Client == nil {
		return avdom.Avatar{}, errBadClient
	}

	if id == "" {
		return avdom.Avatar{}, errNotFound
	}

	ref := r.col().Doc(id)

	if patch.WalletAddress != nil {
		return r.updateWithWalletAddress(
			ctx,
			ref,
			patch,
		)
	}

	return r.updateWithoutWalletAddress(
		ctx,
		ref,
		patch,
	)
}

func (r *AvatarRepositoryFS) updateWithWalletAddress(
	ctx context.Context,
	ref *firestore.DocumentRef,
	patch avdom.AvatarPatch,
) (avdom.Avatar, error) {
	walletAddress := *patch.WalletAddress
	if walletAddress == "" {
		return avdom.Avatar{}, errInvalidWalletAddr
	}

	avatarIcon := nilIfEmptyPtr(patch.AvatarIcon)
	profile := nilIfEmptyPtr(patch.Profile)
	externalLink := nilIfEmptyPtr(patch.ExternalLink)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snap, err := tx.Get(ref)
			if status.Code(err) == codes.NotFound {
				return errNotFound
			}
			if err != nil {
				return err
			}

			updatedAt := time.Now().UTC()

			current, err := r.docToDomain(snap)
			if err != nil {
				return err
			}

			// Transaction内の最新値にPatchを適用して検証する。
			if _, err := current.ApplyPatch(
				patch,
				updatedAt,
			); err != nil {
				return err
			}

			existing := getStringField(
				snap,
				"walletAddress",
			)
			if existing != "" {
				return errWalletAlreadyBound
			}

			updates := []firestore.Update{
				{
					Path:  "walletAddress",
					Value: walletAddress,
				},
			}

			if patch.AvatarName != nil {
				updates = append(
					updates,
					firestore.Update{
						Path:  "avatarName",
						Value: *patch.AvatarName,
					},
				)
			}

			if patch.AvatarIcon != nil {
				var value any
				if avatarIcon != nil {
					value = *avatarIcon
				}

				updates = append(
					updates,
					firestore.Update{
						Path:  "avatarIcon",
						Value: value,
					},
				)
			}

			if patch.Profile != nil {
				var value any
				if profile != nil {
					value = *profile
				}

				updates = append(
					updates,
					firestore.Update{
						Path:  "profile",
						Value: value,
					},
				)
			}

			if patch.ExternalLink != nil {
				var value any
				if externalLink != nil {
					value = *externalLink
				}

				updates = append(
					updates,
					firestore.Update{
						Path:  "externalLink",
						Value: value,
					},
				)
			}

			updates = append(
				updates,
				firestore.Update{
					Path:  "updatedAt",
					Value: updatedAt,
				},
			)

			if err := tx.Update(ref, updates); err != nil {
				if status.Code(err) == codes.NotFound {
					return errNotFound
				}

				return err
			}

			return nil
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, errWalletAlreadyBound):
			return avdom.Avatar{}, errConflict
		case errors.Is(err, errNotFound):
			return avdom.Avatar{}, errNotFound
		default:
			return avdom.Avatar{}, err
		}
	}

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}

	return r.docToDomain(snap)
}

func (r *AvatarRepositoryFS) updateWithoutWalletAddress(
	ctx context.Context,
	ref *firestore.DocumentRef,
	patch avdom.AvatarPatch,
) (avdom.Avatar, error) {
	currentSnap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}

	updates := make([]firestore.Update, 0, 5)

	if patch.AvatarName != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "avatarName",
				Value: *patch.AvatarName,
			},
		)
	}

	if patch.AvatarIcon != nil {
		updates = append(
			updates,
			firestore.Update{
				Path: "avatarIcon",
				Value: optionalAvatarString(
					*patch.AvatarIcon,
				),
			},
		)
	}

	if patch.Profile != nil {
		updates = append(
			updates,
			firestore.Update{
				Path: "profile",
				Value: optionalAvatarString(
					*patch.Profile,
				),
			},
		)
	}

	if patch.ExternalLink != nil {
		updates = append(
			updates,
			firestore.Update{
				Path: "externalLink",
				Value: optionalAvatarString(
					*patch.ExternalLink,
				),
			},
		)
	}

	if len(updates) == 0 {
		return r.docToDomain(currentSnap)
	}

	updatedAt := time.Now().UTC()

	current, err := r.docToDomain(currentSnap)
	if err != nil {
		return avdom.Avatar{}, err
	}

	// Firestoreへ書き込む直前に更新後の集約を検証する。
	if _, err := current.ApplyPatch(
		patch,
		updatedAt,
	); err != nil {
		return avdom.Avatar{}, err
	}

	updates = append(
		updates,
		firestore.Update{
			Path:  "updatedAt",
			Value: updatedAt,
		},
	)

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

func (r *AvatarRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
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

func (r *AvatarRepositoryFS) docToDomain(
	doc *firestore.DocumentSnapshot,
) (avdom.Avatar, error) {
	var raw struct {
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

	if raw.AvatarIcon != nil &&
		*raw.AvatarIcon != "" {
		value := *raw.AvatarIcon
		a.AvatarIcon = &value
	}

	if raw.WalletAddress != nil &&
		*raw.WalletAddress != "" {
		value := *raw.WalletAddress
		a.WalletAddress = &value
	}

	if raw.Profile != nil &&
		*raw.Profile != "" {
		value := *raw.Profile
		a.Profile = &value
	}

	if raw.ExternalLink != nil &&
		*raw.ExternalLink != "" {
		value := *raw.ExternalLink
		a.ExternalLink = &value
	}

	return a, nil
}

func (r *AvatarRepositoryFS) domainToDocData(
	a avdom.Avatar,
) map[string]any {
	data := map[string]any{
		"userId":     a.UserID,
		"avatarName": a.AvatarName,
		"createdAt":  a.CreatedAt.UTC(),
		"updatedAt":  a.UpdatedAt.UTC(),
	}

	if a.AvatarIcon != nil &&
		*a.AvatarIcon != "" {
		data["avatarIcon"] = *a.AvatarIcon
	}

	if a.WalletAddress != nil &&
		*a.WalletAddress != "" {
		data["walletAddress"] = *a.WalletAddress
	}

	if a.Profile != nil &&
		*a.Profile != "" {
		data["profile"] = *a.Profile
	}

	if a.ExternalLink != nil &&
		*a.ExternalLink != "" {
		data["externalLink"] = *a.ExternalLink
	}

	return data
}

func optionalAvatarString(v string) any {
	if v == "" {
		return nil
	}

	return v
}

func nilIfEmptyPtr(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}

	return p
}
