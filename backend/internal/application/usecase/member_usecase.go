// backend/internal/application/usecase/member_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// DTOs
// -----------------------------------------------------------------------------

type MemberRecord struct {
	DocID  string
	Member memdom.Member
}

// -----------------------------------------------------------------------------
// Usecase 本体
// -----------------------------------------------------------------------------

type MemberUsecase struct {
	repo              memdom.Repository
	now               func() time.Time
	invitationCommand InvitationCommandPort
}

func NewMemberUsecase(repo memdom.Repository) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func NewMemberUsecaseWithInvitationCommand(
	repo memdom.Repository,
	invitationCommand InvitationCommandPort,
) *MemberUsecase {
	return &MemberUsecase{
		repo:              repo,
		now:               time.Now,
		invitationCommand: invitationCommand,
	}
}

// -----------------------------------------------------------------------------
// Commands
// -----------------------------------------------------------------------------

type CreateMemberInput struct {
	// UID is Firebase Auth UID.
	//
	// 初回会社登録者など、Firebase Auth user がすでに確定している member 作成では必須。
	// 招待前 member 作成では空を許可し、招待承諾時に BindFirebaseUID で後付けする。
	UID string

	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string

	CompanyID string
	Status    string
	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (MemberRecord, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	cid := CompanyIDFromContext(ctx)
	companyID := in.CompanyID
	if cid != "" {
		companyID = cid
	}

	m := memdom.Member{
		UID:            in.UID,
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		FirstNameKana:  in.FirstNameKana,
		LastNameKana:   in.LastNameKana,
		Email:          in.Email,
		Permissions:    dedupStrings(in.Permissions),
		AssignedBrands: dedupStrings(in.AssignedBrands),
		CompanyID:      companyID,
		Status:         in.Status,
		CreatedAt:      *createdAt,
		UpdatedAt:      nil,
	}

	rec, err := u.repo.Create(ctx, m)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

type UpdateMemberInput struct {
	// UID is Firebase Auth UID.
	UID string

	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string

	CompanyID *string
	Status    *string
}

func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (MemberRecord, error) {
	uid := in.UID
	if uid == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	patch := memdom.MemberPatch{
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		FirstNameKana:  in.FirstNameKana,
		LastNameKana:   in.LastNameKana,
		Email:          in.Email,
		Permissions:    in.Permissions,
		AssignedBrands: in.AssignedBrands,
		Status:         in.Status,
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		patch.CompanyID = &cid
	} else if in.CompanyID != nil {
		companyID := *in.CompanyID
		patch.CompanyID = &companyID
	}

	now := u.now().UTC()
	patch.UpdatedAt = &now

	rec, err := u.repo.Update(ctx, uid, patch)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

type BindFirebaseUIDInput struct {
	UID string
}

func (u *MemberUsecase) BindFirebaseUID(ctx context.Context, in BindFirebaseUIDInput) (MemberRecord, error) {
	uid := in.UID
	if uid == "" {
		return MemberRecord{}, fmt.Errorf("firebase uid is empty")
	}

	rec, err := u.repo.GetByUID(ctx, uid)
	if err != nil {
		return MemberRecord{}, err
	}

	if rec.Member.UID != "" && rec.Member.UID != uid {
		return MemberRecord{}, memdom.ErrConflict
	}

	patch := memdom.MemberPatch{
		UID: &uid,
	}

	now := u.now().UTC()
	patch.UpdatedAt = &now

	updated, err := u.repo.Update(ctx, uid, patch)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  updated.DocID,
		Member: updated.Member,
	}, nil
}

func (u *MemberUsecase) Delete(ctx context.Context, uid string) error {
	if uid == "" {
		return memdom.ErrNotFound
	}

	return u.repo.Delete(ctx, uid)
}

// -----------------------------------------------------------------------------
// Invitation (招待メール送信)
// -----------------------------------------------------------------------------

func (u *MemberUsecase) SendInvitation(ctx context.Context, memberID string) error {
	if u.invitationCommand == nil {
		return errors.New("invitation command is not configured")
	}

	if memberID == "" {
		return fmt.Errorf("memberID is empty")
	}

	_, err := u.invitationCommand.CreateInvitationAndSend(ctx, memberID)
	if err != nil {
		return fmt.Errorf("send invitation failed: %w", err)
	}

	return nil
}
