// backend/internal/application/port/invitation_delivery.go
package port

import (
	"context"

	invdom "narratives/internal/domain/invitation"
)

// InvitationDeliveryQueuePortは、delivery IDをメール送信queueへ投入します。
//
// 実装側は、Cloud Tasksなどへtokenやemailを直接渡さず、
// delivery IDだけをpayloadとして使用します。
//
// delivery.NextAttemptAtが現在より未来の場合は、
// その時刻以降に処理されるようscheduleを設定します。
type InvitationDeliveryQueuePort interface {
	EnqueueInvitationDelivery(
		ctx context.Context,
		delivery invdom.InvitationDelivery,
	) error
}

// InvitationMailMessageは、招待メール送信adapterへ渡す入力です。
//
// IdempotencyKeyにはdelivery IDを設定します。
// メールadapterは、可能な場合、この値を外部providerの冪等キーとして
// 使用します。
type InvitationMailMessage struct {
	IdempotencyKey string
	ToEmail        string
	Token          string
	Info           invdom.InvitationInfo
}

// InvitationMailSendResultは、メールproviderから得られた送信結果です。
//
// Retryableは、送信エラーが一時的で再試行可能な場合にtrueとします。
type InvitationMailSendResult struct {
	ProviderMessageID string
	Retryable         bool
}

// InvitationDeliveryMailerPortは、招待メールを外部providerへ送信します。
//
// Firestoreのdelivery stateやtoken stateは更新しません。
// 状態更新はapplication usecaseがDeliveryRepositoryを通して行います。
type InvitationDeliveryMailerPort interface {
	SendInvitationEmail(
		ctx context.Context,
		message InvitationMailMessage,
	) (InvitationMailSendResult, error)
}
