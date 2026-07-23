// backend/internal/adapters/out/firestore/invitation_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	itdom "narratives/internal/domain/invitation"
	memdom "narratives/internal/domain/member"
)

const invitationTokensCollectionName = "invitationTokens"

// InvitationTokenRepositoryFS is a Firestore-based implementation of
// invitation.Repository.
//
// Uses the "invitationTokens" collection.
//
// Firestoreドキュメント構造：
// - コレクション: "invitationTokens"
// - ドキュメントID: token
// - フィールド:
//   - deliveryId      : string
//   - memberId        : string
//   - email           : string
//   - companyId       : string
//   - assignedBrands  : []string
//   - permissions     : []string
//   - createdAt       : Firestore Timestamp
//   - expiresAt       : Firestore Timestamp (optional)
//   - deliveredAt     : Firestore Timestamp (optional)
//   - usedAt          : Firestore Timestamp (optional)
//   - revokedAt       : Firestore Timestamp (optional)
//   - updatedAt       : Firestore Timestamp
//
// deliveredAtが設定されるまでは、tokenは利用できません。
// deliveryの最終失敗時はrevokedAtを設定して失効させます。
//
// 日時フィールドはFirestore Timestamp型のみを正式仕様とする。
// string形式の日時データには対応しない。
type InvitationTokenRepositoryFS struct {
	Client *firestore.Client
}

type invitationTokenDocument struct {
	DeliveryID       string     `firestore:"deliveryId"`
	MemberID         string     `firestore:"memberId"`
	Email            string     `firestore:"email"`
	CompanyID        string     `firestore:"companyId"`
	AssignedBrandIDs []string   `firestore:"assignedBrands"`
	Permissions      []string   `firestore:"permissions"`
	CreatedAt        time.Time  `firestore:"createdAt"`
	ExpiresAt        *time.Time `firestore:"expiresAt,omitempty"`
	DeliveredAt      *time.Time `firestore:"deliveredAt,omitempty"`
	UsedAt           *time.Time `firestore:"usedAt,omitempty"`
	RevokedAt        *time.Time `firestore:"revokedAt,omitempty"`
	UpdatedAt        time.Time  `firestore:"updatedAt"`
}

func NewInvitationTokenRepositoryFS(
	client *firestore.Client,
) *InvitationTokenRepositoryFS {
	return &InvitationTokenRepositoryFS{
		Client: client,
	}
}

func (r *InvitationTokenRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(invitationTokensCollectionName)
}

func (r *InvitationTokenRepositoryFS) membersCol() *firestore.CollectionRef {
	return r.Client.Collection(membersCollectionName)
}

func (r *InvitationTokenRepositoryFS) memberUIDsCol() *firestore.CollectionRef {
	return r.Client.Collection(memberUIDsCollectionName)
}

var _ itdom.Repository = (*InvitationTokenRepositoryFS)(nil)

// FindByToken retrieves a usable invitation token document by token string.
// tokenはFirestoreのdocument IDとして扱います。
//
// deliveredAt未設定、使用済み、失効済み、期限切れのtokenは
// 利用可能なtokenとして返しません。
func (r *InvitationTokenRepositoryFS) FindByToken(
	ctx context.Context,
	token string,
) (itdom.InvitationToken, error) {
	if r == nil || r.Client == nil {
		return itdom.InvitationToken{}, errors.New(
			"firestore client is nil",
		)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return itdom.InvitationToken{},
			itdom.ErrInvitationTokenNotFound
	}

	doc, err := r.col().Doc(token).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return itdom.InvitationToken{},
				itdom.ErrInvitationTokenNotFound
		}

		return itdom.InvitationToken{}, fmt.Errorf(
			"get invitation token %q: %w",
			token,
			err,
		)
	}

	invitationToken, err := readInvitationTokenSnapshot(doc)
	if err != nil {
		return itdom.InvitationToken{}, err
	}

	if err := invitationToken.ValidateUsable(
		time.Now().UTC(),
	); err != nil {
		return itdom.InvitationToken{}, err
	}

	return invitationToken, nil
}

