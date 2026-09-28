// backend/internal/application/usecase/identity_verification_usecase.go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	identitydom "narratives/internal/domain/identity_verification"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrIdentityVerificationNotConfigured = errors.New(
		"identity verification: usecase is not configured",
	)
)

// ============================================================
// Usecase
// ============================================================

// IdentityVerificationUsecase はユーザー本人確認を管理します。
//
// 現段階では実 KYC provider との連携は行わず、
// マイナンバーカードによる本人確認を模した mock 登録のみを提供します。
//
// 個人番号、カード画像、暗証番号、IC チップから取得した生データなどは
// 本 Usecase では一切扱いません。
type IdentityVerificationUsecase struct {
	repo identitydom.Repository
	now  func() time.Time
}

func NewIdentityVerificationUsecase(
	repo identitydom.Repository,
) *IdentityVerificationUsecase {
	return &IdentityVerificationUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// SetNowFunc はテスト用にサーバー時刻を差し替えます。
func (u *IdentityVerificationUsecase) SetNowFunc(
	now func() time.Time,
) {
	if u == nil || now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Status
// ============================================================

// IdentityVerificationStatusResult は本人確認状態取得 API などで利用する
// 公開可能な本人確認情報です。
//
// ProviderVerificationID は内部識別子のため返しません。
type IdentityVerificationStatusResult struct {
	Status identitydom.Status `json:"status"`

	Method   identitydom.Method   `json:"method,omitempty"`
	Provider identitydom.Provider `json:"provider,omitempty"`

	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

// GetStatus は userID の本人確認状態を返します。
//
// identityVerifications/{userId} が存在しない場合はエラーにせず、
// StatusUnverified を返します。
func (u *IdentityVerificationUsecase) GetStatus(
	ctx context.Context,
	userID string,
) (IdentityVerificationStatusResult, error) {
	if err := u.validateConfigured(); err != nil {
		return IdentityVerificationStatusResult{}, err
	}

	normalizedUserID, err := normalizeIdentityVerificationUserID(
		userID,
	)
	if err != nil {
		return IdentityVerificationStatusResult{}, err
	}

	verification, err := u.repo.GetByUserID(
		ctx,
		normalizedUserID,
	)
	if err != nil {
		if errors.Is(err, identitydom.ErrNotFound) {
			return IdentityVerificationStatusResult{
				Status: identitydom.StatusUnverified,
			}, nil
		}

		return IdentityVerificationStatusResult{}, err
	}

	return identityVerificationStatusResult(
		verification,
	), nil
}

// IsVerified は userID が本人確認済みかを返します。
//
// 本人確認 document が存在しない場合は false, nil を返します。
//
// Trade の返品相談など、本人確認必須機能の Application 層から
// このメソッドを利用できます。
func (u *IdentityVerificationUsecase) IsVerified(
	ctx context.Context,
	userID string,
) (bool, error) {
	if err := u.validateConfigured(); err != nil {
		return false, err
	}

	normalizedUserID, err := normalizeIdentityVerificationUserID(
		userID,
	)
	if err != nil {
		return false, err
	}

	verification, err := u.repo.GetByUserID(
		ctx,
		normalizedUserID,
	)
	if err != nil {
		if errors.Is(err, identitydom.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return verification.IsVerified(), nil
}

// ============================================================
// Mock verification
// ============================================================

// VerifyWithMockMyNumberCard はマイナンバーカードによる本人確認を
// 模した mock 本人確認を登録します。
//
// 現段階では外部 KYC provider への通信は行いません。
//
// 挙動:
//
//   - 未登録
//     -> mock の verified document を作成
//
//   - unverified document が存在
//     -> verified へ遷移
//
//   - 既に verified
//     -> そのまま既存値を返す
//
// これにより通常の再送について冪等に動作します。
func (u *IdentityVerificationUsecase) VerifyWithMockMyNumberCard(
	ctx context.Context,
	userID string,
) (identitydom.IdentityVerification, error) {
	if err := u.validateConfigured(); err != nil {
		return identitydom.IdentityVerification{}, err
	}

	normalizedUserID, err := normalizeIdentityVerificationUserID(
		userID,
	)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	existing, err := u.repo.GetByUserID(
		ctx,
		normalizedUserID,
	)
	if err == nil {
		if existing.IsVerified() {
			return existing, nil
		}

		now := u.nowUTC()

		if err := existing.VerifyWithMockMyNumberCard(
			buildMockIdentityVerificationID(normalizedUserID),
			now,
		); err != nil {
			return identitydom.IdentityVerification{}, err
		}

		return u.repo.Upsert(
			ctx,
			existing,
		)
	}

	if !errors.Is(err, identitydom.ErrNotFound) {
		return identitydom.IdentityVerification{}, err
	}

	now := u.nowUTC()

	verification, err :=
		identitydom.NewMockMyNumberCardVerified(
			normalizedUserID,
			buildMockIdentityVerificationID(
				normalizedUserID,
			),
			now,
		)
	if err != nil {
		return identitydom.IdentityVerification{}, err
	}

	return u.repo.Upsert(
		ctx,
		verification,
	)
}

// ============================================================
// Helpers
// ============================================================

func (u *IdentityVerificationUsecase) validateConfigured() error {
	if u == nil || u.repo == nil || u.now == nil {
		return ErrIdentityVerificationNotConfigured
	}

	return nil
}

func (u *IdentityVerificationUsecase) nowUTC() time.Time {
	return u.now().UTC()
}

func normalizeIdentityVerificationUserID(
	userID string,
) (string, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return "", identitydom.ErrInvalidUserID
	}

	return normalizedUserID, nil
}

func identityVerificationStatusResult(
	verification identitydom.IdentityVerification,
) IdentityVerificationStatusResult {
	result := IdentityVerificationStatusResult{
		Status:   verification.Status,
		Method:   verification.Method,
		Provider: verification.Provider,
	}

	if verification.VerifiedAt != nil {
		verifiedAt := verification.VerifiedAt.UTC()
		result.VerifiedAt = &verifiedAt
	}

	return result
}

// buildMockIdentityVerificationID は mock provider 用の内部確認IDを
// userID から決定的に生成します。
//
// Firebase UID 自体は ProviderVerificationID として保存せず、
// SHA-256 による不可逆な識別子へ変換します。
//
// 同一 userID では常に同一の ID となるため、mock KYC の再実行時にも
// provider 側識別子が不必要に変化しません。
func buildMockIdentityVerificationID(
	userID string,
) string {
	sum := sha256.Sum256(
		[]byte(
			"amol:identity-verification:mock:" +
				userID,
		),
	)

	return "mock_" + hex.EncodeToString(sum[:])
}
