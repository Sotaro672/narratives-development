// backend/internal/application/usecase/invitation_delivery_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	invdom "narratives/internal/domain/invitation"
)

const (
	defaultInvitationDeliveryDispatchLimit = 50
	maxInvitationDeliveryDispatchLimit     = 200

	defaultInvitationDeliveryLeaseDuration = 5 * time.Minute
	defaultInvitationDeliveryRetryDelay    = time.Minute
	maxInvitationDeliveryRetryDelay        = 6 * time.Hour

	maxInvitationDeliveryErrorLength = 2000
)

// ==============================
// Inbound Port
// ==============================

// InvitationDeliveryUsecasePortは、招待メールdeliveryのdispatchと
// 個別送信処理を提供します。
type InvitationDeliveryUsecasePort interface {
	// DispatchDueは、送信時刻を迎えたdeliveryをqueueへ投入します。
	//
	// 戻り値は、正常にqueueへ投入できた件数です。
	DispatchDue(
		ctx context.Context,
		limit int,
	) (int, error)

	// Processは、指定deliveryをclaimし、メール送信と状態更新を行います。
	//
	// 同じdelivery IDに対して複数回呼ばれても、処理中または完了済みなら
	// 再送せず正常終了します。
	Process(
		ctx context.Context,
		deliveryID string,
	) error
}

// ==============================
// Outbound Port
// ==============================

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
// 状態更新はInvitationDeliveryUsecaseがDeliveryRepositoryを通して行います。
type InvitationDeliveryMailerPort interface {
	SendInvitationEmail(
		ctx context.Context,
		message InvitationMailMessage,
	) (InvitationMailSendResult, error)
}

// ==============================
// Usecase
// ==============================

type InvitationDeliveryUsecase struct {
	deliveryRepo  invdom.DeliveryRepository
	mailer        InvitationDeliveryMailerPort
	deliveryQueue InvitationDeliveryQueuePort

	now           func() time.Time
	leaseDuration time.Duration
	retryDelay    time.Duration
	maxRetryDelay time.Duration
}

// NewInvitationDeliveryUsecaseは、招待メールdelivery usecaseを生成します。
func NewInvitationDeliveryUsecase(
	deliveryRepo invdom.DeliveryRepository,
	mailer InvitationDeliveryMailerPort,
	deliveryQueue InvitationDeliveryQueuePort,
) *InvitationDeliveryUsecase {
	return &InvitationDeliveryUsecase{
		deliveryRepo:  deliveryRepo,
		mailer:        mailer,
		deliveryQueue: deliveryQueue,

		now:           time.Now,
		leaseDuration: defaultInvitationDeliveryLeaseDuration,
		retryDelay:    defaultInvitationDeliveryRetryDelay,
		maxRetryDelay: maxInvitationDeliveryRetryDelay,
	}
}

// ==============================
// Dispatch
// ==============================

// DispatchDueは、送信時刻を迎えたdeliveryをqueueへ投入します。
//
// 一部のdeliveryでqueue投入に失敗した場合も、残りのdeliveryの投入は
// 継続します。最初に発生したエラーを返します。
func (u *InvitationDeliveryUsecase) DispatchDue(
	ctx context.Context,
	limit int,
) (int, error) {
	if u == nil {
		return 0, fmt.Errorf(
			"invitation delivery usecase is nil",
		)
	}

	if u.deliveryRepo == nil {
		return 0, fmt.Errorf(
			"invitation delivery repository is not configured",
		)
	}

	if u.deliveryQueue == nil {
		return 0, fmt.Errorf(
			"invitation delivery queue is not configured",
		)
	}

	limit = normalizeInvitationDeliveryDispatchLimit(limit)
	now := u.currentTime()

	deliveries, err := u.deliveryRepo.ListDueInvitationDeliveries(
		ctx,
		now,
		limit,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"list due invitation deliveries failed: %w",
			err,
		)
	}

	enqueuedCount := 0
	var firstErr error

	for _, delivery := range deliveries {
		normalized, err := delivery.Normalize()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"normalize invitation delivery %q failed: %w",
					delivery.ID,
					err,
				)
			}

			continue
		}

		if normalized.IsTerminal() ||
			!normalized.IsDue(now) {
			continue
		}

		if err := u.deliveryQueue.EnqueueInvitationDelivery(
			ctx,
			normalized,
		); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"enqueue invitation delivery %q failed: %w",
					normalized.ID,
					err,
				)
			}

			continue
		}

		enqueuedCount++
	}

	return enqueuedCount, firstErr
}