func (r *InvitationTokenRepositoryFS) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (itdom.InvitationInfo, error) {
	if r == nil || r.Client == nil {
		return itdom.InvitationInfo{}, errors.New(
			"firestore client is nil",
		)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return itdom.InvitationInfo{},
			itdom.ErrInvitationTokenNotFound
	}

	invitationToken, err := r.FindByToken(ctx, token)
	if err != nil {
		return itdom.InvitationInfo{}, err
	}

	info, err := invitationToken.InvitationInfo()
	if err != nil {
		if errors.Is(
			err,
			itdom.ErrInvitationMemberIDRequired,
		) {
			return itdom.InvitationInfo{},
				itdom.ErrInvitationTokenNotFound
		}

		return itdom.InvitationInfo{}, fmt.Errorf(
			"resolve invitation info from token %q: %w",
			token,
			err,
		)
	}

	return info, nil
}

// CompleteInvitationは、次の処理を同一Firestore transaction内で
// 実行します。
//
//   - invitation tokenの利用可能性確認
//   - token emailとFirebase認証済みemailの照合
//   - Memberの存在・company境界確認
//   - Firebase UIDの一意性確認
//   - Memberのactive化とUID紐付け
//   - memberUIDs mappingの作成または更新
//   - invitation tokenのusedAt更新
//
// Member更新とtoken消費の一方だけが確定することはありません。
func (r *InvitationTokenRepositoryFS) CompleteInvitation(
	ctx context.Context,
	completion itdom.InvitationCompletion,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	normalizedCompletion, err := completion.Normalize()
	if err != nil {
		return err
	}

	tokenRef := r.col().Doc(
		normalizedCompletion.Token,
	)

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			tokenDoc, err := tx.Get(tokenRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return itdom.ErrInvitationTokenNotFound
				}

				return fmt.Errorf(
					"get invitation token %q in transaction: %w",
					normalizedCompletion.Token,
					err,
				)
			}

			invitationToken, err :=
				readInvitationTokenSnapshot(tokenDoc)
			if err != nil {
				return err
			}

			now := time.Now().UTC()

			if err := invitationToken.ValidateUsable(now); err != nil {
				return err
			}

			info, err := invitationToken.InvitationInfo()
			if err != nil {
				if errors.Is(
					err,
					itdom.ErrInvitationMemberIDRequired,
				) {
					return itdom.ErrInvitationMemberNotFound
				}

				return fmt.Errorf(
					"resolve invitation info in transaction: %w",
					err,
				)
			}

			if info.Email != normalizedCompletion.Email {
				return itdom.ErrInvitationEmailMismatch
			}

			memberRef := r.membersCol().Doc(
				info.MemberID,
			)

			memberDoc, err := tx.Get(memberRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return itdom.ErrInvitationMemberNotFound
				}

				return fmt.Errorf(
					"get invited member %q in transaction: %w",
					info.MemberID,
					err,
				)
			}

			member, err := readMemberSnapshot(memberDoc)
			if err != nil {
				return fmt.Errorf(
					"decode invited member %q: %w",
					info.MemberID,
					err,
				)
			}

			if strings.TrimSpace(member.CompanyID) !=
				info.CompanyID {
				return itdom.ErrInvitationCompanyMismatch
			}

			newUID := normalizedCompletion.UID
			newUIDRef := r.memberUIDsCol().Doc(newUID)

			newUIDMapping, newUIDExists, err :=
				getMemberUIDMappingInTransaction(
					tx,
					newUIDRef,
				)
			if err != nil {
				return err
			}

			if newUIDExists &&
				newUIDMapping.MemberID != info.MemberID {
				return itdom.ErrInvitationUIDAlreadyInUse
			}

			memberIDs, err := findMemberIDsByUIDInTransaction(
				tx,
				r.membersCol(),
				newUID,
				2,
			)
			if err != nil {
				return err
			}

			for _, memberID := range memberIDs {
				if memberID != info.MemberID {
					return itdom.ErrInvitationUIDAlreadyInUse
				}
			}

			oldUID := strings.TrimSpace(member.UID)

			var (
				oldUIDRef    *firestore.DocumentRef
				deleteOldUID bool
			)

			if oldUID != "" && oldUID != newUID {
				oldUIDRef = r.memberUIDsCol().Doc(oldUID)

				oldUIDMapping, oldUIDExists, err :=
					getMemberUIDMappingInTransaction(
						tx,
						oldUIDRef,
					)
				if err != nil {
					return err
				}

				deleteOldUID = oldUIDExists &&
					oldUIDMapping.MemberID == info.MemberID
			}

			statusValue := "active"
			companyID := info.CompanyID
			email := normalizedCompletion.Email

			permissions := append(
				[]string(nil),
				info.Permissions...,
			)

			assignedBrandIDs := append(
				[]string(nil),
				info.AssignedBrandIDs...,
			)

			updatedMember, err := applyMemberPatch(
				member,
				memdom.MemberPatch{
					UID:            &normalizedCompletion.UID,
					LastName:       &normalizedCompletion.LastName,
					LastNameKana:   &normalizedCompletion.LastNameKana,
					FirstName:      &normalizedCompletion.FirstName,
					FirstNameKana:  &normalizedCompletion.FirstNameKana,
					Email:          &email,
					CompanyID:      &companyID,
					Status:         &statusValue,
					Permissions:    &permissions,
					AssignedBrands: &assignedBrandIDs,
				},
				now,
			)
			if err != nil {
				return fmt.Errorf(
					"apply invited member patch %q: %w",
					info.MemberID,
					err,
				)
			}

			if err := tx.Set(
				memberRef,
				updatedMember,
			); err != nil {
				return fmt.Errorf(
					"update invited member %q in transaction: %w",
					info.MemberID,
					err,
				)
			}

			if deleteOldUID {
				if err := tx.Delete(oldUIDRef); err != nil {
					return fmt.Errorf(
						"delete old member UID mapping %q: %w",
						oldUID,
						err,
					)
				}
			}

			createdAt := now
			if newUIDExists &&
				!newUIDMapping.CreatedAt.IsZero() {
				createdAt = newUIDMapping.CreatedAt.UTC()
			}

			uidMapping := memberUIDDocument{
				MemberID:  info.MemberID,
				CreatedAt: createdAt,
				UpdatedAt: now,
			}

			if newUIDExists {
				if err := tx.Set(
					newUIDRef,
					uidMapping,
				); err != nil {
					return fmt.Errorf(
						"update member UID mapping %q: %w",
						newUID,
						err,
					)
				}
			} else {
				if err := tx.Create(
					newUIDRef,
					uidMapping,
				); err != nil {
					return fmt.Errorf(
						"create member UID mapping %q: %w",
						newUID,
						err,
					)
				}
			}

			if err := tx.Update(
				tokenRef,
				[]firestore.Update{
					{
						Path:  "usedAt",
						Value: now,
					},
					{
						Path:  "updatedAt",
						Value: now,
					},
				},
			); err != nil {
				return fmt.Errorf(
					"consume invitation token %q in transaction: %w",
					normalizedCompletion.Token,
					err,
				)
			}

			return nil
		},
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			itdom.ErrInvitationTokenNotFound,
		):
			return itdom.ErrInvitationTokenNotFound

		case errors.Is(
			err,
			itdom.ErrInvitationTokenExpired,
		):
			return itdom.ErrInvitationTokenExpired

		case errors.Is(
			err,
			itdom.ErrInvitationTokenUsed,
		):
			return itdom.ErrInvitationTokenUsed

		case errors.Is(
			err,
			itdom.ErrInvitationTokenRevoked,
		):
			return itdom.ErrInvitationTokenRevoked

		case errors.Is(
			err,
			itdom.ErrInvitationTokenNotDelivered,
		):
			return itdom.ErrInvitationTokenNotDelivered

		case errors.Is(
			err,
			itdom.ErrInvitationMemberNotFound,
		):
			return itdom.ErrInvitationMemberNotFound

		case errors.Is(
			err,
			itdom.ErrInvitationCompanyMismatch,
		):
			return itdom.ErrInvitationCompanyMismatch

		case errors.Is(
			err,
			itdom.ErrInvitationEmailMismatch,
		):
			return itdom.ErrInvitationEmailMismatch

		case errors.Is(
			err,
			itdom.ErrInvitationUIDAlreadyInUse,
		),
			status.Code(err) == codes.AlreadyExists:
			return itdom.ErrInvitationUIDAlreadyInUse

		default:
			return fmt.Errorf(
				"complete invitation transaction: %w",
				err,
			)
		}
	}

	return nil
}

