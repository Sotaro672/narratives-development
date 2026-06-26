// backend/internal/application/usecase/inquiry_usecase.go
package usecase

import (
	"context"
	"fmt"
	"time"

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

// InquiryReplyRepository is the minimal repository required for storing
// inquiry replies in Firestore subcollection:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// Reply is intentionally separated from Inquiry.Content.
// Inquiry.Content must remain the first inquiry body only.
type InquiryReplyRepository interface {
	Create(ctx context.Context, reply inquirydom.Reply) (inquirydom.Reply, error)
	ListByInquiryID(ctx context.Context, inquiryID string) ([]inquirydom.Reply, error)

	// MarkAsReadByInquiryID marks replies under the given inquiry as read.
	//
	// Repository implementations must not mark the reader's own replies as read.
	// A reply should be skipped when:
	//
	//	reply.SenderType == readerSenderType && reply.SenderID == readerSenderID
	//
	// Repository implementations should update:
	// - isRead = true
	// - updatedAt = readAt
	MarkAsReadByInquiryID(
		ctx context.Context,
		inquiryID string,
		readerSenderType inquirydom.ReplySenderType,
		readerSenderID string,
		readAt time.Time,
	) error
}

// InquiryUsecase は Inquiry の command を扱います。
//
// 画像は Inquiry.Images として Inquiry 集約内で管理します。
// Firebase Storage への保存・削除は frontend / application 層の責務とし、
// domain / repository では fileUrl と objectPath のメタデータのみ扱います。
//
// 返信は Inquiry.Content へ追記せず、Firestore subcollection:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// に保存します。
type InquiryUsecase struct {
	repo      inquirydom.Repository
	replyRepo InquiryReplyRepository

	mailer   InquiryCreatedMailer
	mailFrom string

	// mailTo は後方互換 / fallback 用です。
	// 原則として問い合わせ作成者 avatar の UserID から Firebase Auth email を解決します。
	mailTo string

	avatarEmailResolver AvatarEmailResolver
	authUserGetter      AuthUserEmailGetter

	now func() time.Time
}

// NewInquiryUsecase はユースケースを初期化します。
func NewInquiryUsecase(repo inquirydom.Repository) *InquiryUsecase {
	return &InquiryUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// NewInquiryUsecaseWithMailer はメール送信ありの InquiryUsecase を初期化します。
//
// mailer が nil、または mailFrom が空の場合、Create 時のメール送信は行いません。
// mailTo は fallback 用です。通常は SetAvatarEmailResolver と SetAuthUserEmailGetter で
// avatar.UserID から送信先 email を解決してください。
func NewInquiryUsecaseWithMailer(
	repo inquirydom.Repository,
	mailer InquiryCreatedMailer,
	mailFrom string,
	mailTo string,
) *InquiryUsecase {
	return &InquiryUsecase{
		repo:     repo,
		mailer:   mailer,
		mailFrom: mailFrom,
		mailTo:   mailTo,
		now:      time.Now,
	}
}

// SetInquiryCreatedMailer は既存の InquiryUsecase にメール送信設定を追加します。
//
// bootstrap / wire 側で既存 constructor を変えにくい場合に使います。
// mailTo は fallback 用です。通常は avatar.UserID から送信先 email を解決します。
func (uc *InquiryUsecase) SetInquiryCreatedMailer(
	mailer InquiryCreatedMailer,
	mailFrom string,
	mailTo string,
) {
	if uc == nil {
		return
	}

	uc.mailer = mailer
	uc.mailFrom = mailFrom
	uc.mailTo = mailTo
}

// SetReplyRepository は Inquiry reply subcollection 保存用 repository を設定します。
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
func (uc *InquiryUsecase) SetReplyRepository(repo InquiryReplyRepository) {
	if uc == nil {
		return
	}

	uc.replyRepo = repo
}

// SetAvatarEmailResolver は Inquiry.AvatarID から avatar を取得する resolver を設定します。
func (uc *InquiryUsecase) SetAvatarEmailResolver(resolver AvatarEmailResolver) {
	if uc == nil {
		return
	}

	uc.avatarEmailResolver = resolver
}

// SetAuthUserEmailGetter は Firebase Auth UID から email を取得する getter を設定します。
func (uc *InquiryUsecase) SetAuthUserEmailGetter(getter AuthUserEmailGetter) {
	if uc == nil {
		return
	}

	uc.authUserGetter = getter
}

// SetNowFunc はテスト用に現在時刻関数を差し替えます。
func (uc *InquiryUsecase) SetNowFunc(now func() time.Time) {
	if uc == nil || now == nil {
		return
	}

	uc.now = now
}

// Create は Inquiry を作成します。
//
// 作成後、メール設定がある場合は問い合わせ作成通知メールを送信します。
// メール送信に失敗した場合、Inquiry 作成自体は完了済みのため、作成済み Inquiry と error を返します。
func (uc *InquiryUsecase) Create(ctx context.Context, inq inquirydom.Inquiry) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	created, err := uc.repo.Create(ctx, inq)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	if err := uc.sendInquiryCreatedMail(ctx, created); err != nil {
		return created, err
	}

	return created, nil
}

// CreateInquiryReplyInput は company member / avatar が問い合わせへ返信する入力です。
//
// Console からの返信では SenderType=member, SenderID=memberId を使います。
// Mall / SNS から avatar が返信する場合は SenderType=avatar, SenderID=avatarId を使います。
type CreateInquiryReplyInput struct {
	InquiryID  string
	SenderType inquirydom.ReplySenderType
	SenderID   string
	Content    string
	Images     []inquirydom.ImageFile
}

// CreateReply は Inquiry の reply subcollection に返信を作成します。
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// Inquiry.Content へ返信本文を追記しません。
// Inquiry 本体は updatedAt / updatedBy のみ更新します。
func (uc *InquiryUsecase) CreateReply(
	ctx context.Context,
	in CreateInquiryReplyInput,
) (inquirydom.Reply, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Reply{}, fmt.Errorf("inquiry usecase: repository is nil")
	}
	if uc.replyRepo == nil {
		return inquirydom.Reply{}, fmt.Errorf("inquiry usecase: reply repository is nil")
	}

	inquiryID := in.InquiryID
	senderID := in.SenderID

	if inquiryID == "" {
		return inquirydom.Reply{}, inquirydom.ErrInvalidReplyInquiryID
	}
	if senderID == "" {
		return inquirydom.Reply{}, inquirydom.ErrInvalidReplySenderID
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	if current.Status == inquirydom.InquiryStatusClosed {
		return inquirydom.Reply{}, inquirydom.ErrInquiryAlreadyClosed
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

	created, err := uc.replyRepo.Create(ctx, reply)
	if err != nil {
		return inquirydom.Reply{}, err
	}

	updatedBy := senderID
	if _, err := uc.repo.Update(ctx, inquiryID, inquirydom.InquiryPatch{
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	}); err != nil {
		return inquirydom.Reply{}, err
	}

	return created, nil
}

// CreateReplyByMember は company member が問い合わせへ返信します。
//
// Console 用の shorthand です。
func (uc *InquiryUsecase) CreateReplyByMember(
	ctx context.Context,
	inquiryID string,
	memberID string,
	content string,
	images []inquirydom.ImageFile,
) (inquirydom.Reply, error) {
	return uc.CreateReply(ctx, CreateInquiryReplyInput{
		InquiryID:  inquiryID,
		SenderType: inquirydom.ReplySenderTypeMember,
		SenderID:   memberID,
		Content:    content,
		Images:     images,
	})
}

// CreateReplyByAvatar は avatar が問い合わせへ返信します。
//
// Mall / SNS 用の shorthand です。
func (uc *InquiryUsecase) CreateReplyByAvatar(
	ctx context.Context,
	inquiryID string,
	avatarID string,
	content string,
	images []inquirydom.ImageFile,
) (inquirydom.Reply, error) {
	return uc.CreateReply(ctx, CreateInquiryReplyInput{
		InquiryID:  inquiryID,
		SenderType: inquirydom.ReplySenderTypeAvatar,
		SenderID:   avatarID,
		Content:    content,
		Images:     images,
	})
}

// ListReplies は Inquiry の reply subcollection を取得します。
//
// 保存先:
//
//	inquiries/{inquiryId}/replies/{replyId}
func (uc *InquiryUsecase) ListReplies(
	ctx context.Context,
	inquiryID string,
) ([]inquirydom.Reply, error) {
	if uc == nil || uc.replyRepo == nil {
		return nil, fmt.Errorf("inquiry usecase: reply repository is nil")
	}
	if inquiryID == "" {
		return nil, inquirydom.ErrInvalidReplyInquiryID
	}

	return uc.replyRepo.ListByInquiryID(ctx, inquiryID)
}

// CountUnreadInquiriesForMemberInput は company member 向けの未読件数集計入力です。
//
// CountUnreadByCompanyIDForMember は以下を合算します。
// - Inquiry 本体の isRead=false
// - Inquiry 配下 replies の isRead=false
//
// ただし、memberId 自身が送信した reply は未読件数に含めません。
type CountUnreadInquiriesForMemberInput struct {
	CompanyID string
	MemberID  string
	Filter    inquirydom.Filter
}

// CountUnreadInquiriesForAvatarInput は avatar 向けの未読件数集計入力です。
//
// CountUnreadByCompanyIDForAvatar は以下を合算します。
// - avatarId が受け取る対象 Inquiry 配下 replies の isRead=false
//
// ただし、avatarId 自身が送信した reply は未読件数に含めません。
// また、avatar 自身が起票した Inquiry 本体は未読件数に含めません。
type CountUnreadInquiriesForAvatarInput struct {
	CompanyID string
	AvatarID  string
	Filter    inquirydom.Filter
}

// CountUnreadByCompanyIDForMember は Inquiry 本体と replies を対象に未読件数を返します。
//
// reply の count 条件:
//
//	!reply.IsRead
//	&& !(reply.SenderType == member && reply.SenderID == memberId)
//
// NOTE:
// company scope / filter に一致する Inquiry のみを対象に replies を集計します。
func (uc *InquiryUsecase) CountUnreadByCompanyIDForMember(
	ctx context.Context,
	in CountUnreadInquiriesForMemberInput,
) (int, error) {
	if uc == nil || uc.repo == nil {
		return 0, fmt.Errorf("inquiry usecase: repository is nil")
	}

	companyID := in.CompanyID
	memberID := in.MemberID

	if companyID == "" {
		return 0, inquirydom.ErrNotFound
	}
	if memberID == "" {
		return 0, inquirydom.ErrInvalidReplySenderID
	}

	total := 0
	pageNumber := 1
	perPage := 200

	for {
		result, err := uc.repo.ListByCompanyID(
			ctx,
			companyID,
			in.Filter,
			inquirydom.Sort{},
			inquirydom.Page{
				Number:  pageNumber,
				PerPage: perPage,
			},
		)
		if err != nil {
			return 0, err
		}

		for _, inquiry := range result.Items {
			if !inquiry.IsRead {
				total++
			}

			replyUnreadCount, err := uc.countUnreadRepliesExcludingSender(
				ctx,
				inquiry.ID,
				inquirydom.ReplySenderTypeMember,
				memberID,
			)
			if err != nil {
				return 0, err
			}

			total += replyUnreadCount
		}

		if result.TotalPages <= 0 || pageNumber >= result.TotalPages {
			break
		}

		pageNumber++
	}

	return total, nil
}

// CountUnreadByCompanyIDForAvatar は avatar 向けに replies を対象に未読件数を返します。
//
// avatar 側では、avatar 自身が起票した Inquiry 本体を未読件数には含めません。
// reply の count 条件:
//
//	!reply.IsRead
//	&& !(reply.SenderType == avatar && reply.SenderID == avatarId)
//
// NOTE:
// company scope / filter に一致し、かつ avatarId に紐づく Inquiry のみを対象に replies を集計します。
func (uc *InquiryUsecase) CountUnreadByCompanyIDForAvatar(
	ctx context.Context,
	in CountUnreadInquiriesForAvatarInput,
) (int, error) {
	if uc == nil || uc.repo == nil {
		return 0, fmt.Errorf("inquiry usecase: repository is nil")
	}

	companyID := in.CompanyID
	avatarID := in.AvatarID

	if companyID == "" {
		return 0, inquirydom.ErrNotFound
	}
	if avatarID == "" {
		return 0, inquirydom.ErrInvalidAvatarID
	}

	filter := in.Filter
	filter.AvatarID = &avatarID

	total := 0
	pageNumber := 1
	perPage := 200

	for {
		result, err := uc.repo.ListByCompanyID(
			ctx,
			companyID,
			filter,
			inquirydom.Sort{},
			inquirydom.Page{
				Number:  pageNumber,
				PerPage: perPage,
			},
		)
		if err != nil {
			return 0, err
		}

		for _, inquiry := range result.Items {
			replyUnreadCount, err := uc.countUnreadRepliesExcludingSender(
				ctx,
				inquiry.ID,
				inquirydom.ReplySenderTypeAvatar,
				avatarID,
			)
			if err != nil {
				return 0, err
			}

			total += replyUnreadCount
		}

		if result.TotalPages <= 0 || pageNumber >= result.TotalPages {
			break
		}

		pageNumber++
	}

	return total, nil
}

// CountUnreadByCompanyID は後方互換用の shorthand です。
//
// memberId / avatarId を指定しない既存呼び出しでは、自分の reply を除外できないため、
// reply は集計せず Inquiry 本体の未読数のみ返します。
func (uc *InquiryUsecase) CountUnreadByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
) (int, error) {
	if uc == nil || uc.repo == nil {
		return 0, fmt.Errorf("inquiry usecase: repository is nil")
	}
	if companyID == "" {
		return 0, inquirydom.ErrNotFound
	}

	return uc.repo.CountUnreadByCompanyID(ctx, companyID, filter)
}

func (uc *InquiryUsecase) countUnreadRepliesExcludingSender(
	ctx context.Context,
	inquiryID string,
	excludedSenderType inquirydom.ReplySenderType,
	excludedSenderID string,
) (int, error) {
	if uc == nil || uc.replyRepo == nil {
		return 0, nil
	}
	if inquiryID == "" {
		return 0, inquirydom.ErrInvalidReplyInquiryID
	}
	if excludedSenderType == "" {
		return 0, inquirydom.ErrInvalidReplySenderType
	}
	if excludedSenderID == "" {
		return 0, inquirydom.ErrInvalidReplySenderID
	}

	replies, err := uc.replyRepo.ListByInquiryID(ctx, inquiryID)
	if err != nil {
		return 0, err
	}

	count := 0

	for _, reply := range replies {
		if reply.IsRead {
			continue
		}

		if reply.SenderType == excludedSenderType && reply.SenderID == excludedSenderID {
			continue
		}

		count++
	}

	return count, nil
}

// ResolveInquiryInput は company member が問い合わせを対処済みにする入力です。
type ResolveInquiryInput struct {
	InquiryID string
	MemberID  string
}

// ResolveByMember は company member が Inquiry を resolved にします。
//
// company member は close せず、対処済みとして resolved にします。
func (uc *InquiryUsecase) ResolveByMember(
	ctx context.Context,
	in ResolveInquiryInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	memberID := in.MemberID

	if inquiryID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	if memberID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidResolvedBy
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	now := uc.nowUTC()
	if err := current.ResolveByMember(memberID, now); err != nil {
		return inquirydom.Inquiry{}, err
	}

	return uc.repo.Update(ctx, current.ID, inquirydom.InquiryPatch{
		Status:     &current.Status,
		ResolvedAt: current.ResolvedAt,
		ResolvedBy: current.ResolvedBy,
		UpdatedAt:  &current.UpdatedAt,
		UpdatedBy:  current.UpdatedBy,
	})
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
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	memberID := in.MemberID

	if inquiryID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	if memberID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidUpdatedBy
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	now := uc.nowUTC()
	if err := current.ReopenByMember(memberID, now); err != nil {
		return inquirydom.Inquiry{}, err
	}

	return uc.repo.Update(ctx, current.ID, inquirydom.InquiryPatch{
		Status:     &current.Status,
		ResolvedAt: current.ResolvedAt,
		ResolvedBy: current.ResolvedBy,
		ClosedAt:   current.ClosedAt,
		ClosedBy:   current.ClosedBy,
		UpdatedAt:  &current.UpdatedAt,
		UpdatedBy:  current.UpdatedBy,
	})
}

// CloseInquiryByAvatarInput は avatar が問い合わせを close する入力です。
type CloseInquiryByAvatarInput struct {
	InquiryID string
	AvatarID  string
}

// CloseByAvatar は avatar が Inquiry を closed にします。
//
// Inquiry を起票した avatar のみ close できます。
func (uc *InquiryUsecase) CloseByAvatar(
	ctx context.Context,
	in CloseInquiryByAvatarInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	avatarID := in.AvatarID

	if inquiryID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	if avatarID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidAvatarID
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	now := uc.nowUTC()
	if err := current.CloseByAvatar(avatarID, now); err != nil {
		return inquirydom.Inquiry{}, err
	}

	return uc.repo.Update(ctx, current.ID, inquirydom.InquiryPatch{
		Status:    &current.Status,
		ClosedAt:  current.ClosedAt,
		ClosedBy:  current.ClosedBy,
		UpdatedAt: &current.UpdatedAt,
		UpdatedBy: current.UpdatedBy,
	})
}

// MarkInquiryAsReadInput は Inquiry を既読にする入力です.
//
// ReaderSenderType / ReaderSenderID は reply の既読化で自分の reply を除外するために使います。
//
// Console 側で company member が読む場合:
//
//	ReaderSenderType: inquirydom.ReplySenderTypeMember
//	ReaderSenderID:   memberId
//
// Mall / SNS 側で avatar が読む場合:
//
//	ReaderSenderType: inquirydom.ReplySenderTypeAvatar
//	ReaderSenderID:   avatarId
type MarkInquiryAsReadInput struct {
	InquiryID        string
	ReaderSenderType inquirydom.ReplySenderType
	ReaderSenderID   string
}

// MarkAsRead は Inquiry と配下の replies を既読にします。
//
// replies の既読化では、自分が送信した reply は除外します。
func (uc *InquiryUsecase) MarkAsRead(
	ctx context.Context,
	in MarkInquiryAsReadInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := in.InquiryID
	readerSenderType := in.ReaderSenderType
	readerSenderID := in.ReaderSenderID

	if inquiryID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	if readerSenderType == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidReplySenderType
	}
	if readerSenderID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidReplySenderID
	}

	current, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	now := uc.nowUTC()
	if err := current.MarkAsRead(now); err != nil {
		return inquirydom.Inquiry{}, err
	}

	updated, err := uc.repo.Update(ctx, current.ID, inquirydom.InquiryPatch{
		IsRead:    &current.IsRead,
		UpdatedAt: &current.UpdatedAt,
	})
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	if uc.replyRepo != nil {
		if err := uc.replyRepo.MarkAsReadByInquiryID(
			ctx,
			inquiryID,
			readerSenderType,
			readerSenderID,
			now,
		); err != nil {
			return inquirydom.Inquiry{}, err
		}
	}

	return updated, nil
}

