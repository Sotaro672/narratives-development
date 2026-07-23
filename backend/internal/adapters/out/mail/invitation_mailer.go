// backend/internal/adapters/out/mail/invitation_mailer.go
package mail

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	invitationuc "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	invitationdom "narratives/internal/domain/invitation"
	permdom "narratives/internal/domain/permission"
)

const fixedInvitationConsoleBaseURL = "https://narratives-console-dev.web.app"

type BrandNameResolver interface {
	GetByID(
		ctx context.Context,
		id string,
	) (branddom.Brand, error)
}

// EmailClientは、既存のAuthMailer、ContactMailerなどが使用する
// 従来のメール送信契約です。
//
// 既存メール機能との互換性を維持するため、このinterfaceは変更しません。
type EmailClient interface {
	Send(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
	) error
}

// EmailSendResultは、招待メール送信時に必要なprovider側の結果です。
type EmailSendResult struct {
	ProviderMessageID string
	Retryable         bool
}

// InvitationEmailClientは、delivery/outbox方式の招待メール専用契約です。
//
// SendWithResultは、provider message IDと、エラーが再試行可能かどうかを
// 呼出元へ返します。
type InvitationEmailClient interface {
	SendWithResult(
		ctx context.Context,
		from string,
		to string,
		subject string,
		body string,
		idempotencyKey string,
	) (EmailSendResult, error)
}

type InvitationMailer struct {
	client            InvitationEmailClient
	fromAddress       string
	consoleBaseURL    string
	companyRepo       companydom.Repository
	brandNameResolver BrandNameResolver
}

var _ invitationuc.InvitationDeliveryMailerPort = (*InvitationMailer)(nil)

func NewInvitationMailer(
	client InvitationEmailClient,
	fromAddress string,
	consoleBaseURL string,
	companyRepo companydom.Repository,
	brandResolver BrandNameResolver,
) *InvitationMailer {
	baseURL := strings.TrimSpace(consoleBaseURL)
	if baseURL == "" {
		baseURL = fixedInvitationConsoleBaseURL
	}

	return &InvitationMailer{
		client:            client,
		fromAddress:       strings.TrimSpace(fromAddress),
		consoleBaseURL:    strings.TrimRight(baseURL, "/"),
		companyRepo:       companyRepo,
		brandNameResolver: brandResolver,
	}
}

func (m *InvitationMailer) buildInvitationURL(
	token string,
) string {
	base := fixedInvitationConsoleBaseURL

	if m != nil {
		configuredBaseURL := strings.TrimSpace(
			m.consoleBaseURL,
		)
		if configuredBaseURL != "" {
			base = configuredBaseURL
		}
	}

	base = strings.TrimRight(base, "/")
	token = strings.TrimSpace(token)

	values := url.Values{}
	values.Set("token", token)

	return fmt.Sprintf(
		"%s/invitation?%s",
		base,
		values.Encode(),
	)
}

func (m *InvitationMailer) resolveCompanyDisplayName(
	ctx context.Context,
	companyID string,
) string {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return ""
	}

	if m == nil || m.companyRepo == nil {
		return companyID
	}

	companyEntity, err := m.companyRepo.GetByID(
		ctx,
		companyID,
	)
	if err != nil {
		return companyID
	}

	companyName := strings.TrimSpace(
		companyEntity.Name,
	)
	if companyName == "" {
		return companyID
	}

	return companyName
}

func (m *InvitationMailer) resolveBrandDisplayNames(
	ctx context.Context,
	brandIDs []string,
) []string {
	normalizedBrandIDs := normalizeInvitationDisplayValues(
		brandIDs,
	)
	if len(normalizedBrandIDs) == 0 {
		return nil
	}

	if m == nil || m.brandNameResolver == nil {
		return normalizedBrandIDs
	}

	results := make(
		[]string,
		0,
		len(normalizedBrandIDs),
	)

	for _, brandID := range normalizedBrandIDs {
		brandEntity, err := m.brandNameResolver.GetByID(
			ctx,
			brandID,
		)
		if err != nil {
			results = append(results, brandID)
			continue
		}

		brandName := strings.TrimSpace(
			brandEntity.Name,
		)
		if brandName == "" {
			results = append(results, brandID)
			continue
		}

		results = append(results, brandName)
	}

	return results
}

