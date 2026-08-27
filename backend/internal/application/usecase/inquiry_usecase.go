// backend/internal/application/usecase/inquiry_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	applicationport "narratives/internal/application/port"
	avatardom "narratives/internal/domain/avatar"
	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryCreatedMailer is the mail port required by InquiryUsecase.
//
// Implemented by adapters/out/mail.InquiryMailer.
type InquiryCreatedMailer interface {
	SendInquiryCreatedNotification(
		ctx context.Context,
		from string,
		to string,
		inq inquirydom.Inquiry,
	) error
}

// AvatarEmailResolver is the minimal avatar reader required for resolving
// Inquiry.AvatarID -> Avatar.UserID -> Firebase Auth email.
//
// Avatar.UserID is treated as Firebase Auth UID.
type AvatarEmailResolver interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// InquiryUsecase は Inquiry の command を扱います.
//
// 画像は Inquiry.Images として Inquiry 集約内で管理します。
// Firebase Storage への保存・削除は frontend / application 層の責務とし、
// domain / repository では fileUrl と objectPath のメタデータのみ扱います.
//
// 返信は Inquiry.Content へ追記せず、Firestore subcollection:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// に保存します。
type InquiryUsecase struct {
	repo      inquirydom.Repository
	replyRepo inquirydom.ReplyRepository

	mailer   InquiryCreatedMailer
	mailFrom string

	avatarEmailResolver AvatarEmailResolver
	authUserGetter      applicationport.AuthUserReader

	now func() time.Time
}

// NewInquiryUsecase は InquiryUsecase の唯一の生成入口です.
//
// InquiryUsecase が必要とする依存はここに集約します。
// replyRepo は reply 作成・一覧取得・既読化・avatar 未読件数集計で必須です.
//
// mailer / mailFrom / avatarEmailResolver / authUserGetter はメール送信用です。
// メール送信を使わない場合は nil / 空文字を渡してください。
func NewInquiryUsecase(
	repo inquirydom.Repository,
	replyRepo inquirydom.ReplyRepository,
	mailer InquiryCreatedMailer,
	mailFrom string,
	avatarEmailResolver AvatarEmailResolver,
	authUserGetter applicationport.AuthUserReader,
) *InquiryUsecase {
	return &InquiryUsecase{
		repo:                repo,
		replyRepo:           replyRepo,
		mailer:              mailer,
		mailFrom:            mailFrom,
		avatarEmailResolver: avatarEmailResolver,
		authUserGetter:      authUserGetter,
		now:                 time.Now,
	}
}

// SetNowFunc はテスト用に現在時刻関数を差し替えます。
func (uc *InquiryUsecase) SetNowFunc(now func() time.Time) {
	if uc == nil || now == nil {
		return
	}

	uc.now = now
}

// Create は Inquiry を作成します.
//
// 作成後、メール設定がある場合は問い合わせ作成通知メールを送信します。
// メール送信に失敗した場合、Inquiry 作成自体は完了済みのため、作成済み Inquiry と error を返します。
func (uc *InquiryUsecase) Create(
	ctx context.Context,
	inq inquirydom.Inquiry,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	created, err := uc.repo.Create(ctx, inq)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	if err := uc.sendInquiryCreatedMail(
		ctx,
		created,
	); err != nil {
		return created, err
	}

	return created, nil
}

// CreateInquiryReplyInput は company member / avatar が問い合わせへ返信する入力です.
//
// Console からの返信では SenderType=member, SenderID=memberId を使います。
// Mall から avatar が返信する場合は SenderType=avatar, SenderID=avatarId を使います。
type CreateInquiryReplyInput struct {
	InquiryID  string
	SenderType inquirydom.ReplySenderType
	SenderID   string
	Content    string
	Images     []inquirydom.ImageFile
}

