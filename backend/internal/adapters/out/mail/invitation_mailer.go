// backend/internal/adapters/out/mail/invitation_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	memdom "narratives/internal/domain/member"
	permdom "narratives/internal/domain/permission"
)

// CompanyNameResolver は CompanyID から会社名を取得するためのポートです。
// internal/domain/company.Service などがこのインターフェースを実装します。
type CompanyNameResolver interface {
	GetCompanyNameByID(ctx context.Context, id string) (string, error)
}

// BrandNameResolver は BrandID からブランド名を取得するためのポートです。
// internal/domain/brand.Service などがこのインターフェースを実装します。
type BrandNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// InvitationMailerPort はアプリケーション層（usecase）が利用する
// 「招待メール送信用アウトバウンドポート」のインターフェースを表します。
//
//   - toEmail : 送信先メールアドレス（Member.Email）
//   - token   : 招待トークン（例: "INV_xxx"）
//   - info    : 招待対象メンバーのコンテキスト情報（companyId / brands / permissions など）
type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// EmailClient は実際のメール送信クライアント（SMTP / SendGrid / SES など）を
// 抽象化した下位レベルのインターフェースです。
type EmailClient interface {
	Send(ctx context.Context, from, to, subject, body string) error
}

// InvitationMailer は InvitationMailerPort の具象実装で、
// 内部で EmailClient を利用してメール送信を行います。
type InvitationMailer struct {
	client              EmailClient
	fromAddress         string
	consoleBaseURL      string // 例: "https://console.example.com"
	companyNameResolver CompanyNameResolver
	brandNameResolver   BrandNameResolver
}

// NewInvitationMailer は InvitationMailer のコンストラクタです。
//
//   - client         : SMTP / SendGrid などの具体的な EmailClient 実装
//   - fromAddress    : 送信元メールアドレス
//   - consoleBaseURL : "https://console.example.com" のような Console のベースURL
//   - companyResolver: CompanyID から会社名を引くためのリゾルバ
//   - brandResolver  : BrandID からブランド名を引くためのリゾルバ
//
// 呼び出し例:
//
//	mailer := mail.NewInvitationMailer(
//	  smtpClient,
//	  "no-reply@example.com",
//	  "https://console.example.com",
//	  companyService, // company.Service (CompanyNameResolver)
//	  brandService,   // brand.Service   (BrandNameResolver)
//	)
func NewInvitationMailer(
	client EmailClient,
	fromAddress,
	consoleBaseURL string,
	companyResolver CompanyNameResolver,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	base := strings.TrimRight(consoleBaseURL, "/")
	return &InvitationMailer{
		client:              client,
		fromAddress:         fromAddress,
		consoleBaseURL:      base,
		companyNameResolver: companyResolver,
		brandNameResolver:   brandResolver,
	}
}

// buildInvitationURL は招待メール内に記載する URL を組み立てます。
// 仕様: https://console.example.com/invitation?token=INV_xxx
func (m *InvitationMailer) buildInvitationURL(token string) string {
	token = strings.TrimSpace(token)
	return fmt.Sprintf("%s/invitation?token=%s", m.consoleBaseURL, token)
}

// resolveCompanyDisplayName は companyId から表示用の会社名を解決します。
// - 会社名取得に成功し、空でなければ会社名
// - 失敗/空 の場合はフォールバックとして companyId を返します。
func (m *InvitationMailer) resolveCompanyDisplayName(ctx context.Context, companyID string) string {
	id := strings.TrimSpace(companyID)
	if id == "" {
		return ""
	}
	if m.companyNameResolver == nil {
		return id
	}

	name, err := m.companyNameResolver.GetCompanyNameByID(ctx, id)
	if err != nil {
		// 会社名解決に失敗してもメール送信自体は継続したいので、
		// ログ等は上位で行う前提でここではフォールバックのみ行う。
		return id
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return id
	}
	return name
}

// resolveBrandDisplayNames は AssignedBrandIDs から表示用のブランド名一覧を解決します。
// - brandResolver が nil の場合: 正規化した ID のまま返却
// - 個別のブランド名取得に失敗した場合: 該当 ID をそのまま表示に使う
func (m *InvitationMailer) resolveBrandDisplayNames(ctx context.Context, brandIDs []string) []string {
	if len(brandIDs) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(brandIDs))
	for _, id := range brandIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			normalized = append(normalized, id)
		}
	}

	// リゾルバがなければ ID 表示のまま
	if m.brandNameResolver == nil {
		return normalized
	}

	results := make([]string, 0, len(normalized))
	for _, id := range normalized {
		name, err := m.brandNameResolver.GetNameByID(ctx, id)
		if err != nil {
			// 1件失敗しても他は進めたいので、該当 ID をそのまま使用
			results = append(results, id)
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			// 空名も ID にフォールバック
			results = append(results, id)
			continue
		}
		results = append(results, name)
	}
	return results
}

// resolvePermissionDisplayNamesJa は権限名スライスを日本語表示名に変換します。
// - permdom.DisplayNameJaFromPermissionName を利用
// - マッピングに失敗したものは元の name をフォールバック表示
func (m *InvitationMailer) resolvePermissionDisplayNamesJa(permissionNames []string) []string {
	if len(permissionNames) == 0 {
		return nil
	}

	out := make([]string, 0, len(permissionNames))
	for _, name := range permissionNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if ja, ok := permdom.DisplayNameJaFromPermissionName(name); ok && ja != "" {
			out = append(out, ja)
		} else {
			// カタログに無いものは key をそのまま表示
			out = append(out, name)
		}
	}
	return out
}

// SendInvitationEmail は、usecase から呼び出される招待メール送信処理です。
//
// 招待メール本文に「https://console.example.com/invitation?token=INV_xxx」を
// 埋め込んで送信します。
func (m *InvitationMailer) SendInvitationEmail(
	ctx context.Context,
	toEmail string,
	token string,
	info memdom.InvitationInfo,
) error {
	invitationURL := m.buildInvitationURL(token)

	subject := "【Narratives】メンバー招待のお知らせ"

	// CompanyID → 会社名に変換（失敗時はIDフォールバック）
	companyDisplay := m.resolveCompanyDisplayName(ctx, info.CompanyID)

	// AssignedBrandIDs → ブランド名に変換（失敗時はIDフォールバック）
	brandNames := m.resolveBrandDisplayNames(ctx, info.AssignedBrandIDs)
	brandsDisplay := strings.Join(brandNames, ", ")

	// Permissions → 日本語権限名に変換（失敗時はキーをフォールバック表示）
	permLabelsJa := m.resolvePermissionDisplayNamesJa(info.Permissions)
	permsDisplay := strings.Join(permLabelsJa, ", ")

	// メール本文（プレーンテキスト例）
	// ★ 氏名行（「◯◯ 様」）は削除
	body := fmt.Sprintf(
		`管理者から「Narratives Console」へのメンバー招待が届いています。

下記のリンクを開き、パスワード設定およびプロフィール情報の登録を行ってください。

  招待リンク: %s

【所属情報（参考表示）】
  Company   : %s
  Brands    : %s
  Permissions: %s

※本メールに心当たりがない場合は、このメッセージは破棄してください。

-- 
Narratives Console`,
		invitationURL,
		strings.TrimSpace(companyDisplay),
		brandsDisplay,
		permsDisplay,
	)

	return m.client.Send(ctx, m.fromAddress, strings.TrimSpace(toEmail), subject, body)
}