// ==============================
// Process
// ==============================

// Processは、deliveryをclaimして招待メールを送信します。
//
// 処理順序：
//
//   - deliveryをprocessingとしてclaim
//   - 招待メールを送信
//   - 成功時はdeliveryをdeliveredにし、token.deliveredAtを設定
//   - 一時失敗時は同じtokenのままretryable_failedへ変更
//   - 最大試行回数到達または恒久失敗時はfailedにし、tokenを失効
func (u *InvitationDeliveryUsecase) Process(
	ctx context.Context,
	deliveryID string,
) error {
	if u == nil {
		return fmt.Errorf(
			"invitation delivery usecase is nil",
		)
	}

	if u.deliveryRepo == nil {
		return fmt.Errorf(
			"invitation delivery repository is not configured",
		)
	}

	if u.mailer == nil {
		return fmt.Errorf(
			"invitation delivery mailer is not configured",
		)
	}

	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return invdom.ErrInvitationDeliveryIDRequired
	}

	claimedAt := u.currentTime()
	processingUntil := claimedAt.Add(
		u.normalizedLeaseDuration(),
	)

	delivery, err := u.deliveryRepo.ClaimInvitationDelivery(
		ctx,
		deliveryID,
		claimedAt,
		processingUntil,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryNotFound,
		):
			// 既に削除されたtaskや古い重複taskは再試行しない。
			return nil

		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryNotClaimable,
		):
			// 別workerが処理中、または既に処理済み。
			return nil

		case errors.Is(
			err,
			invdom.ErrInvitationDeliveryAttemptLimit,
		):
			// ListDueInvitationDeliveries側で通常は除外される。
			return nil

		default:
			return fmt.Errorf(
				"claim invitation delivery %q failed: %w",
				deliveryID,
				err,
			)
		}
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize claimed invitation delivery %q failed: %w",
			deliveryID,
			err,
		)
	}

	if delivery.Status !=
		invdom.InvitationDeliveryStatusProcessing {
		return nil
	}

	info, err := delivery.InvitationInfo()
	if err != nil {
		return u.markInvalidDeliveryFailed(
			ctx,
			delivery,
			fmt.Errorf(
				"resolve invitation info from delivery: %w",
				err,
			),
		)
	}

	result, sendErr := u.mailer.SendInvitationEmail(
		ctx,
		InvitationMailMessage{
			IdempotencyKey: delivery.ID,
			ToEmail:        delivery.Email,
			Token:          delivery.Token,
			Info:           info,
		},
	)

	completedAt := u.currentTime()

	if sendErr == nil {
		if err := u.deliveryRepo.MarkInvitationDeliveryDelivered(
			ctx,
			delivery.ID,
			delivery.AttemptCount,
			strings.TrimSpace(result.ProviderMessageID),
			completedAt,
		); err != nil {
			return fmt.Errorf(
				"mark invitation delivery %q delivered failed: %w",
				delivery.ID,
				err,
			)
		}

		return nil
	}

	lastError := invitationDeliveryErrorText(sendErr)

	if result.Retryable &&
		delivery.AttemptCount < delivery.MaxAttempts {
		return u.markRetryableFailure(
			ctx,
			delivery,
			lastError,
			completedAt,
		)
	}

	if err := u.deliveryRepo.MarkInvitationDeliveryFailed(
		ctx,
		delivery.ID,
		delivery.AttemptCount,
		lastError,
		completedAt,
	); err != nil {
		return fmt.Errorf(
			"mark invitation delivery %q failed: %w",
			delivery.ID,
			err,
		)
	}

	// 恒久失敗または最大試行回数到達はbusiness上の終端状態です。
	// Cloud Tasksに同じ処理を再試行させないため、nilを返します。
	return nil
}