// CreateReply は Inquiry の reply subcollection に返信を作成します.
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// Inquiry.Content へ返信本文を追記しません。
// Inquiry 本体は updatedAt / updatedBy を更新します。
// company member の返信時、status=open の Inquiry は in_progress へ遷移させます。
func (uc *InquiryUsecase) CreateReply(
	ctx context.Context,
	in CreateInquiryReplyInput,
) (inquirydom.Reply, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Reply{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	if uc.replyRepo == nil {
		return inquirydom.Reply{},
			fmt.Errorf("inquiry usecase: reply repository is nil")
	}

	inquiryID := in.InquiryID
	senderID := in.SenderID

	if inquiryID == "" {
		return inquirydom.Reply{},
			inquirydom.ErrInvalidReplyInquiryID
	}

	if senderID == "" {
		return inquirydom.Reply{},
			inquirydom.ErrInvalidReplySenderID
	}

	current, err := uc.repo.GetByID(
		ctx,
		inquiryID,
	)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	if current.Status ==
		inquirydom.InquiryStatusClosed {
		return inquirydom.Reply{},
			inquirydom.ErrInquiryAlreadyClosed
	}

	now := uc.nowUTC()

	replyID := newInquiryReplyID(now)
	reply, err := inquirydom.NewReply(
		replyID,
		inquiryID,
		in.SenderType,
		senderID,
		in.Content,
		in.Images,
		now,
		senderID,
	)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	created, err := uc.replyRepo.Create(
		ctx,
		reply,
	)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	updatedBy := senderID
	patch := inquirydom.InquiryPatch{
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	}

	if in.SenderType ==
		inquirydom.ReplySenderTypeMember &&
		current.Status ==
			inquirydom.InquiryStatusOpen {
		status :=
			inquirydom.InquiryStatusInProgress
		patch.Status = &status
	}

	if _, err := uc.repo.Update(
		ctx,
		inquiryID,
		patch,
	); err != nil {
		return inquirydom.Reply{}, err
	}

	return created, nil
}

// CreateReplyByMember は company member が問い合わせへ返信します.
//
// Console 用の shorthand です。
func (uc *InquiryUsecase) CreateReplyByMember(
	ctx context.Context,
	inquiryID string,
	memberID string,
	content string,
	images []inquirydom.ImageFile,
) (inquirydom.Reply, error) {
	return uc.CreateReply(
		ctx,
		CreateInquiryReplyInput{
			InquiryID:  inquiryID,
			SenderType: inquirydom.ReplySenderTypeMember,
			SenderID:   memberID,
			Content:    content,
			Images:     images,
		},
	)
}

// EnsureReplyByMember は company member 名義の reply を決定的 ID で冪等に作成します.
//
// 返金完了通知など、同一 application operation の再試行で同じ reply を
// 重複作成してはならない処理で使用します.
//
// replyID は caller が operation に対して決定的に生成した値を渡します。
// Create が ErrConflict を返した場合は同じ replyID の既存 reply を取得し、
// immutable fields が今回の要求と一致する場合のみ成功済みとして扱います。
func (uc *InquiryUsecase) EnsureReplyByMember(
	ctx context.Context,
	replyID string,
	inquiryID string,
	memberID string,
	content string,
	images []inquirydom.ImageFile,
) (inquirydom.Reply, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Reply{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}
	if uc.replyRepo == nil {
		return inquirydom.Reply{},
			fmt.Errorf("inquiry usecase: reply repository is nil")
	}
	if replyID == "" {
		return inquirydom.Reply{},
			inquirydom.ErrInvalidReplyID
	}
	if inquiryID == "" {
		return inquirydom.Reply{},
			inquirydom.ErrInvalidReplyInquiryID
	}
	if memberID == "" {
		return inquirydom.Reply{},
			inquirydom.ErrInvalidReplySenderID
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Reply{}, err
	}
	if current.Status == inquirydom.InquiryStatusClosed {
		return inquirydom.Reply{},
			inquirydom.ErrInquiryAlreadyClosed
	}

	now := uc.nowUTC()
	expected, err := inquirydom.NewReply(
		replyID,
		inquiryID,
		inquirydom.ReplySenderTypeMember,
		memberID,
		content,
		images,
		now,
		memberID,
	)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	created, err := uc.replyRepo.Create(ctx, expected)
	createdNow := err == nil
	if err != nil {
		if !errors.Is(err, inquirydom.ErrConflict) {
			return inquirydom.Reply{}, err
		}

		existing, getErr := uc.replyRepo.GetByID(
			ctx,
			inquiryID,
			replyID,
		)
		if getErr != nil {
			return inquirydom.Reply{}, getErr
		}
		if !sameEnsuredInquiryReply(existing, expected) {
			return inquirydom.Reply{},
				inquirydom.ErrConflict
		}

		created = existing
	}

	// A retry after the reply was already created only needs to repair the
	// Inquiry-side transition when it is still open. If the Inquiry already
	// progressed or resolved, avoid changing updatedAt again.
	if !createdNow &&
		current.Status != inquirydom.InquiryStatusOpen {
		return created, nil
	}

	updatedBy := memberID
	patch := inquirydom.InquiryPatch{
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	}
	if current.Status == inquirydom.InquiryStatusOpen {
		status := inquirydom.InquiryStatusInProgress
		patch.Status = &status
	}

	if _, err := uc.repo.Update(
		ctx,
		inquiryID,
		patch,
	); err != nil {
		return inquirydom.Reply{}, err
	}

	return created, nil
}

func sameEnsuredInquiryReply(
	existing inquirydom.Reply,
	expected inquirydom.Reply,
) bool {
	if existing.ID != expected.ID ||
		existing.InquiryID != expected.InquiryID ||
		existing.SenderType != expected.SenderType ||
		existing.SenderID != expected.SenderID ||
		existing.Content != expected.Content ||
		existing.CreatedBy != expected.CreatedBy ||
		existing.DeletedAt != nil ||
		len(existing.Images) != len(expected.Images) {
		return false
	}

	for i := range existing.Images {
		if !reflect.DeepEqual(
			existing.Images[i],
			expected.Images[i],
		) {
			return false
		}
	}

	return true
}

// CreateReplyByAvatar は avatar が問い合わせへ返信します.
//
// Mall 用の shorthand です。
func (uc *InquiryUsecase) CreateReplyByAvatar(
	ctx context.Context,
	inquiryID string,
	avatarID string,
	content string,
	images []inquirydom.ImageFile,
) (inquirydom.Reply, error) {
	return uc.CreateReply(
		ctx,
		CreateInquiryReplyInput{
			InquiryID:  inquiryID,
			SenderType: inquirydom.ReplySenderTypeAvatar,
			SenderID:   avatarID,
			Content:    content,
			Images:     images,
		},
	)
}

// ListReplies は Inquiry の reply subcollection を取得します.
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
func (uc *InquiryUsecase) ListReplies(
	ctx context.Context,
	inquiryID string,
) ([]inquirydom.Reply, error) {
	if uc == nil || uc.replyRepo == nil {
		return nil,
			fmt.Errorf("inquiry usecase: reply repository is nil")
	}

	if inquiryID == "" {
		return nil,
			inquirydom.ErrInvalidReplyInquiryID
	}

	return uc.replyRepo.ListByInquiryID(
		ctx,
		inquiryID,
	)
}

// CountUnreadByAvatarIDInput は avatar 向けの未読件数集計入力です.
//
// avatarId に紐づく Inquiry 配下 replies の未読数を返します。
// avatar 側では、avatar 自身が起票した Inquiry 本体を未読件数に含めません。
// member が avatar 宛に返信した unread reply を count 対象にします。
type CountUnreadByAvatarIDInput struct {
	AvatarID string
	Filter   inquirydom.Filter
}

// CountInquiryBadgeByAvatarIDInput は avatar 向け Chat badge 件数集計入力です.
//
// Chat badge は未読 reply 数と、avatar の close 操作待ちである
// status=resolved の Inquiry 数を合算します。
type CountInquiryBadgeByAvatarIDInput struct {
	AvatarID string
	Filter   inquirydom.Filter
}

// InquiryBadgeCount は avatar 向け Chat badge の内訳です.
//
// UnreadReplyCount は member 等から届いた未読 reply 数です。
// ClosePendingCount は status=resolved の Inquiry 数です。
// TotalCount は両者の合計です。
type InquiryBadgeCount struct {
	UnreadReplyCount int `json:"unreadReplyCount"`

	ClosePendingCount int `json:"closePendingCount"`

	TotalCount int `json:"totalCount"`
}

// CountUnreadByAvatarID は avatarId のみで avatar 向け未読件数を返します.
//
// Inquiry 本体の IsRead は avatar 自身が起票した初回本文に対する既読状態のため、
// avatar 向け未読数には含めません.
//
// count 条件は ReplyRepository 側で以下として扱います:
//
//	!reply.IsRead
//	&& !(reply.SenderType == avatar && reply.SenderID == avatarID)
func (uc *InquiryUsecase) CountUnreadByAvatarID(
	ctx context.Context,
	in CountUnreadByAvatarIDInput,
) (int, error) {
	if uc == nil || uc.replyRepo == nil {
		return 0,
			fmt.Errorf("inquiry usecase: reply repository is nil")
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return 0,
			inquirydom.ErrInvalidAvatarID
	}

	filter := in.Filter
	filter.AvatarID = &avatarID

	return uc.replyRepo.CountUnreadByAvatarID(
		ctx,
		avatarID,
		filter,
	)
}

// CountBadgeByAvatarID は avatar 向け Chat badge 件数を返します.
//
// badge は以下を独立した attention として合算します。
// - avatar が受け取った未読 reply 数
// - status=resolved で avatar の close 操作待ちとなっている Inquiry 数
//
// resolved Inquiry に未読 reply が存在する場合は両方を加算します。
// Inquiry の既読状態と close 操作待ちは別概念として扱います。
func (uc *InquiryUsecase) CountBadgeByAvatarID(
	ctx context.Context,
	in CountInquiryBadgeByAvatarIDInput,
) (InquiryBadgeCount, error) {
	if uc == nil || uc.repo == nil {
		return InquiryBadgeCount{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	avatarID := in.AvatarID
	if avatarID == "" {
		return InquiryBadgeCount{},
			inquirydom.ErrInvalidAvatarID
	}

	unreadReplyCount, err :=
		uc.CountUnreadByAvatarID(
			ctx,
			CountUnreadByAvatarIDInput{
				AvatarID: avatarID,
				Filter:   in.Filter,
			},
		)
	if err != nil {
		return InquiryBadgeCount{}, err
	}

	closePendingCount := 0
	resolvedStatus :=
		inquirydom.InquiryStatusResolved

	if in.Filter.Status == nil ||
		*in.Filter.Status == resolvedStatus {
		closeFilter := in.Filter
		closeFilter.AvatarID = &avatarID
		closeFilter.Status = &resolvedStatus

		result, err :=
			uc.repo.ListByAvatarID(
				ctx,
				avatarID,
				closeFilter,
				inquirydom.Sort{},
				inquirydom.Page{
					Number:  1,
					PerPage: 1,
				},
			)
		if err != nil {
			return InquiryBadgeCount{}, err
		}

		closePendingCount =
			result.TotalCount
	}

	return InquiryBadgeCount{
		UnreadReplyCount: unreadReplyCount,

		ClosePendingCount: closePendingCount,

		TotalCount: unreadReplyCount +
			closePendingCount,
	}, nil
}

// ResolveInquiryInput は company member が問い合わせを対処済みにする入力です。
type ResolveInquiryInput struct {
	InquiryID string
	MemberID  string
}

// ResolveByMember は company member が Inquiry を resolved にします.
//
// company member は close せず、対処済みとして resolved にします。
func (uc *InquiryUsecase) ResolveByMember(
	ctx context.Context,
	in ResolveInquiryInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	memberID := in.MemberID

	if inquiryID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidID
	}

	if memberID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidResolvedBy
	}

	current, err :=
		uc.repo.GetByID(
			ctx,
			inquiryID,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			err
	}

	now := uc.nowUTC()

	if err := current.ResolveByMember(
		memberID,
		now,
	); err != nil {
		return inquirydom.Inquiry{},
			err
	}

	return uc.repo.Update(
		ctx,
		current.ID,
		inquirydom.InquiryPatch{
			Status: &current.Status,

			ResolvedAt: current.ResolvedAt,

			ResolvedBy: current.ResolvedBy,

			UpdatedAt: &current.UpdatedAt,

			UpdatedBy: current.UpdatedBy,
		},
	)
}

// ReopenInquiryInput は company member が問い合わせを open に戻す入力です。
type ReopenInquiryInput struct {
	InquiryID string
	MemberID  string
}

// ReopenByMember は company member が Inquiry を open に戻します。
func (uc *InquiryUsecase) ReopenByMember(
	ctx context.Context,
	in ReopenInquiryInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	memberID := in.MemberID

	if inquiryID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidID
	}

	if memberID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidUpdatedBy
	}

	current, err :=
		uc.repo.GetByID(
			ctx,
			inquiryID,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			err
	}

	now := uc.nowUTC()

	if err := current.ReopenByMember(
		memberID,
		now,
	); err != nil {
		return inquirydom.Inquiry{},
			err
	}

	return uc.repo.Update(
		ctx,
		current.ID,
		inquirydom.InquiryPatch{
			Status: &current.Status,

			ResolvedAt: current.ResolvedAt,

			ResolvedBy: current.ResolvedBy,

			ClosedAt: current.ClosedAt,

			ClosedBy: current.ClosedBy,

			UpdatedAt: &current.UpdatedAt,

			UpdatedBy: current.UpdatedBy,
		},
	)
}

