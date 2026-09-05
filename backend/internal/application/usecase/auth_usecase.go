// backend/internal/application/usecase/auth_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	accountdom "narratives/internal/domain/account"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	permdom "narratives/internal/domain/permission"
)

// -------------------------------------------------------
// 開発環境用デフォルト口座
// -------------------------------------------------------

const (
	defaultTestBankName      = "テスト銀行"
	defaultTestBranchName    = "テスト支店"
	defaultTestAccountNumber = 1234567
	defaultTestCountry       = "JP"
)

// -------------------------------------------------------
// フロントから受け取るプロフィール
// - omitted と empty string を区別するため pointer にする
// -------------------------------------------------------

type SignUpProfile struct {
	LastName      *string `json:"lastName,omitempty"`
	FirstName     *string `json:"firstName,omitempty"`
	LastNameKana  *string `json:"lastNameKana,omitempty"`
	FirstNameKana *string `json:"firstNameKana,omitempty"`
	CompanyName   *string `json:"companyName,omitempty"`
}

// -------------------------------------------------------
// Bootstrap Service
// -------------------------------------------------------

type BootstrapService struct {
	Members   memberdom.Repository
	Companies companydom.Repository

	// Accounts は開発環境で会社作成時に
	// デフォルトのテスト口座を自動登録するために利用する。
	Accounts *AccountUsecase

	// AutoCreateTestAccount が true の環境でのみ
	// テスト口座を自動登録する。
	AutoCreateTestAccount bool
}

// MemberCompanyIDReader は、Firebase Auth UID から companyID だけを取得するための
// adapter 側の任意拡張です。
type MemberCompanyIDReader interface {
	GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error)
}

// -------------------------------------------------------
// Bootstrap（管理アカウントの初回ログイン時に呼ばれる想定）
//
// 方針:
// - Firestore document ID と Firebase Auth UID は分離する
// - members/{autoDocID}.uid = Firebase Auth UID として保存する
// - Company.admin には Firebase Auth UID ではなく members document ID を保存する
// - 既存データで Company.admin に Firebase Auth UID が残っている場合は bootstrap 時に自動補正する
// - 新規作成時のみ firstName / lastName を必須にする
// - 開発環境では Company に口座が存在しない場合のみテスト口座を自動登録する
// -------------------------------------------------------

func (s *BootstrapService) Bootstrap(
	ctx context.Context,
	uid string,
	email string,
	profile *SignUpProfile,
) error {
	now := time.Now().UTC()

	if uid == "" {
		return errors.New("bootstrap: uid is empty")
	}
	if s == nil || s.Members == nil || s.Companies == nil {
		return errors.New("bootstrap: service not initialized")
	}

	// ---------------------------------------------------------
	// 0) 既に member がいる場合は冪等処理
	//
	// 過去データで Company.admin に Firebase UID が保存されている場合、
	// 現在の members document ID へ自動補正する。
	//
	// 開発環境でテスト口座の作成だけが失敗していた場合に備え、
	// Account が1件も存在しない場合はここでも補完する。
	// ---------------------------------------------------------
	if r, ok := any(s.Members).(MemberCompanyIDReader); ok {
		companyID, err := r.GetCompanyIDByFirebaseUID(ctx, uid)
		if err == nil {
			if companyID != "" {
				existingMember, memberErr := s.Members.GetByUID(ctx, uid)
				if memberErr != nil {
					if !isAuthNotFoundLike(memberErr) {
						return memberErr
					}
				} else if existingMember.DocID != "" {
					if err := s.ensureCompanyAdminMemberID(ctx, companyID, existingMember.DocID); err != nil {
						return err
					}

					if err := s.ensureDefaultTestAccount(
						ctx,
						companyID,
						existingMember.DocID,
						"",
						email,
					); err != nil {
						return err
					}

					return nil
				}
			}
		} else if !isAuthNotFoundLike(err) {
			return err
		}
	}

	// ---------------------------------------------------------
	// 1) profile 取り出し（nil-safe）
	// ---------------------------------------------------------

	var p SignUpProfile
	if profile != nil {
		p = *profile
	}

	companyName := ""
	if p.CompanyName != nil {
		companyName = *p.CompanyName
	}

	firstName := ""
	if p.FirstName != nil {
		firstName = *p.FirstName
	}

	lastName := ""
	if p.LastName != nil {
		lastName = *p.LastName
	}

	firstNameKana := ""
	if p.FirstNameKana != nil {
		firstNameKana = *p.FirstNameKana
	}

	lastNameKana := ""
	if p.LastNameKana != nil {
		lastNameKana = *p.LastNameKana
	}

	// ---------------------------------------------------------
	// 2) 新規作成時は名前必須
	// ---------------------------------------------------------

	if firstName == "" || lastName == "" {
		return errors.New("member: invalid firstName")
	}

	// ---------------------------------------------------------
	// 3) companyName がある場合は Company ID を確保して Company を作る
	//
	// Member document ID はまだ発行されていないため、admin には一時的に
	// Firebase UID を設定する。Member 作成直後に members document ID へ補正する。
	// ---------------------------------------------------------

	companyID := ""
	if companyName != "" {
		issuedID, err := s.Companies.NewID(ctx)
		if err != nil {
			return err
		}

		companyEntity, err := companydom.NewCompany(
			issuedID,
			companyName,
			uid,
			uid,
			uid,
			now,
			now,
			true,
			nil,
			nil,
		)
		if err != nil {
			return err
		}

		createdCompany, err := s.Companies.Create(ctx, companyEntity)
		if err != nil {
			return err
		}

		companyID = createdCompany.ID
		if companyID == "" {
			companyID = issuedID
		}
	}

	// ---------------------------------------------------------
	// 4) Member 新規作成
	//    Firestore docID は repository 側の自動ID
	//    Firebase Auth UID は member.uid フィールドに保存する
	// ---------------------------------------------------------

	allPermNames := permdom.AllPermissionNames()

	memberEntity, err := memberdom.New(
		now,
		memberdom.WithUID(uid),
		memberdom.WithName(firstName, lastName),
		memberdom.WithNameKana(firstNameKana, lastNameKana),
		memberdom.WithEmail(email),
		memberdom.WithStatus("active"),
		memberdom.WithPermissions(allPermNames),
	)
	if err != nil {
		return err
	}

	if companyID != "" {
		memberEntity.CompanyID = companyID
	}

	createdMember, err := s.Members.Create(ctx, memberEntity)
	if err != nil {
		return err
	}

	// ---------------------------------------------------------
	// 5) Company.admin を members document ID に統一
	// ---------------------------------------------------------

	if companyID != "" {
		if err := s.ensureCompanyAdminMemberID(ctx, companyID, createdMember.DocID); err != nil {
			return err
		}
	}

	// ---------------------------------------------------------
	// 6) 開発環境ではテスト口座を自動登録
	//
	// UIには口座入力を表示せず、Bootstrap 内で自動登録する。
	// Stripe Connected Account 自体も既存 AccountUsecase の
	// createStripeAccount を通して作成する。
	// ---------------------------------------------------------

	if companyID != "" {
		if err := s.ensureDefaultTestAccount(
			ctx,
			companyID,
			createdMember.DocID,
			companyName,
			email,
		); err != nil {
			return err
		}
	}

	return nil
}

