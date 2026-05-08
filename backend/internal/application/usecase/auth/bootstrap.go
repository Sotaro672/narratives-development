// backend/internal/application/usecase/auth/bootstrap.go
package auth

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	permdom "narratives/internal/domain/permission"
)

// -------------------------------------------------------
// Repository Interfaces
// -------------------------------------------------------

type MemberRepository interface {
	// GetByFirebaseUID returns a member whose uid field matches Firebase Auth UID.
	// Firestore document ID and Firebase Auth UID are intentionally separated.
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (*memberdom.Member, error)

	// Create creates a new member using repository default document ID behavior.
	// The member.UID field must contain Firebase Auth UID when the member is
	// already associated with a Firebase Auth user.
	Create(ctx context.Context, m *memberdom.Member) error
}

type CompanyRepository interface {
	NewID(ctx context.Context) (string, error)
	Save(ctx context.Context, c *companydom.Company) error
}

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
	Members   MemberRepository
	Companies CompanyRepository
}

// -------------------------------------------------------
// Bootstrap（管理アカウントの初回ログイン時に呼ばれる想定）
//
// 方針:
// - Firestore document ID と Firebase Auth UID は分離する
// - members/{autoDocID}.uid = Firebase Auth UID として保存する
// - 既存 member 確認は docID ではなく uid フィールドで行う
// - 新規作成時のみ firstName / lastName を必須にする
// -------------------------------------------------------

func (s *BootstrapService) Bootstrap(
	ctx context.Context,
	uid string,
	email string,
	profile *SignUpProfile,
) error {
	now := time.Now().UTC()

	uid = strings.TrimSpace(uid)
	email = strings.TrimSpace(email)

	log.Printf("[bootstrap] request received: uid=%s email=%s", uid, email)

	if uid == "" {
		return errors.New("bootstrap: uid is empty")
	}
	if s == nil || s.Members == nil || s.Companies == nil {
		return errors.New("bootstrap: service not initialized")
	}

	// ---------------------------------------------------------
	// 0) 既に member がいるなら（冪等）基本は何もしない
	//    新方針では docID ではなく uid フィールドで検索する
	// ---------------------------------------------------------
	if m, err := s.Members.GetByFirebaseUID(ctx, uid); err == nil && m != nil {
		log.Printf("[bootstrap] member already exists: uid=%s companyID=%s (noop)", uid, m.CompanyID)
		return nil
	} else if err != nil && !isNotFoundLike(err) {
		// NotFound 以外は異常として返す
		log.Printf("[bootstrap] failed to check existing member by uid: uid=%s err=%v", uid, err)
		return err
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
		companyName = strings.TrimSpace(*p.CompanyName)
	}

	firstName := ""
	if p.FirstName != nil {
		firstName = strings.TrimSpace(*p.FirstName)
	}

	lastName := ""
	if p.LastName != nil {
		lastName = strings.TrimSpace(*p.LastName)
	}

	firstNameKana := ""
	if p.FirstNameKana != nil {
		firstNameKana = strings.TrimSpace(*p.FirstNameKana)
	}

	lastNameKana := ""
	if p.LastNameKana != nil {
		lastNameKana = strings.TrimSpace(*p.LastNameKana)
	}

	// ---------------------------------------------------------
	// 2) 新規作成時は名前必須
	// ---------------------------------------------------------
	if firstName == "" || lastName == "" {
		log.Printf("[bootstrap] invalid profile for new member: uid=%s firstName=%q lastName=%q", uid, firstName, lastName)
		return errors.New("member: invalid firstName")
	}

	// ---------------------------------------------------------
	// 3) companyName がある場合は先に Company を作る
	// ---------------------------------------------------------
	companyID := ""
	if companyName != "" {
		issuedID, err := s.Companies.NewID(ctx)
		if err != nil {
			log.Printf("[bootstrap] failed to issue companyID: uid=%s err=%v", uid, err)
			return err
		}

		companyEntity, err := companydom.NewCompanyWithNow(
			issuedID,
			companyName,
			uid, // admin
			uid, // createdBy
			uid, // updatedBy
			true,
			now,
		)
		if err != nil {
			log.Printf("[bootstrap] failed to create company entity: uid=%s companyName=%s err=%v", uid, companyName, err)
			return err
		}

		if err := s.Companies.Save(ctx, &companyEntity); err != nil {
			log.Printf("[bootstrap] failed to save company: uid=%s companyID=%s err=%v", uid, issuedID, err)
			return err
		}

		companyID = issuedID
		log.Printf("[bootstrap] company created: uid=%s companyID=%s name=%s", uid, companyID, companyName)
	} else {
		log.Printf("[bootstrap] no companyName provided, will create member only: uid=%s", uid)
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
		log.Printf("[bootstrap] failed to create member entity: uid=%s err=%v", uid, err)
		return err
	}

	if companyID != "" {
		memberEntity.CompanyID = companyID
	}

	if err := s.Members.Create(ctx, &memberEntity); err != nil {
		log.Printf("[bootstrap] failed to save member: uid=%s err=%v", uid, err)
		return err
	}

	log.Printf("[bootstrap] member created: uid=%s, companyID=%s, permissions=%d, status=%s",
		uid, companyID, len(memberEntity.Permissions), memberEntity.Status)

	return nil
}

// -------------------------------------------------------
// helpers
// -------------------------------------------------------

// ここは実装依存（Firestore NotFound / domain ErrNotFound 等）なので “それっぽい” 判定で安全側に倒す
func isNotFoundLike(err error) bool {
	if err == nil {
		return false
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
