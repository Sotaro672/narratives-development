// backend/internal/application/usecase/inquiry_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
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

// InquiryAggregate は Inquiry とその画像一覧をまとめたビューです。
//
// inquiryImage ドメインは inquiry ドメインへ統合済みのため、
// Images は Inquiry.Images から取得します。
type InquiryAggregate struct {
	Inquiry inquirydom.Inquiry     `json:"inquiry"`
	Images  []inquirydom.ImageFile `json:"images"`
}

// InquiryUsecase は Inquiry 集約を扱います。
//
// 画像は Inquiry.Images として Inquiry 集約内で管理します。
// Firebase Storage への保存・削除は frontend / application 層の責務とし、
// domain / repository では fileUrl と objectPath のメタデータのみ扱います。
type InquiryUsecase struct {
	repo inquirydom.Repository

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
		mailFrom: strings.TrimSpace(mailFrom),
		mailTo:   strings.TrimSpace(mailTo),
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
	uc.mailFrom = strings.TrimSpace(mailFrom)
	uc.mailTo = strings.TrimSpace(mailTo)
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

// ListByCompanyID は companyID に紐づく Inquiry 一覧を返します。
//
// 現在 Inquiry 自体は companyId を保持しないため、repository 実装側では
// 互換メソッドとして扱います。将来的には companyId -> productIds を解決する
// query service 側へ分離する想定です。
func (uc *InquiryUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
	sort inquirydom.Sort,
	page inquirydom.Page,
) (inquirydom.PageResult[inquirydom.Inquiry], error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.PageResult[inquirydom.Inquiry]{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.ListByCompanyID(ctx, companyID, filter, sort, page)
}

// CountUnreadByCompanyID は companyID に紐づく未読 Inquiry 件数を返します。
//
// 現在 Inquiry 自体は companyId を保持しないため、repository 実装側では
// 互換メソッドとして扱います。将来的には companyId -> productIds を解決する
// query service 側へ分離する想定です。
func (uc *InquiryUsecase) CountUnreadByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
) (int, error) {
	if uc == nil || uc.repo == nil {
		return 0, fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.CountUnreadByCompanyID(ctx, companyID, filter)
}

// GetByID は Inquiry を返します。
func (uc *InquiryUsecase) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	return uc.repo.GetByID(ctx, id)
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

// ResolveInquiryInput は company member が問い合わせを対処済みにする入力です。
type ResolveInquiryInput struct {
	InquiryID string
	MemberID  string
}

// ResolveByMember は company member が Inquiry を resolved にします。
//
// company member は close せず、対処済みとして resolved にします。
// 最終 close は avatar 側で行います。
func (uc *InquiryUsecase) ResolveByMember(
	ctx context.Context,
	in ResolveInquiryInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := strings.TrimSpace(in.InquiryID)
	memberID := strings.TrimSpace(in.MemberID)

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

// ReopenByMember は company member が resolved 状態の Inquiry を open に戻します。
//
// 追加対応が必要になった場合に使います。
func (uc *InquiryUsecase) ReopenByMember(
	ctx context.Context,
	in ReopenInquiryInput,
) (inquirydom.Inquiry, error) {
	if uc == nil || uc.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	inquiryID := strings.TrimSpace(in.InquiryID)
	memberID := strings.TrimSpace(in.MemberID)

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

	clearResolvedAt := time.Time{}
	clearResolvedBy := ""

	return uc.repo.Update(ctx, current.ID, inquirydom.InquiryPatch{
		Status:     &current.Status,
		ResolvedAt: &clearResolvedAt,
		ResolvedBy: &clearResolvedBy,
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

	inquiryID := strings.TrimSpace(in.InquiryID)
	avatarID := strings.TrimSpace(in.AvatarID)

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

// GetImages は Inquiry に紐づく画像一覧を返します。
//
// inquiryImage ドメインは廃止済みのため、別 repository へは問い合わせず、
// Inquiry.Images をそのまま返します。
func (uc *InquiryUsecase) GetImages(ctx context.Context, inquiryID string) ([]inquirydom.ImageFile, error) {
	if uc == nil || uc.repo == nil {
		return nil, fmt.Errorf("inquiry usecase: repository is nil")
	}

	in, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if len(in.Images) == 0 {
		return []inquirydom.ImageFile{}, nil
	}
	return in.Images, nil
}

// GetAggregate は Inquiry と画像一覧をまとめて返します。
//
// 画像は Inquiry.Images を正として扱います。
func (uc *InquiryUsecase) GetAggregate(ctx context.Context, id string) (InquiryAggregate, error) {
	if uc == nil || uc.repo == nil {
		return InquiryAggregate{}, fmt.Errorf("inquiry usecase: repository is nil")
	}

	in, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	images := in.Images
	if images == nil {
		images = []inquirydom.ImageFile{}
	}

	return InquiryAggregate{
		Inquiry: in,
		Images:  images,
	}, nil
}

func (uc *InquiryUsecase) sendInquiryCreatedMail(ctx context.Context, inq inquirydom.Inquiry) error {
	if uc == nil || uc.mailer == nil {
		return nil
	}

	from := strings.TrimSpace(uc.mailFrom)
	if from == "" {
		return nil
	}

	to, err := uc.resolveInquiryMailTo(ctx, inq)
	if err != nil {
		return fmt.Errorf("inquiry usecase: failed to resolve inquiry mail recipient: %w", err)
	}

	to = strings.TrimSpace(to)
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
		avatarID := strings.TrimSpace(inq.AvatarID)
		if avatarID != "" {
			avatar, err := uc.avatarEmailResolver.GetByID(ctx, avatarID)
			if err != nil {
				return "", err
			}

			uid := strings.TrimSpace(avatar.UserID)
			if uid != "" {
				email, err := uc.authUserGetter.GetEmailByUID(ctx, uid)
				if err != nil {
					return "", err
				}

				email = strings.TrimSpace(email)
				if email != "" {
					return email, nil
				}
			}
		}
	}

	return strings.TrimSpace(uc.mailTo), nil
}

func (uc *InquiryUsecase) nowUTC() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}
