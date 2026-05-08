// backend/internal/adapters/out/mail/invitation_mailer.go
package mail

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	memdom "narratives/internal/domain/member"
	permdom "narratives/internal/domain/permission"
)

const fixedInvitationConsoleBaseURL = "https://narratives-console-dev.web.app"

type CompanyNameResolver interface {
	GetCompanyNameByID(ctx context.Context, id string) (string, error)
}

type BrandNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

type EmailClient interface {
	Send(ctx context.Context, from, to, subject, body string) error
}

type InvitationMailer struct {
	client              EmailClient
	fromAddress         string
	consoleBaseURL      string
	companyNameResolver CompanyNameResolver
	brandNameResolver   BrandNameResolver
}

func NewInvitationMailer(
	client EmailClient,
	fromAddress,
	consoleBaseURL string,
	companyResolver CompanyNameResolver,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	return &InvitationMailer{
		client:              client,
		fromAddress:         fromAddress,
		consoleBaseURL:      fixedInvitationConsoleBaseURL,
		companyNameResolver: companyResolver,
		brandNameResolver:   brandResolver,
	}
}

func (m *InvitationMailer) buildInvitationURL(token string) string {
	base := strings.TrimRight(fixedInvitationConsoleBaseURL, "/")
	v := url.Values{}
	v.Set("token", token)
	return fmt.Sprintf("%s/invitation?%s", base, v.Encode())
}

func (m *InvitationMailer) resolveCompanyDisplayName(ctx context.Context, companyID string) string {
	id := companyID
	if id == "" {
		return ""
	}
	if m.companyNameResolver == nil {
		return id
	}

	name, err := m.companyNameResolver.GetCompanyNameByID(ctx, id)
	if err != nil {
		return id
	}

	if name == "" {
		return id
	}
	return name
}

func (m *InvitationMailer) resolveBrandDisplayNames(ctx context.Context, brandIDs []string) []string {
	if len(brandIDs) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(brandIDs))
	for _, id := range brandIDs {
		if id != "" {
			normalized = append(normalized, id)
		}
	}

	if len(normalized) == 0 {
		return nil
	}

	if m.brandNameResolver == nil {
		return normalized
	}

	results := make([]string, 0, len(normalized))
	for _, id := range normalized {
		name, err := m.brandNameResolver.GetNameByID(ctx, id)
		if err != nil {
			results = append(results, id)
			continue
		}
		if name == "" {
			results = append(results, id)
			continue
		}
		results = append(results, name)
	}
	return results
}

func (m *InvitationMailer) resolvePermissionDisplayNamesJa(permissionNames []string) []string {
	if len(permissionNames) == 0 {
		return nil
	}

	out := make([]string, 0, len(permissionNames))
	for _, name := range permissionNames {
		if name == "" {
			continue
		}
		if ja, ok := permdom.DisplayNameJaFromPermissionName(name); ok && ja != "" {
			out = append(out, ja)
		} else {
			out = append(out, name)
		}
	}
	return out
}

func (m *InvitationMailer) SendInvitationEmail(
	ctx context.Context,
	toEmail string,
	token string,
	info memdom.InvitationInfo,
) error {
	if m == nil {
		return fmt.Errorf("invitation mailer is nil")
	}
	if m.client == nil {
		return fmt.Errorf("email client is not configured")
	}
	if m.fromAddress == "" {
		return fmt.Errorf("from address is empty")
	}
	if toEmail == "" {
		return fmt.Errorf("to email is empty")
	}
	if token == "" {
		return fmt.Errorf("invitation token is empty")
	}

	invitationURL := m.buildInvitationURL(token)

	subject := "【Narratives】メンバー招待のお知らせ"

	companyDisplay := m.resolveCompanyDisplayName(ctx, info.CompanyID)
	brandNames := m.resolveBrandDisplayNames(ctx, info.AssignedBrandIDs)
	brandsDisplay := strings.Join(brandNames, ", ")
	permLabelsJa := m.resolvePermissionDisplayNamesJa(info.Permissions)
	permsDisplay := strings.Join(permLabelsJa, ", ")

	body := fmt.Sprintf(
		`管理者から「Narratives Console」へのメンバー招待が届いています。

下記のリンクを開き、パスワード設定およびプロフィール情報の登録を行ってください。

  招待リンク: %s

【所属情報（参考表示）】
  Company    : %s
  Brands     : %s
  Permissions: %s

※本メールに心当たりがない場合は、このメッセージは破棄してください。

-- 
Narratives Console`,
		invitationURL,
		companyDisplay,
		brandsDisplay,
		permsDisplay,
	)

	if err := m.client.Send(ctx, m.fromAddress, toEmail, subject, body); err != nil {
		return fmt.Errorf("send invitation email failed: to=%s: %w", toEmail, err)
	}

	return nil
}