// CloseInquiryByAvatarInput は avatar が問い合わせを close する入力です。
type CloseInquiryByAvatarInput struct {
	InquiryID string
	AvatarID  string
}

// CloseByAvatar は avatar が Inquiry を closed にします.
//
// Inquiry を起票した avatar のみ close できます。
func (uc *InquiryUsecase) CloseByAvatar(
	ctx context.Context,
	in CloseInquiryByAvatarInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	avatarID := in.AvatarID

	if inquiryID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidID
	}

	if avatarID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidAvatarID
	}

	current, err :=
		uc.repo.GetByID(
			ctx,
			inquiryID,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			err
	}

	now := uc.nowUTC()

	if err := current.CloseByAvatar(
		avatarID,
		now,
	); err != nil {
		return inquirydom.Inquiry{},
			err
	}

	return uc.repo.Update(
		ctx,
		current.ID,
		inquirydom.InquiryPatch{
			Status: &current.Status,

			ClosedAt: current.ClosedAt,

			ClosedBy: current.ClosedBy,

			UpdatedAt: &current.UpdatedAt,

			UpdatedBy: current.UpdatedBy,
		},
	)
}

// MarkInquiryAsReadInput は Inquiry を既読にする入力です.
//
// ReaderSenderType / ReaderSenderID は reply の既読化で自分の reply を除外するために使います.
//
// Console 側で company member が読む場合:
//
//	ReaderSenderType: inquirydom.ReplySenderTypeMember
//	ReaderSenderID:   memberId
//
// Mall 側で avatar が読む場合:
//
//	ReaderSenderType: inquirydom.ReplySenderTypeAvatar
//	ReaderSenderID:   avatarId
type MarkInquiryAsReadInput struct {
	InquiryID string

	ReaderSenderType inquirydom.ReplySenderType

	ReaderSenderID string
}

