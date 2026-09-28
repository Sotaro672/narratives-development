// backend/internal/domain/identity_verification/entity.go
package identityverification

import (
	"errors"
	"strings"
	"time"
)

// ============================================================
// Types
// ============================================================

type Status string
type Method string
type Provider string

const (
	StatusUnverified Status = "unverified"
	StatusVerified   Status = "verified"
)

const (
	MethodMyNumberCard Method = "my_number_card"
)

const (
	ProviderMock Provider = "mock"
)

// ============================================================
// Policy
// ============================================================

const (
	MaxProviderVerificationIDLength = 512
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidUserID = errors.New(
		"identity verification: invalid userId",
	)
	ErrInvalidStatus = errors.New(
		"identity verification: invalid status",
	)
	ErrInvalidMethod = errors.New(
		"identity verification: invalid method",
	)
	ErrInvalidProvider = errors.New(
		"identity verification: invalid provider",
	)
	ErrInvalidProviderVerificationID = errors.New(
		"identity verification: invalid provider verification id",
	)
	ErrInvalidVerifiedAt = errors.New(
		"identity verification: invalid verifiedAt",
	)
	ErrInvalidCreatedAt = errors.New(
		"identity verification: invalid createdAt",
	)
	ErrInvalidUpdatedAt = errors.New(
		"identity verification: invalid updatedAt",
	)
	ErrInvalidState = errors.New(
		"identity verification: invalid state",
	)
)

// ============================================================
// Entity
// ============================================================

