package auth

import (
	"context"
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
	GetByID(ctx context.Context, id string) (*memberdom.Member, error)
	Save(ctx context.Context, m *memberdom.Member) error
}

type CompanyRepository interface {
	NewID(ctx context.Context) (string, error)
	Save(ctx context.Context, c *companydom.Company) error
}

// -------------------------------------------------------
// フロントから受け取るプロフィール
// -------------------------------------------------------

type SignUpProfile struct {
	LastName      string `json:"lastName"`
	FirstName     string `json:"firstName"`
	LastNameKana  string `json:"lastNameKana"`
	FirstNameKana string `json:"firstNameKana"`
	CompanyName   string `json:"companyName"`
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
// -------------------------------------------------------

func (s *BootstrapService) Bootstrap(
	ctx context.Context,
	uid string,
	email string,
	profile *SignUpProfile,
) error {
	now := time.Now().UTC()
	log.Printf("[bootstrap] request received: uid=%s email=%s", uid, email)

	// nil セーフ
	var p SignUpProfile
	if profile != nil {
		p = *profile
	}

	companyName := strings.TrimSpace(p.CompanyName)

	//---------------------------------------------------------
	// 0) companyName がある場合は先に Company を作る
	//    - ここで失敗したら member は作らない（中途半端状態を防ぐ）
	//---------------------------------------------------------
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

	//---------------------------------------------------------
	// 1) Member 新規作成
	//    - status = "active"
	//    - permissions = 全権限（backend catalog 由来）
	//    - companyName がある場合は companyID を紐付けて保存
	//---------------------------------------------------------

	allPermNames := permdom.AllPermissionNames()

	memberEntity, err := memberdom.New(
		uid,
		now,
		memberdom.WithName(p.FirstName, p.LastName),
		memberdom.WithNameKana(p.FirstNameKana, p.LastNameKana),
		memberdom.WithEmail(email),
		memberdom.WithStatus("active"),
		memberdom.WithPermissions(allPermNames),
	)
	if err != nil {
		log.Printf("[bootstrap] failed to create member entity: uid=%s err=%v", uid, err)
		return err
	}

	// ★ company が作れている場合のみ紐付け（保存は1回で済ませる）
	if companyID != "" {
		memberEntity.CompanyID = companyID
	}

	if err := s.Members.Save(ctx, &memberEntity); err != nil {
		log.Printf("[bootstrap] failed to save member: uid=%s err=%v", uid, err)
		return err
	}

	log.Printf("[bootstrap] member created: uid=%s, companyID=%s, permissions=%d, status=%s",
		uid, companyID, len(memberEntity.Permissions), memberEntity.Status)

	return nil
}