// Update は Inquiry を部分更新します。
//
// 画像追加・更新・削除は InquiryPatch.Images に更新後の Images 全体を渡して行います。
func (uc *InquiryUsecase) Update(
	ctx context.Context,
	id string,
	patch inquirydom.InquiryPatch,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.Update(ctx, id, patch)
}

// Delete は Inquiry を削除します。
func (uc *InquiryUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.repo == nil {
		return fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.Delete(ctx, id)
}

func (uc *InquiryUsecase) sendInquiryCreatedMail(ctx context.Context, inq inquirydom.Inquiry) error {
	if uc == nil || uc.mailer == nil {
		return nil
	}

	from := uc.mailFrom
	if from == "" {
		return nil
	}

	to, err := uc.resolveInquiryMailTo(ctx, inq)
	if err != nil {
		return fmt.Errorf("inquiry usecase: failed to resolve inquiry mail recipient: %w", err)
	}

	if to == "" {
		return nil
	}

	if err := uc.mailer.SendInquiryCreatedNotification(ctx, from, to, inq); err != nil {
		return fmt.Errorf("inquiry usecase: failed to send inquiry created mail: %w", err)
	}

	return nil
}

func (uc *InquiryUsecase) resolveInquiryMailTo(ctx context.Context, inq inquirydom.Inquiry) (string, error) {
	if uc == nil {
		return "", nil
	}

	if uc.avatarEmailResolver != nil && uc.authUserGetter != nil {
		avatarID := inq.AvatarID
		if avatarID != "" {
			avatar, err := uc.avatarEmailResolver.GetByID(ctx, avatarID)
			if err != nil {
				return "", err
			}

			uid := avatar.UserID
			if uid != "" {
				email, err := uc.authUserGetter.GetEmailByUID(ctx, uid)
				if err != nil {
					return "", err
				}

				if email != "" {
					return email, nil
				}
			}
		}
	}

	return uc.mailTo, nil
}

func (uc *InquiryUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}

func newInquiryReplyID(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return fmt.Sprintf("reply_%d", now.UTC().UnixNano())
}