// IdentityVerification はユーザー単位の本人確認状態を表します。
//
// Firestore:
//
//	identityVerifications/{userId}
//
// userId は Firebase Auth UID を使用します。
//
// このエンティティは個人番号、マイナンバーカード画像、暗証番号、
// ICチップから取得した生データなどを保持しません。
// ProviderVerificationID は外部本人確認サービス等の内部参照用IDであり、
// APIレスポンスには公開しません。
type IdentityVerification struct {
	UserID string `json:"userId"`

	Status Status `json:"status"`
	Method Method `json:"method,omitempty"`

	Provider Provider `json:"provider,omitempty"`

	ProviderVerificationID string `json:"-"`

	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ============================================================
// Validation
// ============================================================

func IsValidStatus(status Status) bool {
	switch status {
	case StatusUnverified, StatusVerified:
		return true
	default:
		return false
	}
}

func IsValidMethod(method Method) bool {
	switch method {
	case MethodMyNumberCard:
		return true
	default:
		return false
	}
}

func IsValidProvider(provider Provider) bool {
	switch provider {
	case ProviderMock:
		return true
	default:
		return false
	}
}

func (v IdentityVerification) Validate() error {
	if strings.TrimSpace(v.UserID) == "" {
		return ErrInvalidUserID
	}
	if !IsValidStatus(v.Status) {
		return ErrInvalidStatus
	}
	if v.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if v.UpdatedAt.IsZero() || v.UpdatedAt.Before(v.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	switch v.Status {
	case StatusUnverified:
		if v.Method != "" ||
			v.Provider != "" ||
			v.ProviderVerificationID != "" ||
			v.VerifiedAt != nil {
			return ErrInvalidState
		}

	case StatusVerified:
		if !IsValidMethod(v.Method) {
			return ErrInvalidMethod
		}
		if !IsValidProvider(v.Provider) {
			return ErrInvalidProvider
		}

		providerVerificationID := strings.TrimSpace(
			v.ProviderVerificationID,
		)
		if providerVerificationID == "" ||
			len([]rune(providerVerificationID)) >
				MaxProviderVerificationIDLength {
			return ErrInvalidProviderVerificationID
		}

		if v.VerifiedAt == nil || v.VerifiedAt.IsZero() {
			return ErrInvalidVerifiedAt
		}

		verifiedAt := v.VerifiedAt.UTC()
		if verifiedAt.Before(v.CreatedAt) {
			return ErrInvalidVerifiedAt
		}
		if v.UpdatedAt.Before(verifiedAt) {
			return ErrInvalidUpdatedAt
		}
	}

	return nil
}

// ============================================================
// Constructors
// ============================================================

// New は永続化済みの本人確認情報を復元します。
func New(
	userID string,
	status Status,
	method Method,
	provider Provider,
	providerVerificationID string,
	verifiedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) (IdentityVerification, error) {
	userID = strings.TrimSpace(userID)
	providerVerificationID = strings.TrimSpace(
		providerVerificationID,
	)

	var normalizedVerifiedAt *time.Time
	if verifiedAt != nil {
		value := verifiedAt.UTC()
		normalizedVerifiedAt = &value
	}

	verification := IdentityVerification{
		UserID:                 userID,
		Status:                 status,
		Method:                 method,
		Provider:               provider,
		ProviderVerificationID: providerVerificationID,
		VerifiedAt:             normalizedVerifiedAt,
		CreatedAt:              createdAt.UTC(),
		UpdatedAt:              updatedAt.UTC(),
	}

	if err := verification.Validate(); err != nil {
		return IdentityVerification{}, err
	}

	return verification, nil
}

// NewUnverified は未本人確認状態を生成します。
// Firestore上では未登録を unverified と扱う設計でも利用できます。
func NewUnverified(
	userID string,
	now time.Time,
) (IdentityVerification, error) {
	if now.IsZero() {
		return IdentityVerification{}, ErrInvalidCreatedAt
	}

	now = now.UTC()

	return New(
		userID,
		StatusUnverified,
		"",
		"",
		"",
		nil,
		now,
		now,
	)
}

// NewMockMyNumberCardVerified は現段階のモックKYC登録用です。
// 実際の個人番号やマイナンバーカード情報は受け取りません。
func NewMockMyNumberCardVerified(
	userID string,
	providerVerificationID string,
	now time.Time,
) (IdentityVerification, error) {
	if now.IsZero() {
		return IdentityVerification{}, ErrInvalidVerifiedAt
	}

	now = now.UTC()

	return New(
		userID,
		StatusVerified,
		MethodMyNumberCard,
		ProviderMock,
		providerVerificationID,
		&now,
		now,
		now,
	)
}

// ============================================================
// Behavior
// ============================================================

// IsVerified は本人確認が完了しているかを返します。
func (v IdentityVerification) IsVerified() bool {
	return v.Status == StatusVerified &&
		IsValidMethod(v.Method) &&
		IsValidProvider(v.Provider) &&
		strings.TrimSpace(v.ProviderVerificationID) != "" &&
		v.VerifiedAt != nil &&
		!v.VerifiedAt.IsZero()
}

// VerifyWithMockMyNumberCard は既存の未確認レコードを、
// モックのマイナンバーカード本人確認済み状態へ遷移させます。
//
// 既に本人確認済みの場合は状態を変更せず成功扱いとし、
// POSTの再試行を冪等に扱えるようにします。
func (v *IdentityVerification) VerifyWithMockMyNumberCard(
	providerVerificationID string,
	now time.Time,
) error {
	if v == nil {
		return ErrInvalidState
	}
	if v.IsVerified() {
		return nil
	}
	if strings.TrimSpace(v.UserID) == "" {
		return ErrInvalidUserID
	}
	if v.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if now.IsZero() {
		return ErrInvalidVerifiedAt
	}

	providerVerificationID = strings.TrimSpace(
		providerVerificationID,
	)
	if providerVerificationID == "" ||
		len([]rune(providerVerificationID)) >
			MaxProviderVerificationIDLength {
		return ErrInvalidProviderVerificationID
	}

	now = now.UTC()

	v.Status = StatusVerified
	v.Method = MethodMyNumberCard
	v.Provider = ProviderMock
	v.ProviderVerificationID = providerVerificationID
	v.VerifiedAt = &now
	v.UpdatedAt = now

	return v.Validate()
}
