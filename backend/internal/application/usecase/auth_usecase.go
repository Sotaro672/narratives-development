// backend/internal/application/usecase/auth_usecase.go
package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	permdom "narratives/internal/domain/permission"
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
}

// MemberCompanyIDReader は、Firebase Auth UID から companyID だけを取得するための
// adapter 側の任意拡張です。
//
// member.Repository 本体は GetByID / ListByCompanyID に統一しているため、
// uid から companyID を逆引きする処理は repository port 本体には含めません。
type MemberCompanyIDReader interface {
	GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error)
}

// -------------------------------------------------------
// Bootstrap（管理アカウントの初回ログイン時に呼ばれる想定）
//
// 方針:
// - Firestore document ID と Firebase Auth UID は分離する
// - members/{autoDocID}.uid = Firebase Auth UID として保存する
// - 既存 member 確認は repository port 本体の GetByFirebaseUID では行わない
// - uid から companyID を解決できる adapter の場合のみ、ListByCompanyID + Filter.UID で冪等確認する
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
	//
	// repository port から GetByFirebaseUID は削除済み。
	// そのため、adapter 側が GetCompanyIDByFirebaseUID を実装している場合のみ、
	// companyID を取得したうえで ListByCompanyID + Filter.UID により既存 member を確認する。
	// ---------------------------------------------------------
	if r, ok := any(s.Members).(MemberCompanyIDReader); ok {
		companyID, err := r.GetCompanyIDByFirebaseUID(ctx, uid)
		if err == nil {
			companyID = strings.TrimSpace(companyID)

			if companyID != "" {
				res, listErr := s.Members.ListByCompanyID(
					ctx,
					companyID,
					memberdom.Filter{
						UID: uid,
					},
					common.Page{
						Number:  1,
						PerPage: 1,
					},
				)
				if listErr != nil && !isAuthNotFoundLike(listErr) {
					log.Printf("[bootstrap] failed to check existing member by uid: uid=%s companyID=%s err=%v", uid, companyID, listErr)
					return listErr
				}

				if len(res.Items) > 0 {
					rec := res.Items[0]
					log.Printf(
						"[bootstrap] member already exists: uid=%s memberDocID=%s companyID=%s (noop)",
						uid,
						rec.DocID,
						rec.Member.CompanyID,
					)
					return nil
				}
			}
		} else if !isAuthNotFoundLike(err) {
			log.Printf("[bootstrap] failed to resolve companyID by uid: uid=%s err=%v", uid, err)
			return err
		}
	} else {
		log.Printf("[bootstrap] member repository does not implement MemberCompanyIDReader: uid=%s", uid)
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

		savedCompany, err := s.Companies.Save(ctx, companyEntity, nil)
		if err != nil {
			log.Printf("[bootstrap] failed to save company: uid=%s companyID=%s err=%v", uid, issuedID, err)
			return err
		}

		companyID = savedCompany.ID
		if companyID == "" {
			companyID = issuedID
		}

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

	createdRecord, err := s.Members.Create(ctx, memberEntity)
	if err != nil {
		log.Printf("[bootstrap] failed to save member: uid=%s err=%v", uid, err)
		return err
	}

	log.Printf(
		"[bootstrap] member created: uid=%s, memberDocID=%s, companyID=%s, permissions=%d, status=%s",
		uid,
		createdRecord.DocID,
		createdRecord.Member.CompanyID,
		len(createdRecord.Member.Permissions),
		createdRecord.Member.Status,
	)

	return nil
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