// -------------------------------------------------------
// Company admin member ID 補正
// -------------------------------------------------------

func (s *BootstrapService) ensureCompanyAdminMemberID(
	ctx context.Context,
	companyID string,
	memberID string,
) error {
	if companyID == "" {
		return errors.New("bootstrap: companyID is empty")
	}
	if memberID == "" {
		return errors.New("bootstrap: memberID is empty")
	}

	company, err := s.Companies.GetByID(ctx, companyID)
	if err != nil {
		return err
	}

	if company.Admin == memberID {
		return nil
	}

	now := time.Now().UTC()
	updatedBy := memberID

	_, err = s.Companies.Update(
		ctx,
		companyID,
		companydom.CompanyPatch{
			Admin:     &memberID,
			UpdatedAt: &now,
			UpdatedBy: &updatedBy,
		},
	)

	return err
}

// -------------------------------------------------------
// 開発環境用テスト口座
// -------------------------------------------------------

func (s *BootstrapService) ensureDefaultTestAccount(
	ctx context.Context,
	companyID string,
	memberID string,
	companyName string,
	email string,
) error {
	if !s.AutoCreateTestAccount {
		return nil
	}

	if s.Accounts == nil ||
		s.Accounts.repo == nil ||
		s.Accounts.accountGateway == nil {
		return errors.New("bootstrap: account service not initialized")
	}

	if companyID == "" {
		return accountdom.ErrInvalidCompanyID
	}

	if memberID == "" {
		return accountdom.ErrInvalidMemberID
	}

	accountID := defaultTestAccountID(companyID)

	accounts, err := s.Accounts.repo.ListByCompanyID(ctx, companyID)
	if err != nil {
		return err
	}

	hasExistingAccount := false

	for _, account := range accounts {
		if account.ID == accountID {
			if account.Status == accountdom.StatusDeleted {
				// 明示的に削除されたデフォルト口座は復活させない。
				return nil
			}

			return s.applyDefaultTestAccountValues(ctx, accountID, memberID)
		}

		if account.Status != accountdom.StatusDeleted {
			hasExistingAccount = true
		}
	}

	// すでに別の利用可能な口座がある場合は
	// デフォルトテスト口座を追加しない。
	if hasExistingAccount {
		return nil
	}

	_, err = s.Accounts.createStripeAccount(
		ctx,
		accountID,
		companyID,
		memberID,
		companyName,
		email,
		defaultTestCountry,
	)
	if err != nil {
		return err
	}

	return s.applyDefaultTestAccountValues(ctx, accountID, memberID)
}

func (s *BootstrapService) applyDefaultTestAccountValues(
	ctx context.Context,
	accountID string,
	memberID string,
) error {
	bankName := defaultTestBankName
	branchName := defaultTestBranchName
	accountNumber := defaultTestAccountNumber
	accountType := accountdom.TypeFutsu
	status := accountdom.StatusActive

	_, err := s.Accounts.repo.Update(
		ctx,
		accountID,
		accountdom.AccountPatch{
			MemberID:      &memberID,
			BankName:      &bankName,
			BranchName:    &branchName,
			AccountNumber: &accountNumber,
			AccountType:   &accountType,
			Status:        &status,
			UpdatedBy:     &memberID,
		},
	)

	return err
}

func defaultTestAccountID(companyID string) string {
	return accountdom.AccountIDPrefix + "default_" + companyID
}

// -------------------------------------------------------
// helpers
// -------------------------------------------------------

func isAuthNotFoundLike(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, memberdom.ErrNotFound) {
		return true
	}

	msg := strings.ToLower(err.Error())
	if msg == "" {
		return false
	}

	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "no documents") ||
		(strings.Contains(msg, "document") && strings.Contains(msg, "missing"))
}
