// backend/internal/domain/invitation/repository_port.go
package invitation

import (
	"context"
	"time"
)

// Repositoryは、利用者によるtoken検証と招待完了処理を提供します。
type Repository interface {
	ResolveInvitationInfoByToken(ctx context.Context, token string) (InvitationInfo, error)
	CompleteInvitation(ctx context.Context, completion InvitationCompletion) error
}

// DeliveryRepositoryは、招待tokenとdelivery outboxを管理します。
//
// CreateOrReuseInvitationDeliveryは、同一Memberに対する未使用かつ未失効のtokenを複数作成してはいけません。
// 同一Memberにpending、processing、retryable_failedのdeliveryが存在する場合は、同じdeliveryとtokenを再利用します。
// MarkInvitationDeliveryDeliveredは、deliveryのdelivered化とinvitationToken.deliveredAt更新を同一transactionで実行します。
// MarkInvitationDeliveryFailedは、deliveryのfailed化とinvitationToken.revokedAt更新を同一transactionで実行します。
// RevokeMemberInvitationsは、指定Memberに紐づく未使用・未失効のinvitation tokenをすべて失効させ、対応するdeliveryも終了状態にします。
type DeliveryRepository interface {
	CreateOrReuseInvitationDelivery(ctx context.Context, info InvitationInfo) (InvitationDelivery, error)
	ListDueInvitationDeliveries(ctx context.Context, now time.Time, limit int) ([]InvitationDelivery, error)
	ClaimInvitationDelivery(ctx context.Context, deliveryID string, now time.Time, processingUntil time.Time) (InvitationDelivery, error)
	MarkInvitationDeliveryDelivered(ctx context.Context, deliveryID string, expectedAttemptCount int, providerMessageID string, deliveredAt time.Time) error
	MarkInvitationDeliveryRetryableFailed(ctx context.Context, deliveryID string, expectedAttemptCount int, lastError string, nextAttemptAt time.Time, failedAt time.Time) error
	MarkInvitationDeliveryFailed(ctx context.Context, deliveryID string, expectedAttemptCount int, lastError string, failedAt time.Time) error
	RevokeMemberInvitations(ctx context.Context, memberID string, companyID string, revokedAt time.Time) error
}
