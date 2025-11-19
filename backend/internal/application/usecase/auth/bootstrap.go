// backend/internal/application/usecase/auth/bootstrap.go
package auth

import (
	"context"
	"log"
	"strings"
	"time"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
)

// -------------------------------------------------------
// Repository Interfaces
// -------------------------------------------------------

type MemberRepository interface {
	// Firestore 版とも整合性を取るため、*Member を返す
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

	// ログ: リクエスト到達
	log.Printf("[bootstrap] request received: uid=%s email=%s", uid, email)

	// nil セーフ
	var p SignUpProfile
	if profile != nil {
		p = *profile
	}

	//---------------------------------------------------------
	// 1) Member 新規作成
	//---------------------------------------------------------

	memberEntity, err := memberdom.New(
		uid,
		now,
		memberdom.WithName(p.FirstName, p.LastName),
		memberdom.WithNameKana(p.FirstNameKana, p.LastNameKana),
		memberdom.WithEmail(email),
	)
	if err != nil {
		log.Printf("[bootstrap] failed to create member entity: uid=%s err=%v", uid, err)
		return err
	}

	// まず Member を保存（この時点では CompanyID 空のまま）
	if err := s.Members.Save(ctx, &memberEntity); err != nil {
		log.Printf("[bootstrap] failed to save member: uid=%s err=%v", uid, err)
		return err
	}

	log.Printf("[bootstrap] member created: uid=%s", uid)

	//---------------------------------------------------------
	// 2) 会社名が空ならここで終了（Company は作成しない）
	//---------------------------------------------------------

	companyName := strings.TrimSpace(p.CompanyName)
	if companyName == "" {
		log.Printf("[bootstrap] no companyName provided, finish with member only: uid=%s", uid)
		return nil
	}

	//---------------------------------------------------------
	// 3) Company を新規作成
	//---------------------------------------------------------

	companyID, err := s.Companies.NewID(ctx)
	if err != nil {
		log.Printf("[bootstrap] failed to issue companyID: uid=%s err=%v", uid, err)
		return err
	}

	companyEntity, err := companydom.NewCompanyWithNow(
		companyID,
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
		log.Printf("[bootstrap] failed to save company: uid=%s companyID=%s err=%v", uid, companyID, err)
		return err
	}

	log.Printf("[bootstrap] company created: uid=%s companyID=%s name=%s", uid, companyID, companyName)

	//---------------------------------------------------------
	// 4) Member に companyId を紐付けて更新
	//---------------------------------------------------------

	memberEntity.CompanyID = companyID

	if err := memberEntity.TouchUpdated(now, &uid); err != nil {
		log.Printf("[bootstrap] failed to touch member updated: uid=%s err=%v", uid, err)
		return err
	}

	if err := s.Members.Save(ctx, &memberEntity); err != nil {
		log.Printf("[bootstrap] failed to update member with companyID: uid=%s companyID=%s err=%v", uid, companyID, err)
		return err
	}

	log.Printf("[bootstrap] member linked to company: uid=%s companyID=%s", uid, companyID)

	return nil
}
