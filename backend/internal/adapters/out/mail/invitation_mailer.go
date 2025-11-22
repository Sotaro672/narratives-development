// backend/internal/adapters/out/mail/invitation_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	memdom "narratives/internal/domain/member"
)

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
	client         EmailClient
	fromAddress    string
	consoleBaseURL string // 例: "https://console.example.com"
}

// NewInvitationMailer は InvitationMailer のコンストラクタです。
//
//   - client         : SMTP / SendGrid などの具体的な EmailClient 実装
//   - fromAddress    : 送信元メールアドレス
//   - consoleBaseURL : "https://console.example.com" のような Console のベースURL
//
// 呼び出し例:
//
//	mailer := mail.NewInvitationMailer(smtpClient, "no-reply@example.com", "https://console.example.com")
func NewInvitationMailer(client EmailClient, fromAddress, consoleBaseURL string) *InvitationMailer {
	base := strings.TrimRight(consoleBaseURL, "/")
	return &InvitationMailer{
		client:         client,
		fromAddress:    fromAddress,
		consoleBaseURL: base,
	}
}

// buildInvitationURL は招待メール内に記載する URL を組み立てます。
// 仕様: https://console.example.com/invitation?token=INV_xxx
func (m *InvitationMailer) buildInvitationURL(token string) string {
	token = strings.TrimSpace(token)
	return fmt.Sprintf("%s/invitation?token=%s", m.consoleBaseURL, token)
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
	_ = ctx // 将来的にログ出力などで使う場合を考慮して受け取っておく

	invitationURL := m.buildInvitationURL(token)

	subject := "【Narratives】メンバー招待のお知らせ"

	// メール本文（プレーンテキスト例）
	// ★ 氏名行（「◯◯ 様」）は削除
	body := fmt.Sprintf(
		`管理者から「Narratives Console」へのメンバー招待が届いています。

下記のリンクを開き、パスワード設定およびプロフィール情報の登録を行ってください。

  招待リンク: %s

【所属情報（参考表示）】
  Company ID : %s
  Brands     : %s
  Permissions: %s

※本メールに心当たりがない場合は、このメッセージは破棄してください。

-- 
Narratives Console`,
		invitationURL,
		strings.TrimSpace(info.CompanyID),
		strings.Join(info.AssignedBrandIDs, ", "),
		strings.Join(info.Permissions, ", "),
	)

	return m.client.Send(ctx, m.fromAddress, strings.TrimSpace(toEmail), subject, body)
}
