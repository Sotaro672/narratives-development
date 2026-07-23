// backend/internal/application/usecase/invitation_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	invdom "narratives/internal/domain/invitation"
	memdom "narratives/internal/domain/member"
)

// ==============================
// Inbound Ports
// ==============================

// InvitationQueryPort は、招待リンクのtokenからInvitationInfoを取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(
		ctx context.Context,
		token string,
	) (*invdom.InvitationInfo, error)
}

// InvitationCommandPort は、招待deliveryを作成または再利用し、
// メール送信queueへ投入するユースケースです。
//
// tokenは内部処理だけで使用し、呼出元には返しません。
type InvitationCommandPort interface {
	CreateInvitationAndSend(
		ctx context.Context,
		memberDocID string,
	) error
}

// InvitationCompletePort は、招待完了処理を行うユースケースです。
type InvitationCompletePort interface {
	CompleteInvitation(
		ctx context.Context,
		in CompleteInvitationInput,
	) error
}

// InvitationUsecasePort は、招待に関するQuery、Command、Completeをまとめた入口です。
type InvitationUsecasePort interface {
	InvitationQueryPort
	InvitationCommandPort
	InvitationCompletePort
}

// ==============================
// Outbound Ports
// ==============================

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

// ==============================
// Usecase
// ==============================

type invitationUsecase struct {
	invitationTokenRepo    invdom.Repository
	invitationDeliveryRepo invdom.DeliveryRepository
	memberRepo             memdom.Repository
	deliveryQueue          InvitationDeliveryQueuePort
}

// NewInvitationUsecase は、招待ユースケースの唯一の生成入口です。
//
// invitationTokenRepo:
//   - tokenの検証
//   - 招待完了
//
// invitationDeliveryRepo:
//   - tokenとdelivery outboxの作成または再利用
//   - delivery stateの永続化
//
// deliveryQueue:
//   - delivery IDの非同期メール送信queueへの投入
func NewInvitationUsecase(
	invitationTokenRepo invdom.Repository,
	invitationDeliveryRepo invdom.DeliveryRepository,
	memberRepo memdom.Repository,
	deliveryQueue InvitationDeliveryQueuePort,
) InvitationUsecasePort {
	return &invitationUsecase{
		invitationTokenRepo:    invitationTokenRepo,
		invitationDeliveryRepo: invitationDeliveryRepo,
		memberRepo:             memberRepo,
		deliveryQueue:          deliveryQueue,
	}
}

// ==============================
// Query
// ==============================

// POST /invitations/validate
func (u *invitationUsecase) GetInvitationInfo(
	ctx context.Context,
	token string,
) (*invdom.InvitationInfo, error) {
	if u == nil {
		return nil, fmt.Errorf("invitation usecase is nil")
	}

	if u.invitationTokenRepo == nil {
		return nil, fmt.Errorf(
			"invitation token repository is not configured",
		)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, invdom.ErrInvitationTokenNotFound
	}

	info, err := u.invitationTokenRepo.ResolveInvitationInfoByToken(
		ctx,
		token,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationTokenNotFound,
		),
			errors.Is(
				err,
				invdom.ErrInvitationTokenExpired,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenUsed,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenRevoked,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenNotDelivered,
			):
			return nil, invdom.ErrInvitationTokenNotFound

		default:
			return nil, err
		}
	}

	normalizedInfo, err := info.Normalize()
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrInvitationMemberIDRequired,
		) {
			return nil, invdom.ErrInvitationTokenNotFound
		}

		return nil, err
	}

	return &normalizedInfo, nil
}

// ==============================
// Command: Create & Enqueue
// ==============================