func (u *InvitationDeliveryUsecase) markRetryableFailure(
	ctx context.Context,
	delivery invdom.InvitationDelivery,
	lastError string,
	failedAt time.Time,
) error {
	nextAttemptAt := failedAt.Add(
		u.retryDelayForAttempt(delivery.AttemptCount),
	)

	retryDelivery, err := delivery.MarkRetryableFailed(
		lastError,
		nextAttemptAt,
		failedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"build retryable invitation delivery %q failed: %w",
			delivery.ID,
			err,
		)
	}

	if err := u.deliveryRepo.MarkInvitationDeliveryRetryableFailed(
		ctx,
		delivery.ID,
		delivery.AttemptCount,
		lastError,
		nextAttemptAt,
		failedAt,
	); err != nil {
		return fmt.Errorf(
			"mark invitation delivery %q retryable failed: %w",
			delivery.ID,
			err,
		)
	}

	if u.deliveryQueue == nil {
		// outbox stateは保存済みです。
		// DispatchDueによる回収を可能にするため、状態は戻しません。
		return fmt.Errorf(
			"invitation delivery queue is not configured",
		)
	}

	if err := u.deliveryQueue.EnqueueInvitationDelivery(
		ctx,
		retryDelivery,
	); err != nil {
		// retryable_failedとnextAttemptAtは保存済みです。
		// 後続のDispatchDueが同じdeliveryを再投入できます。
		return fmt.Errorf(
			"enqueue retryable invitation delivery %q failed: %w",
			delivery.ID,
			err,
		)
	}

	return nil
}

func (u *InvitationDeliveryUsecase) markInvalidDeliveryFailed(
	ctx context.Context,
	delivery invdom.InvitationDelivery,
	cause error,
) error {
	failedAt := u.currentTime()
	lastError := invitationDeliveryErrorText(cause)

	if err := u.deliveryRepo.MarkInvitationDeliveryFailed(
		ctx,
		delivery.ID,
		delivery.AttemptCount,
		lastError,
		failedAt,
	); err != nil {
		return fmt.Errorf(
			"mark invalid invitation delivery %q failed: %w",
			delivery.ID,
			err,
		)
	}

	return nil
}

// ==============================
// Helpers
// ==============================

func (u *InvitationDeliveryUsecase) currentTime() time.Time {
	if u != nil && u.now != nil {
		return u.now().UTC()
	}

	return time.Now().UTC()
}

func (u *InvitationDeliveryUsecase) normalizedLeaseDuration() time.Duration {
	if u == nil || u.leaseDuration <= 0 {
		return defaultInvitationDeliveryLeaseDuration
	}

	return u.leaseDuration
}

func (u *InvitationDeliveryUsecase) retryDelayForAttempt(
	attemptCount int,
) time.Duration {
	baseDelay := defaultInvitationDeliveryRetryDelay
	maxDelay := maxInvitationDeliveryRetryDelay

	if u != nil {
		if u.retryDelay > 0 {
			baseDelay = u.retryDelay
		}

		if u.maxRetryDelay > 0 {
			maxDelay = u.maxRetryDelay
		}
	}

	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}

	if attemptCount <= 1 {
		return baseDelay
	}

	delay := baseDelay

	for attempt := 1; attempt < attemptCount; attempt++ {
		if delay >= maxDelay {
			return maxDelay
		}

		if delay > maxDelay/2 {
			return maxDelay
		}

		delay *= 2
	}

	if delay > maxDelay {
		return maxDelay
	}

	return delay
}

func normalizeInvitationDeliveryDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultInvitationDeliveryDispatchLimit
	}

	if limit > maxInvitationDeliveryDispatchLimit {
		return maxInvitationDeliveryDispatchLimit
	}

	return limit
}

func invitationDeliveryErrorText(err error) string {
	if err == nil {
		return "invitation delivery failed"
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "invitation delivery failed"
	}

	runes := []rune(message)
	if len(runes) <= maxInvitationDeliveryErrorLength {
		return message
	}

	return string(runes[:maxInvitationDeliveryErrorLength])
}