// MarkAsRead は Inquiry と配下の replies を既読にします.
//
// replies の既読化では、自分が送信した reply は除外します。
func (uc *InquiryUsecase) MarkAsRead(
	ctx context.Context,
	in MarkInquiryAsReadInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	readerSenderType :=
		in.ReaderSenderType
	readerSenderID :=
		in.ReaderSenderID

	if inquiryID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidID
	}

	if readerSenderType == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidReplySenderType
	}

	if readerSenderID == "" {
		return inquirydom.Inquiry{},
			inquirydom.ErrInvalidReplySenderID
	}

	current, err :=
		uc.repo.GetByID(
			ctx,
			inquiryID,
		)
	if err != nil {
		return inquirydom.Inquiry{},
			err
	}

	now := uc.nowUTC()

	if err := current.MarkAsRead(
		now,
	); err != nil {
		return inquirydom.Inquiry{},
			err
	}

	updated, err :=
		uc.repo.Update(
			ctx,
			current.ID,
			inquirydom.InquiryPatch{
				IsRead: &current.IsRead,

				UpdatedAt: &current.UpdatedAt,
			},
		)
	if err != nil {
		return inquirydom.Inquiry{},
			err
	}

	if uc.replyRepo != nil {
		if err :=
			uc.replyRepo.MarkAsReadByInquiryID(
				ctx,
				inquiryID,
				readerSenderType,
				readerSenderID,
				now,
			); err != nil {
			return inquirydom.Inquiry{},
				err
		}
	}

	return updated, nil
}