// POST /invitations
func (u *invitationUsecase) CreateInvitationAndSend(
	ctx context.Context,
	memberDocID string,
) error {
	if u == nil {
		return fmt.Errorf("invitation usecase is nil")
	}

	if u.invitationDeliveryRepo == nil {
		return fmt.Errorf(
			"invitation delivery repository is not configured",
		)
	}

	if u.memberRepo == nil {
		return fmt.Errorf(
			"member repository is not configured",
		)
	}

	if u.deliveryQueue == nil {
		return fmt.Errorf(
			"invitation delivery queue is not configured",
		)
	}

	memberDocID = strings.TrimSpace(memberDocID)
	if memberDocID == "" {
		return fmt.Errorf("memberDocID is empty")
	}

	companyID := strings.TrimSpace(
		CompanyIDFromContext(ctx),
	)
	if companyID == "" {
		return fmt.Errorf("companyID is empty")
	}

	rec, err := u.memberRepo.GetByID(
		ctx,
		memberDocID,
	)
	if err != nil {
		return fmt.Errorf(
			"find member by doc id failed: %w",
			err,
		)
	}

	rec.DocID = strings.TrimSpace(rec.DocID)
	if rec.DocID == "" {
		return memdom.ErrNotFound
	}

	memberCompanyID := strings.TrimSpace(
		rec.Member.CompanyID,
	)
	if memberCompanyID != companyID {
		return memdom.ErrNotFound
	}

	info, err := invdom.NewInvitationInfo(
		rec.DocID,
		memberCompanyID,
		rec.Member.AssignedBrands,
		rec.Member.Permissions,
		rec.Member.Email,
	)
	if err != nil {
		return fmt.Errorf(
			"create invitation info failed: %w",
			err,
		)
	}

	if !strings.EqualFold(
		strings.TrimSpace(rec.Member.Status),
		"active",
	) {
		status := "inactive"

		if _, err := u.memberRepo.Update(
			ctx,
			rec.DocID,
			memdom.MemberPatch{
				Status: &status,
			},
		); err != nil {
			return fmt.Errorf(
				"update member status before invitation failed: %w",
				err,
			)
		}
	}

	delivery, err :=
		u.invitationDeliveryRepo.CreateOrReuseInvitationDelivery(
			ctx,
			info,
		)
	if err != nil {
		return fmt.Errorf(
			"create or reuse invitation delivery failed: %w",
			err,
		)
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize invitation delivery failed: %w",
			err,
		)
	}

	switch delivery.Status {
	case invdom.InvitationDeliveryStatusPending,
		invdom.InvitationDeliveryStatusRetryableFailed:
		if err := u.deliveryQueue.EnqueueInvitationDelivery(
			ctx,
			delivery,
		); err != nil {
			return fmt.Errorf(
				"enqueue invitation delivery failed: %w",
				err,
			)
		}

		return nil

	case invdom.InvitationDeliveryStatusProcessing:
		// 既にworkerが処理中のため、重複queue投入は行わない。
		return nil

	case invdom.InvitationDeliveryStatusDelivered:
		// 既に同じtokenのメール送信が完了している。
		return nil

	case invdom.InvitationDeliveryStatusFailed:
		return fmt.Errorf(
			"invitation delivery is already failed",
		)

	default:
		return invdom.ErrInvitationDeliveryStatusInvalid
	}
}

// ==============================
// Command: Complete
// ==============================

type CompleteInvitationInput struct {
	Token         string
	UID           string
	LastName      string
	LastNameKana  string
	FirstName     string
	FirstNameKana string
	Email         string
}

func (u *invitationUsecase) CompleteInvitation(
	ctx context.Context,
	in CompleteInvitationInput,
) error {
	if u == nil {
		return fmt.Errorf("invitation usecase is nil")
	}

	if u.invitationTokenRepo == nil {
		return fmt.Errorf(
			"invitation token repository is not configured",
		)
	}

	completion, err := invdom.NewInvitationCompletion(
		in.Token,
		in.UID,
		in.LastName,
		in.LastNameKana,
		in.FirstName,
		in.FirstNameKana,
		in.Email,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationTokenRequired,
		),
			errors.Is(
				err,
				invdom.ErrInvitationUIDRequired,
			):
			return fmt.Errorf("token_or_uid_required")

		case errors.Is(
			err,
			invdom.ErrInvitationNameFieldsRequired,
		):
			return fmt.Errorf("name_fields_required")

		case errors.Is(
			err,
			invdom.ErrInvitationEmailRequired,
		):
			return fmt.Errorf("email_required")

		default:
			return fmt.Errorf(
				"create invitation completion failed: %w",
				err,
			)
		}
	}

	if err := u.invitationTokenRepo.CompleteInvitation(
		ctx,
		completion,
	); err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationTokenNotFound,
		),
			errors.Is(
				err,
				invdom.ErrInvitationTokenExpired,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenUsed,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenRevoked,
			),
			errors.Is(
				err,
				invdom.ErrInvitationTokenNotDelivered,
			):
			return invdom.ErrInvitationTokenNotFound

		case errors.Is(
			err,
			invdom.ErrInvitationMemberNotFound,
		),
			errors.Is(
				err,
				invdom.ErrInvitationCompanyMismatch,
			):
			return memdom.ErrNotFound

		case errors.Is(
			err,
			invdom.ErrInvitationEmailMismatch,
		):
			return fmt.Errorf("email_mismatch")

		case errors.Is(
			err,
			invdom.ErrInvitationUIDAlreadyInUse,
		):
			return fmt.Errorf("firebase_uid_already_in_use")

		default:
			return fmt.Errorf(
				"complete invitation transaction failed: %w",
				err,
			)
		}
	}

	return nil
}
