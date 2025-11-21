package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
)

// ─────────────────────────────────────────────────────────────
// Ports (依存性逆転: usecase が利用する外部サービスのインターフェース)
// ─────────────────────────────────────────────────────────────

// InvitationMailer は招待メール送信用のポートです。
// adapters/out/mail.InvitationMailerPort と同じシグネチャにしておけば、
// Go の構造的型付けによりそのまま差し込めます（import せずに済む）。
type InvitationMailer interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// ─────────────────────────────────────────────────────────────
// Usecase
// ─────────────────────────────────────────────────────────────

type MemberUsecase struct {
	repo             memdom.Repository
	now              func() time.Time
	invitationMailer InvitationMailer // ★ 招待メール用ポート
}

// NewMemberUsecase は招待メールなしの最小構成コンストラクタ。
// 既存の呼び出しコードとの互換性維持用。
func NewMemberUsecase(repo memdom.Repository) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// NewMemberUsecaseWithMailer は招待メール機能付きのコンストラクタ。
// adapters 側で SendGrid 付き InvitationMailer を生成して渡す想定。
func NewMemberUsecaseWithMailer(repo memdom.Repository, mailer InvitationMailer) *MemberUsecase {
	return &MemberUsecase{
		repo:             repo,
		now:              time.Now,
		invitationMailer: mailer,
	}
}

// ─────────────────────────────────────────────────────────────
// Auth/Multitenancy: companyId の取得（ミドルウェアで注入された値を拾う）
//
// 依存方向を守るために adapters に依存せず、context から汎用キーで取得します。
// ミドルウェア側では以下いずれかのキーで string を詰めてください。
//   - "companyId"
//   - "auth.companyId"
//
// 見つからない場合は空文字を返します（＝強制上書き不可）。
// ─────────────────────────────────────────────────────────────
func companyIDFromContext(ctx context.Context) string {
	// 代表キー
	if v := ctx.Value("companyId"); v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	// 互換キー
	if v := ctx.Value("auth.companyId"); v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// ─────────────────────────────────────────────────────────────
// Queries
// ─────────────────────────────────────────────────────────────

func (u *MemberUsecase) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) GetByEmail(ctx context.Context, email string) (memdom.Member, error) {
	return u.repo.GetByEmail(ctx, strings.TrimSpace(email))
}

func (u *MemberUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) Count(ctx context.Context, f memdom.Filter) (int, error) {
	// ★ クライアント指定は無視して、サーバが信頼する companyId を強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}
	return u.repo.Count(ctx, f)
}

// ★戻り値型を common.PageResult[memdom.Member] に統一
func (u *MemberUsecase) List(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	// ★ クライアント指定は無視して、サーバが信頼する companyId を強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}
	return u.repo.List(ctx, f, s, p)
}

// ─────────────────────────────────────────────────────────────
// Commands
// ─────────────────────────────────────────────────────────────

type CreateMemberInput struct {
	ID             string
	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string

	// ★ 新規追加
	CompanyID string // 所属会社ID（任意だが、サーバで強制上書き）
	Status    string // "active" | "inactive"（任意、空なら未指定）

	// CreatedAt を指定しない場合は現在時刻
	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (memdom.Member, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	// ★ クライアント指定は無視して、サーバが信頼する companyId を強制適用
	cid := companyIDFromContext(ctx)
	companyID := strings.TrimSpace(in.CompanyID)
	if cid != "" {
		companyID = cid
	}

	m := memdom.Member{
		ID:             strings.TrimSpace(in.ID),
		FirstName:      strings.TrimSpace(in.FirstName),
		LastName:       strings.TrimSpace(in.LastName),
		FirstNameKana:  strings.TrimSpace(in.FirstNameKana),
		LastNameKana:   strings.TrimSpace(in.LastNameKana),
		Email:          strings.TrimSpace(in.Email),
		Permissions:    dedupStrings(in.Permissions),
		AssignedBrands: dedupStrings(in.AssignedBrands),

		// ★ 強制反映
		CompanyID: companyID,
		Status:    strings.TrimSpace(in.Status),

		CreatedAt: *createdAt,
		UpdatedAt: nil,
	}
	return u.repo.Create(ctx, m)
}

type UpdateMemberInput struct {
	ID             string
	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string

	// ★ 新規追加
	CompanyID *string // クライアント指定は無視される（サーバ強制）
	Status    *string
}

// Update は現在の Member を読み出して上書きし、repo.Save() に投げる。
// UpdatedAt は repo.Save/upsert 側で NOW() に更新される前提。
func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (memdom.Member, error) {
	current, err := u.repo.GetByID(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return memdom.Member{}, err
	}

	if in.FirstName != nil {
		current.FirstName = strings.TrimSpace(*in.FirstName)
	}
	if in.LastName != nil {
		current.LastName = strings.TrimSpace(*in.LastName)
	}
	if in.FirstNameKana != nil {
		current.FirstNameKana = strings.TrimSpace(*in.FirstNameKana)
	}
	if in.LastNameKana != nil {
		current.LastNameKana = strings.TrimSpace(*in.LastNameKana)
	}
	if in.Email != nil {
		current.Email = strings.TrimSpace(*in.Email)
	}
	if in.Permissions != nil {
		current.Permissions = dedupStrings(*in.Permissions)
	}
	if in.AssignedBrands != nil {
		current.AssignedBrands = dedupStrings(*in.AssignedBrands)
	}

	// ★ companyId はクライアント指定ではなくサーバが強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		current.CompanyID = cid
	}
	// ステータスは任意更新（空指定は無視）
	if in.Status != nil {
		current.Status = strings.TrimSpace(*in.Status)
	}

	return u.repo.Save(ctx, current, nil)
}

func (u *MemberUsecase) Save(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	// Save は Upsert。CreatedAt がゼロなら現在時刻を付与。
	if m.CreatedAt.IsZero() {
		m.CreatedAt = u.now().UTC()
	}
	// ★ Save 経由でも companyId をサーバ値で強制（セーフティネット）
	if cid := companyIDFromContext(ctx); cid != "" {
		m.CompanyID = cid
	}
	return u.repo.Save(ctx, m, nil)
}

func (u *MemberUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}

// ─────────────────────────────────────────────────────────────
// Invitation (招待メール送信)
// ─────────────────────────────────────────────────────────────

// SendInvitation は、指定した memberID のメンバーに招待メールを送信します。
// - メンバーを取得
// - 招待トークンを生成（暫定実装）
// - InvitationInfo を組み立てて InvitationMailer に委譲
func (u *MemberUsecase) SendInvitation(ctx context.Context, memberID string) error {
	if u.invitationMailer == nil {
		return errors.New("invitation mailer is not configured")
	}

	m, err := u.repo.GetByID(ctx, strings.TrimSpace(memberID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(m.Email) == "" {
		return fmt.Errorf("member %s has no email", m.ID)
	}

	token, err := generateInvitationToken()
	if err != nil {
		return fmt.Errorf("failed to generate invitation token: %w", err)
	}

	info := memdom.InvitationInfo{
		MemberID:         m.ID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: append([]string(nil), m.AssignedBrands...),
		Permissions:      append([]string(nil), m.Permissions...),
	}

	return u.invitationMailer.SendInvitationEmail(ctx, m.Email, token, info)
}

// generateInvitationToken は暫定的な招待トークン生成。
// 将来的に専用のドメインサービスや永続化と組み合わせる前提。
func generateInvitationToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// シンプルに 32 桁 hex に "INV_" プレフィックスを付与
	return fmt.Sprintf("INV_%x", buf), nil
}
