// backend/internal/application/usecase/auth/bootstrap.go
package auth

import (
	"context"
	"strings"
	"time"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
)

// MemberRepository は member 用のアプリケーション層インターフェース
type MemberRepository interface {
	Save(ctx context.Context, m *memberdom.Member) error
	GetByID(ctx context.Context, id string) (*memberdom.Member, error)
}

// CompanyRepository は company 用のアプリケーション層インターフェース
type CompanyRepository interface {
	NewID(ctx context.Context) (string, error)
	Save(ctx context.Context, c *companydom.Company) error
}

// ★ フロントから受け取るプロフィール
type SignUpProfile struct {
	LastName      string `json:"lastName"`
	FirstName     string `json:"firstName"`
	LastNameKana  string `json:"lastNameKana"`
	FirstNameKana string `json:"firstNameKana"`
	CompanyName   string `json:"companyName"`
}

type BootstrapService struct {
	Members   MemberRepository
	Companies CompanyRepository
	// 必要なら PermissionRepo 等を足す
}

func (s *BootstrapService) Bootstrap(
	ctx context.Context,
	uid string,
	email string,
	profile *SignUpProfile,
) error {
	now := time.Now().UTC()

	// nil セーフなプロフィール
	var p SignUpProfile
	if profile != nil {
		p = *profile
	}

	// 1) Member を新規作成（CreatedAt のみ必須、UpdatedAt は nil のままでOK）
	//    ※ memberdom.New を使ってエンティティ整合性を担保
	memberEntity, err := memberdom.New(
		uid,
		now,
		memberdom.WithName(p.FirstName, p.LastName),
		memberdom.WithNameKana(p.FirstNameKana, p.LastNameKana),
		memberdom.WithEmail(email),
		// 必要ならここで WithPermissions(...) / WithStatus(...) などを追加
	)
	if err != nil {
		return err
	}

	if err := s.Members.Save(ctx, &memberEntity); err != nil {
		return err
	}

	// 2) 会社名がなければここで終了（companies は作成しない）
	name := strings.TrimSpace(p.CompanyName)
	if name == "" {
		return nil
	}

	// 3) Company を作成（Company は CreatedAt / UpdatedAt が必須の time.Time）
	companyID, err := s.Companies.NewID(ctx)
	if err != nil {
		return err
	}

	// entity.go の NewCompanyWithNow を正として利用
	//   func NewCompanyWithNow(
	//       id, name, admin, createdBy, updatedBy string,
	//       isActive bool,
	//       now time.Time,
	//   ) (Company, error)
	companyEntity, err := companydom.NewCompanyWithNow(
		companyID,
		name,
		uid, // admin
		uid, // createdBy
		uid, // updatedBy
		true,
		now,
	)
	if err != nil {
		return err
	}

	if err := s.Companies.Save(ctx, &companyEntity); err != nil {
		return err
	}

	// 4) Member に companyId を付与して UpdatedAt/UpdatedBy を更新
	//    memberEntity は値なのでポインタにして書き換える
	memberPtr := &memberEntity
	memberPtr.CompanyID = companyID

	if err := memberPtr.TouchUpdated(now, &uid); err != nil {
		return err
	}

	if err := s.Members.Save(ctx, memberPtr); err != nil {
		return err
	}

	return nil
}