// Update は Inquiry を部分更新します.
//
// 画像追加・更新・削除は InquiryPatch.Images に更新後の Images 全体を渡して行います。
func (uc *InquiryUsecase) Update(
	ctx context.Context,
	id string,
	patch inquirydom.InquiryPatch,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{},
			fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.Update(
		ctx,
		id,
		patch,
	)
}

// Delete は Inquiry を削除します。
func (uc *InquiryUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if uc == nil || uc.repo == nil {
		return fmt.Errorf(
			"inquiry usecase: repository is nil",
		)
	}

	return uc.repo.Delete(
		ctx,
		id,
	)
}

func (uc *InquiryUsecase) sendInquiryCreatedMail(
	ctx context.Context,
	inq inquirydom.Inquiry,
) error {
	if uc == nil || uc.mailer == nil {
		return nil
	}

	from := uc.mailFrom
	if from == "" {
		return nil
	}

	to, err := uc.resolveInquiryMailTo(
		ctx,
		inq,
	)
	if err != nil {
		return fmt.Errorf(
			"inquiry usecase: failed to resolve inquiry mail recipient: %w",
			err,
		)
	}

	if to == "" {
		return nil
	}

	if err :=
		uc.mailer.SendInquiryCreatedNotification(
			ctx,
			from,
			to,
			inq,
		); err != nil {
		return fmt.Errorf(
			"inquiry usecase: failed to send inquiry created mail: %w",
			err,
		)
	}

	return nil
}

func (uc *InquiryUsecase) resolveInquiryMailTo(
	ctx context.Context,
	inq inquirydom.Inquiry,
) (string, error) {
	if uc == nil {
		return "", nil
	}

	if uc.avatarEmailResolver == nil ||
		uc.authUserGetter == nil {
		return "", nil
	}

	avatarID := inq.AvatarID
	if avatarID == "" {
		return "", nil
	}

	avatar, err :=
		uc.avatarEmailResolver.GetByID(
			ctx,
			avatarID,
		)
	if err != nil {
		return "", err
	}

	uid := avatar.UserID
	if uid == "" {
		return "", nil
	}

	email, err :=
		uc.authUserGetter.GetEmailByUID(
			ctx,
			uid,
		)
	if err != nil {
		return "", err
	}

	return email, nil
}

func (uc *InquiryUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}

func newInquiryReplyID(
	now time.Time,
) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return fmt.Sprintf(
		"reply_%d",
		now.UTC().UnixNano(),
	)
}