func (m *InvitationMailer) resolvePermissionDisplayNamesJa(
	permissionNames []string,
) []string {
	normalizedPermissionNames :=
		normalizeInvitationDisplayValues(
			permissionNames,
		)

	if len(normalizedPermissionNames) == 0 {
		return nil
	}

	results := make(
		[]string,
		0,
		len(normalizedPermissionNames),
	)

	for _, permissionName := range normalizedPermissionNames {
		displayName, ok :=
			permdom.DisplayNameJaFromPermissionName(
				permissionName,
			)

		displayName = strings.TrimSpace(displayName)
		if ok && displayName != "" {
			results = append(results, displayName)
			continue
		}

		results = append(results, permissionName)
	}

	return results
}

// SendInvitationEmailは、InvitationDeliveryUsecaseから受け取った
// delivery情報を使用して招待メールを送信します。
//
// Firestore上のdelivery stateやtoken stateは更新しません。
// 状態更新はInvitationDeliveryUsecaseとDeliveryRepositoryが担当します。
func (m *InvitationMailer) SendInvitationEmail(
	ctx context.Context,
	message invitationuc.InvitationMailMessage,
) (invitationuc.InvitationMailSendResult, error) {
	if m == nil {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("invitation mailer is nil")
	}

	if m.client == nil {
		return invitationuc.InvitationMailSendResult{
				Retryable: false,
			}, fmt.Errorf(
				"invitation email client is not configured",
			)
	}

	fromAddress := strings.TrimSpace(m.fromAddress)
	if fromAddress == "" {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, fmt.Errorf("from address is empty")
	}

	idempotencyKey := strings.TrimSpace(
		message.IdempotencyKey,
	)
	if idempotencyKey == "" {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, invitationdom.ErrInvitationDeliveryIDRequired
	}

	toEmail := strings.ToLower(
		strings.TrimSpace(message.ToEmail),
	)
	if toEmail == "" {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, invitationdom.ErrInvitationEmailRequired
	}

	token := strings.TrimSpace(message.Token)
	if token == "" {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, invitationdom.ErrInvitationTokenRequired
	}

	info, err := message.Info.Normalize()
	if err != nil {
		return invitationuc.InvitationMailSendResult{
				Retryable: false,
			}, fmt.Errorf(
				"normalize invitation mail info: %w",
				err,
			)
	}

	if info.Email != toEmail {
		return invitationuc.InvitationMailSendResult{
			Retryable: false,
		}, invitationdom.ErrInvitationEmailMismatch
	}

	invitationURL := m.buildInvitationURL(token)
	subject := "【AMOL】メンバー招待のお知らせ"

	companyDisplay := m.resolveCompanyDisplayName(
		ctx,
		info.CompanyID,
	)

	brandNames := m.resolveBrandDisplayNames(
		ctx,
		info.AssignedBrandIDs,
	)
	brandsDisplay := strings.Join(
		brandNames,
		", ",
	)

	permissionLabels :=
		m.resolvePermissionDisplayNamesJa(
			info.Permissions,
		)
	permissionsDisplay := strings.Join(
		permissionLabels,
		", ",
	)

	body := fmt.Sprintf(
		`管理者から「AMOL Console」へのメンバー招待が届いています。

下記のリンクを開き、パスワード設定およびプロフィール情報の登録を行ってください。

  招待リンク: %s

【所属情報（参考表示）】
  Company    : %s
  Brands     : %s
  Permissions: %s

※本メールに心当たりがない場合は、このメッセージは破棄してください。

--
AMOL Console`,
		invitationURL,
		companyDisplay,
		brandsDisplay,
		permissionsDisplay,
	)

	sendResult, err := m.client.SendWithResult(
		ctx,
		fromAddress,
		toEmail,
		subject,
		body,
		idempotencyKey,
	)
	if err != nil {
		return invitationuc.InvitationMailSendResult{
				ProviderMessageID: strings.TrimSpace(
					sendResult.ProviderMessageID,
				),
				Retryable: sendResult.Retryable,
			}, fmt.Errorf(
				"send invitation email failed: to=%s: %w",
				toEmail,
				err,
			)
	}

	return invitationuc.InvitationMailSendResult{
		ProviderMessageID: strings.TrimSpace(
			sendResult.ProviderMessageID,
		),
		Retryable: false,
	}, nil
}

func normalizeInvitationDisplayValues(
	values []string,
) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(
		map[string]struct{},
		len(values),
	)
	normalized := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