func invitationTokenToDocument(
	invitationToken itdom.InvitationToken,
	updatedAt time.Time,
) invitationTokenDocument {
	return invitationTokenDocument{
		DeliveryID: invitationToken.DeliveryID,
		MemberID:   invitationToken.MemberID,
		Email:      invitationToken.Email,
		CompanyID:  invitationToken.CompanyID,
		AssignedBrandIDs: append(
			[]string(nil),
			invitationToken.AssignedBrandIDs...,
		),
		Permissions: append(
			[]string(nil),
			invitationToken.Permissions...,
		),
		CreatedAt: invitationToken.CreatedAt.UTC(),
		ExpiresAt: copyInvitationTimePointer(
			invitationToken.ExpiresAt,
		),
		DeliveredAt: copyInvitationTimePointer(
			invitationToken.DeliveredAt,
		),
		UsedAt: copyInvitationTimePointer(
			invitationToken.UsedAt,
		),
		RevokedAt: copyInvitationTimePointer(
			invitationToken.RevokedAt,
		),
		UpdatedAt: updatedAt.UTC(),
	}
}

func readInvitationTokenSnapshot(
	doc *firestore.DocumentSnapshot,
) (itdom.InvitationToken, error) {
	if doc == nil {
		return itdom.InvitationToken{}, errors.New(
			"invitation token document snapshot is nil",
		)
	}

	var stored invitationTokenDocument
	if err := doc.DataTo(&stored); err != nil {
		return itdom.InvitationToken{}, fmt.Errorf(
			"decode invitation token %q: %w",
			doc.Ref.ID,
			err,
		)
	}

	if stored.CreatedAt.IsZero() {
		return itdom.InvitationToken{}, fmt.Errorf(
			"invitation token %q has no valid createdAt timestamp",
			doc.Ref.ID,
		)
	}

	if stored.UpdatedAt.IsZero() {
		return itdom.InvitationToken{}, fmt.Errorf(
			"invitation token %q has no valid updatedAt timestamp",
			doc.Ref.ID,
		)
	}

	info := itdom.InvitationInfo{
		MemberID:         stored.MemberID,
		CompanyID:        stored.CompanyID,
		AssignedBrandIDs: stored.AssignedBrandIDs,
		Permissions:      stored.Permissions,
		Email:            stored.Email,
	}

	invitationToken, err := itdom.NewInvitationToken(
		doc.Ref.ID,
		stored.DeliveryID,
		info,
		stored.CreatedAt.UTC(),
		copyInvitationTimePointer(stored.ExpiresAt),
	)
	if err != nil {
		return itdom.InvitationToken{}, fmt.Errorf(
			"normalize invitation token %q: %w",
			doc.Ref.ID,
			err,
		)
	}

	invitationToken.DeliveredAt = copyInvitationTimePointer(
		stored.DeliveredAt,
	)

	invitationToken.UsedAt = copyInvitationTimePointer(
		stored.UsedAt,
	)

	invitationToken.RevokedAt = copyInvitationTimePointer(
		stored.RevokedAt,
	)

	updatedAt := stored.UpdatedAt.UTC()
	invitationToken.UpdatedAt = &updatedAt

	return invitationToken, nil
}

func copyInvitationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
