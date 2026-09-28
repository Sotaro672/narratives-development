// backend/internal/adapters/out/firestore/identity_verification_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identitydom "narratives/internal/domain/identity_verification"
)

const identityVerificationsCollection = "identityVerifications"

// Compile-time interface check.
var _ identitydom.Repository = (*IdentityVerificationRepositoryFS)(nil)

// IdentityVerificationRepositoryFS は本人確認情報の Firestore 実装です。
//
// Persistence:
//
//	identityVerifications/{userId}
//
// document ID と UserID は常に同一です。
//
// この Repository は個人番号、マイナンバーカード画像、暗証番号、
// IC チップから取得した生データなどを保存しません。
// ProviderVerificationID は KYC provider 側の確認処理を参照するための
// 内部識別子としてのみ保存します。
type IdentityVerificationRepositoryFS struct {
	Client *firestore.Client
}

// identityVerificationDocument は Firestore に保存する schema です。
//
// ProviderVerificationID は Domain Entity では json:"-" ですが、
// Firestore には内部参照用の値として保存します。
type identityVerificationDocument struct {
	UserID string `firestore:"userId"`

	Status identitydom.Status `firestore:"status"`
	Method identitydom.Method `firestore:"method,omitempty"`

	Provider identitydom.Provider `firestore:"provider,omitempty"`

	ProviderVerificationID string `firestore:"providerVerificationId,omitempty"`

	VerifiedAt *time.Time `firestore:"verifiedAt,omitempty"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func NewIdentityVerificationRepositoryFS(
	client *firestore.Client,
) *IdentityVerificationRepositoryFS {
	return &IdentityVerificationRepositoryFS{
		Client: client,
	}
}

func (r *IdentityVerificationRepositoryFS) ensureClient() error {
	if r == nil || r.Client == nil {
		return errors.New(
			"identity verification: firestore client is nil",
		)
	}

	return nil
}

func (r *IdentityVerificationRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(identityVerificationsCollection)
}

func (r *IdentityVerificationRepositoryFS) doc(
	userID string,
) *firestore.DocumentRef {
	return r.col().Doc(userID)
}

func normalizeIdentityVerificationUserID(
	userID string,
) (string, error) {
	normalized := strings.TrimSpace(userID)
	if normalized == "" {
		return "", identitydom.ErrInvalidUserID
	}

	return normalized, nil
}

// ============================================================
// Read
// ============================================================

// GetByUserID は userId に対応する本人確認情報を取得します。
//
// document が存在しない場合は identityverification.ErrNotFound を返します。
// Application 層では ErrNotFound を「本人確認未登録 = unverified」として
// 扱うことができます。
func (r *IdentityVerificationRepositoryFS) GetByUserID(
	ctx context.Context,
	userID string,
) (identitydom.IdentityVerification, error) {
	if err := r.ensureClient(); err != nil {
		return identitydom.IdentityVerification{}, err
	}

	normalizedUserID, err := normalizeIdentityVerificationUserID(
		userID,
	)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	snapshot, err := r.doc(normalizedUserID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return identitydom.IdentityVerification{},
				identitydom.ErrNotFound
		}

		return identitydom.IdentityVerification{}, err
	}

	verification, err := identityVerificationDocumentToDomain(
		snapshot,
	)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	if verification.UserID != normalizedUserID {
		return identitydom.IdentityVerification{},
			identitydom.ErrInvalidUserID
	}

	return verification, nil
}

// ============================================================
// Write
// ============================================================

// Upsert は userId に対応する本人確認情報を作成または更新します。
//
// Firestore document ID は verification.UserID です。
//
// 更新時は Transaction 内で既存 document を取得し、以下を保証します:
//
//   - document ID と UserID が一致する
//   - UserID は変更されない
//   - CreatedAt は変更されない
//   - verified から unverified へ巻き戻さない
//
// 同一内容の保存は成功するため、モック KYC 登録処理を冪等にできます。
func (r *IdentityVerificationRepositoryFS) Upsert(
	ctx context.Context,
	verification identitydom.IdentityVerification,
) (identitydom.IdentityVerification, error) {
	if err := r.ensureClient(); err != nil {
		return identitydom.IdentityVerification{}, err
	}

	normalizedUserID, err := normalizeIdentityVerificationUserID(
		verification.UserID,
	)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	if normalizedUserID != verification.UserID {
		return identitydom.IdentityVerification{},
			identitydom.ErrInvalidUserID
	}

	normalizeIdentityVerificationTimestamps(&verification)

	if err := verification.Validate(); err != nil {
		return identitydom.IdentityVerification{}, err
	}

	ref := r.doc(normalizedUserID)
	incomingDocument := identityVerificationDomainToDocument(
		verification,
	)

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snapshot, getErr := tx.Get(ref)

			if getErr != nil {
				if status.Code(getErr) == codes.NotFound {
					return tx.Create(
						ref,
						incomingDocument,
					)
				}

				return getErr
			}

			current, convertErr :=
				identityVerificationDocumentToDomain(snapshot)
			if convertErr != nil {
				return convertErr
			}

			if current.UserID != normalizedUserID ||
				verification.UserID != current.UserID {
				return identitydom.ErrInvalidUserID
			}

			normalizeIdentityVerificationTimestamps(&current)

			// CreatedAt は最初の本人確認レコード作成後は不変です。
			if !current.CreatedAt.Equal(verification.CreatedAt) {
				return identitydom.ErrInvalidCreatedAt
			}

			// 本人確認済み状態を Repository 経由で未確認状態へ
			// 巻き戻すことは禁止します。
			if current.IsVerified() && !verification.IsVerified() {
				return identitydom.ErrInvalidState
			}

			return tx.Set(
				ref,
				incomingDocument,
			)
		},
	)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	// Firestore に永続化された値を正として返します。
	return r.GetByUserID(
		ctx,
		normalizedUserID,
	)
}

// ============================================================
// Conversion
// ============================================================

func identityVerificationDomainToDocument(
	verification identitydom.IdentityVerification,
) identityVerificationDocument {
	normalizeIdentityVerificationTimestamps(&verification)

	return identityVerificationDocument{
		UserID: verification.UserID,

		Status: verification.Status,
		Method: verification.Method,

		Provider: verification.Provider,

		ProviderVerificationID: verification.ProviderVerificationID,

		VerifiedAt: verification.VerifiedAt,

		CreatedAt: verification.CreatedAt,
		UpdatedAt: verification.UpdatedAt,
	}
}

func identityVerificationDocumentToDomain(
	snapshot *firestore.DocumentSnapshot,
) (identitydom.IdentityVerification, error) {
	if snapshot == nil ||
		snapshot.Ref == nil ||
		snapshot.Ref.ID == "" {
		return identitydom.IdentityVerification{},
			identitydom.ErrNotFound
	}

	var document identityVerificationDocument
	if err := snapshot.DataTo(&document); err != nil {
		return identitydom.IdentityVerification{}, err
	}

	documentUserID := strings.TrimSpace(document.UserID)
	if documentUserID == "" ||
		documentUserID != snapshot.Ref.ID {
		return identitydom.IdentityVerification{},
			identitydom.ErrInvalidUserID
	}

	normalizeIdentityVerificationDocumentTimestamps(
		&document,
	)

	return identitydom.New(
		documentUserID,
		document.Status,
		document.Method,
		document.Provider,
		document.ProviderVerificationID,
		document.VerifiedAt,
		document.CreatedAt,
		document.UpdatedAt,
	)
}

// ============================================================
// Timestamp normalization
// ============================================================

// Firestore Timestamp は microsecond precision で扱われるため、
// 書き込み前と読み込み後で同じ精度へ揃えます。
func normalizeIdentityVerificationTimestamps(
	verification *identitydom.IdentityVerification,
) {
	if verification == nil {
		return
	}

	verification.CreatedAt =
		normalizeIdentityVerificationTimestamp(
			verification.CreatedAt,
		)

	verification.UpdatedAt =
		normalizeIdentityVerificationTimestamp(
			verification.UpdatedAt,
		)

	if verification.VerifiedAt != nil {
		verifiedAt := normalizeIdentityVerificationTimestamp(
			*verification.VerifiedAt,
		)
		verification.VerifiedAt = &verifiedAt
	}
}

func normalizeIdentityVerificationDocumentTimestamps(
	document *identityVerificationDocument,
) {
	if document == nil {
		return
	}

	document.CreatedAt =
		normalizeIdentityVerificationTimestamp(
			document.CreatedAt,
		)

	document.UpdatedAt =
		normalizeIdentityVerificationTimestamp(
			document.UpdatedAt,
		)

	if document.VerifiedAt != nil {
		verifiedAt := normalizeIdentityVerificationTimestamp(
			*document.VerifiedAt,
		)
		document.VerifiedAt = &verifiedAt
	}
}

func normalizeIdentityVerificationTimestamp(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC().Truncate(time.Microsecond)
}
